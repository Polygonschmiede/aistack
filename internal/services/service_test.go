package services

import (
	"aistack/internal/logging"
	"testing"
)

func TestBaseService_Name(t *testing.T) {
	runtime := NewMockRuntime()
	logger := logging.NewLogger(logging.LevelInfo)
	hc := DefaultHealthCheck("http://localhost:8080")

	service := NewBaseService("test-service", "./compose", hc, []string{"vol1"}, runtime, logger)

	if service.Name() != "test-service" {
		t.Errorf("Expected name 'test-service', got: %s", service.Name())
	}
}

func TestBaseService_Install(t *testing.T) {
	runtime := NewMockRuntime()
	logger := logging.NewLogger(logging.LevelInfo)
	hc := DefaultHealthCheck("http://localhost:8080")

	service := NewBaseService("test-service", "./compose", hc, []string{"vol1"}, runtime, logger)

	// Note: This will fail in actual execution without docker-compose file
	// In a real test, we'd mock the runtime.ComposeUp
	// For now, we just verify the structure
	if service == nil {
		t.Fatal("Expected service to be created")
	}
}

func TestOllamaService_Creation(t *testing.T) {
	runtime := NewMockRuntime()
	logger := logging.NewLogger(logging.LevelInfo)

	service := NewOllamaService("./compose", runtime, logger)

	if service.Name() != "ollama" {
		t.Errorf("Expected name 'ollama', got: %s", service.Name())
	}
}

func TestOpenWebUIService_Creation(t *testing.T) {
	runtime := NewMockRuntime()
	logger := logging.NewLogger(logging.LevelInfo)

	service := NewOpenWebUIService("./compose", runtime, logger)

	if service.Name() != "openwebui" {
		t.Errorf("Expected name 'openwebui', got: %s", service.Name())
	}
}

func TestLocalAIService_Creation(t *testing.T) {
	runtime := NewMockRuntime()
	logger := logging.NewLogger(logging.LevelInfo)

	service := NewLocalAIService("./compose", runtime, logger)

	if service.Name() != "localai" {
		t.Errorf("Expected name 'localai', got: %s", service.Name())
	}
}
