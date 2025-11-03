package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"aistack/internal/logging"
)

// LocalAIModel represents a cached model entry for LocalAI
type LocalAIModel struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	Updated string `json:"updated"`
}

// LocalAIModelsRegistry persists the localai_models.json contract
type LocalAIModelsRegistry struct {
	path   string
	logger *logging.Logger
}

// NewLocalAIModelsRegistry creates a registry rooted in the given state directory
func NewLocalAIModelsRegistry(stateDir string, logger *logging.Logger) *LocalAIModelsRegistry {
	return &LocalAIModelsRegistry{
		path:   filepath.Join(stateDir, "localai_models.json"),
		logger: logger,
	}
}

// Ensure ensures the registry file exists with a valid structure
func (r *LocalAIModelsRegistry) Ensure() error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o750); err != nil {
		return fmt.Errorf("failed to create registry directory: %w", err)
	}

	cleanPath := filepath.Clean(r.path)
	if _, err := os.Stat(cleanPath); err == nil {
		return nil
	}

	doc := struct {
		Models  []LocalAIModel `json:"models"`
		Updated string         `json:"updated"`
	}{
		Models:  []LocalAIModel{},
		Updated: time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(cleanPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	r.logger.Info("localai.registry.created", "Initialized LocalAI model registry", map[string]interface{}{
		"path": r.path,
	})

	return nil
}

// Update replaces the registry contents with the provided model list
func (r *LocalAIModelsRegistry) Update(models []LocalAIModel) error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o750); err != nil {
		return fmt.Errorf("failed to create registry directory: %w", err)
	}

	doc := struct {
		Models  []LocalAIModel `json:"models"`
		Updated string         `json:"updated"`
	}{
		Models:  models,
		Updated: time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(filepath.Clean(r.path), data, 0o600); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	r.logger.Info("localai.registry.updated", "Updated LocalAI model registry", map[string]interface{}{
		"path":   r.path,
		"models": len(models),
	})

	return nil
}
