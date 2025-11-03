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
	updater *ServiceUpdater
}

// NewLocalAIService creates a new LocalAI service
func NewLocalAIService(composeDir string, runtime Runtime, logger *logging.Logger) *LocalAIService {
	healthCheck := DefaultHealthCheck("http://localhost:8080/healthz")
	volumes := []string{"localai_models"}

	base := NewBaseService("localai", composeDir, healthCheck, volumes, runtime, logger)

	// Get state directory from env or use default
	stateDir := os.Getenv("AISTACK_STATE_DIR")
	if stateDir == "" {
		stateDir = "/var/lib/aistack"
	}

	updater := NewServiceUpdater(base, runtime, LocalAIImageName, healthCheck, logger, stateDir)

	return &LocalAIService{
		BaseService: base,
		updater:     updater,
	}
}

// Update updates the LocalAI service to the latest version
func (s *LocalAIService) Update() error {
	return s.updater.Update()
}
