package services

import (
	"aistack/internal/logging"
)

// OllamaService manages the Ollama container service
// Story T-006: Compose-Template: Ollama Service (Health & Ports)
type OllamaService struct {
	*BaseService
}

// NewOllamaService creates a new Ollama service
func NewOllamaService(composeDir string, runtime Runtime, logger *logging.Logger) *OllamaService {
	healthCheck := DefaultHealthCheck("http://localhost:11434/api/tags")
	volumes := []string{"ollama_data"}

	base := NewBaseService("ollama", composeDir, healthCheck, volumes, runtime, logger)

	return &OllamaService{
		BaseService: base,
	}
}
