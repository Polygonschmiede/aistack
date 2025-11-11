//go:build linux

package suspend

import (
	"fmt"
	"os/exec"
	"time"

	"aistack/internal/logging"
)

// Executor handles suspend logic and execution
type Executor struct {
	detector *Detector
	manager  *Manager
	logger   *logging.Logger
	dryRun   bool // If true, log but don't actually suspend
}

// NewExecutor creates a new suspend executor
func NewExecutor(logger *logging.Logger, dryRun bool) *Executor {
	return &Executor{
		detector: NewDetector(logger),
		manager:  NewManager(logger),
		logger:   logger,
		dryRun:   dryRun,
	}
}

// CheckAndSuspend is the main entry point for suspend checking
// It should be called periodically (e.g. every 60 seconds by systemd timer)
func (e *Executor) CheckAndSuspend() error {
	e.logger.Debug("suspend.check.start", "Starting suspend check", nil)

	// Load state
	state, err := e.manager.LoadState()
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	// Check if suspend is enabled
	if !state.Enabled {
		e.logger.Debug("suspend.disabled", "Auto-suspend is disabled, skipping check", nil)
		return nil
	}

	// Detect current activity
	status, err := e.detector.DetectActivity()
	if err != nil {
		return fmt.Errorf("detect activity: %w", err)
	}

	// If system is active, update last_active_timestamp
	if !status.IsIdle {
		e.logger.Info("suspend.activity_detected", "System is active, resetting idle timer", map[string]interface{}{
			"cpu_percent": status.CPUPercent,
			"gpu_percent": status.GPUPercent,
		})

		state.LastActiveTimestamp = time.Now().Unix()
		if err := e.manager.SaveState(state); err != nil {
			return fmt.Errorf("save state: %w", err)
		}

		return nil
	}

	// System is idle, check if timeout has been reached
	idleDuration := e.manager.GetIdleDuration(state)
	idleSeconds := int(idleDuration.Seconds())

	if !e.manager.ShouldSuspend(state) {
		remainingSeconds := IdleTimeoutSeconds - idleSeconds
		e.logger.Info("suspend.idle_detected", "System is idle but timeout not reached", map[string]interface{}{
			"idle_seconds":      idleSeconds,
			"remaining_seconds": remainingSeconds,
			"cpu_percent":       status.CPUPercent,
			"gpu_percent":       status.GPUPercent,
		})
		return nil
	}

	// Timeout reached, execute suspend
	e.logger.Info("suspend.executing", "Idle timeout reached, suspending system", map[string]interface{}{
		"idle_seconds": idleSeconds,
		"cpu_percent":  status.CPUPercent,
		"gpu_percent":  status.GPUPercent,
	})

	if e.dryRun {
		e.logger.Info("suspend.dry_run", "Dry-run mode: would suspend now", nil)
		return nil
	}

	// Execute systemctl suspend
	cmd := exec.Command("systemctl", "suspend")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("execute suspend: %w", err)
	}

	e.logger.Info("suspend.done", "System suspend command executed", nil)

	return nil
}
