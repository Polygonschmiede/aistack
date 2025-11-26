package services

import (
	"fmt"

	"aistack/internal/fsutil"
	"aistack/internal/gpulock"
	"aistack/internal/logging"
)

const (
	// LocalAIImageName is the Docker image for LocalAI
	LocalAIImageName = "quay.io/go-skynet/local-ai:latest"
)

// LocalAIService manages the LocalAI container service
// Story T-008: Compose-Template: LocalAI Service (Health & Volume)
// Story T-021: GPU-Mutex (Dateisperre + Lease)
type LocalAIService struct {
	*BaseService
	updater  *ServiceUpdater
	registry *LocalAIModelsRegistry
	gpuLock  *gpulock.Manager
}

// NewLocalAIService creates a new LocalAI service
func NewLocalAIService(composeDir string, runtime Runtime, logger *logging.Logger, lock *VersionLock, gpuLock *gpulock.Manager) *LocalAIService {
	healthCheck := DefaultHealthCheck("http://localhost:8080/healthz")
	volumes := []string{"localai_models"}

	base := NewBaseService("localai", composeDir, healthCheck, volumes, runtime, logger)

	stateDir := fsutil.GetStateDir(defaultStateDir)
	updater := NewServiceUpdater(base, runtime, LocalAIImageName, healthCheck, logger, stateDir, lock)
	registry := NewLocalAIModelsRegistry(stateDir, logger)

	service := &LocalAIService{
		BaseService: base,
		updater:     updater,
		registry:    registry,
		gpuLock:     gpuLock,
	}

	base.SetPreStartHook(func() error {
		if err := updater.EnforceImagePolicy(); err != nil {
			return err
		}
		if err := registry.Ensure(); err != nil {
			return err
		}

		// Acquire GPU lock
		// Story T-021: GPU-Mutex (Dateisperre + Lease)
		if err := gpuLock.Acquire(gpulock.HolderLocalAI); err != nil {
			return fmt.Errorf("failed to acquire GPU lock: %w", err)
		}

		return nil
	})

	base.SetPostStopHook(func() error {
		// Release GPU lock
		return gpuLock.Release(gpulock.HolderLocalAI)
	})

	return service
}

// Update updates the LocalAI service to the latest version
func (s *LocalAIService) Update() error {
	if err := s.registry.Ensure(); err != nil {
		return err
	}
	return s.updater.Update()
}
