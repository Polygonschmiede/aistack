package idle

import (
	"testing"

	"aistack/internal/logging"
)

func TestEngine_Creation(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	config := DefaultIdleConfig()
	engine := NewEngine(config, logger)

	if engine == nil {
		t.Fatal("Expected engine to be created")
	}
}

func TestEngine_GetState_WarmingUp(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	config := DefaultIdleConfig()
	config.MinSamplesRequired = 5
	engine := NewEngine(config, logger)

	// Add insufficient samples
	for i := 0; i < 3; i++ {
		engine.AddMetrics(5.0, 3.0)
	}

	state := engine.GetState()

	if state.Status != StatusWarmingUp {
		t.Errorf("Expected status %s, got %s", StatusWarmingUp, state.Status)
	}

	if len(state.GatingReasons) != 1 || state.GatingReasons[0] != GatingReasonWarmingUp {
		t.Errorf("Expected gating reason %s, got %v", GatingReasonWarmingUp, state.GatingReasons)
	}
}

func TestEngine_GetState_Idle(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	config := DefaultIdleConfig()
	config.MinSamplesRequired = 3
	config.CPUThresholdPct = 10.0
	config.GPUThresholdPct = 5.0
	engine := NewEngine(config, logger)

	// Add idle samples (below thresholds)
	for i := 0; i < 5; i++ {
		engine.AddMetrics(5.0, 3.0) // CPU: 5%, GPU: 3%
	}

	state := engine.GetState()

	if state.Status != StatusIdle {
		t.Errorf("Expected status %s, got %s", StatusIdle, state.Status)
	}

	// CPU idle % should be 95% (100 - 5)
	if state.CPUIdlePct < 94.0 || state.CPUIdlePct > 96.0 {
		t.Errorf("Expected CPU idle ~95%%, got %.2f%%", state.CPUIdlePct)
	}

	// GPU idle % should be 97% (100 - 3)
	if state.GPUIdlePct < 96.0 || state.GPUIdlePct > 98.0 {
		t.Errorf("Expected GPU idle ~97%%, got %.2f%%", state.GPUIdlePct)
	}

	// Should have gating reason "below_timeout" since idle_for_s < threshold
	if len(state.GatingReasons) == 0 {
		t.Error("Expected gating reasons for idle state below timeout")
	}
}

func TestEngine_GetState_Active_HighCPU(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	config := DefaultIdleConfig()
	config.MinSamplesRequired = 3
	config.CPUThresholdPct = 10.0
	config.GPUThresholdPct = 5.0
	engine := NewEngine(config, logger)

	// Add samples with high CPU
	for i := 0; i < 5; i++ {
		engine.AddMetrics(50.0, 3.0) // CPU high, GPU low
	}

	state := engine.GetState()

	if state.Status != StatusActive {
		t.Errorf("Expected status %s, got %s", StatusActive, state.Status)
	}

	// Should have gating reason for high CPU
	hasHighCPU := false
	for _, reason := range state.GatingReasons {
		if reason == GatingReasonHighCPU {
			hasHighCPU = true
			break
		}
	}

	if !hasHighCPU {
		t.Errorf("Expected gating reason %s, got %v", GatingReasonHighCPU, state.GatingReasons)
	}
}

func TestEngine_GetState_Active_HighGPU(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	config := DefaultIdleConfig()
	config.MinSamplesRequired = 3
	config.CPUThresholdPct = 10.0
	config.GPUThresholdPct = 5.0
	engine := NewEngine(config, logger)

	// Add samples with high GPU
	for i := 0; i < 5; i++ {
		engine.AddMetrics(5.0, 25.0) // CPU low, GPU high
	}

	state := engine.GetState()

	if state.Status != StatusActive {
		t.Errorf("Expected status %s, got %s", StatusActive, state.Status)
	}

	// Should have gating reason for high GPU
	hasHighGPU := false
	for _, reason := range state.GatingReasons {
		if reason == GatingReasonHighGPU {
			hasHighGPU = true
			break
		}
	}

	if !hasHighGPU {
		t.Errorf("Expected gating reason %s, got %v", GatingReasonHighGPU, state.GatingReasons)
	}
}

func TestEngine_GetState_Active_HighBoth(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	config := DefaultIdleConfig()
	config.MinSamplesRequired = 3
	engine := NewEngine(config, logger)

	// Add samples with both CPU and GPU high
	for i := 0; i < 5; i++ {
		engine.AddMetrics(50.0, 25.0)
	}

	state := engine.GetState()

	if state.Status != StatusActive {
		t.Errorf("Expected status %s, got %s", StatusActive, state.Status)
	}

	// Should have both gating reasons
	hasHighCPU := false
	hasHighGPU := false
	for _, reason := range state.GatingReasons {
		if reason == GatingReasonHighCPU {
			hasHighCPU = true
		}
		if reason == GatingReasonHighGPU {
			hasHighGPU = true
		}
	}

	if !hasHighCPU || !hasHighGPU {
		t.Errorf("Expected both CPU and GPU gating reasons, got %v", state.GatingReasons)
	}
}

func TestEngine_ShouldSuspend_WarmingUp(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	config := DefaultIdleConfig()
	engine := NewEngine(config, logger)

	state := IdleState{
		Status: StatusWarmingUp,
	}

	if engine.ShouldSuspend(state) {
		t.Error("Should not suspend during warming up")
	}
}

func TestEngine_ShouldSuspend_Active(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	config := DefaultIdleConfig()
	engine := NewEngine(config, logger)

	state := IdleState{
		Status: StatusActive,
	}

	if engine.ShouldSuspend(state) {
		t.Error("Should not suspend when active")
	}
}

func TestEngine_ShouldSuspend_IdleWithGating(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	config := DefaultIdleConfig()
	engine := NewEngine(config, logger)

	state := IdleState{
		Status:        StatusIdle,
		GatingReasons: []string{GatingReasonBelowTimeout},
	}

	if engine.ShouldSuspend(state) {
		t.Error("Should not suspend when there are gating reasons")
	}
}

func TestEngine_ShouldSuspend_IdleReady(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	config := DefaultIdleConfig()
	config.IdleTimeoutSeconds = 300
	engine := NewEngine(config, logger)

	state := IdleState{
		Status:         StatusIdle,
		IdleForSeconds: 350, // Above threshold
		GatingReasons:  []string{},
	}

	if !engine.ShouldSuspend(state) {
		t.Error("Should suspend when idle and above threshold with no gating reasons")
	}
}

func TestEngine_Reset(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	config := DefaultIdleConfig()
	engine := NewEngine(config, logger)

	// Add samples
	for i := 0; i < 5; i++ {
		engine.AddMetrics(10.0, 5.0)
	}

	// Reset
	engine.Reset()

	// Window should be empty
	if engine.window.SampleCount() != 0 {
		t.Errorf("Expected 0 samples after reset, got %d", engine.window.SampleCount())
	}
}
