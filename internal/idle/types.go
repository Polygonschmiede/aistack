package idle

import (
	"os"
	"path/filepath"
	"time"
)

// IdleConfig holds configuration for idle detection and suspend behavior
//
//nolint:revive // exported name intentionally mirrors package name for clarity (idle.IdleConfig)
type IdleConfig struct {
	// WindowSeconds is the size of the sliding window for idle calculation
	WindowSeconds int `json:"window_seconds"`

	// IdleTimeoutSeconds is the duration of idle time before triggering suspend
	IdleTimeoutSeconds int `json:"idle_timeout_seconds"`

	// CPUThresholdPct is the CPU utilization threshold (%) below which CPU is considered idle
	CPUThresholdPct float64 `json:"cpu_threshold_pct"`

	// GPUThresholdPct is the GPU utilization threshold (%) below which GPU is considered idle
	GPUThresholdPct float64 `json:"gpu_threshold_pct"`

	// MinSamplesRequired is the minimum number of samples before calculating idle state
	MinSamplesRequired int `json:"min_samples_required"`

	// EnableSuspend enables actual suspend execution (false for dry-run)
	EnableSuspend bool `json:"enable_suspend"`

	// StateFilePath is the path to the idle state JSON file
	StateFilePath string `json:"state_file_path"`
}

// DefaultIdleConfig returns default idle configuration
func DefaultIdleConfig() IdleConfig {
	stateDir := "/var/lib/aistack"

	if envDir := os.Getenv("AISTACK_STATE_DIR"); envDir != "" {
		stateDir = envDir
	} else if os.Geteuid() != 0 {
		if home, err := os.UserHomeDir(); err == nil {
			stateDir = filepath.Join(home, ".local", "state", "aistack")
		} else {
			stateDir = filepath.Join(os.TempDir(), "aistack")
		}
	}

	return IdleConfig{
		WindowSeconds:      60,  // 60 second sliding window
		IdleTimeoutSeconds: 300, // 5 minutes idle timeout
		CPUThresholdPct:    10.0,
		GPUThresholdPct:    5.0,
		MinSamplesRequired: 6, // At least 6 samples (60s / 10s sample interval)
		EnableSuspend:      true,
		StateFilePath:      filepath.Join(stateDir, "idle_state.json"),
	}
}

// IdleState represents the current idle state of the system
//
//nolint:revive // exported name intentionally mirrors package name (idle.IdleState)
type IdleState struct {
	// Status is the current idle status (warming_up, active, idle)
	Status string `json:"status"`

	// IdleForSeconds is how long the system has been continuously idle
	IdleForSeconds int `json:"idle_for_s"`

	// ThresholdSeconds is the configured idle timeout threshold
	ThresholdSeconds int `json:"threshold_s"`

	// CPUIdlePct is the current CPU idle percentage
	CPUIdlePct float64 `json:"cpu_idle_pct"`

	// GPUIdlePct is the current GPU idle percentage
	GPUIdlePct float64 `json:"gpu_idle_pct"`

	// GatingReasons lists reasons preventing suspend
	GatingReasons []string `json:"gating_reasons"`

	// LastUpdate is the timestamp of the last state update
	LastUpdate time.Time `json:"last_update"`
}

// Idle status constants
const (
	StatusWarmingUp = "warming_up"
	StatusActive    = "active"
	StatusIdle      = "idle"
)

// MetricSample represents a single metric sample for idle calculation
type MetricSample struct {
	Timestamp time.Time
	CPUUtil   float64
	GPUUtil   float64
}

// GatingReason represents reasons that prevent suspend
const (
	GatingReasonInhibit      = "inhibit"
	GatingReasonBelowTimeout = "below_timeout"
	GatingReasonHighCPU      = "high_cpu"
	GatingReasonHighGPU      = "high_gpu"
	GatingReasonWarmingUp    = "warming_up"
)
