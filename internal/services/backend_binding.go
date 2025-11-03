package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"aistack/internal/logging"
)

// BackendType represents the type of backend service
type BackendType string

const (
	BackendOllama  BackendType = "ollama"
	BackendLocalAI BackendType = "localai"
)

// UIBinding represents the backend binding configuration for Open WebUI
// Story T-019: Backend-Switch (Ollama ↔ LocalAI)
type UIBinding struct {
	ActiveBackend BackendType `json:"active_backend"`
	URL           string      `json:"url"`
}

// DefaultUIBinding returns the default binding configuration (Ollama)
func DefaultUIBinding() *UIBinding {
	return &UIBinding{
		ActiveBackend: BackendOllama,
		URL:           "http://aistack-ollama:11434",
	}
}

// BackendBindingManager manages the backend binding state for Open WebUI
type BackendBindingManager struct {
	stateDir string
	logger   *logging.Logger
}

// NewBackendBindingManager creates a new backend binding manager
func NewBackendBindingManager(stateDir string, logger *logging.Logger) *BackendBindingManager {
	return &BackendBindingManager{
		stateDir: stateDir,
		logger:   logger,
	}
}

// GetBinding returns the current backend binding
func (m *BackendBindingManager) GetBinding() (*UIBinding, error) {
	bindingPath := m.getBindingPath()

	// If file doesn't exist, return default
	if _, err := os.Stat(bindingPath); os.IsNotExist(err) {
		return DefaultUIBinding(), nil
	}

	data, err := os.ReadFile(bindingPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read binding: %w", err)
	}

	var binding UIBinding
	if err := json.Unmarshal(data, &binding); err != nil {
		return nil, fmt.Errorf("failed to unmarshal binding: %w", err)
	}

	return &binding, nil
}

// SetBinding sets the backend binding and persists it
func (m *BackendBindingManager) SetBinding(backend BackendType) error {
	var url string
	switch backend {
	case BackendOllama:
		url = "http://aistack-ollama:11434"
	case BackendLocalAI:
		url = "http://aistack-localai:8080"
	default:
		return fmt.Errorf("invalid backend type: %s", backend)
	}

	binding := &UIBinding{
		ActiveBackend: backend,
		URL:           url,
	}

	// Ensure state directory exists
	if err := os.MkdirAll(m.stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(binding, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal binding: %w", err)
	}

	// Write to file
	bindingPath := m.getBindingPath()
	if err := os.WriteFile(bindingPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write binding: %w", err)
	}

	m.logger.Info("ui.backend.changed", "Backend binding updated", map[string]interface{}{
		"backend": backend,
		"url":     url,
	})

	return nil
}

// SwitchBackend switches the backend and returns the old and new backend types
func (m *BackendBindingManager) SwitchBackend(newBackend BackendType) (BackendType, error) {
	// Get current binding
	currentBinding, err := m.GetBinding()
	if err != nil {
		return "", fmt.Errorf("failed to get current binding: %w", err)
	}

	oldBackend := currentBinding.ActiveBackend

	// Don't switch if already on the requested backend
	if oldBackend == newBackend {
		m.logger.Info("ui.backend.no_change", "Backend already set", map[string]interface{}{
			"backend": newBackend,
		})
		return oldBackend, nil
	}

	// Set new binding
	if err := m.SetBinding(newBackend); err != nil {
		return "", fmt.Errorf("failed to set binding: %w", err)
	}

	m.logger.Info("ui.backend.switched", "Backend switched", map[string]interface{}{
		"from": oldBackend,
		"to":   newBackend,
	})

	return oldBackend, nil
}

// getBindingPath returns the path to the binding state file
func (m *BackendBindingManager) getBindingPath() string {
	return filepath.Join(m.stateDir, "ui_binding.json")
}

// GetBackendURL returns the URL for a given backend type
func GetBackendURL(backend BackendType) (string, error) {
	switch backend {
	case BackendOllama:
		return "http://aistack-ollama:11434", nil
	case BackendLocalAI:
		return "http://aistack-localai:8080", nil
	default:
		return "", fmt.Errorf("invalid backend type: %s", backend)
	}
}
