package services

import (
	"aistack/internal/logging"
	"testing"
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
	manager.services["ollama"] = NewOllamaService("./compose", runtime, logger, nil)
	manager.services["openwebui"] = NewOpenWebUIService("./compose", runtime, logger, nil)
	manager.services["localai"] = NewLocalAIService("./compose", runtime, logger, nil)

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
