package idle

import (
	"testing"
)

func TestDefaultIdleConfig(t *testing.T) {
	config := DefaultIdleConfig()

	if config.WindowSeconds != 60 {
		t.Errorf("Expected window seconds 60, got %d", config.WindowSeconds)
	}

	if config.IdleTimeoutSeconds != 300 {
		t.Errorf("Expected idle timeout 300s, got %d", config.IdleTimeoutSeconds)
	}

	if config.CPUThresholdPct != 10.0 {
		t.Errorf("Expected CPU threshold 10%%, got %.2f%%", config.CPUThresholdPct)
	}

	if config.GPUThresholdPct != 5.0 {
		t.Errorf("Expected GPU threshold 5%%, got %.2f%%", config.GPUThresholdPct)
	}

	if config.MinSamplesRequired != 6 {
		t.Errorf("Expected min samples 6, got %d", config.MinSamplesRequired)
	}

	if !config.EnableSuspend {
		t.Error("Expected suspend to be enabled by default")
	}
}

func TestIdleStateConstants(t *testing.T) {
	// Ensure constants are defined correctly
	if StatusWarmingUp != "warming_up" {
		t.Errorf("Expected StatusWarmingUp to be 'warming_up', got '%s'", StatusWarmingUp)
	}

	if StatusActive != "active" {
		t.Errorf("Expected StatusActive to be 'active', got '%s'", StatusActive)
	}

	if StatusIdle != "idle" {
		t.Errorf("Expected StatusIdle to be 'idle', got '%s'", StatusIdle)
	}
}
