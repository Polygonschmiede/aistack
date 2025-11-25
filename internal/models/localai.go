package models

import (
	"fmt"
	"os"
	"path/filepath"

	"aistack/internal/logging"
)

// LocalAIManager manages LocalAI models
// Story T-023: LocalAI cache management
type LocalAIManager struct {
	stateManager *StateManager
	logger       *logging.Logger
	modelsPath   string // Path to LocalAI models directory
}

// NewLocalAIManager creates a new LocalAI model manager
func NewLocalAIManager(stateDir string, modelsPath string, logger *logging.Logger) *LocalAIManager {
	return &LocalAIManager{
		stateManager: NewStateManager(stateDir, ProviderLocalAI, logger),
		logger:       logger,
		modelsPath:   modelsPath,
	}
}

// List returns all LocalAI models by scanning the models directory
func (m *LocalAIManager) List() ([]ModelInfo, error) {
	if m.modelsPath == "" {
		return []ModelInfo{}, nil
	}

	// Check if directory exists
	if _, err := os.Stat(m.modelsPath); os.IsNotExist(err) {
		return []ModelInfo{}, nil
	}

	entries, err := os.ReadDir(m.modelsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read models directory: %w", err)
	}

	models := make([]ModelInfo, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Get file info
		fullPath := filepath.Join(m.modelsPath, entry.Name())
		info, err := entry.Info()
		if err != nil {
			m.logger.Warn("model.list.stat_failed", "Failed to get file info", map[string]interface{}{
				"file":  entry.Name(),
				"error": err.Error(),
			})
			continue
		}

		models = append(models, ModelInfo{
			Name:     entry.Name(),
			Size:     info.Size(),
			Path:     fullPath,
			LastUsed: info.ModTime(), // Use file modification time as fallback
		})
	}

	return models, nil
}

// SyncState synchronizes the state with actual LocalAI models
func (m *LocalAIManager) SyncState() error {
	models, err := m.List()
	if err != nil {
		return err
	}

	return SyncStateWithModels(m.stateManager, models)
}

// Delete removes a LocalAI model
func (m *LocalAIManager) Delete(modelName string) error {
	m.logger.Info("model.delete.started", "Deleting model", map[string]interface{}{
		"provider": ProviderLocalAI,
		"model":    modelName,
	})

	modelPath := filepath.Join(m.modelsPath, modelName)

	// Check if file exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("model file not found: %s", modelName)
	}

	// Remove file
	if err := os.Remove(modelPath); err != nil {
		return fmt.Errorf("failed to remove model file: %w", err)
	}

	// Remove from state
	if err := m.stateManager.RemoveModel(modelName); err != nil {
		m.logger.Warn("model.delete.state_update_failed", "Failed to update state", map[string]interface{}{
			"error": err.Error(),
		})
	}

	m.logger.Info("model.delete.completed", "Model deleted", map[string]interface{}{
		"model": modelName,
		"path":  modelPath,
	})

	return nil
}

// GetStats returns cache statistics
// Story T-023: Cache overview
func (m *LocalAIManager) GetStats() (*CacheStats, error) {
	// Sync state with actual models
	if err := m.SyncState(); err != nil {
		m.logger.Warn("model.stats.sync_failed", "Failed to sync state", map[string]interface{}{
			"error": err.Error(),
		})
	}

	return m.stateManager.GetStats()
}

// EvictOldest removes the oldest model to free up space
// Story T-023: Evict oldest functionality
func (m *LocalAIManager) EvictOldest() (*ModelInfo, error) {
	return evictOldestModel(m.SyncState, m.stateManager, m.Delete, m.logger)
}

// UpdateLastUsed updates the last used timestamp for a model
func (m *LocalAIManager) UpdateLastUsed(modelName string) error {
	return m.stateManager.UpdateLastUsed(modelName)
}
