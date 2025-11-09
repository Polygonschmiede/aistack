package idle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"aistack/internal/logging"
)

// StateManager handles idle state persistence
type StateManager struct {
	filePath string
	logger   *logging.Logger
}

// NewStateManager creates a new state manager
func NewStateManager(filePath string, logger *logging.Logger) *StateManager {
	return &StateManager{
		filePath: filePath,
		logger:   logger,
	}
}

// Save writes the idle state to the configured file
func (sm *StateManager) Save(state IdleState) error {
	// Ensure directory exists
	dir := filepath.Dir(sm.filePath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to file atomically (write to temp, then rename)
	tempPath := sm.filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempPath, sm.filePath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	sm.logger.Debug("idle.state.saved", "Idle state saved", map[string]interface{}{
		"path":   sm.filePath,
		"status": state.Status,
	})

	return nil
}

// Load reads the idle state from the configured file
func (sm *StateManager) Load() (IdleState, error) {
	data, err := os.ReadFile(sm.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return IdleState{}, fmt.Errorf("state file not found: %w", err)
		}
		return IdleState{}, fmt.Errorf("failed to read state file: %w", err)
	}

	var state IdleState
	if err := json.Unmarshal(data, &state); err != nil {
		return IdleState{}, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Remove "inhibit" from gating reasons - it's a runtime check, not a state property
	// The inhibitor check is performed fresh on each idle-check run
	state.GatingReasons = removeReason(state.GatingReasons, GatingReasonInhibit)

	sm.logger.Debug("idle.state.loaded", "Idle state loaded", map[string]interface{}{
		"path":   sm.filePath,
		"status": state.Status,
	})

	return state, nil
}

// removeReason removes a specific reason from the gating reasons list
func removeReason(reasons []string, reason string) []string {
	result := make([]string, 0, len(reasons))
	for _, r := range reasons {
		if r != reason {
			result = append(result, r)
		}
	}
	return result
}

// Delete removes the state file
func (sm *StateManager) Delete() error {
	if err := os.Remove(sm.filePath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete state file: %w", err)
	}

	sm.logger.Debug("idle.state.deleted", "Idle state file deleted", map[string]interface{}{
		"path": sm.filePath,
	})

	return nil
}

// Exists checks if the state file exists
func (sm *StateManager) Exists() bool {
	_, err := os.Stat(sm.filePath)
	return err == nil
}
