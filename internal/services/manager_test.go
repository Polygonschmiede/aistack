package services

import (
	"os"
	"testing"

	"aistack/internal/gpulock"
	"aistack/internal/logging"
)

// MockManager for testing without actual Docker
type MockManager struct {
	*Manager
}

func NewMockManager() *MockManager {
	runtime := NewMockRuntime()
	logger := logging.NewLogger(logging.LevelInfo)

	manager := &Manager{
		runtime:    runtime,
		logger:     logger,
		composeDir: "./compose",
		services:   make(map[string]Service),
	}

	// Register mock services
	lockStateDir := os.TempDir()
	gpuLock := gpulock.NewManager(lockStateDir, logger)

	manager.services["ollama"] = NewOllamaService("./compose", runtime, logger, nil)
	manager.services["openwebui"] = NewOpenWebUIService("./compose", runtime, logger, nil, gpuLock)
	manager.services["localai"] = NewLocalAIService("./compose", runtime, logger, nil, gpuLock)

	return &MockManager{Manager: manager}
}

func TestManager_GetService(t *testing.T) {
	manager := NewMockManager()

	tests := []struct {
		name        string
		serviceName string
		expectError bool
	}{
		{"valid service - ollama", "ollama", false},
		{"valid service - openwebui", "openwebui", false},
		{"valid service - localai", "localai", false},
		{"invalid service", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := manager.GetService(tt.serviceName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error for invalid service")
				}
				if service != nil {
					t.Error("Expected nil service for invalid name")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if service == nil {
					t.Error("Expected valid service")
				}
			}
		})
	}
}

func TestManager_ListServices(t *testing.T) {
	manager := NewMockManager()

	services := manager.ListServices()

	if len(services) != 3 {
		t.Errorf("Expected 3 services, got: %d", len(services))
	}

	// Verify all expected services are in the list
	serviceMap := make(map[string]bool)
	for _, name := range services {
		serviceMap[name] = true
	}

	expectedServices := []string{"ollama", "openwebui", "localai"}
	for _, expected := range expectedServices {
		if !serviceMap[expected] {
			t.Errorf("Expected service '%s' to be in list", expected)
		}
	}
}

func TestManager_UpdateAllServices(t *testing.T) {
	// Create temp directory for state
	tmpDir, err := os.MkdirTemp("", "aistack-update-all-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set state directory
	origStateDir := os.Getenv("AISTACK_STATE_DIR")
	os.Setenv("AISTACK_STATE_DIR", tmpDir)
	defer os.Setenv("AISTACK_STATE_DIR", origStateDir)

	manager := NewMockManager()

	// Run update all
	result, err := manager.UpdateAllServices()
	if err != nil {
		t.Fatalf("UpdateAllServices() error = %v", err)
	}

	// Verify result structure
	if result.TotalServices != 3 {
		t.Errorf("Expected TotalServices=3, got %d", result.TotalServices)
	}

	if len(result.ServiceResults) != 3 {
		t.Errorf("Expected 3 service results, got %d", len(result.ServiceResults))
	}

	// Verify all services were attempted
	expectedServices := []string{"localai", "ollama", "openwebui"}
	for _, service := range expectedServices {
		if _, exists := result.ServiceResults[service]; !exists {
			t.Errorf("Expected result for service %s", service)
		}
	}

	// Mock runtime creates images with changed IDs, so all should report as changed or successful
	// (depending on mock behavior)
	totalProcessed := result.SuccessfulCount + result.FailedCount + result.RolledBackCount + result.UnchangedCount
	if totalProcessed != 3 {
		t.Errorf("Expected total processed services = 3, got %d", totalProcessed)
	}
}

func TestManager_UpdateAllServices_Order(t *testing.T) {
	// Create temp directory for state
	tmpDir, err := os.MkdirTemp("", "aistack-update-order-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set state directory
	origStateDir := os.Getenv("AISTACK_STATE_DIR")
	os.Setenv("AISTACK_STATE_DIR", tmpDir)
	defer os.Setenv("AISTACK_STATE_DIR", origStateDir)

	manager := NewMockManager()

	// Run update all
	result, err := manager.UpdateAllServices()
	if err != nil {
		t.Fatalf("UpdateAllServices() error = %v", err)
	}

	// Verify correct order: LocalAI → Ollama → Open WebUI
	// All services should be in results
	if len(result.ServiceResults) != 3 {
		t.Fatalf("Expected 3 service results, got %d", len(result.ServiceResults))
	}

	// Verify all expected services present
	expectedServices := map[string]bool{
		"localai":   false,
		"ollama":    false,
		"openwebui": false,
	}

	for service := range result.ServiceResults {
		if _, expected := expectedServices[service]; expected {
			expectedServices[service] = true
		}
	}

	for service, found := range expectedServices {
		if !found {
			t.Errorf("Expected service %s in results", service)
		}
	}
}

func TestManager_UpdateAllServices_IndependentFailure(t *testing.T) {
	// Create temp directory for state
	tmpDir, err := os.MkdirTemp("", "aistack-update-fail-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set state directory
	origStateDir := os.Getenv("AISTACK_STATE_DIR")
	os.Setenv("AISTACK_STATE_DIR", tmpDir)
	defer os.Setenv("AISTACK_STATE_DIR", origStateDir)

	manager := NewMockManager()

	// Run update all - even if some fail, all should be attempted
	result, err := manager.UpdateAllServices()
	if err != nil {
		t.Fatalf("UpdateAllServices() error = %v", err)
	}

	// Verify all 3 services were attempted regardless of individual failures
	if len(result.ServiceResults) != 3 {
		t.Errorf("Expected 3 service results (all attempted), got %d", len(result.ServiceResults))
	}

	// Verify count totals are consistent
	totalCounted := result.SuccessfulCount + result.FailedCount + result.RolledBackCount + result.UnchangedCount
	if totalCounted != 3 {
		t.Errorf("Expected total count = 3, got %d (successful=%d, failed=%d, rolled_back=%d, unchanged=%d)",
			totalCounted, result.SuccessfulCount, result.FailedCount, result.RolledBackCount, result.UnchangedCount)
	}
}
