package suspend

import (
	"os"
	"testing"
	"time"

	"aistack/internal/logging"
)

func TestStateLoadAndSave(t *testing.T) {
	// Create temp directory for testing
	tempDir := t.TempDir()

	// Override state dir
	oldEnv := os.Getenv("AISTACK_STATE_DIR")
	defer func() {
		if oldEnv != "" {
			os.Setenv("AISTACK_STATE_DIR", oldEnv)
		} else {
			os.Unsetenv("AISTACK_STATE_DIR")
		}
	}()
	os.Setenv("AISTACK_STATE_DIR", tempDir)

	logger := logging.NewLogger(logging.LevelDebug)
	manager := NewManager(logger)

	// Load state (should create default)
	state, err := manager.LoadState()
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	// Verify default state
	if !state.Enabled {
		t.Error("Expected default state to be enabled")
	}
	if state.LastActiveTimestamp == 0 {
		t.Error("Expected default state to have valid timestamp")
	}

	// Modify and save
	state.Enabled = false
	state.LastActiveTimestamp = time.Now().Unix()
	if err := manager.SaveState(state); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Load again and verify
	state2, err := manager.LoadState()
	if err != nil {
		t.Fatalf("LoadState (second) failed: %v", err)
	}

	if state2.Enabled {
		t.Error("Expected state to be disabled after save")
	}
	if state2.LastActiveTimestamp != state.LastActiveTimestamp {
		t.Error("LastActiveTimestamp mismatch after save/load")
	}
}

func TestEnableDisable(t *testing.T) {
	// Create temp directory for testing
	tempDir := t.TempDir()
	oldEnv := os.Getenv("AISTACK_STATE_DIR")
	defer func() {
		if oldEnv != "" {
			os.Setenv("AISTACK_STATE_DIR", oldEnv)
		} else {
			os.Unsetenv("AISTACK_STATE_DIR")
		}
	}()
	os.Setenv("AISTACK_STATE_DIR", tempDir)

	logger := logging.NewLogger(logging.LevelDebug)
	manager := NewManager(logger)

	// Initial state (should be enabled by default)
	state, err := manager.LoadState()
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}
	if !state.Enabled {
		t.Error("Expected initial state to be enabled")
	}

	// Disable
	if err := manager.Disable(); err != nil {
		t.Fatalf("Disable failed: %v", err)
	}

	// Verify disabled
	state, err = manager.LoadState()
	if err != nil {
		t.Fatalf("LoadState after disable failed: %v", err)
	}
	if state.Enabled {
		t.Error("Expected state to be disabled")
	}

	// Enable
	if err := manager.Enable(); err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	// Verify enabled
	state, err = manager.LoadState()
	if err != nil {
		t.Fatalf("LoadState after enable failed: %v", err)
	}
	if !state.Enabled {
		t.Error("Expected state to be enabled")
	}
}

func TestShouldSuspend(t *testing.T) {
	logger := logging.NewLogger(logging.LevelDebug)
	manager := NewManager(logger)

	tests := []struct {
		name            string
		enabled         bool
		idleSeconds     int
		expectedSuspend bool
	}{
		{
			name:            "disabled - should not suspend",
			enabled:         false,
			idleSeconds:     400,
			expectedSuspend: false,
		},
		{
			name:            "enabled but not idle long enough",
			enabled:         true,
			idleSeconds:     200,
			expectedSuspend: false,
		},
		{
			name:            "enabled and idle timeout reached",
			enabled:         true,
			idleSeconds:     400,
			expectedSuspend: true,
		},
		{
			name:            "enabled and exactly at timeout",
			enabled:         true,
			idleSeconds:     IdleTimeoutSeconds,
			expectedSuspend: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &State{
				Enabled:             tt.enabled,
				LastActiveTimestamp: time.Now().Unix() - int64(tt.idleSeconds),
			}

			shouldSuspend := manager.ShouldSuspend(state)
			if shouldSuspend != tt.expectedSuspend {
				t.Errorf("ShouldSuspend() = %v, want %v", shouldSuspend, tt.expectedSuspend)
			}
		})
	}
}

func TestGetIdleDuration(t *testing.T) {
	logger := logging.NewLogger(logging.LevelDebug)
	manager := NewManager(logger)

	now := time.Now()
	state := &State{
		Enabled:             true,
		LastActiveTimestamp: now.Add(-5 * time.Minute).Unix(),
	}

	duration := manager.GetIdleDuration(state)

	// Allow some tolerance (1 second) for test execution time
	expected := 5 * time.Minute
	tolerance := 1 * time.Second

	if duration < expected-tolerance || duration > expected+tolerance {
		t.Errorf("GetIdleDuration() = %v, want ~%v", duration, expected)
	}
}
