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
	e.logger.Info("power.suspend.execute_start", "Starting suspend execution", map[string]interface{}{
		"ignore_inhibitors": ignoreInhibitors,
		"enable_suspend":    e.config.EnableSuspend,
		"idle_for_s":        state.IdleForSeconds,
		"threshold_s":       state.ThresholdSeconds,
		"status":            state.Status,
		"gating_reasons":    state.GatingReasons,
	})

	// Check if any other gating reasons exist first (before checking inhibitors)
	if ignoreInhibitors {
		e.logger.Info("power.suspend.filter_inhibitors", "Filtering inhibit gating reason", map[string]interface{}{
			"gating_reasons_before": state.GatingReasons,
		})
		filtered := make([]string, 0, len(state.GatingReasons))
		for _, r := range state.GatingReasons {
			if r != GatingReasonInhibit {
				filtered = append(filtered, r)
			}
		}
		state.GatingReasons = filtered
		e.logger.Info("power.suspend.filter_inhibitors_done", "Filtered inhibit gating reason", map[string]interface{}{
			"gating_reasons_after": state.GatingReasons,
		})
	}

	if len(state.GatingReasons) > 0 {
		e.logger.Warn("power.suspend.blocked_by_gating", "Suspend blocked by gating reasons", map[string]interface{}{
			"idle_for_s":     state.IdleForSeconds,
			"gating_reasons": state.GatingReasons,
			"count":          len(state.GatingReasons),
		})
		return fmt.Errorf("suspend blocked by gating reasons: %s", strings.Join(state.GatingReasons, ", "))
	}

	e.logger.Info("power.suspend.gating_check_passed", "No gating reasons blocking suspend", nil)

	// Check if suspend is enabled
	if !e.config.EnableSuspend {
		e.logger.Warn("power.suspend.disabled", "Suspend disabled in configuration (dry-run mode)", map[string]interface{}{
			"idle_for_s": state.IdleForSeconds,
			"reason":     "dry_run",
		})
		return nil
	}

	e.logger.Info("power.suspend.config_check_passed", "Suspend enabled in configuration", nil)

	// Check for inhibitors (unless explicitly ignored)
	if !ignoreInhibitors {
		e.logger.Info("power.suspend.check_inhibitors", "Checking systemd inhibitors", nil)
		hasInhibit, inhibitors, err := e.checkInhibitors()
		if err != nil {
			e.logger.Warn("power.inhibit.check.failed", "Failed to check inhibitors (continuing anyway)", map[string]interface{}{
				"error": err.Error(),
			})
			// Continue anyway - don't block on check failure
		}

		e.logger.Info("power.inhibit.check_result", "Inhibitor check completed", map[string]interface{}{
			"has_inhibit":  hasInhibit,
			"inhibitors":   inhibitors,
			"count":        len(inhibitors),
			"check_failed": err != nil,
		})

		if hasInhibit {
			e.logger.Warn("power.suspend.blocked_by_inhibitors", "Suspend blocked by systemd inhibitors", map[string]interface{}{
				"idle_for_s":     state.IdleForSeconds,
				"reason":         GatingReasonInhibit,
				"inhibitors":     inhibitors,
				"inhibitor_list": strings.Join(inhibitors, ", "),
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

		e.logger.Info("power.suspend.inhibitor_check_passed", "No inhibitors blocking suspend", nil)
	} else {
		e.logger.Info("power.inhibit.check.skipped", "Skipping inhibitor check (force mode enabled)", nil)
	}

	// All gates passed - request suspend
	e.logger.Info("power.suspend.all_checks_passed", "All pre-suspend checks passed, executing suspend", map[string]interface{}{
		"idle_for_s":     state.IdleForSeconds,
		"threshold_s":    state.ThresholdSeconds,
		"cpu_idle_pct":   state.CPUIdlePct,
		"gpu_idle_pct":   state.GPUIdlePct,
		"enable_actual":  e.config.EnableSuspend,
		"force_mode":     ignoreInhibitors,
		"gating_reasons": state.GatingReasons,
		"gating_count":   len(state.GatingReasons),
	})

	// Execute systemctl suspend
	e.logger.Info("power.suspend.executing", "Executing systemctl suspend command", nil)
	if err := e.executeSuspend(); err != nil {
		e.logger.Error("power.suspend.failed", "Failed to execute suspend command", map[string]interface{}{
			"error":      err.Error(),
			"error_type": fmt.Sprintf("%T", err),
		})
		return fmt.Errorf("failed to execute suspend: %w", err)
	}

	e.logger.Info("power.suspend.done", "Suspend command executed successfully", nil)
	return nil
}

// ActiveInhibitors returns whether there are active systemd inhibitors blocking suspend
func (e *Executor) ActiveInhibitors() (bool, []string, error) {
	return e.checkInhibitors()
}

// checkInhibitors checks for active systemd inhibitors
func (e *Executor) checkInhibitors() (bool, []string, error) {
	e.logger.Info("power.inhibit.check_start", "Starting inhibitor check", nil)

	// Run systemd-inhibit --list to check for active inhibitors
	cmd := exec.Command("systemd-inhibit", "--list", "--no-pager", "--no-legend")
	output, err := cmd.CombinedOutput()
	if err != nil {
		e.logger.Warn("power.inhibit.command_failed", "systemd-inhibit command failed", map[string]interface{}{
			"error":  err.Error(),
			"output": string(output),
		})
		// If command doesn't exist or fails, assume no inhibitors
		return false, nil, err
	}

	e.logger.Info("power.inhibit.command_output", "systemd-inhibit raw output", map[string]interface{}{
		"output":       string(output),
		"output_lines": len(strings.Split(strings.TrimSpace(string(output)), "\n")),
	})

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	inhibitors := make([]string, 0)
	matchedLines := make([]string, 0)

	// Parse output for sleep/shutdown inhibitors
	for i, line := range lines {
		e.logger.Debug("power.inhibit.parse_line", "Parsing inhibitor line", map[string]interface{}{
			"line_number": i,
			"line":        line,
		})

		if strings.Contains(line, "sleep") || strings.Contains(line, "shutdown") {
			fields := strings.Fields(line)
			e.logger.Info("power.inhibit.line_matched", "Found inhibitor line", map[string]interface{}{
				"line":        line,
				"fields":      fields,
				"field_count": len(fields),
			})

			if len(fields) > 0 {
				inhibitors = append(inhibitors, fields[0]) // Add the inhibitor name
				matchedLines = append(matchedLines, line)
			}
		}
	}

	hasInhibit := len(inhibitors) > 0

	e.logger.Info("power.inhibit.check_complete", "Inhibitor check completed", map[string]interface{}{
		"has_inhibit":   hasInhibit,
		"inhibitors":    inhibitors,
		"count":         len(inhibitors),
		"matched_lines": matchedLines,
		"total_lines":   len(lines),
	})

	return hasInhibit, inhibitors, nil
}

// executeSuspend executes the actual suspend command
func (e *Executor) executeSuspend() error {
	e.logger.Info("power.suspend.command_start", "Executing systemctl suspend", nil)

	cmd := exec.Command("systemctl", "suspend")
	output, err := cmd.CombinedOutput()

	e.logger.Info("power.suspend.command_result", "systemctl suspend command completed", map[string]interface{}{
		"success":    err == nil,
		"output":     string(output),
		"output_len": len(output),
		"has_error":  err != nil,
	})

	if err != nil {
		e.logger.Error("power.suspend.command_error", "systemctl suspend failed", map[string]interface{}{
			"error":      err.Error(),
			"error_type": fmt.Sprintf("%T", err),
			"output":     string(output),
			"exit_code":  cmd.ProcessState,
		})
		return fmt.Errorf("systemctl suspend failed: %w (output: %s)", err, string(output))
	}

	e.logger.Info("power.suspend.command_success", "systemctl suspend succeeded", nil)
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
