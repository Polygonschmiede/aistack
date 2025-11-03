package idle

import (
	"os/exec"
	"testing"

	"aistack/internal/logging"
)

func TestExecutor_Creation(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	config := DefaultIdleConfig()
	executor := NewExecutor(config, logger)

	if executor == nil {
		t.Fatal("Expected executor to be created")
	}
}

func TestExecutor_Execute_DryRun(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	config := DefaultIdleConfig()
	config.EnableSuspend = false // Dry-run mode

	executor := NewExecutor(config, logger)

	state := IdleState{
		Status:           StatusIdle,
		IdleForSeconds:   350,
		ThresholdSeconds: 300,
		GatingReasons:    []string{}, // No gating reasons
	}

	// Should not error in dry-run mode
	err := executor.Execute(&state)
	if err != nil {
		t.Errorf("Expected no error in dry-run mode, got: %v", err)
	}
}

func TestExecutor_Execute_WithGatingReasons(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	config := DefaultIdleConfig()
	config.EnableSuspend = false

	executor := NewExecutor(config, logger)

	state := IdleState{
		Status:           StatusIdle,
		IdleForSeconds:   350,
		ThresholdSeconds: 300,
		GatingReasons:    []string{GatingReasonBelowTimeout},
	}

	// Should return error due to gating reasons
	err := executor.Execute(&state)
	if err == nil {
		t.Error("Expected error when gating reasons present")
	}
}

func TestExecutor_Execute_WarmingUp(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	config := DefaultIdleConfig()
	config.EnableSuspend = false

	executor := NewExecutor(config, logger)

	state := IdleState{
		Status:           StatusWarmingUp,
		IdleForSeconds:   0,
		ThresholdSeconds: 300,
		GatingReasons:    []string{GatingReasonWarmingUp},
	}

	// Should not error but also not suspend
	err := executor.Execute(&state)
	// In warming_up state with gating reasons, should return error
	if err == nil {
		t.Error("Expected error in warming up state")
	}
}

func TestExecutor_Execute_Active(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	config := DefaultIdleConfig()
	config.EnableSuspend = false

	executor := NewExecutor(config, logger)

	state := IdleState{
		Status:           StatusActive,
		IdleForSeconds:   0,
		ThresholdSeconds: 300,
		GatingReasons:    []string{GatingReasonHighCPU},
	}

	// Should return error due to active state
	err := executor.Execute(&state)
	if err == nil {
		t.Error("Expected error when system is active")
	}
}

func TestExecutor_CheckCanSuspend(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	config := DefaultIdleConfig()
	executor := NewExecutor(config, logger)

	// This test will only pass on systems with systemd
	// On macOS or non-systemd systems, it should fail gracefully
	if _, err := exec.LookPath("systemctl"); err != nil {
		t.Skip("systemctl not available; skipping suspend capability check")
	}

	err := executor.CheckCanSuspend()

	// We don't assert on error here as it depends on the system
	// Just verify the method doesn't panic
	t.Logf("CheckCanSuspend result: %v", err)
}

// TestExecutor_InhibitCheck tests the inhibitor checking logic
// This is a basic test since we can't easily mock systemd-inhibit
func TestExecutor_InhibitCheck(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	config := DefaultIdleConfig()
	executor := NewExecutor(config, logger)

	// Try to check for inhibitors
	// This will likely fail on non-systemd systems, which is fine
	if _, err := exec.LookPath("systemd-inhibit"); err != nil {
		t.Skip("systemd-inhibit not available; skipping inhibitor check")
	}

	hasInhibit, inhibitors, err := executor.checkInhibitors()

	// Log the results for debugging
	t.Logf("Has inhibitors: %v, Inhibitors: %v, Error: %v", hasInhibit, inhibitors, err)

	// We don't assert here as behavior depends on the system
	// Just verify it doesn't panic
}
