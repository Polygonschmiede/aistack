package idle

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"aistack/internal/logging"
)

func TestStateManager_SaveAndLoad(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	tempFile := filepath.Join(os.TempDir(), "test_idle_state.json")
	defer os.Remove(tempFile)

	manager := NewStateManager(tempFile, logger)

	// Create test state
	state := IdleState{
		Status:           StatusIdle,
		IdleForSeconds:   120,
		ThresholdSeconds: 300,
		CPUIdlePct:       95.0,
		GPUIdlePct:       97.0,
		GatingReasons:    []string{GatingReasonBelowTimeout},
		LastUpdate:       time.Now(),
	}

	// Save
	if err := manager.Save(state); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Load
	loaded, err := manager.Load()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Verify
	if loaded.Status != state.Status {
		t.Errorf("Expected status %s, got %s", state.Status, loaded.Status)
	}

	if loaded.IdleForSeconds != state.IdleForSeconds {
		t.Errorf("Expected idle_for_s %d, got %d", state.IdleForSeconds, loaded.IdleForSeconds)
	}

	if loaded.ThresholdSeconds != state.ThresholdSeconds {
		t.Errorf("Expected threshold_s %d, got %d", state.ThresholdSeconds, loaded.ThresholdSeconds)
	}

	if loaded.CPUIdlePct != state.CPUIdlePct {
		t.Errorf("Expected CPU idle %.2f%%, got %.2f%%", state.CPUIdlePct, loaded.CPUIdlePct)
	}

	if loaded.GPUIdlePct != state.GPUIdlePct {
		t.Errorf("Expected GPU idle %.2f%%, got %.2f%%", state.GPUIdlePct, loaded.GPUIdlePct)
	}

	if len(loaded.GatingReasons) != len(state.GatingReasons) {
		t.Errorf("Expected %d gating reasons, got %d", len(state.GatingReasons), len(loaded.GatingReasons))
	}
}

func TestStateManager_LoadNonexistent(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	tempFile := filepath.Join(os.TempDir(), "nonexistent_idle_state.json")

	manager := NewStateManager(tempFile, logger)

	// Try to load nonexistent file
	_, err := manager.Load()
	if err == nil {
		t.Error("Expected error when loading nonexistent file")
	}
}

func TestStateManager_Exists(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	tempFile := filepath.Join(os.TempDir(), "test_exists_idle_state.json")
	defer os.Remove(tempFile)

	manager := NewStateManager(tempFile, logger)

	// Should not exist initially
	if manager.Exists() {
		t.Error("Expected file to not exist initially")
	}

	// Save state
	state := IdleState{
		Status: StatusActive,
	}
	if err := manager.Save(state); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Should exist now
	if !manager.Exists() {
		t.Error("Expected file to exist after save")
	}
}

func TestStateManager_Delete(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	tempFile := filepath.Join(os.TempDir(), "test_delete_idle_state.json")
	defer os.Remove(tempFile)

	manager := NewStateManager(tempFile, logger)

	// Save state
	state := IdleState{
		Status: StatusActive,
	}
	if err := manager.Save(state); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Verify exists
	if !manager.Exists() {
		t.Error("Expected file to exist before delete")
	}

	// Delete
	if err := manager.Delete(); err != nil {
		t.Fatalf("Failed to delete state: %v", err)
	}

	// Verify deleted
	if manager.Exists() {
		t.Error("Expected file to not exist after delete")
	}

	// Delete again should not error
	if err := manager.Delete(); err != nil {
		t.Errorf("Expected no error when deleting already-deleted file, got: %v", err)
	}
}

func TestStateManager_AtomicWrite(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	tempFile := filepath.Join(os.TempDir(), "test_atomic_idle_state.json")
	defer os.Remove(tempFile)

	manager := NewStateManager(tempFile, logger)

	// Write multiple times
	for i := 0; i < 10; i++ {
		state := IdleState{
			Status:         StatusIdle,
			IdleForSeconds: i * 10,
		}

		if err := manager.Save(state); err != nil {
			t.Fatalf("Failed to save state iteration %d: %v", i, err)
		}

		// Immediately read back
		loaded, err := manager.Load()
		if err != nil {
			t.Fatalf("Failed to load state iteration %d: %v", i, err)
		}

		if loaded.IdleForSeconds != i*10 {
			t.Errorf("Iteration %d: expected idle_for_s %d, got %d", i, i*10, loaded.IdleForSeconds)
		}
	}
}

func TestStateManager_RemovesInhibitOnLoad(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	tempFile := filepath.Join(os.TempDir(), "test_inhibit_removal_state.json")
	defer os.Remove(tempFile)

	manager := NewStateManager(tempFile, logger)

	// Create state with "inhibit" gating reason (simulating what executor does)
	state := IdleState{
		Status:           StatusIdle,
		IdleForSeconds:   900,
		ThresholdSeconds: 300,
		CPUIdlePct:       95.0,
		GPUIdlePct:       98.0,
		GatingReasons:    []string{GatingReasonInhibit, GatingReasonBelowTimeout},
		LastUpdate:       time.Now(),
	}

	// Save state with "inhibit"
	if err := manager.Save(state); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Load state - "inhibit" should be removed
	loaded, err := manager.Load()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Verify "inhibit" was removed
	for _, reason := range loaded.GatingReasons {
		if reason == GatingReasonInhibit {
			t.Error("Expected 'inhibit' gating reason to be removed on load, but it's still present")
		}
	}

	// Verify other gating reasons are preserved
	hasExpectedReason := false
	for _, reason := range loaded.GatingReasons {
		if reason == GatingReasonBelowTimeout {
			hasExpectedReason = true
		}
	}

	if !hasExpectedReason {
		t.Error("Expected 'below_timeout' gating reason to be preserved, but it was removed")
	}

	// Verify count is correct (1 reason left instead of 2)
	if len(loaded.GatingReasons) != 1 {
		t.Errorf("Expected 1 gating reason after removing 'inhibit', got %d: %v",
			len(loaded.GatingReasons), loaded.GatingReasons)
	}
}

func TestRemoveReason(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		remove   string
		expected []string
	}{
		{
			name:     "remove single occurrence",
			input:    []string{"inhibit", "below_timeout"},
			remove:   "inhibit",
			expected: []string{"below_timeout"},
		},
		{
			name:     "remove from middle",
			input:    []string{"high_cpu", "inhibit", "below_timeout"},
			remove:   "inhibit",
			expected: []string{"high_cpu", "below_timeout"},
		},
		{
			name:     "remove non-existent",
			input:    []string{"high_cpu", "below_timeout"},
			remove:   "inhibit",
			expected: []string{"high_cpu", "below_timeout"},
		},
		{
			name:     "remove from empty",
			input:    []string{},
			remove:   "inhibit",
			expected: []string{},
		},
		{
			name:     "remove only element",
			input:    []string{"inhibit"},
			remove:   "inhibit",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeReason(tt.input, tt.remove)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
			}

			for i, expected := range tt.expected {
				if i >= len(result) || result[i] != expected {
					t.Errorf("Expected result[%d] = %s, got %v", i, expected, result)
				}
			}
		})
	}
}
