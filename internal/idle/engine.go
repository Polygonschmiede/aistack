package idle

import (
	"time"

	"aistack/internal/logging"
)

// Engine is the idle detection engine
type Engine struct {
	config IdleConfig
	window *SlidingWindow
	logger *logging.Logger
}

// NewEngine creates a new idle detection engine
func NewEngine(config IdleConfig, logger *logging.Logger) *Engine {
	return &Engine{
		config: config,
		window: NewSlidingWindow(config.WindowSeconds, config.MinSamplesRequired),
		logger: logger,
	}
}

// AddMetrics adds new CPU/GPU metrics to the idle detection engine
func (e *Engine) AddMetrics(cpuUtil, gpuUtil float64) {
	sample := MetricSample{
		Timestamp: time.Now(),
		CPUUtil:   cpuUtil,
		GPUUtil:   gpuUtil,
	}

	e.window.AddSample(sample)

	e.logger.Debug("idle.metrics.added", "Added metrics sample", map[string]interface{}{
		"cpu_util": cpuUtil,
		"gpu_util": gpuUtil,
		"samples":  e.window.SampleCount(),
	})
}

// GetState calculates and returns the current idle state
func (e *Engine) GetState() IdleState {
	// Check if we have enough samples
	if !e.window.HasEnoughSamples() {
		return IdleState{
			Status:           StatusWarmingUp,
			IdleForSeconds:   0,
			ThresholdSeconds: e.config.IdleTimeoutSeconds,
			CPUIdlePct:       0,
			GPUIdlePct:       0,
			GatingReasons:    []string{GatingReasonWarmingUp},
			LastUpdate:       time.Now(),
		}
	}

	// Check if idle
	idle, cpuAvg, gpuAvg := e.window.IsIdle(e.config.CPUThresholdPct, e.config.GPUThresholdPct)
	idleDuration := e.window.GetIdleDuration(e.config.CPUThresholdPct, e.config.GPUThresholdPct)

	// Calculate CPU/GPU idle percentages (inverse of utilization)
	cpuIdlePct := 100.0 - cpuAvg
	gpuIdlePct := 100.0 - gpuAvg

	// Determine status and gating reasons
	var status string
	gatingReasons := make([]string, 0)

	if !idle {
		status = StatusActive

		// Determine why we're not idle
		if cpuAvg >= e.config.CPUThresholdPct {
			gatingReasons = append(gatingReasons, GatingReasonHighCPU)
		}
		if gpuAvg >= e.config.GPUThresholdPct {
			gatingReasons = append(gatingReasons, GatingReasonHighGPU)
		}
	} else {
		// System is idle, but check if we've been idle long enough
		idleSeconds := int(idleDuration.Seconds())
		if idleSeconds < e.config.IdleTimeoutSeconds {
			status = StatusIdle
			gatingReasons = append(gatingReasons, GatingReasonBelowTimeout)
		} else {
			status = StatusIdle
			// No gating reasons - ready for suspend
		}
	}

	state := IdleState{
		Status:           status,
		IdleForSeconds:   int(idleDuration.Seconds()),
		ThresholdSeconds: e.config.IdleTimeoutSeconds,
		CPUIdlePct:       cpuIdlePct,
		GPUIdlePct:       gpuIdlePct,
		GatingReasons:    gatingReasons,
		LastUpdate:       time.Now(),
	}

	e.logger.Debug("idle.state.calculated", "Calculated idle state", map[string]interface{}{
		"status":         state.Status,
		"idle_for_s":     state.IdleForSeconds,
		"cpu_idle_pct":   state.CPUIdlePct,
		"gpu_idle_pct":   state.GPUIdlePct,
		"gating_reasons": state.GatingReasons,
	})

	return state
}

// ShouldSuspend checks if the system should suspend based on current state
func (e *Engine) ShouldSuspend(state IdleState) bool {
	// Don't suspend if warming up
	if state.Status == StatusWarmingUp {
		return false
	}

	// Don't suspend if not idle
	if state.Status != StatusIdle {
		return false
	}

	// Don't suspend if there are gating reasons
	if len(state.GatingReasons) > 0 {
		return false
	}

	// Check if idle duration meets threshold
	return state.IdleForSeconds >= e.config.IdleTimeoutSeconds
}

// Reset resets the idle detection engine
func (e *Engine) Reset() {
	e.window.Reset()
	e.logger.Info("idle.engine.reset", "Idle engine reset", nil)
}
