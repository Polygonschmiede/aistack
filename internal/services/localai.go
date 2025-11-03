package services

import (
	"os"

	"aistack/internal/logging"
)

const (
	// LocalAIImageName is the Docker image for LocalAI
	LocalAIImageName = "quay.io/go-skynet/local-ai:latest"
)

// LocalAIService manages the LocalAI container service
// Story T-008: Compose-Template: LocalAI Service (Health & Volume)
type LocalAIService struct {
	*BaseService
	updater  *ServiceUpdater
	registry *LocalAIModelsRegistry
}

// NewLocalAIService creates a new LocalAI service
func NewLocalAIService(composeDir string, runtime Runtime, logger *logging.Logger, lock *VersionLock) *LocalAIService {
	healthCheck := DefaultHealthCheck("http://localhost:8080/healthz")
	volumes := []string{"localai_models"}

	base := NewBaseService("localai", composeDir, healthCheck, volumes, runtime, logger)

	// Get state directory from env or use default
	stateDir := os.Getenv("AISTACK_STATE_DIR")
	if stateDir == "" {
		stateDir = "/var/lib/aistack"
	}

	updater := NewServiceUpdater(base, runtime, LocalAIImageName, healthCheck, logger, stateDir, lock)
	registry := NewLocalAIModelsRegistry(stateDir, logger)

	base.SetPreStartHook(func() error {
		if err := updater.EnforceImagePolicy(); err != nil {
			return err
		}
		return registry.Ensure()
	})

	return &LocalAIService{
		BaseService: base,
		updater:     updater,
		registry:    registry,
	}
}

// Update updates the LocalAI service to the latest version
func (s *LocalAIService) Update() error {
	if err := s.registry.Ensure(); err != nil {
		return err
	}
	return s.updater.Update()
}
