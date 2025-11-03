package services

import (
	"aistack/internal/logging"
	"fmt"
)

// Manager coordinates all services
type Manager struct {
	runtime    Runtime
	logger     *logging.Logger
	composeDir string
	services   map[string]Service
}

// NewManager creates a new service manager
func NewManager(composeDir string, logger *logging.Logger) (*Manager, error) {
	// Detect container runtime
	runtime, err := DetectRuntime()
	if err != nil {
		return nil, fmt.Errorf("failed to detect container runtime: %w", err)
	}

	manager := &Manager{
		runtime:    runtime,
		logger:     logger,
		composeDir: composeDir,
		services:   make(map[string]Service),
	}

	// Register services
	manager.services["ollama"] = NewOllamaService(composeDir, runtime, logger)
	manager.services["openwebui"] = NewOpenWebUIService(composeDir, runtime, logger)
	manager.services["localai"] = NewLocalAIService(composeDir, runtime, logger)

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
