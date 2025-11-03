package services

import (
	"aistack/internal/logging"
)

// OpenWebUIService manages the Open WebUI container service
// Story T-007: Compose-Template: Open WebUI mit Backend-Binding
type OpenWebUIService struct {
	*BaseService
}

// NewOpenWebUIService creates a new Open WebUI service
func NewOpenWebUIService(composeDir string, runtime Runtime, logger *logging.Logger) *OpenWebUIService {
	healthCheck := DefaultHealthCheck("http://localhost:3000/")
	volumes := []string{"openwebui_data"}

	base := NewBaseService("openwebui", composeDir, healthCheck, volumes, runtime, logger)

	return &OpenWebUIService{
		BaseService: base,
	}
}
