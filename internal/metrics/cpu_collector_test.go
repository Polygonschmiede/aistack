package metrics

import (
	"aistack/internal/logging"
	"testing"
)

func TestCPUCollector_Creation(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	collector := NewCPUCollector(logger)

	if collector == nil {
		t.Fatal("Expected collector to be created")
	}

	if collector.logger == nil {
		t.Error("Expected logger to be set")
	}
}

func TestCPUCollector_CalculateUtilization(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	collector := NewCPUCollector(logger)

	// Simulate CPU stats with 50% utilization
	prev := &CPUStats{
		User:   100,
		System: 50,
		Idle:   150,
	}

	current := &CPUStats{
		User:   150, // +50
		System: 75,  // +25
		Idle:   175, // +25  (total delta = 100, idle delta = 25)
	}

	util := collector.calculateUtilization(prev, current)

	// Expected: (1 - 25/100) * 100 = 75%
	if util < 74 || util > 76 {
		t.Errorf("Expected utilization around 75%%, got %.2f%%", util)
	}
}

func TestCPUCollector_CalculateUtilization_ZeroDelta(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	collector := NewCPUCollector(logger)

	stats := &CPUStats{
		User:   100,
		System: 50,
		Idle:   150,
	}

	// Same stats = no delta = 0% utilization
	util := collector.calculateUtilization(stats, stats)

	if util != 0.0 {
		t.Errorf("Expected 0%% utilization for same stats, got %.2f%%", util)
	}
}

func TestCPUCollector_IsRAPLEnabled(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	collector := NewCPUCollector(logger)

	// On most systems, RAPL won't be available
	// Just verify the method doesn't panic
	_ = collector.IsRAPLEnabled()
}
