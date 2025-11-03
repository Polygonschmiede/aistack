package services

import (
	"aistack/internal/logging"
	"fmt"
	"path/filepath"
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
	healthCheck HealthCheck
	volumes     []string
	runtime     Runtime
	logger      *logging.Logger
	netManager  *NetworkManager
}

// NewBaseService creates a new base service
func NewBaseService(name, composeDir string, healthCheck HealthCheck, volumes []string, runtime Runtime, logger *logging.Logger) *BaseService {
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
	s.logger.Info("service.start", fmt.Sprintf("Starting %s service", s.name), map[string]interface{}{
		"service": s.name,
	})

	if err := s.runtime.ComposeUp(s.composeFile); err != nil {
		s.logger.Error("service.start.error", "Failed to start service", map[string]interface{}{
			"service": s.name,
			"error":   err.Error(),
		})
		return fmt.Errorf("failed to start %s: %w", s.name, err)
	}

	s.logger.Info("service.started", fmt.Sprintf("%s service started", s.name), map[string]interface{}{
		"service": s.name,
	})

	return nil
}

// Stop stops the service
func (s *BaseService) Stop() error {
	s.logger.Info("service.stop", fmt.Sprintf("Stopping %s service", s.name), map[string]interface{}{
		"service": s.name,
	})

	if err := s.runtime.ComposeDown(s.composeFile); err != nil {
		s.logger.Error("service.stop.error", "Failed to stop service", map[string]interface{}{
			"service": s.name,
			"error":   err.Error(),
		})
		return fmt.Errorf("failed to stop %s: %w", s.name, err)
	}

	s.logger.Info("service.stopped", fmt.Sprintf("%s service stopped", s.name), map[string]interface{}{
		"service": s.name,
	})

	return nil
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

	// TODO: If !keepData, remove volumes
	// This would require additional runtime methods for volume removal

	s.logger.Info("service.removed", fmt.Sprintf("%s service removed", s.name), map[string]interface{}{
		"service": s.name,
	})

	return nil
}
