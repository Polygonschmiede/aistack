package services

import (
	"os"
	"path/filepath"
	"testing"

	"aistack/internal/logging"
)

func TestDefaultUIBinding(t *testing.T) {
	binding := DefaultUIBinding()

	if binding == nil {
		t.Fatal("Expected non-nil binding")
	}

	if binding.ActiveBackend != BackendOllama {
		t.Errorf("Expected default backend Ollama, got %s", binding.ActiveBackend)
	}

	if binding.URL != "http://aistack-ollama:11434" {
		t.Errorf("Expected Ollama URL, got %s", binding.URL)
	}
}

func TestBackendBindingManager_GetBinding_NotExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-binding-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewBackendBindingManager(tmpDir, logger)

	// When no binding exists, should return default
	binding, err := manager.GetBinding()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if binding.ActiveBackend != BackendOllama {
		t.Errorf("Expected default backend Ollama, got %s", binding.ActiveBackend)
	}
}

func TestBackendBindingManager_SetBinding_Ollama(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-binding-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewBackendBindingManager(tmpDir, logger)

	// Set binding to Ollama
	if err = manager.SetBinding(BackendOllama); err != nil {
		t.Fatalf("SetBinding failed: %v", err)
	}

	// Verify file was created
	bindingPath := filepath.Join(tmpDir, "ui_binding.json")
	if _, err := os.Stat(bindingPath); os.IsNotExist(err) {
		t.Error("Binding file was not created")
	}

	// Verify binding content
	binding, err := manager.GetBinding()
	if err != nil {
		t.Fatalf("GetBinding failed: %v", err)
	}

	if binding.ActiveBackend != BackendOllama {
		t.Errorf("Expected backend Ollama, got %s", binding.ActiveBackend)
	}

	if binding.URL != "http://aistack-ollama:11434" {
		t.Errorf("Expected Ollama URL, got %s", binding.URL)
	}
}

func TestBackendBindingManager_SetBinding_LocalAI(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-binding-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewBackendBindingManager(tmpDir, logger)

	// Set binding to LocalAI
	if err = manager.SetBinding(BackendLocalAI); err != nil {
		t.Fatalf("SetBinding failed: %v", err)
	}

	// Verify binding content
	binding, err := manager.GetBinding()
	if err != nil {
		t.Fatalf("GetBinding failed: %v", err)
	}

	if binding.ActiveBackend != BackendLocalAI {
		t.Errorf("Expected backend LocalAI, got %s", binding.ActiveBackend)
	}

	if binding.URL != "http://aistack-localai:8080" {
		t.Errorf("Expected LocalAI URL, got %s", binding.URL)
	}
}

func TestBackendBindingManager_SetBinding_Invalid(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-binding-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewBackendBindingManager(tmpDir, logger)

	// Try to set invalid backend
	err = manager.SetBinding("invalid")
	if err == nil {
		t.Error("Expected error for invalid backend, got nil")
	}
}

func TestBackendBindingManager_SwitchBackend(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-binding-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewBackendBindingManager(tmpDir, logger)

	// Initial state is Ollama (default)
	binding, err := manager.GetBinding()
	if err != nil {
		t.Fatal(err)
	}
	if binding.ActiveBackend != BackendOllama {
		t.Errorf("Expected initial backend Ollama, got %s", binding.ActiveBackend)
	}

	// Switch to LocalAI
	oldBackend, err := manager.SwitchBackend(BackendLocalAI)
	if err != nil {
		t.Fatalf("SwitchBackend failed: %v", err)
	}

	if oldBackend != BackendOllama {
		t.Errorf("Expected old backend Ollama, got %s", oldBackend)
	}

	// Verify new binding
	binding, err = manager.GetBinding()
	if err != nil {
		t.Fatal(err)
	}
	if binding.ActiveBackend != BackendLocalAI {
		t.Errorf("Expected backend LocalAI, got %s", binding.ActiveBackend)
	}

	// Switch back to Ollama
	oldBackend, err = manager.SwitchBackend(BackendOllama)
	if err != nil {
		t.Fatalf("SwitchBackend failed: %v", err)
	}

	if oldBackend != BackendLocalAI {
		t.Errorf("Expected old backend LocalAI, got %s", oldBackend)
	}

	// Verify binding
	binding, err = manager.GetBinding()
	if err != nil {
		t.Fatal(err)
	}
	if binding.ActiveBackend != BackendOllama {
		t.Errorf("Expected backend Ollama, got %s", binding.ActiveBackend)
	}
}

func TestBackendBindingManager_SwitchBackend_NoChange(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-binding-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewBackendBindingManager(tmpDir, logger)

	// Set to Ollama
	if err = manager.SetBinding(BackendOllama); err != nil {
		t.Fatal(err)
	}

	// Try to switch to Ollama again (no change)
	oldBackend, err := manager.SwitchBackend(BackendOllama)
	if err != nil {
		t.Fatalf("SwitchBackend failed: %v", err)
	}

	if oldBackend != BackendOllama {
		t.Errorf("Expected old backend Ollama, got %s", oldBackend)
	}

	// Verify binding unchanged
	binding, err := manager.GetBinding()
	if err != nil {
		t.Fatal(err)
	}
	if binding.ActiveBackend != BackendOllama {
		t.Errorf("Expected backend Ollama, got %s", binding.ActiveBackend)
	}
}

func TestGetBackendURL(t *testing.T) {
	tests := []struct {
		backend     BackendType
		expectedURL string
		expectError bool
	}{
		{BackendOllama, "http://aistack-ollama:11434", false},
		{BackendLocalAI, "http://aistack-localai:8080", false},
		{"invalid", "", true},
	}

	for _, test := range tests {
		url, err := GetBackendURL(test.backend)

		if test.expectError {
			if err == nil {
				t.Errorf("Expected error for backend %s, got nil", test.backend)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for backend %s: %v", test.backend, err)
			}
			if url != test.expectedURL {
				t.Errorf("Expected URL %s for backend %s, got %s", test.expectedURL, test.backend, url)
			}
		}
	}
}
