package services

import (
	"aistack/internal/logging"
)

// LocalAIService manages the LocalAI container service
// Story T-008: Compose-Template: LocalAI Service (Health & Volume)
type LocalAIService struct {
	*BaseService
}

// NewLocalAIService creates a new LocalAI service
func NewLocalAIService(composeDir string, runtime Runtime, logger *logging.Logger) *LocalAIService {
	healthCheck := DefaultHealthCheck("http://localhost:8080/healthz")
	volumes := []string{"localai_models"}

	base := NewBaseService("localai", composeDir, healthCheck, volumes, runtime, logger)

	return &LocalAIService{
		BaseService: base,
	}
}
