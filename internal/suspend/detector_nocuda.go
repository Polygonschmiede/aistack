//go:build linux && !cuda

package suspend

import (
	"fmt"
	"time"

	"aistack/internal/logging"
)

// Thresholds for idle detection
const (
	CPUIdleThreshold = 10.0 // CPU utilization below 10% = idle
	GPUIdleThreshold = 5.0  // GPU utilization below 5% = idle
)

// ActivityStatus represents system activity at a point in time
type ActivityStatus struct {
	IsIdle     bool      // True if system is idle (CPU and GPU below thresholds)
	CPUPercent float64   // CPU utilization percentage (0-100)
	GPUPercent float64   // GPU utilization percentage (0-100, -1 if no GPU)
	Timestamp  time.Time // When this status was measured
}

// Detector handles activity detection for suspend decisions
type Detector struct {
	logger *logging.Logger
}

// NewDetector creates a new activity detector
func NewDetector(logger *logging.Logger) *Detector {
	return &Detector{
		logger: logger,
	}
}

// DetectActivity measures current system activity (stub for non-CUDA Linux builds)
// This version only monitors CPU, as GPU monitoring requires CUDA/NVML
func (d *Detector) DetectActivity() (ActivityStatus, error) {
	d.logger.Debug("suspend.detect.start", "Starting activity detection (no GPU support)", nil)

	// Measure CPU
	cpuPercent, err := d.measureCPU()
	if err != nil {
		return ActivityStatus{}, fmt.Errorf("measure CPU: %w", err)
	}

	// No GPU monitoring in non-CUDA builds
	gpuPercent := -1.0

	// Determine if idle (CPU below threshold, GPU monitoring disabled)
	cpuIdle := cpuPercent < CPUIdleThreshold
	isIdle := cpuIdle

	status := ActivityStatus{
		IsIdle:     isIdle,
		CPUPercent: cpuPercent,
		GPUPercent: gpuPercent,
		Timestamp:  time.Now(),
	}

	d.logger.Debug("suspend.detect.done", "Activity detection completed (no GPU)", map[string]interface{}{
		"cpu_percent": cpuPercent,
		"is_idle":     isIdle,
	})

	return status, nil
}

// measureCPU measures CPU utilization over 1 second
func (d *Detector) measureCPU() (float64, error) {
	sample1, err := readCPUSample()
	if err != nil {
		return 0, fmt.Errorf("read first CPU sample: %w", err)
	}

	time.Sleep(1 * time.Second)

	sample2, err := readCPUSample()
	if err != nil {
		return 0, fmt.Errorf("read second CPU sample: %w", err)
	}

	percent := calculateCPUPercent(sample1, sample2)
	return percent, nil
}
