package metrics

import (
	"testing"
)

func TestCPUStats_Total(t *testing.T) {
	stats := CPUStats{
		User:    100,
		Nice:    10,
		System:  50,
		Idle:    200,
		IOWait:  20,
		IRQ:     5,
		SoftIRQ: 3,
		Steal:   2,
	}

	expected := uint64(390)
	if total := stats.Total(); total != expected {
		t.Errorf("Expected total %d, got %d", expected, total)
	}
}

func TestCPUStats_IdleTime(t *testing.T) {
	stats := CPUStats{
		Idle:   200,
		IOWait: 20,
	}

	expected := uint64(220)
	if idle := stats.IdleTime(); idle != expected {
		t.Errorf("Expected idle %d, got %d", expected, idle)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.SampleInterval.Seconds() != 10 {
		t.Errorf("Expected sample interval 10s, got %v", config.SampleInterval)
	}

	if config.BaselinePowerW != 50.0 {
		t.Errorf("Expected baseline power 50W, got %f", config.BaselinePowerW)
	}

	if !config.EnableGPU {
		t.Error("Expected GPU to be enabled by default")
	}

	if !config.EnableCPUPower {
		t.Error("Expected CPU power to be enabled by default")
	}
}
