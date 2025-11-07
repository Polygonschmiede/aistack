package services

import (
	"fmt"
	"os"

	"aistack/internal/config"
	"aistack/internal/gpulock"
	"aistack/internal/logging"
)

// Manager coordinates all services
type Manager struct {
	runtime    Runtime
	logger     *logging.Logger
	composeDir string
	services   map[string]Service
	imageLock  *VersionLock
	gpuLock    *gpulock.Manager
}

// NewManager creates a new service manager
func NewManager(composeDir string, logger *logging.Logger) (*Manager, error) {
	// Detect container runtime
	runtime, err := DetectRuntime()
	if err != nil {
		return nil, fmt.Errorf("failed to detect container runtime: %w", err)
	}

	lock, err := loadVersionLock()
	if err != nil {
		return nil, err
	}

	// Get state directory from env or use default
	stateDir := os.Getenv("AISTACK_STATE_DIR")
	if stateDir == "" {
		stateDir = defaultStateDir
	}

	// Create GPU lock manager
	gpuLockManager := gpulock.NewManager(stateDir, logger)

	manager := &Manager{
		runtime:    runtime,
		logger:     logger,
		composeDir: composeDir,
		services:   make(map[string]Service),
		imageLock:  lock,
		gpuLock:    gpuLockManager,
	}

	// Register services
	manager.services["ollama"] = NewOllamaService(composeDir, runtime, logger, lock)
	manager.services["openwebui"] = NewOpenWebUIService(composeDir, runtime, logger, lock, gpuLockManager)
	manager.services["localai"] = NewLocalAIService(composeDir, runtime, logger, lock, gpuLockManager)

	return manager, nil
}

// GetService returns a service by name
func (m *Manager) GetService(name string) (Service, error) {
	service, exists := m.services[name]
	if !exists {
		return nil, fmt.Errorf("unknown service: %s", name)
	}
	return service, nil
}

// ListServices returns all available service names
func (m *Manager) ListServices() []string {
	names := make([]string, 0, len(m.services))
	for name := range m.services {
		names = append(names, name)
	}
	return names
}

// InstallProfile installs services based on a profile
func (m *Manager) InstallProfile(profile string) error {
	m.logger.Info("profile.install", "Installing profile", map[string]interface{}{
		"profile": profile,
	})

	var servicesToInstall []string

	switch profile {
	case "standard-gpu":
		servicesToInstall = []string{"ollama", "openwebui", "localai"}
	case "minimal":
		servicesToInstall = []string{"ollama"}
	default:
		return fmt.Errorf("unknown profile: %s", profile)
	}

	for _, serviceName := range servicesToInstall {
		service, err := m.GetService(serviceName)
		if err != nil {
			return err
		}

		if err := service.Install(); err != nil {
			return fmt.Errorf("failed to install %s: %w", serviceName, err)
		}
	}

	m.logger.Info("profile.installed", "Profile installed successfully", map[string]interface{}{
		"profile":  profile,
		"services": servicesToInstall,
	})

	return nil
}

// StatusAll returns status of all services
func (m *Manager) StatusAll() ([]ServiceStatus, error) {
	statuses := make([]ServiceStatus, 0, len(m.services))

	for _, service := range m.services {
		status, err := service.Status()
		if err != nil {
			m.logger.Warn("service.status.error", "Failed to get service status", map[string]interface{}{
				"service": service.Name(),
				"error":   err.Error(),
			})
			continue
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// UpdateAllResult represents the result of updating all services
// Story T-029: Container-Update "all" mit Health-Gate
type UpdateAllResult struct {
	TotalServices   int                     `json:"total_services"`
	SuccessfulCount int                     `json:"successful_count"`
	FailedCount     int                     `json:"failed_count"`
	RolledBackCount int                     `json:"rolled_back_count"`
	UnchangedCount  int                     `json:"unchanged_count"`
	ServiceResults  map[string]UpdateResult `json:"service_results"`
}

// UpdateResult represents the result of a single service update
type UpdateResult struct {
	Success      bool   `json:"success"`
	Changed      bool   `json:"changed"`
	RolledBack   bool   `json:"rolled_back"`
	Health       string `json:"health"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// UpdateAllServices updates all services sequentially with health-gating
// Order: LocalAI → Ollama → Open WebUI (as specified in T-029)
// Story T-029: Each service is updated independently; failure in one does not affect others
// Story T-035: Enforces update policy (pinned vs rolling mode)
func (m *Manager) UpdateAllServices() (*UpdateAllResult, error) {
	// Check update policy before proceeding
	if err := m.checkUpdatePolicy(); err != nil {
		return nil, err
	}

	m.logger.Info("services.update_all.start", "Starting sequential update of all services", nil)

	result := &UpdateAllResult{
		TotalServices:  3,
		ServiceResults: make(map[string]UpdateResult),
	}

	// Define update order as per T-029: LocalAI → Ollama → Open WebUI
	updateOrder := []string{"localai", "ollama", "openwebui"}

	for _, serviceName := range updateOrder {
		service, err := m.GetService(serviceName)
		if err != nil {
			m.logger.Error("services.update_all.service_not_found", "Service not found", map[string]interface{}{
				"service": serviceName,
				"error":   err.Error(),
			})
			result.ServiceResults[serviceName] = UpdateResult{
				Success:      false,
				ErrorMessage: err.Error(),
			}
			result.FailedCount++
			continue
		}

		m.logger.Info("services.update_all.updating", "Updating service", map[string]interface{}{
			"service": serviceName,
		})

		// Update service
		updateErr := service.Update()

		// Get status after update
		status, statusErr := service.Status()
		health := "unknown"
		if statusErr == nil {
			health = string(status.Health)
		}

		// Determine result
		serviceResult := UpdateResult{
			Health: health,
		}

		if updateErr != nil {
			// Check if it was a rollback
			if updateErr.Error() == "update failed health check, rolled back to previous version" {
				serviceResult.RolledBack = true
				serviceResult.Success = false
				serviceResult.ErrorMessage = updateErr.Error()
				result.RolledBackCount++
				m.logger.Warn("services.update_all.rolled_back", "Service update failed and rolled back", map[string]interface{}{
					"service": serviceName,
					"error":   updateErr.Error(),
				})
			} else {
				serviceResult.Success = false
				serviceResult.ErrorMessage = updateErr.Error()
				result.FailedCount++
				m.logger.Error("services.update_all.failed", "Service update failed", map[string]interface{}{
					"service": serviceName,
					"error":   updateErr.Error(),
				})
			}
		} else {
			// Check if image actually changed by looking at the update plan
			stateDir := os.Getenv("AISTACK_STATE_DIR")
			if stateDir == "" {
				stateDir = defaultStateDir
			}

			plan, loadErr := LoadUpdatePlan(serviceName, stateDir)
			_ = loadErr // Error can be safely ignored, we just check if plan exists
			if plan != nil && plan.HealthAfterSwap == healthStatusUnchanged {
				serviceResult.Success = true
				serviceResult.Changed = false
				result.UnchangedCount++
				m.logger.Info("services.update_all.unchanged", "Service image unchanged", map[string]interface{}{
					"service": serviceName,
				})
			} else {
				serviceResult.Success = true
				serviceResult.Changed = true
				result.SuccessfulCount++
				m.logger.Info("services.update_all.success", "Service updated successfully", map[string]interface{}{
					"service": serviceName,
					"health":  health,
				})
			}
		}

		result.ServiceResults[serviceName] = serviceResult
	}

	m.logger.Info("services.update_all.complete", "All services update process completed", map[string]interface{}{
		"total":       result.TotalServices,
		"successful":  result.SuccessfulCount,
		"failed":      result.FailedCount,
		"rolled_back": result.RolledBackCount,
		"unchanged":   result.UnchangedCount,
	})

	return result, nil
}

// checkUpdatePolicy checks if updates are allowed based on configuration
// Returns error if updates.mode is "pinned" and updates are blocked
// Story T-035: Enforce update policy based on configuration
func (m *Manager) checkUpdatePolicy() error {
	cfg, err := config.Load()
	if err != nil {
		// If config can't be loaded, allow updates (fail open for backwards compatibility)
		m.logger.Warn("update.policy.check.failed", "Failed to load config, allowing updates", map[string]interface{}{
			"error": err.Error(),
		})
		return nil
	}

	// Check if updates are pinned
	if cfg.Updates.Mode == "pinned" {
		m.logger.Info("update.policy.blocked", "Updates blocked by policy", map[string]interface{}{
			"mode": cfg.Updates.Mode,
		})
		return fmt.Errorf("updates are disabled: updates.mode is set to 'pinned' (change to 'rolling' in config to allow updates)")
	}

	m.logger.Debug("update.policy.allowed", "Updates allowed by policy", map[string]interface{}{
		"mode": cfg.Updates.Mode,
	})
	return nil
}
