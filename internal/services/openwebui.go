package services

import (
	"fmt"
	"os"

	"aistack/internal/gpulock"
	"aistack/internal/logging"
)

const (
	// OpenWebUIImageName is the Docker image for Open WebUI
	OpenWebUIImageName = "ghcr.io/open-webui/open-webui:main"
)

// OpenWebUIService manages the Open WebUI container service
// Story T-007: Compose-Template: Open WebUI mit Backend-Binding
// Story T-019: Backend-Switch (Ollama ↔ LocalAI)
// Story T-021: GPU-Mutex (Dateisperre + Lease)
type OpenWebUIService struct {
	*BaseService
	updater        *ServiceUpdater
	bindingManager *BackendBindingManager
	gpuLock        *gpulock.Manager
}

// NewOpenWebUIService creates a new Open WebUI service
func NewOpenWebUIService(composeDir string, runtime Runtime, logger *logging.Logger, lock *VersionLock, gpuLock *gpulock.Manager) *OpenWebUIService {
	healthCheck := DefaultHealthCheck("http://localhost:3000/")
	volumes := []string{"openwebui_data"}

	base := NewBaseService("openwebui", composeDir, healthCheck, volumes, runtime, logger)

	// Get state directory from env or use default
	stateDir := os.Getenv("AISTACK_STATE_DIR")
	if stateDir == "" {
		stateDir = defaultStateDir
	}

	updater := NewServiceUpdater(base, runtime, OpenWebUIImageName, healthCheck, logger, stateDir, lock)
	bindingManager := NewBackendBindingManager(stateDir, logger)

	service := &OpenWebUIService{
		BaseService:    base,
		updater:        updater,
		bindingManager: bindingManager,
		gpuLock:        gpuLock,
	}

	base.SetPreStartHook(func() error {
		if err := updater.EnforceImagePolicy(); err != nil {
			return err
		}

		binding, err := bindingManager.GetBinding()
		if err != nil {
			return fmt.Errorf("failed to load backend binding: %w", err)
		}

		if err := os.Setenv("OLLAMA_BASE_URL", binding.URL); err != nil {
			return fmt.Errorf("failed to set OLLAMA_BASE_URL: %w", err)
		}

		// Acquire GPU lock
		// Story T-021: GPU-Mutex (Dateisperre + Lease)
		if err := gpuLock.Acquire(gpulock.HolderOpenWebUI); err != nil {
			return fmt.Errorf("failed to acquire GPU lock: %w", err)
		}

		return nil
	})

	base.SetPostStopHook(func() error {
		// Release GPU lock
		return gpuLock.Release(gpulock.HolderOpenWebUI)
	})

	return service
}

// Update updates the Open WebUI service to the latest version
func (s *OpenWebUIService) Update() error {
	return s.updater.Update()
}

// SwitchBackend switches the Open WebUI backend between Ollama and LocalAI
// Story T-019: Backend-Switch (Ollama ↔ LocalAI)
func (s *OpenWebUIService) SwitchBackend(backend BackendType) error {
	s.logger.Info("openwebui.backend.switch.start", "Switching backend", map[string]interface{}{
		"backend": backend,
	})

	// Switch backend in state
	oldBackend, err := s.bindingManager.SwitchBackend(backend)
	if err != nil {
		return fmt.Errorf("failed to switch backend: %w", err)
	}

	// If no change, don't restart
	if oldBackend == backend {
		s.logger.Info("openwebui.backend.switch.no_change", "Backend unchanged", map[string]interface{}{
			"backend": backend,
		})
		return nil
	}

	// Get backend URL
	backendURL, err := GetBackendURL(backend)
	if err != nil {
		return fmt.Errorf("failed to get backend URL: %w", err)
	}

	// Set environment variable for docker compose
	if err := os.Setenv("OLLAMA_BASE_URL", backendURL); err != nil {
		return fmt.Errorf("failed to set environment variable: %w", err)
	}

	// Restart service to apply new backend
	s.logger.Info("openwebui.backend.switch.restart", "Restarting with new backend", map[string]interface{}{
		"from": oldBackend,
		"to":   backend,
		"url":  backendURL,
	})

	if err := s.Stop(); err != nil {
		s.logger.Warn("openwebui.backend.switch.stop_error", "Error stopping service", map[string]interface{}{
			"error": err.Error(),
		})
	}

	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to start service with new backend: %w", err)
	}

	s.logger.Info("openwebui.backend.switch.success", "Backend switched successfully", map[string]interface{}{
		"from": oldBackend,
		"to":   backend,
		"url":  backendURL,
	})

	return nil
}

// GetCurrentBackend returns the currently configured backend
func (s *OpenWebUIService) GetCurrentBackend() (BackendType, error) {
	binding, err := s.bindingManager.GetBinding()
	if err != nil {
		return "", fmt.Errorf("failed to get binding: %w", err)
	}
	return binding.ActiveBackend, nil
}
