package models

import (
	"fmt"

	"aistack/internal/logging"
)

// evictOldestModel centralizes the shared eviction flow between model managers.
func evictOldestModel(
	syncState func() error,
	stateManager *StateManager,
	deleteModel func(string) error,
	logger *logging.Logger,
) (*ModelInfo, error) {
	if err := syncState(); err != nil {
		return nil, err
	}

	oldestModels, err := stateManager.GetOldestModels()
	if err != nil {
		return nil, err
	}

	if len(oldestModels) == 0 {
		return nil, fmt.Errorf("no models to evict")
	}

	oldest := oldestModels[0]

	logger.Info("model.evict.started", "Evicting oldest model", map[string]interface{}{
		"model":     oldest.Name,
		"last_used": oldest.LastUsed,
		"size":      oldest.Size,
	})

	if err := deleteModel(oldest.Name); err != nil {
		return nil, fmt.Errorf("failed to delete oldest model: %w", err)
	}

	logger.Info("model.evict.completed", "Oldest model evicted", map[string]interface{}{
		"model": oldest.Name,
		"size":  oldest.Size,
	})

	return &oldest, nil
}
