package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"aistack/internal/logging"
)

// UpdatePlan tracks an update operation for rollback capability
// Story T-018: Ollama Update & Rollback (Service-specific)
type UpdatePlan struct {
	ServiceName     string    `json:"service_name"`
	OldImageID      string    `json:"old_image_id"`
	NewImage        string    `json:"new_image"`
	NewImageID      string    `json:"new_image_id,omitempty"`
	PullReference   string    `json:"pull_reference,omitempty"`
	StartedAt       time.Time `json:"started_at"`
	CompletedAt     time.Time `json:"completed_at,omitempty"`
	Status          string    `json:"status"` // pending, completed, rolled_back, failed
	HealthAfterSwap string    `json:"health_after_swap,omitempty"`
}

// ServiceUpdater handles service updates with rollback capability
type ServiceUpdater struct {
	service     Service
	runtime     Runtime
	logger      *logging.Logger
	stateDir    string
	imageName   string
	healthCheck HealthChecker
	imageLock   *VersionLock
}

// NewServiceUpdater creates a new service updater
func NewServiceUpdater(service Service, runtime Runtime, imageName string, healthCheck HealthChecker, logger *logging.Logger, stateDir string, lock *VersionLock) *ServiceUpdater {
	return &ServiceUpdater{
		service:     service,
		runtime:     runtime,
		logger:      logger,
		stateDir:    stateDir,
		imageName:   imageName,
		healthCheck: healthCheck,
		imageLock:   lock,
	}
}

// Update performs a service update with health validation and rollback on failure
// Story T-018: Implements update with health-gating and automatic rollback
func (u *ServiceUpdater) Update() error {
	ref, err := u.resolveImageReference()
	if err != nil {
		return err
	}

	u.logger.Info("service.update.start", fmt.Sprintf("Starting update for %s", u.service.Name()), map[string]interface{}{
		"service": u.service.Name(),
		"image":   ref.TagRef,
		"pull":    ref.PullRef,
	})

	// Create update plan
	plan := &UpdatePlan{
		ServiceName:   u.service.Name(),
		NewImage:      ref.TagRef,
		PullReference: ref.PullRef,
		StartedAt:     time.Now(),
		Status:        "pending",
	}

	// Get current image ID for rollback
	oldImageID, err := u.runtime.GetImageID(ref.TagRef)
	if err != nil {
		// Service might not be installed yet, this is OK
		u.logger.Warn("service.update.no_old_image", "No existing image found", map[string]interface{}{
			"service": u.service.Name(),
		})
		oldImageID = ""
	}
	plan.OldImageID = oldImageID

	// Save plan for potential rollback
	if err := u.savePlan(plan); err != nil {
		return fmt.Errorf("failed to save update plan: %w", err)
	}

	// Pull new image
	u.logger.Info("service.update.pull", fmt.Sprintf("Pulling new image: %s", ref.PullRef), map[string]interface{}{
		"service": u.service.Name(),
		"image":   ref.PullRef,
	})

	if err := u.runtime.PullImage(ref.PullRef); err != nil {
		plan.Status = "failed"
		plan.CompletedAt = time.Now()
		_ = u.savePlan(plan)
		return fmt.Errorf("failed to pull image: %w", err)
	}

	if ref.PullRef != ref.TagRef {
		if err := u.runtime.TagImage(ref.PullRef, ref.TagRef); err != nil {
			plan.Status = "failed"
			plan.CompletedAt = time.Now()
			_ = u.savePlan(plan)
			return fmt.Errorf("failed to tag image %s as %s: %w", ref.PullRef, ref.TagRef, err)
		}
	}

	// Get new image ID
	newImageID, err := u.runtime.GetImageID(ref.TagRef)
	if err != nil {
		plan.Status = "failed"
		plan.CompletedAt = time.Now()
		_ = u.savePlan(plan)
		return fmt.Errorf("failed to get new image ID: %w", err)
	}
	plan.NewImageID = newImageID

	// Check if image actually changed
	if oldImageID != "" && oldImageID == newImageID {
		u.logger.Info("service.update.no_change", "Image unchanged, no update needed", map[string]interface{}{
			"service":  u.service.Name(),
			"image_id": newImageID,
		})
		plan.Status = "completed"
		plan.HealthAfterSwap = "unchanged"
		plan.CompletedAt = time.Now()
		_ = u.savePlan(plan)
		return nil
	}

	// Restart service with new image
	u.logger.Info("service.update.restart", "Restarting service with new image", map[string]interface{}{
		"service": u.service.Name(),
	})

	if err := u.service.Stop(); err != nil {
		u.logger.Warn("service.update.stop_error", "Error stopping service", map[string]interface{}{
			"service": u.service.Name(),
			"error":   err.Error(),
		})
	}

	if err := u.service.Start(); err != nil {
		plan.Status = "failed"
		plan.CompletedAt = time.Now()
		_ = u.savePlan(plan)
		return fmt.Errorf("failed to start service with new image: %w", err)
	}

	// Wait a bit for service to initialize
	time.Sleep(5 * time.Second)

	// Perform health check
	u.logger.Info("service.update.health_check", "Performing health check", map[string]interface{}{
		"service": u.service.Name(),
	})

	health, err := u.healthCheck.Check()
	plan.HealthAfterSwap = string(health)

	if err != nil || health == HealthRed {
		u.logger.Error("service.update.health_failed", "Health check failed after update", map[string]interface{}{
			"service": u.service.Name(),
			"health":  health,
			"error":   err,
		})

		// Attempt rollback
		if rollbackErr := u.Rollback(plan); rollbackErr != nil {
			plan.Status = "failed"
			plan.CompletedAt = time.Now()
			_ = u.savePlan(plan)
			return fmt.Errorf("update failed and rollback also failed: health_err=%w, rollback_err=%v", err, rollbackErr)
		}

		plan.Status = "rolled_back"
		plan.CompletedAt = time.Now()
		_ = u.savePlan(plan)
		return fmt.Errorf("update failed health check, rolled back to previous version")
	}

	// Update succeeded
	u.logger.Info("service.update.success", "Update completed successfully", map[string]interface{}{
		"service":      u.service.Name(),
		"new_image_id": newImageID,
		"health":       health,
	})

	plan.Status = "completed"
	plan.CompletedAt = time.Now()
	_ = u.savePlan(plan)

	return nil
}

// Rollback rolls back to the previous image version
func (u *ServiceUpdater) Rollback(plan *UpdatePlan) error {
	if plan.OldImageID == "" {
		return fmt.Errorf("no previous image to rollback to")
	}

	u.logger.Info("service.update.rollback", "Rolling back to previous image", map[string]interface{}{
		"service":      u.service.Name(),
		"old_image_id": plan.OldImageID,
	})

	// Stop current service
	if err := u.service.Stop(); err != nil {
		u.logger.Warn("service.update.rollback.stop_error", "Error stopping service during rollback", map[string]interface{}{
			"service": u.service.Name(),
			"error":   err.Error(),
		})
	}

	if err := u.runtime.TagImage(plan.OldImageID, plan.NewImage); err != nil {
		return fmt.Errorf("failed to retag image during rollback: %w", err)
	}

	// Start service (will use old image)
	if err := u.service.Start(); err != nil {
		return fmt.Errorf("failed to start service during rollback: %w", err)
	}

	// Wait for initialization
	time.Sleep(5 * time.Second)

	// Verify health
	health, err := u.healthCheck.Check()
	if err != nil || health == HealthRed {
		return fmt.Errorf("rollback failed health check: %w", err)
	}

	u.logger.Info("service.update.rollback.success", "Rollback completed successfully", map[string]interface{}{
		"service": u.service.Name(),
		"health":  health,
	})

	return nil
}

// savePlan saves the update plan to disk
func (u *ServiceUpdater) savePlan(plan *UpdatePlan) error {
	// Ensure state directory exists
	if err := os.MkdirAll(u.stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	planPath := filepath.Join(u.stateDir, fmt.Sprintf("%s_update_plan.json", u.service.Name()))

	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	if err := os.WriteFile(planPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write plan: %w", err)
	}

	return nil
}

func (u *ServiceUpdater) resolveImageReference() (ImageReference, error) {
	if u.imageLock == nil {
		return ImageReference{PullRef: u.imageName, TagRef: u.imageName}, nil
	}

	ref, err := u.imageLock.Resolve(u.service.Name(), u.imageName)
	if err != nil {
		return ImageReference{}, err
	}

	if ref.PullRef == "" {
		ref.PullRef = u.imageName
	}
	if ref.TagRef == "" {
		ref.TagRef = u.imageName
	}

	return ref, nil
}

// EnforceImagePolicy ensures the configured image reference is present and tagged
func (u *ServiceUpdater) EnforceImagePolicy() error {
	ref, err := u.resolveImageReference()
	if err != nil {
		return err
	}

	if ref.PullRef == ref.TagRef {
		return nil
	}

	if err := u.runtime.PullImage(ref.PullRef); err != nil {
		return fmt.Errorf("failed to pull image %s: %w", ref.PullRef, err)
	}

	if err := u.runtime.TagImage(ref.PullRef, ref.TagRef); err != nil {
		return fmt.Errorf("failed to tag image %s as %s: %w", ref.PullRef, ref.TagRef, err)
	}

	return nil
}

// LoadUpdatePlan loads the most recent update plan for a service
func LoadUpdatePlan(serviceName, stateDir string) (*UpdatePlan, error) {
	planPath := filepath.Join(stateDir, fmt.Sprintf("%s_update_plan.json", serviceName))

	data, err := os.ReadFile(planPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No plan exists, this is OK
		}
		return nil, fmt.Errorf("failed to read plan: %w", err)
	}

	var plan UpdatePlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}

	return &plan, nil
}
