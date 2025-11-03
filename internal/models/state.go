package models

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"aistack/internal/logging"
)

const (
	// StateFileName is the name of the models state file
	StateFileName = "models_state.json"
)

// StateManager manages the models state persistence
// Story T-022, T-023: Model state tracking
type StateManager struct {
	stateDir string
	provider Provider
	logger   *logging.Logger
}

// NewStateManager creates a new state manager
func NewStateManager(stateDir string, provider Provider, logger *logging.Logger) *StateManager {
	return &StateManager{
		stateDir: stateDir,
		provider: provider,
		logger:   logger,
	}
}

// getStatePath returns the full path to the state file for the provider
func (m *StateManager) getStatePath() string {
	return filepath.Join(m.stateDir, fmt.Sprintf("%s_%s", m.provider, StateFileName))
}

// Load loads the models state from disk
func (m *StateManager) Load() (*State, error) {
	statePath := m.getStatePath()

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty state if file doesn't exist
			return &State{
				Provider: m.provider,
				Items:    []ModelInfo{},
				Updated:  time.Now().UTC(),
			}, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// Save saves the models state to disk
func (m *StateManager) Save(state *State) error {
	// Ensure state directory exists
	if err := os.MkdirAll(m.stateDir, 0o750); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Update timestamp
	state.Updated = time.Now().UTC()
	state.Provider = m.provider

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	statePath := m.getStatePath()

	// Atomic write: write to temp file, then rename
	tmpPath := statePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}

	if err := os.Rename(tmpPath, statePath); err != nil {
		if removeErr := os.Remove(tmpPath); removeErr != nil && !os.IsNotExist(removeErr) {
			m.logger.Warn("models.state.tmp_cleanup_failed", "Failed to remove temp state file", map[string]interface{}{
				"error": removeErr.Error(),
				"path":  tmpPath,
			})
		}
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	m.logger.Info("models.state.saved", "Models state saved", map[string]interface{}{
		"provider": m.provider,
		"count":    len(state.Items),
	})

	return nil
}

// AddModel adds or updates a model in the state
func (m *StateManager) AddModel(model ModelInfo) error {
	state, err := m.Load()
	if err != nil {
		return err
	}

	// Check if model already exists
	found := false
	for i, item := range state.Items {
		if item.Name == model.Name {
			state.Items[i] = model
			found = true
			break
		}
	}

	// Add new model if not found
	if !found {
		state.Items = append(state.Items, model)
	}

	return m.Save(state)
}

// RemoveModel removes a model from the state
func (m *StateManager) RemoveModel(modelName string) error {
	state, err := m.Load()
	if err != nil {
		return err
	}

	// Filter out the model
	filtered := make([]ModelInfo, 0)
	for _, item := range state.Items {
		if item.Name != modelName {
			filtered = append(filtered, item)
		}
	}

	state.Items = filtered
	return m.Save(state)
}

// UpdateLastUsed updates the last used timestamp for a model
func (m *StateManager) UpdateLastUsed(modelName string) error {
	state, err := m.Load()
	if err != nil {
		return err
	}

	// Find and update the model
	for i, item := range state.Items {
		if item.Name == modelName {
			state.Items[i].LastUsed = time.Now().UTC()
			return m.Save(state)
		}
	}

	return fmt.Errorf("model not found: %s", modelName)
}

// GetStats returns cache statistics
// Story T-023: Cache overview
func (m *StateManager) GetStats() (*CacheStats, error) {
	state, err := m.Load()
	if err != nil {
		return nil, err
	}

	stats := &CacheStats{
		Provider:   m.provider,
		TotalSize:  0,
		ModelCount: len(state.Items),
	}

	// Calculate total size and find oldest model
	var oldestModel *ModelInfo
	for i := range state.Items {
		stats.TotalSize += state.Items[i].Size

		// Find oldest model by last_used
		if oldestModel == nil || state.Items[i].LastUsed.Before(oldestModel.LastUsed) {
			// Make a copy to avoid pointer to loop variable
			model := state.Items[i]
			oldestModel = &model
		}
	}

	stats.OldestModel = oldestModel

	return stats, nil
}

// GetOldestModels returns models sorted by last_used (oldest first)
// Story T-023: Evict oldest functionality
func (m *StateManager) GetOldestModels() ([]ModelInfo, error) {
	state, err := m.Load()
	if err != nil {
		return nil, err
	}

	// Create a copy to avoid modifying the original
	models := make([]ModelInfo, len(state.Items))
	copy(models, state.Items)

	// Sort by last_used (oldest first)
	sort.Slice(models, func(i, j int) bool {
		return models[i].LastUsed.Before(models[j].LastUsed)
	})

	return models, nil
}

// Clear removes all models from the state
func (m *StateManager) Clear() error {
	state := &State{
		Provider: m.provider,
		Items:    []ModelInfo{},
		Updated:  time.Now().UTC(),
	}

	return m.Save(state)
}
