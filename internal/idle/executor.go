package idle

import (
	"fmt"
	"os/exec"
	"strings"

	"aistack/internal/logging"
)

// Executor handles suspend execution with inhibit checking
type Executor struct {
	config IdleConfig
	logger *logging.Logger
}

// NewExecutor creates a new suspend executor
func NewExecutor(config IdleConfig, logger *logging.Logger) *Executor {
	return &Executor{
		config: config,
		logger: logger,
	}
}

// Execute attempts to execute suspend with appropriate gate checks
func (e *Executor) Execute(state *IdleState) error {
	return e.ExecuteWithOptions(state, false)
}

// ExecuteWithOptions attempts to execute suspend with optional inhibitor bypass
func (e *Executor) ExecuteWithOptions(state *IdleState, ignoreInhibitors bool) error {
	// Check if any other gating reasons exist first (before checking inhibitors)
	if ignoreInhibitors {
		filtered := make([]string, 0, len(state.GatingReasons))
		for _, r := range state.GatingReasons {
			if r != GatingReasonInhibit {
				filtered = append(filtered, r)
			}
		}
		state.GatingReasons = filtered
	}

	if len(state.GatingReasons) > 0 {
		e.logger.Info("power.suspend.skipped", "Suspend skipped due to gating reasons", map[string]interface{}{
			"idle_for_s":     state.IdleForSeconds,
			"gating_reasons": state.GatingReasons,
		})
		return fmt.Errorf("suspend blocked by gating reasons: %s", strings.Join(state.GatingReasons, ", "))
	}

	// Check if suspend is enabled
	if !e.config.EnableSuspend {
		e.logger.Info("power.suspend.skipped", "Suspend skipped (dry-run mode)", map[string]interface{}{
			"idle_for_s": state.IdleForSeconds,
			"reason":     "dry_run",
		})
		return nil
	}

	// Check for inhibitors (unless explicitly ignored)
	if !ignoreInhibitors {
		hasInhibit, inhibitors, err := e.checkInhibitors()
		if err != nil {
			e.logger.Warn("power.inhibit.check.failed", "Failed to check inhibitors", map[string]interface{}{
				"error": err.Error(),
			})
			// Continue anyway - don't block on check failure
		}

		if hasInhibit {
			e.logger.Info("power.suspend.skipped", "Suspend skipped due to inhibitors", map[string]interface{}{
				"idle_for_s": state.IdleForSeconds,
				"reason":     GatingReasonInhibit,
				"inhibitors": inhibitors,
			})

			// Add inhibit to gating reasons
			if state.GatingReasons == nil {
				state.GatingReasons = make([]string, 0)
			}
			if !containsReason(state.GatingReasons, GatingReasonInhibit) {
				state.GatingReasons = append(state.GatingReasons, GatingReasonInhibit)
			}

			return fmt.Errorf("suspend blocked by inhibitors: %s", strings.Join(inhibitors, ", "))
		}
	} else {
		e.logger.Info("power.inhibit.check.skipped", "Skipping inhibitor check (force mode)", nil)
	}

	// All gates passed - request suspend
	e.logger.Info("power.suspend.requested", "Suspend requested", map[string]interface{}{
		"idle_for_s":    state.IdleForSeconds,
		"threshold_s":   state.ThresholdSeconds,
		"cpu_idle_pct":  state.CPUIdlePct,
		"gpu_idle_pct":  state.GPUIdlePct,
		"enable_actual": e.config.EnableSuspend,
	})

	// Execute systemctl suspend
	if err := e.executeSuspend(); err != nil {
		e.logger.Error("power.suspend.failed", "Failed to execute suspend", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to execute suspend: %w", err)
	}

	e.logger.Info("power.suspend.done", "Suspend executed successfully", nil)
	return nil
}

// ActiveInhibitors returns whether there are active systemd inhibitors blocking suspend
func (e *Executor) ActiveInhibitors() (bool, []string, error) {
	return e.checkInhibitors()
}

// checkInhibitors checks for active systemd inhibitors
func (e *Executor) checkInhibitors() (bool, []string, error) {
	// Run systemd-inhibit --list to check for active inhibitors
	cmd := exec.Command("systemd-inhibit", "--list", "--no-pager", "--no-legend")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If command doesn't exist or fails, assume no inhibitors
		return false, nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	inhibitors := make([]string, 0)

	// Parse output for sleep/shutdown inhibitors
	for _, line := range lines {
		if strings.Contains(line, "sleep") || strings.Contains(line, "shutdown") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				inhibitors = append(inhibitors, fields[0]) // Add the inhibitor name
			}
		}
	}

	hasInhibit := len(inhibitors) > 0

	e.logger.Debug("power.inhibit.checked", "Checked for inhibitors", map[string]interface{}{
		"has_inhibit": hasInhibit,
		"inhibitors":  inhibitors,
	})

	return hasInhibit, inhibitors, nil
}

// executeSuspend executes the actual suspend command
func (e *Executor) executeSuspend() error {
	cmd := exec.Command("systemctl", "suspend")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl suspend failed: %w (output: %s)", err, string(output))
	}
	return nil
}

// CheckCanSuspend performs a dry-run check to see if suspend is possible
func (e *Executor) CheckCanSuspend() error {
	// Check if systemctl is available
	if _, err := exec.LookPath("systemctl"); err != nil {
		return fmt.Errorf("systemctl not found: %w", err)
	}

	// Check if suspend is supported (this won't actually suspend)
	cmd := exec.Command("systemctl", "can-suspend")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("suspend not supported by system: %w", err)
	}

	return nil
}

func containsReason(reasons []string, target string) bool {
	for _, r := range reasons {
		if r == target {
			return true
		}
	}
	return false
}
