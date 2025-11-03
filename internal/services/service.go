package services

import (
	"aistack/internal/logging"
	"fmt"
	"path/filepath"
	"strings"
)

// Service represents a container service
type Service interface {
	Name() string
	Install() error
	Start() error
	Stop() error
	Status() (ServiceStatus, error)
	Health() (HealthStatus, error)
	Remove(keepData bool) error
	Update() error
	Logs(tail int) (string, error)
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Name    string       `json:"name"`
	State   string       `json:"state"`  // running, stopped, unknown
	Health  HealthStatus `json:"health"` // green, yellow, red
	Message string       `json:"message"`
}

// BaseService provides common service functionality
type BaseService struct {
	name        string
	composeFile string
	healthCheck HealthChecker
	volumes     []string
	runtime     Runtime
	logger      *logging.Logger
	netManager  *NetworkManager
}

// NewBaseService creates a new base service
func NewBaseService(name, composeDir string, healthCheck HealthChecker, volumes []string, runtime Runtime, logger *logging.Logger) *BaseService {
	return &BaseService{
		name:        name,
		composeFile: filepath.Join(composeDir, name+".yaml"),
		healthCheck: healthCheck,
		volumes:     volumes,
		runtime:     runtime,
		logger:      logger,
		netManager:  NewNetworkManager(runtime, logger),
	}
}

// Name returns the service name
func (s *BaseService) Name() string {
	return s.name
}

// Install installs the service (ensures network, volumes, and starts)
func (s *BaseService) Install() error {
	s.logger.Info("service.install.start", fmt.Sprintf("Installing %s service", s.name), map[string]interface{}{
		"service": s.name,
	})

	// Ensure network exists
	if err := s.netManager.EnsureNetwork(); err != nil {
		return fmt.Errorf("failed to ensure network: %w", err)
	}

	// Ensure volumes exist
	if err := s.netManager.EnsureVolumes(s.volumes); err != nil {
		return fmt.Errorf("failed to ensure volumes: %w", err)
	}

	// Start the service
	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	s.logger.Info("service.install.complete", fmt.Sprintf("%s service installed successfully", s.name), map[string]interface{}{
		"service": s.name,
	})

	return nil
}

// Start starts the service using docker compose
func (s *BaseService) Start() error {
	return s.runComposeAction("start", func(composeFile string) error {
		return s.runtime.ComposeUp(composeFile)
	})
}

// Stop stops the service
func (s *BaseService) Stop() error {
	return s.runComposeAction("stop", s.runtime.ComposeDown)
}

func (s *BaseService) runComposeAction(action string, execFn func(string) error) error {
	verb := actionVerb(action)
	baseEvent := fmt.Sprintf("service.%s", action)
	serviceFields := map[string]interface{}{"service": s.name}

	s.logger.Info(baseEvent, fmt.Sprintf("%s %s service", verb, s.name), serviceFields)

	if err := execFn(s.composeFile); err != nil {
		s.logger.Error(baseEvent+".error", fmt.Sprintf("Failed to %s service", action), map[string]interface{}{
			"service": s.name,
			"error":   err.Error(),
		})
		return fmt.Errorf("failed to %s %s: %w", action, s.name, err)
	}

	s.logger.Info(baseEvent+"ed", fmt.Sprintf("%s service %s", s.name, pastTense(action)), serviceFields)
	return nil
}

func actionVerb(action string) string {
	verbs := map[string]string{
		"start": "Starting",
		"stop":  "Stopping",
	}
	if v, ok := verbs[action]; ok {
		return v
	}
	if action == "" {
		return action
	}
	return strings.ToUpper(action[:1]) + action[1:]
}

func pastTense(action string) string {
	forms := map[string]string{
		"start": "started",
		"stop":  "stopped",
	}
	if v, ok := forms[action]; ok {
		return v
	}
	return action + "ed"
}

// Status returns the current status of the service
func (s *BaseService) Status() (ServiceStatus, error) {
	containerName := fmt.Sprintf("aistack-%s", s.name)
	state, err := s.runtime.GetContainerStatus(containerName)
	if err != nil {
		return ServiceStatus{
			Name:    s.name,
			State:   "unknown",
			Health:  HealthRed,
			Message: err.Error(),
		}, nil
	}

	// Get health status
	health := HealthRed
	if state == "running" {
		healthStatus, err := s.Health()
		if err == nil {
			health = healthStatus
		}
	}

	return ServiceStatus{
		Name:   s.name,
		State:  state,
		Health: health,
	}, nil
}

// Health performs a health check on the service
func (s *BaseService) Health() (HealthStatus, error) {
	return s.healthCheck.Check()
}

// Remove removes the service (optionally keeping data volumes)
func (s *BaseService) Remove(keepData bool) error {
	s.logger.Info("service.remove", fmt.Sprintf("Removing %s service", s.name), map[string]interface{}{
		"service":   s.name,
		"keep_data": keepData,
	})

	// First stop the service
	if err := s.Stop(); err != nil {
		// Log but continue - service might already be stopped
		s.logger.Warn("service.remove.stop_error", "Error stopping service during removal", map[string]interface{}{
			"service": s.name,
			"error":   err.Error(),
		})
	}

	// Remove volumes if requested
	if !keepData {
		for _, volume := range s.volumes {
			if err := s.runtime.RemoveVolume(volume); err != nil {
				s.logger.Warn("service.remove.volume_error", "Error removing volume", map[string]interface{}{
					"service": s.name,
					"volume":  volume,
					"error":   err.Error(),
				})
			}
		}
	}

	s.logger.Info("service.removed", fmt.Sprintf("%s service removed", s.name), map[string]interface{}{
		"service": s.name,
	})

	return nil
}

// Update performs a service update - must be implemented by concrete services
func (s *BaseService) Update() error {
	return fmt.Errorf("update not implemented for base service")
}

// Logs retrieves logs from the service container
func (s *BaseService) Logs(tail int) (string, error) {
	containerName := fmt.Sprintf("aistack-%s", s.name)
	logs, err := s.runtime.GetContainerLogs(containerName, tail)
	if err != nil {
		return "", fmt.Errorf("failed to get logs for %s: %w", s.name, err)
	}
	return logs, nil
}
