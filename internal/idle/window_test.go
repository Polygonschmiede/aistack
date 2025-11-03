package idle

import (
	"testing"
	"time"
)

func TestSlidingWindow_AddSample(t *testing.T) {
	window := NewSlidingWindow(60, 3)

	// Add samples
	for i := 0; i < 5; i++ {
		sample := MetricSample{
			Timestamp: time.Now(),
			CPUUtil:   float64(i * 10),
			GPUUtil:   float64(i * 5),
		}
		window.AddSample(sample)
	}

	if window.SampleCount() != 5 {
		t.Errorf("Expected 5 samples, got %d", window.SampleCount())
	}
}

func TestSlidingWindow_HasEnoughSamples(t *testing.T) {
	window := NewSlidingWindow(60, 3)

	// Not enough samples initially
	if window.HasEnoughSamples() {
		t.Error("Expected not enough samples initially")
	}

	// Add samples
	for i := 0; i < 3; i++ {
		window.AddSample(MetricSample{
			Timestamp: time.Now(),
			CPUUtil:   10.0,
			GPUUtil:   5.0,
		})
	}

	// Should have enough now
	if !window.HasEnoughSamples() {
		t.Error("Expected enough samples after adding 3")
	}
}

func TestSlidingWindow_IsIdle_BelowThresholds(t *testing.T) {
	window := NewSlidingWindow(60, 3)

	// Add samples below thresholds
	for i := 0; i < 5; i++ {
		window.AddSample(MetricSample{
			Timestamp: time.Now(),
			CPUUtil:   5.0, // Below 10% threshold
			GPUUtil:   3.0, // Below 5% threshold
		})
	}

	idle, cpuAvg, gpuAvg := window.IsIdle(10.0, 5.0)

	if !idle {
		t.Error("Expected system to be idle when below thresholds")
	}

	if cpuAvg != 5.0 {
		t.Errorf("Expected CPU avg 5%%, got %.2f%%", cpuAvg)
	}

	if gpuAvg != 3.0 {
		t.Errorf("Expected GPU avg 3%%, got %.2f%%", gpuAvg)
	}
}

func TestSlidingWindow_IsIdle_AboveThresholds(t *testing.T) {
	window := NewSlidingWindow(60, 3)

	// Add samples above thresholds
	for i := 0; i < 5; i++ {
		window.AddSample(MetricSample{
			Timestamp: time.Now(),
			CPUUtil:   50.0, // Above 10% threshold
			GPUUtil:   25.0, // Above 5% threshold
		})
	}

	idle, cpuAvg, gpuAvg := window.IsIdle(10.0, 5.0)

	if idle {
		t.Error("Expected system to NOT be idle when above thresholds")
	}

	if cpuAvg != 50.0 {
		t.Errorf("Expected CPU avg 50%%, got %.2f%%", cpuAvg)
	}

	if gpuAvg != 25.0 {
		t.Errorf("Expected GPU avg 25%%, got %.2f%%", gpuAvg)
	}
}

func TestSlidingWindow_IsIdle_MixedUtilization(t *testing.T) {
	window := NewSlidingWindow(60, 3)

	// CPU below, GPU above threshold
	for i := 0; i < 5; i++ {
		window.AddSample(MetricSample{
			Timestamp: time.Now(),
			CPUUtil:   5.0,  // Below 10%
			GPUUtil:   10.0, // Above 5%
		})
	}

	idle, _, _ := window.IsIdle(10.0, 5.0)

	// Should NOT be idle - both must be below threshold
	if idle {
		t.Error("Expected system to NOT be idle when GPU is above threshold")
	}
}

func TestSlidingWindow_GetIdleDuration(t *testing.T) {
	window := NewSlidingWindow(60, 3)

	// Add idle samples
	for i := 0; i < 5; i++ {
		window.AddSample(MetricSample{
			Timestamp: time.Now(),
			CPUUtil:   5.0,
			GPUUtil:   3.0,
		})
	}

	// Get initial idle duration
	duration1 := window.GetIdleDuration(10.0, 5.0)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Get duration again - should have increased
	duration2 := window.GetIdleDuration(10.0, 5.0)

	if duration2 <= duration1 {
		t.Error("Expected idle duration to increase over time")
	}
}

func TestSlidingWindow_GetIdleDuration_Reset(t *testing.T) {
	window := NewSlidingWindow(60, 3)

	// Add idle samples
	for i := 0; i < 5; i++ {
		window.AddSample(MetricSample{
			Timestamp: time.Now(),
			CPUUtil:   5.0,
			GPUUtil:   3.0,
		})
	}

	// Prime idle duration baseline
	_ = window.GetIdleDuration(10.0, 5.0)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Get idle duration again - should be non-zero now
	duration := window.GetIdleDuration(10.0, 5.0)
	if duration == 0 {
		t.Error("Expected non-zero idle duration after delay")
	}

	// Add active sample (above threshold)
	window.AddSample(MetricSample{
		Timestamp: time.Now(),
		CPUUtil:   50.0,
		GPUUtil:   25.0,
	})

	// Idle duration should reset to 0
	duration2 := window.GetIdleDuration(10.0, 5.0)
	if duration2 != 0 {
		t.Errorf("Expected idle duration to reset to 0, got %v", duration2)
	}
}

func TestSlidingWindow_Reset(t *testing.T) {
	window := NewSlidingWindow(60, 3)

	// Add samples
	for i := 0; i < 5; i++ {
		window.AddSample(MetricSample{
			Timestamp: time.Now(),
			CPUUtil:   10.0,
			GPUUtil:   5.0,
		})
	}

	if window.SampleCount() != 5 {
		t.Errorf("Expected 5 samples before reset, got %d", window.SampleCount())
	}

	// Reset
	window.Reset()

	if window.SampleCount() != 0 {
		t.Errorf("Expected 0 samples after reset, got %d", window.SampleCount())
	}

	// Idle duration should also reset
	duration := window.GetIdleDuration(10.0, 5.0)
	if duration != 0 {
		t.Errorf("Expected idle duration to be 0 after reset, got %v", duration)
	}
}
