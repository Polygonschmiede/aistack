package models

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"aistack/internal/logging"
)

func TestLocalAIManager_List(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "localai-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	modelsDir := filepath.Join(tmpDir, "models")
	if mkErr := os.MkdirAll(modelsDir, 0755); mkErr != nil {
		t.Fatal(mkErr)
	}

	// Create test model files
	testFiles := []struct {
		name string
		size int
	}{
		{"model1.gguf", 1024},
		{"model2.gguf", 2048},
		{"model3.bin", 512},
	}

	for _, tf := range testFiles {
		path := filepath.Join(modelsDir, tf.name)
		data := make([]byte, tf.size)
		if writeErr := os.WriteFile(path, data, 0644); writeErr != nil {
			t.Fatal(writeErr)
		}
	}

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewLocalAIManager(tmpDir, modelsDir, logger)

	models, err := manager.List()
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}

	if len(models) != 3 {
		t.Errorf("Expected 3 models, got %d", len(models))
	}

	// Verify model sizes
	sizeMap := make(map[string]int64)
	for _, model := range models {
		sizeMap[model.Name] = model.Size
	}

	if sizeMap["model1.gguf"] != 1024 {
		t.Errorf("Expected model1.gguf size 1024, got %d", sizeMap["model1.gguf"])
	}

	if sizeMap["model2.gguf"] != 2048 {
		t.Errorf("Expected model2.gguf size 2048, got %d", sizeMap["model2.gguf"])
	}

	if sizeMap["model3.bin"] != 512 {
		t.Errorf("Expected model3.bin size 512, got %d", sizeMap["model3.bin"])
	}
}

func TestLocalAIManager_ListEmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "localai-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	modelsDir := filepath.Join(tmpDir, "models")
	if mkErr := os.MkdirAll(modelsDir, 0755); mkErr != nil {
		t.Fatal(mkErr)
	}

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewLocalAIManager(tmpDir, modelsDir, logger)

	models, err := manager.List()
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}

	if len(models) != 0 {
		t.Errorf("Expected 0 models, got %d", len(models))
	}
}

func TestLocalAIManager_ListNonExistentDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "localai-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	modelsDir := filepath.Join(tmpDir, "nonexistent")

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewLocalAIManager(tmpDir, modelsDir, logger)

	models, err := manager.List()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(models) != 0 {
		t.Errorf("Expected 0 models, got %d", len(models))
	}
}

func TestLocalAIManager_Delete(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "localai-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	modelsDir := filepath.Join(tmpDir, "models")
	if mkErr := os.MkdirAll(modelsDir, 0755); mkErr != nil {
		t.Fatal(mkErr)
	}

	// Create test model file
	modelPath := filepath.Join(modelsDir, "test-model.gguf")
	if writeErr := os.WriteFile(modelPath, []byte("test data"), 0644); writeErr != nil {
		t.Fatal(writeErr)
	}

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewLocalAIManager(tmpDir, modelsDir, logger)

	// Delete model
	if delErr := manager.Delete("test-model.gguf"); delErr != nil {
		t.Fatalf("Failed to delete model: %v", delErr)
	}

	// Verify file is gone
	if _, statErr := os.Stat(modelPath); !os.IsNotExist(statErr) {
		t.Errorf("Model file should not exist after delete")
	}
}

func TestLocalAIManager_DeleteNonExistent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "localai-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	modelsDir := filepath.Join(tmpDir, "models")
	if mkErr := os.MkdirAll(modelsDir, 0755); mkErr != nil {
		t.Fatal(mkErr)
	}

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewLocalAIManager(tmpDir, modelsDir, logger)

	// Try to delete non-existent model
	err = manager.Delete("nonexistent.gguf")
	if err == nil {
		t.Errorf("Expected error when deleting non-existent model")
	}
}

func TestLocalAIManager_SyncState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "localai-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	modelsDir := filepath.Join(tmpDir, "models")
	if mkErr := os.MkdirAll(modelsDir, 0755); mkErr != nil {
		t.Fatal(mkErr)
	}

	// Create test model file
	modelPath := filepath.Join(modelsDir, "sync-test.gguf")
	if writeErr := os.WriteFile(modelPath, make([]byte, 1024), 0644); writeErr != nil {
		t.Fatal(writeErr)
	}

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewLocalAIManager(tmpDir, modelsDir, logger)

	// Sync state
	if err = manager.SyncState(); err != nil {
		t.Fatalf("Failed to sync state: %v", err)
	}

	// Load state and verify
	state, err := manager.stateManager.Load()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if len(state.Items) != 1 {
		t.Errorf("Expected 1 item in state, got %d", len(state.Items))
	}

	if state.Items[0].Name != "sync-test.gguf" {
		t.Errorf("Expected model name sync-test.gguf, got %s", state.Items[0].Name)
	}

	if state.Items[0].Size != 1024 {
		t.Errorf("Expected size 1024, got %d", state.Items[0].Size)
	}
}

func TestLocalAIManager_GetStats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "localai-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	modelsDir := filepath.Join(tmpDir, "models")
	if mkErr := os.MkdirAll(modelsDir, 0755); mkErr != nil {
		t.Fatal(mkErr)
	}

	// Create test model files
	testFiles := []struct {
		name string
		size int
	}{
		{"model1.gguf", 1000},
		{"model2.gguf", 2000},
		{"model3.gguf", 1500},
	}

	for _, tf := range testFiles {
		path := filepath.Join(modelsDir, tf.name)
		if writeErr := os.WriteFile(path, make([]byte, tf.size), 0644); writeErr != nil {
			t.Fatal(writeErr)
		}
	}

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewLocalAIManager(tmpDir, modelsDir, logger)

	// Get stats (which syncs state)
	stats, err := manager.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Provider != ProviderLocalAI {
		t.Errorf("Expected provider localai, got %s", stats.Provider)
	}

	if stats.ModelCount != 3 {
		t.Errorf("Expected 3 models, got %d", stats.ModelCount)
	}

	expectedTotal := int64(4500)
	if stats.TotalSize != expectedTotal {
		t.Errorf("Expected total size %d, got %d", expectedTotal, stats.TotalSize)
	}

	if stats.OldestModel == nil {
		t.Fatal("Expected oldest model, got nil")
	}
}

func TestLocalAIManager_EvictOldest(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "localai-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	modelsDir := filepath.Join(tmpDir, "models")
	if mkErr := os.MkdirAll(modelsDir, 0755); mkErr != nil {
		t.Fatal(mkErr)
	}

	// Create test model files with different modification times
	now := time.Now()
	testFiles := []struct {
		name    string
		size    int
		modTime time.Time
	}{
		{"newest.gguf", 1000, now},
		{"oldest.gguf", 3000, now.Add(-48 * time.Hour)},
		{"middle.gguf", 2000, now.Add(-24 * time.Hour)},
	}

	for _, tf := range testFiles {
		path := filepath.Join(modelsDir, tf.name)
		if writeErr := os.WriteFile(path, make([]byte, tf.size), 0644); writeErr != nil {
			t.Fatal(writeErr)
		}
		// Set modification time
		if chtimesErr := os.Chtimes(path, tf.modTime, tf.modTime); chtimesErr != nil {
			t.Fatal(chtimesErr)
		}
	}

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewLocalAIManager(tmpDir, modelsDir, logger)

	// Evict oldest
	evicted, err := manager.EvictOldest()
	if err != nil {
		t.Fatalf("Failed to evict oldest: %v", err)
	}

	if evicted.Name != "oldest.gguf" {
		t.Errorf("Expected to evict oldest.gguf, got %s", evicted.Name)
	}

	if evicted.Size != 3000 {
		t.Errorf("Expected evicted size 3000, got %d", evicted.Size)
	}

	// Verify file is gone
	oldestPath := filepath.Join(modelsDir, "oldest.gguf")
	if _, statErr := os.Stat(oldestPath); !os.IsNotExist(statErr) {
		t.Errorf("Evicted model file should not exist")
	}

	// Verify remaining models
	models, err := manager.List()
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models remaining, got %d", len(models))
	}
}

func TestLocalAIManager_UpdateLastUsed(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "localai-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	modelsDir := filepath.Join(tmpDir, "models")
	if mkErr := os.MkdirAll(modelsDir, 0755); mkErr != nil {
		t.Fatal(mkErr)
	}

	// Create test model file
	modelPath := filepath.Join(modelsDir, "test.gguf")
	if writeErr := os.WriteFile(modelPath, []byte("test"), 0644); writeErr != nil {
		t.Fatal(writeErr)
	}

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewLocalAIManager(tmpDir, modelsDir, logger)

	// Sync state first
	if err = manager.SyncState(); err != nil {
		t.Fatal(err)
	}

	// Get initial last used
	state, err := manager.stateManager.Load()
	if err != nil {
		t.Fatal(err)
	}

	initialLastUsed := state.Items[0].LastUsed

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Update last used
	if err = manager.UpdateLastUsed("test.gguf"); err != nil {
		t.Fatalf("Failed to update last used: %v", err)
	}

	// Load and verify
	state, err = manager.stateManager.Load()
	if err != nil {
		t.Fatal(err)
	}

	updatedLastUsed := state.Items[0].LastUsed

	if !updatedLastUsed.After(initialLastUsed) {
		t.Errorf("LastUsed should be updated to a later time")
	}
}
