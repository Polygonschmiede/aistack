package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"aistack/internal/logging"
)

const (
	// UIStateFileName is the name of the UI state file
	UIStateFileName = "ui_state.json"
)

// UIStateManager manages the UI state persistence
// Story T-024: Persists menu, selection, last_error
type UIStateManager struct {
	stateDir string
	logger   *logging.Logger
}

// NewUIStateManager creates a new UI state manager
func NewUIStateManager(stateDir string, logger *logging.Logger) *UIStateManager {
	return &UIStateManager{
		stateDir: stateDir,
		logger:   logger,
	}
}

// getStatePath returns the full path to the state file
func (m *UIStateManager) getStatePath() string {
	return filepath.Join(m.stateDir, UIStateFileName)
}

// Load loads the UI state from disk
func (m *UIStateManager) Load() (*UIState, error) {
	statePath := m.getStatePath()

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default state if file doesn't exist
			return &UIState{
				CurrentScreen: ScreenMenu,
				Selection:     0,
				LastError:     "",
				Updated:       time.Now().UTC(),
			}, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state UIState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// Save saves the UI state to disk
func (m *UIStateManager) Save(state *UIState) error {
	// Ensure state directory exists
	if err := os.MkdirAll(m.stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Update timestamp
	state.Updated = time.Now().UTC()

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
		os.Remove(tmpPath) // Clean up temp file on error
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	m.logger.Debug("tui.state.saved", "UI state saved", map[string]interface{}{
		"screen":    state.CurrentScreen,
		"selection": state.Selection,
	})

	return nil
}

// SaveError saves an error message to the state
func (m *UIStateManager) SaveError(errorMsg string) error {
	state, err := m.Load()
	if err != nil {
		// If we can't load, create new state with error
		state = &UIState{
			CurrentScreen: ScreenMenu,
			Selection:     0,
			LastError:     errorMsg,
			Updated:       time.Now().UTC(),
		}
	} else {
		state.LastError = errorMsg
	}

	return m.Save(state)
}

// ClearError clears the last error from the state
func (m *UIStateManager) ClearError() error {
	state, err := m.Load()
	if err != nil {
		return err
	}

	state.LastError = ""
	return m.Save(state)
}
