package services

import (
	"os"

	"aistack/internal/logging"
)

const (
	// OpenWebUIImageName is the Docker image for Open WebUI
	OpenWebUIImageName = "ghcr.io/open-webui/open-webui:main"
)

// OpenWebUIService manages the Open WebUI container service
// Story T-007: Compose-Template: Open WebUI mit Backend-Binding
type OpenWebUIService struct {
	*BaseService
	updater *ServiceUpdater
}

// NewOpenWebUIService creates a new Open WebUI service
func NewOpenWebUIService(composeDir string, runtime Runtime, logger *logging.Logger) *OpenWebUIService {
	healthCheck := DefaultHealthCheck("http://localhost:3000/")
	volumes := []string{"openwebui_data"}

	base := NewBaseService("openwebui", composeDir, healthCheck, volumes, runtime, logger)

	// Get state directory from env or use default
	stateDir := os.Getenv("AISTACK_STATE_DIR")
	if stateDir == "" {
		stateDir = "/var/lib/aistack"
	}

	updater := NewServiceUpdater(base, runtime, OpenWebUIImageName, healthCheck, logger, stateDir)

	return &OpenWebUIService{
		BaseService: base,
		updater:     updater,
	}
}

// Update updates the Open WebUI service to the latest version
func (s *OpenWebUIService) Update() error {
	return s.updater.Update()
}
