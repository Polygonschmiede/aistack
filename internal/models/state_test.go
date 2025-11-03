package models

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"aistack/internal/logging"
)

func TestStateManager_SaveAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "models-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewStateManager(tmpDir, ProviderOllama, logger)

	// Create test state
	state := &ModelsState{
		Provider: ProviderOllama,
		Items: []ModelInfo{
			{
				Name:     "test-model-1",
				Size:     1024,
				Path:     "/path/to/model1",
				LastUsed: time.Now().UTC(),
			},
			{
				Name:     "test-model-2",
				Size:     2048,
				Path:     "/path/to/model2",
				LastUsed: time.Now().UTC().Add(-24 * time.Hour),
			},
		},
		Updated: time.Now().UTC(),
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
	if loaded.Provider != ProviderOllama {
		t.Errorf("Expected provider ollama, got %s", loaded.Provider)
	}

	if len(loaded.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(loaded.Items))
	}

	if loaded.Items[0].Name != "test-model-1" {
		t.Errorf("Expected test-model-1, got %s", loaded.Items[0].Name)
	}
}

func TestStateManager_LoadNonExistent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "models-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewStateManager(tmpDir, ProviderOllama, logger)

	// Load non-existent state should return empty state
	state, err := manager.Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if state.Provider != ProviderOllama {
		t.Errorf("Expected provider ollama, got %s", state.Provider)
	}

	if len(state.Items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(state.Items))
	}
}

func TestStateManager_AddModel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "models-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewStateManager(tmpDir, ProviderOllama, logger)

	model := ModelInfo{
		Name:     "new-model",
		Size:     3072,
		Path:     "/path/to/new",
		LastUsed: time.Now().UTC(),
	}

	if err := manager.AddModel(model); err != nil {
		t.Fatalf("Failed to add model: %v", err)
	}

	// Load and verify
	state, err := manager.Load()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if len(state.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(state.Items))
	}

	if state.Items[0].Name != "new-model" {
		t.Errorf("Expected new-model, got %s", state.Items[0].Name)
	}
}

func TestStateManager_UpdateModel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "models-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewStateManager(tmpDir, ProviderOllama, logger)

	// Add initial model
	model := ModelInfo{
		Name:     "update-test",
		Size:     1000,
		Path:     "/old/path",
		LastUsed: time.Now().UTC(),
	}

	if err := manager.AddModel(model); err != nil {
		t.Fatalf("Failed to add model: %v", err)
	}

	// Update model
	updatedModel := ModelInfo{
		Name:     "update-test",
		Size:     2000,
		Path:     "/new/path",
		LastUsed: time.Now().UTC().Add(time.Hour),
	}

	if err := manager.AddModel(updatedModel); err != nil {
		t.Fatalf("Failed to update model: %v", err)
	}

	// Load and verify
	state, err := manager.Load()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if len(state.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(state.Items))
	}

	if state.Items[0].Size != 2000 {
		t.Errorf("Expected size 2000, got %d", state.Items[0].Size)
	}

	if state.Items[0].Path != "/new/path" {
		t.Errorf("Expected path /new/path, got %s", state.Items[0].Path)
	}
}

func TestStateManager_RemoveModel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "models-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewStateManager(tmpDir, ProviderOllama, logger)

	// Add models
	models := []ModelInfo{
		{Name: "model1", Size: 100, LastUsed: time.Now().UTC()},
		{Name: "model2", Size: 200, LastUsed: time.Now().UTC()},
		{Name: "model3", Size: 300, LastUsed: time.Now().UTC()},
	}

	for _, model := range models {
		if err := manager.AddModel(model); err != nil {
			t.Fatalf("Failed to add model: %v", err)
		}
	}

	// Remove middle model
	if err := manager.RemoveModel("model2"); err != nil {
		t.Fatalf("Failed to remove model: %v", err)
	}

	// Load and verify
	state, err := manager.Load()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if len(state.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(state.Items))
	}

	// Verify remaining models
	names := make(map[string]bool)
	for _, item := range state.Items {
		names[item.Name] = true
	}

	if !names["model1"] || !names["model3"] {
		t.Errorf("Expected model1 and model3, got %v", names)
	}

	if names["model2"] {
		t.Errorf("model2 should have been removed")
	}
}

func TestStateManager_GetStats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "models-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewStateManager(tmpDir, ProviderOllama, logger)

	now := time.Now().UTC()
	oldTime := now.Add(-48 * time.Hour)

	// Add models with different ages
	models := []ModelInfo{
		{Name: "new", Size: 1000, LastUsed: now},
		{Name: "old", Size: 2000, LastUsed: oldTime},
		{Name: "medium", Size: 1500, LastUsed: now.Add(-24 * time.Hour)},
	}

	for _, model := range models {
		if err := manager.AddModel(model); err != nil {
			t.Fatalf("Failed to add model: %v", err)
		}
	}

	// Get stats
	stats, err := manager.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Provider != ProviderOllama {
		t.Errorf("Expected provider ollama, got %s", stats.Provider)
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

	if stats.OldestModel.Name != "old" {
		t.Errorf("Expected oldest model 'old', got %s", stats.OldestModel.Name)
	}
}

func TestStateManager_GetOldestModels(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "models-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewStateManager(tmpDir, ProviderOllama, logger)

	now := time.Now().UTC()

	// Add models with different ages
	models := []ModelInfo{
		{Name: "newest", Size: 100, LastUsed: now},
		{Name: "oldest", Size: 300, LastUsed: now.Add(-72 * time.Hour)},
		{Name: "middle", Size: 200, LastUsed: now.Add(-24 * time.Hour)},
	}

	for _, model := range models {
		if err := manager.AddModel(model); err != nil {
			t.Fatalf("Failed to add model: %v", err)
		}
	}

	// Get oldest models
	oldest, err := manager.GetOldestModels()
	if err != nil {
		t.Fatalf("Failed to get oldest models: %v", err)
	}

	if len(oldest) != 3 {
		t.Errorf("Expected 3 models, got %d", len(oldest))
	}

	// Verify sorting (oldest first)
	if oldest[0].Name != "oldest" {
		t.Errorf("Expected first model to be 'oldest', got %s", oldest[0].Name)
	}

	if oldest[1].Name != "middle" {
		t.Errorf("Expected second model to be 'middle', got %s", oldest[1].Name)
	}

	if oldest[2].Name != "newest" {
		t.Errorf("Expected third model to be 'newest', got %s", oldest[2].Name)
	}
}

func TestStateManager_AtomicWrite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "models-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewStateManager(tmpDir, ProviderOllama, logger)

	state := &ModelsState{
		Provider: ProviderOllama,
		Items: []ModelInfo{
			{Name: "test", Size: 100, LastUsed: time.Now().UTC()},
		},
		Updated: time.Now().UTC(),
	}

	// Save state
	if err := manager.Save(state); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Verify temp file doesn't exist
	tmpPath := filepath.Join(tmpDir, "ollama_models_state.json.tmp")
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("Temp file should not exist after save")
	}

	// Verify actual file exists
	statePath := filepath.Join(tmpDir, "ollama_models_state.json")
	if _, err := os.Stat(statePath); err != nil {
		t.Errorf("State file should exist: %v", err)
	}
}

func TestProvider_IsValid(t *testing.T) {
	tests := []struct {
		provider Provider
		valid    bool
	}{
		{ProviderOllama, true},
		{ProviderLocalAI, true},
		{Provider("invalid"), false},
		{Provider(""), false},
	}

	for _, tt := range tests {
		result := tt.provider.IsValid()
		if result != tt.valid {
			t.Errorf("Provider %s: expected IsValid()=%v, got %v", tt.provider, tt.valid, result)
		}
	}
}
