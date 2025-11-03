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
