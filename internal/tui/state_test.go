package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"aistack/internal/logging"
)

func TestUIStateManager_SaveAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tui-state-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewUIStateManager(tmpDir, logger)

	// Create test state
	state := &UIState{
		CurrentScreen: ScreenStatus,
		Selection:     2,
		LastError:     "test error",
		Updated:       time.Now().UTC(),
	}

	// Save state
	if err := manager.Save(state); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Load state
	loaded, err := manager.Load()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Verify
	if loaded.CurrentScreen != ScreenStatus {
		t.Errorf("Expected screen status, got %s", loaded.CurrentScreen)
	}

	if loaded.Selection != 2 {
		t.Errorf("Expected selection 2, got %d", loaded.Selection)
	}

	if loaded.LastError != "test error" {
		t.Errorf("Expected error 'test error', got %s", loaded.LastError)
	}
}

func TestUIStateManager_LoadNonExistent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tui-state-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewUIStateManager(tmpDir, logger)

	// Load non-existent state should return default state
	state, err := manager.Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if state.CurrentScreen != ScreenMenu {
		t.Errorf("Expected default screen menu, got %s", state.CurrentScreen)
	}

	if state.Selection != 0 {
		t.Errorf("Expected default selection 0, got %d", state.Selection)
	}

	if state.LastError != "" {
		t.Errorf("Expected empty error, got %s", state.LastError)
	}
}

func TestUIStateManager_SaveError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tui-state-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewUIStateManager(tmpDir, logger)

	errorMsg := "connection failed"

	// Save error
	if err := manager.SaveError(errorMsg); err != nil {
		t.Fatalf("Failed to save error: %v", err)
	}

	// Load and verify
	state, err := manager.Load()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if state.LastError != errorMsg {
		t.Errorf("Expected error '%s', got '%s'", errorMsg, state.LastError)
	}
}

func TestUIStateManager_ClearError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tui-state-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewUIStateManager(tmpDir, logger)

	// Save state with error
	state := &UIState{
		CurrentScreen: ScreenMenu,
		Selection:     0,
		LastError:     "test error",
		Updated:       time.Now().UTC(),
	}

	if err := manager.Save(state); err != nil {
		t.Fatal(err)
	}

	// Clear error
	if err := manager.ClearError(); err != nil {
		t.Fatalf("Failed to clear error: %v", err)
	}

	// Load and verify
	loaded, err := manager.Load()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if loaded.LastError != "" {
		t.Errorf("Expected empty error, got '%s'", loaded.LastError)
	}
}

func TestUIStateManager_AtomicWrite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tui-state-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewUIStateManager(tmpDir, logger)

	state := &UIState{
		CurrentScreen: ScreenMenu,
		Selection:     0,
		LastError:     "",
		Updated:       time.Now().UTC(),
	}

	// Save state
	if err := manager.Save(state); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Verify temp file doesn't exist
	tmpPath := filepath.Join(tmpDir, "ui_state.json.tmp")
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("Temp file should not exist after save")
	}

	// Verify actual file exists
	statePath := filepath.Join(tmpDir, "ui_state.json")
	if _, err := os.Stat(statePath); err != nil {
		t.Errorf("State file should exist: %v", err)
	}
}

func TestDefaultMenuItems(t *testing.T) {
	items := DefaultMenuItems()

	if len(items) != 8 {
		t.Errorf("Expected 8 menu items, got %d", len(items))
	}

	// Verify first item
	if items[0].Key != "1" {
		t.Errorf("Expected first item key '1', got '%s'", items[0].Key)
	}

	if items[0].Screen != ScreenStatus {
		t.Errorf("Expected first item screen status, got %s", items[0].Screen)
	}

	// Verify help item
	lastItem := items[len(items)-1]
	if lastItem.Key != "?" {
		t.Errorf("Expected last item key '?', got '%s'", lastItem.Key)
	}

	if lastItem.Screen != ScreenHelp {
		t.Errorf("Expected last item screen help, got %s", lastItem.Screen)
	}
}

func TestScreenTypes(t *testing.T) {
	screens := []Screen{
		ScreenMenu,
		ScreenStatus,
		ScreenInstall,
		ScreenModels,
		ScreenPower,
		ScreenLogs,
		ScreenDiagnostics,
		ScreenSettings,
		ScreenHelp,
	}

	// Verify all screens are distinct strings
	screenMap := make(map[Screen]bool)
	for _, screen := range screens {
		if screenMap[screen] {
			t.Errorf("Duplicate screen: %s", screen)
		}
		screenMap[screen] = true

		// Verify non-empty
		if string(screen) == "" {
			t.Errorf("Screen should not be empty")
		}
	}

	if len(screenMap) != len(screens) {
		t.Errorf("Expected %d unique screens, got %d", len(screens), len(screenMap))
	}
}
