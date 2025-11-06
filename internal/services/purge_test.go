package services

import (
	"os"
	"path/filepath"
	"testing"

	"aistack/internal/logging"
)

func TestPurgeManager_PurgeAll(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()

	// Set state dir to temp
	origStateDir := os.Getenv("AISTACK_STATE_DIR")
	defer os.Setenv("AISTACK_STATE_DIR", origStateDir)
	os.Setenv("AISTACK_STATE_DIR", tmpDir)

	// Create mock manager
	logger := logging.NewLogger(logging.LevelInfo)
	runtime := NewMockRuntime()
	mockManager := &Manager{
		runtime: runtime,
		logger:  logger,
	}

	purgeManager := NewPurgeManager(mockManager, logger)

	// Create some files in state directory
	if err := os.WriteFile(filepath.Join(tmpDir, "test_state.json"), []byte("{}"), 0o640); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Perform purge (without removeConfigs)
	log, err := purgeManager.PurgeAll(false)
	if err != nil {
		t.Fatalf("PurgeAll() error = %v", err)
	}

	// Verify uninstall log
	if log.Target != "all" {
		t.Errorf("log.Target = %s, want 'all'", log.Target)
	}

	if log.KeepCache {
		t.Error("log.KeepCache = true, want false")
	}

	// Verify removed items
	if len(log.RemovedItems) == 0 {
		t.Error("log.RemovedItems is empty, expected some items")
	}

	t.Logf("Removed %d items", len(log.RemovedItems))
	t.Logf("Encountered %d errors", len(log.Errors))
}

func TestPurgeManager_CleanStateDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Set state dir to temp
	origStateDir := os.Getenv("AISTACK_STATE_DIR")
	defer os.Setenv("AISTACK_STATE_DIR", origStateDir)
	os.Setenv("AISTACK_STATE_DIR", tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	runtime := NewMockRuntime()
	mockManager := &Manager{
		runtime: runtime,
		logger:  logger,
	}

	purgeManager := NewPurgeManager(mockManager, logger)

	// Create test files
	testFiles := []string{
		"state1.json",
		"state2.json",
		"config.yaml",     // Should be preserved when removeAll=false
		"wol_config.json", // Should be preserved when removeAll=false
	}

	for _, file := range testFiles {
		path := filepath.Join(tmpDir, file)
		if err := os.WriteFile(path, []byte("test"), 0o640); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Clean without removing configs
	log := &UninstallLog{
		RemovedItems: []string{},
		Errors:       []string{},
	}

	if err := purgeManager.cleanStateDirectory(log, false); err != nil {
		t.Fatalf("cleanStateDirectory() error = %v", err)
	}

	// Verify state files removed but config files preserved
	if _, err := os.Stat(filepath.Join(tmpDir, "state1.json")); !os.IsNotExist(err) {
		t.Error("state1.json should be removed")
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "config.yaml")); os.IsNotExist(err) {
		t.Error("config.yaml should be preserved")
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "wol_config.json")); os.IsNotExist(err) {
		t.Error("wol_config.json should be preserved")
	}

	// Clean with removeAll=true
	// Recreate files
	for _, file := range testFiles {
		path := filepath.Join(tmpDir, file)
		if err := os.WriteFile(path, []byte("test"), 0o640); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	log = &UninstallLog{
		RemovedItems: []string{},
		Errors:       []string{},
	}

	if err := purgeManager.cleanStateDirectory(log, true); err != nil {
		t.Fatalf("cleanStateDirectory(removeAll=true) error = %v", err)
	}

	// Verify all files removed
	for _, file := range testFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("%s should be removed when removeAll=true", file)
		}
	}
}

func TestPurgeManager_VerifyClean(t *testing.T) {
	tmpDir := t.TempDir()

	// Set state dir to temp
	origStateDir := os.Getenv("AISTACK_STATE_DIR")
	defer os.Setenv("AISTACK_STATE_DIR", origStateDir)
	os.Setenv("AISTACK_STATE_DIR", tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	runtime := NewMockRuntime()
	mockManager := &Manager{
		runtime: runtime,
		logger:  logger,
	}

	purgeManager := NewPurgeManager(mockManager, logger)

	// Test with empty state dir (clean)
	isClean, leftovers := purgeManager.VerifyClean()
	if !isClean {
		t.Errorf("VerifyClean() = false, want true for empty system")
	}
	if len(leftovers) != 0 {
		t.Errorf("VerifyClean() leftovers = %d, want 0", len(leftovers))
	}

	// Create a leftover file
	if err := os.WriteFile(filepath.Join(tmpDir, "leftover.json"), []byte("test"), 0o640); err != nil {
		t.Fatalf("Failed to create leftover file: %v", err)
	}

	// Test with leftover (not clean)
	isClean, leftovers = purgeManager.VerifyClean()
	if isClean {
		t.Errorf("VerifyClean() = true, want false when leftovers exist")
	}
	if len(leftovers) == 0 {
		t.Errorf("VerifyClean() leftovers = 0, want > 0")
	}
}

func TestPurgeManager_SaveUninstallLog(t *testing.T) {
	tmpDir := t.TempDir()

	// Set state dir to temp
	origStateDir := os.Getenv("AISTACK_STATE_DIR")
	defer os.Setenv("AISTACK_STATE_DIR", origStateDir)
	os.Setenv("AISTACK_STATE_DIR", tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	runtime := NewMockRuntime()
	mockManager := &Manager{
		runtime: runtime,
		logger:  logger,
	}

	purgeManager := NewPurgeManager(mockManager, logger)

	log := &UninstallLog{
		Target:       "test",
		KeepCache:    false,
		RemovedItems: []string{"item1", "item2"},
		Errors:       []string{},
	}

	logPath := filepath.Join(tmpDir, "uninstall_log.json")
	if err := purgeManager.SaveUninstallLog(log, logPath); err != nil {
		t.Fatalf("SaveUninstallLog() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Uninstall log file was not created")
	}

	// Verify file permissions
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("Failed to stat log file: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0o640 {
		t.Errorf("Log file permissions = %o, want 0640", perm)
	}
}

func TestCreateUninstallLogForService(t *testing.T) {
	log := CreateUninstallLogForService("ollama", true, []string{"container:ollama"}, []string{})

	if log.Target != "ollama" {
		t.Errorf("log.Target = %s, want 'ollama'", log.Target)
	}

	if !log.KeepCache {
		t.Error("log.KeepCache = false, want true")
	}

	if len(log.RemovedItems) != 1 {
		t.Errorf("log.RemovedItems length = %d, want 1", len(log.RemovedItems))
	}
}
