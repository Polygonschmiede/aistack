package suspend

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"aistack/internal/logging"
)

const (
	// IdleTimeoutSeconds is the duration of idle time required before suspend
	IdleTimeoutSeconds = 300 // 5 minutes

	// Default state directory
	defaultStateDir = "/var/lib/aistack"
)

// State holds suspend configuration and activity tracking
type State struct {
	Enabled             bool  `json:"enabled"`               // Whether auto-suspend is enabled
	LastActiveTimestamp int64 `json:"last_active_timestamp"` // Unix timestamp of last activity
}

// Manager handles suspend state persistence
type Manager struct {
	logger    *logging.Logger
	stateFile string
}

// getStateDir returns the state directory path (respects AISTACK_STATE_DIR env var)
func getStateDir() string {
	if env := os.Getenv("AISTACK_STATE_DIR"); env != "" {
		if abs, err := filepath.Abs(env); err == nil {
			return abs
		}
	}
	return defaultStateDir
}

// NewManager creates a new state manager
func NewManager(logger *logging.Logger) *Manager {
	stateDir := getStateDir()
	return &Manager{
		logger:    logger,
		stateFile: filepath.Join(stateDir, "suspend_state.json"),
	}
}

// LoadState loads suspend state from disk (creates default if not exists)
func (m *Manager) LoadState() (*State, error) {
	// Try to read existing state
	data, err := os.ReadFile(m.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default state (enabled=true, last_active=now)
			m.logger.Info("suspend.state.init", "Creating default suspend state", map[string]interface{}{
				"enabled": true,
			})
			state := &State{
				Enabled:             true,
				LastActiveTimestamp: time.Now().Unix(),
			}
			// Save default state
			if saveErr := m.SaveState(state); saveErr != nil {
				return nil, fmt.Errorf("save default state: %w", saveErr)
			}
			return state, nil
		}
		return nil, fmt.Errorf("read state file: %w", err)
	}

	// Parse JSON
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse state JSON: %w", err)
	}

	m.logger.Debug("suspend.state.loaded", "Loaded suspend state", map[string]interface{}{
		"enabled":         state.Enabled,
		"last_active_age": time.Since(time.Unix(state.LastActiveTimestamp, 0)).Seconds(),
	})

	return &state, nil
}

// SaveState saves suspend state to disk
func (m *Manager) SaveState(state *State) error {
	// Ensure state directory exists
	stateDir := filepath.Dir(m.stateFile)
	if err := os.MkdirAll(stateDir, 0750); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	// Marshal JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state JSON: %w", err)
	}

	// Write to temp file first (atomic write)
	tempFile := m.stateFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0640); err != nil {
		return fmt.Errorf("write temp state file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, m.stateFile); err != nil {
		return fmt.Errorf("rename state file: %w", err)
	}

	m.logger.Debug("suspend.state.saved", "Saved suspend state", map[string]interface{}{
		"enabled": state.Enabled,
	})

	return nil
}

// Enable enables auto-suspend
func (m *Manager) Enable() error {
	state, err := m.LoadState()
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	if state.Enabled {
		m.logger.Info("suspend.already_enabled", "Auto-suspend already enabled", nil)
		return nil
	}

	state.Enabled = true
	state.LastActiveTimestamp = time.Now().Unix() // Reset activity timestamp

	if err := m.SaveState(state); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	m.logger.Info("suspend.enabled", "Auto-suspend enabled", nil)
	return nil
}

// Disable disables auto-suspend
func (m *Manager) Disable() error {
	state, err := m.LoadState()
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	if !state.Enabled {
		m.logger.Info("suspend.already_disabled", "Auto-suspend already disabled", nil)
		return nil
	}

	state.Enabled = false

	if err := m.SaveState(state); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	m.logger.Info("suspend.disabled", "Auto-suspend disabled", nil)
	return nil
}

// GetIdleDuration calculates how long the system has been idle
func (m *Manager) GetIdleDuration(state *State) time.Duration {
	return time.Since(time.Unix(state.LastActiveTimestamp, 0))
}

// ShouldSuspend returns true if system should suspend now
func (m *Manager) ShouldSuspend(state *State) bool {
	if !state.Enabled {
		return false
	}

	idleDuration := m.GetIdleDuration(state)
	return idleDuration >= time.Duration(IdleTimeoutSeconds)*time.Second
}
