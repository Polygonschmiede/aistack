package services

import (
	"os"
	"path/filepath"
	"testing"

	"aistack/internal/logging"
)

func TestServiceUpdater_Update_NewImage(t *testing.T) {
	// Create temp directory for state
	tmpDir, err := os.MkdirTemp("", "aistack-update-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	mockRuntime := &MockRuntime{
		imageID:    "sha256:oldimage123",
		newImageID: "sha256:newimage456",
	}

	baseService := &BaseService{
		name:    "ollama",
		runtime: mockRuntime,
		logger:  logger,
	}

	healthCheck := &MockHealthCheck{
		shouldPass: true,
	}

	updater := NewServiceUpdater(baseService, mockRuntime, "ollama/ollama:latest", healthCheck, logger, tmpDir, nil)

	// Run update
	if err := updater.Update(); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify plan was saved
	planPath := filepath.Join(tmpDir, "ollama_update_plan.json")
	if _, err := os.Stat(planPath); os.IsNotExist(err) {
		t.Error("Update plan was not saved")
	}

	// Load plan and verify
	plan, err := LoadUpdatePlan("ollama", tmpDir)
	if err != nil {
		t.Fatalf("Failed to load plan: %v", err)
	}

	if plan == nil {
		t.Fatal("Plan is nil")
	}

	if plan.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", plan.Status)
	}

	if plan.HealthAfterSwap != string(HealthGreen) {
		t.Errorf("Expected health 'green', got '%s'", plan.HealthAfterSwap)
	}
}

func TestServiceUpdater_Update_HealthFails(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-update-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	mockRuntime := &MockRuntime{
		imageID:    "sha256:oldimage123",
		newImageID: "sha256:newimage456",
	}

	baseService := &BaseService{
		name:    "ollama",
		runtime: mockRuntime,
		logger:  logger,
	}

	// Health check that fails after update but succeeds after rollback
	healthCheck := &MockHealthCheck{
		shouldPass:     false, // Fails initially
		passAfterCalls: 1,     // Pass after first call (rollback)
		callCount:      0,
	}

	updater := NewServiceUpdater(baseService, mockRuntime, "ollama/ollama:latest", healthCheck, logger, tmpDir, nil)

	// Run update - should fail due to health check and rollback
	if err := updater.Update(); err == nil {
		t.Error("Expected update to fail due to health check, but it succeeded")
	}

	// Verify plan shows rollback
	plan, err := LoadUpdatePlan("ollama", tmpDir)
	if err != nil {
		t.Fatalf("Failed to load plan: %v", err)
	}

	if plan == nil {
		t.Fatal("Plan is nil")
	}

	// After rollback with successful health check, status should be "rolled_back"
	if plan.Status != "rolled_back" {
		t.Errorf("Expected status 'rolled_back', got '%s'", plan.Status)
	}
}

func TestServiceUpdater_Update_NoChange(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-update-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	mockRuntime := &MockRuntime{
		imageID:    "sha256:same123",
		newImageID: "sha256:same123", // Same image
	}

	baseService := &BaseService{
		name:    "ollama",
		runtime: mockRuntime,
		logger:  logger,
	}

	healthCheck := &MockHealthCheck{
		shouldPass: true,
	}

	updater := NewServiceUpdater(baseService, mockRuntime, "ollama/ollama:latest", healthCheck, logger, tmpDir, nil)

	// Run update
	if err := updater.Update(); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify plan shows unchanged
	plan, err := LoadUpdatePlan("ollama", tmpDir)
	if err != nil {
		t.Fatalf("Failed to load plan: %v", err)
	}

	if plan == nil {
		t.Fatal("Plan is nil")
	}

	if plan.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", plan.Status)
	}

	if plan.HealthAfterSwap != "unchanged" {
		t.Errorf("Expected health 'unchanged', got '%s'", plan.HealthAfterSwap)
	}
}

func TestLoadUpdatePlan_NotExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-update-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Try to load non-existent plan
	plan, err := LoadUpdatePlan("nonexistent", tmpDir)
	if err != nil {
		t.Errorf("Expected no error for missing plan, got: %v", err)
	}

	if plan != nil {
		t.Error("Expected nil plan for non-existent service")
	}
}

// MockHealthCheck for testing
type MockHealthCheck struct {
	shouldPass     bool
	passAfterCalls int
	callCount      int
}

func (m *MockHealthCheck) Check() (HealthStatus, error) {
	m.callCount++
	if m.passAfterCalls > 0 && m.callCount > m.passAfterCalls {
		return HealthGreen, nil
	}
	if m.shouldPass {
		return HealthGreen, nil
	}
	return HealthRed, nil
}
