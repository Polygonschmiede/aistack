package services

import (
	"os"

	"aistack/internal/logging"
)

const (
	// OllamaImageName is the Docker image for Ollama
	OllamaImageName = "ollama/ollama:latest"
)

// OllamaService manages the Ollama container service
// Story T-006: Compose-Template: Ollama Service (Health & Ports)
// Story T-017: Ollama Lifecycle Commands (install/start/stop/remove)
// Story T-018: Ollama Update & Rollback (Service-specific)
type OllamaService struct {
	*BaseService
	updater *ServiceUpdater
}

// NewOllamaService creates a new Ollama service
func NewOllamaService(composeDir string, runtime Runtime, logger *logging.Logger) *OllamaService {
	healthCheck := DefaultHealthCheck("http://localhost:11434/api/tags")
	volumes := []string{"ollama_data"}

	base := NewBaseService("ollama", composeDir, healthCheck, volumes, runtime, logger)

	// Get state directory from env or use default
	stateDir := os.Getenv("AISTACK_STATE_DIR")
	if stateDir == "" {
		stateDir = "/var/lib/aistack"
	}

	updater := NewServiceUpdater(base, runtime, OllamaImageName, healthCheck, logger, stateDir)

	return &OllamaService{
		BaseService: base,
		updater:     updater,
	}
}

// Update updates the Ollama service to the latest version
// Story T-018: Implements update with health validation and rollback
func (s *OllamaService) Update() error {
	return s.updater.Update()
}
