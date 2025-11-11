//go:build linux

package suspend

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"aistack/internal/gpu"
	"aistack/internal/logging"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
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
	nvml   gpu.NVMLInterface
}

// NewDetector creates a new activity detector
func NewDetector(logger *logging.Logger) *Detector {
	return &Detector{
		logger: logger,
		nvml:   gpu.NewRealNVML(),
	}
}

// cpuSample represents a snapshot of CPU stats from /proc/stat
type cpuSample struct {
	user    uint64
	nice    uint64
	system  uint64
	idle    uint64
	iowait  uint64
	irq     uint64
	softirq uint64
	steal   uint64
}

// total returns total CPU time (active + idle)
func (s cpuSample) total() uint64 {
	return s.user + s.nice + s.system + s.idle + s.iowait + s.irq + s.softirq + s.steal
}

// active returns active CPU time (total - idle - iowait)
func (s cpuSample) active() uint64 {
	return s.user + s.nice + s.system + s.irq + s.softirq + s.steal
}

// readCPUSample reads CPU stats from /proc/stat
func readCPUSample() (cpuSample, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuSample{}, fmt.Errorf("read /proc/stat: %w", err)
	}

	// Parse first line (aggregate CPU stats)
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return cpuSample{}, fmt.Errorf("empty /proc/stat")
	}

	fields := strings.Fields(lines[0])
	if len(fields) < 8 || fields[0] != "cpu" {
		return cpuSample{}, fmt.Errorf("invalid /proc/stat format")
	}

	// Parse fields: cpu user nice system idle iowait irq softirq steal
	sample := cpuSample{}
	sample.user, _ = strconv.ParseUint(fields[1], 10, 64)
	sample.nice, _ = strconv.ParseUint(fields[2], 10, 64)
	sample.system, _ = strconv.ParseUint(fields[3], 10, 64)
	sample.idle, _ = strconv.ParseUint(fields[4], 10, 64)
	sample.iowait, _ = strconv.ParseUint(fields[5], 10, 64)
	sample.irq, _ = strconv.ParseUint(fields[6], 10, 64)
	sample.softirq, _ = strconv.ParseUint(fields[7], 10, 64)
	if len(fields) >= 9 {
		sample.steal, _ = strconv.ParseUint(fields[8], 10, 64)
	}

	return sample, nil
}

// calculateCPUPercent calculates CPU utilization percentage between two samples
func calculateCPUPercent(before, after cpuSample) float64 {
	totalDelta := after.total() - before.total()
	if totalDelta == 0 {
		return 0.0
	}

	activeDelta := after.active() - before.active()
	return (float64(activeDelta) / float64(totalDelta)) * 100.0
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

// measureGPU measures GPU utilization (returns -1 if no GPU available)
func (d *Detector) measureGPU() float64 {
	// Try to initialize NVML
	ret := d.nvml.Init()
	if ret != nvml.SUCCESS {
		d.logger.Debug("suspend.gpu.unavailable", "NVML initialization failed, assuming no GPU", map[string]interface{}{
			"error": nvml.ErrorString(ret),
		})
		return -1.0
	}
	defer func() {
		if shutdownRet := d.nvml.Shutdown(); shutdownRet != nvml.SUCCESS {
			d.logger.Warn("suspend.gpu.shutdown.failed", "NVML shutdown failed", map[string]interface{}{
				"error": nvml.ErrorString(shutdownRet),
			})
		}
	}()

	// Get device count
	count, ret := d.nvml.DeviceGetCount()
	if ret != nvml.SUCCESS || count == 0 {
		d.logger.Debug("suspend.gpu.unavailable", "No GPU devices found", map[string]interface{}{
			"count": count,
		})
		return -1.0
	}

	// Get first GPU utilization (simple approach for v1)
	device, ret := d.nvml.DeviceGetHandleByIndex(0)
	if ret != nvml.SUCCESS {
		d.logger.Warn("suspend.gpu.device.failed", "Failed to get GPU device handle", map[string]interface{}{
			"error": nvml.ErrorString(ret),
		})
		return -1.0
	}

	util, ret := device.GetUtilizationRates()
	if ret != nvml.SUCCESS {
		d.logger.Warn("suspend.gpu.utilization.failed", "Failed to get GPU utilization", map[string]interface{}{
			"error": nvml.ErrorString(ret),
		})
		return -1.0
	}

	return float64(util.Gpu)
}

// DetectActivity measures current system activity and determines if system is idle
func (d *Detector) DetectActivity() (ActivityStatus, error) {
	d.logger.Debug("suspend.detect.start", "Starting activity detection", nil)

	// Measure CPU
	cpuPercent, err := d.measureCPU()
	if err != nil {
		return ActivityStatus{}, fmt.Errorf("measure CPU: %w", err)
	}

	// Measure GPU (optional)
	gpuPercent := d.measureGPU()

	// Determine if idle (CPU below threshold AND (no GPU OR GPU below threshold))
	cpuIdle := cpuPercent < CPUIdleThreshold
	gpuIdle := gpuPercent < 0 || gpuPercent < GPUIdleThreshold
	isIdle := cpuIdle && gpuIdle

	status := ActivityStatus{
		IsIdle:     isIdle,
		CPUPercent: cpuPercent,
		GPUPercent: gpuPercent,
		Timestamp:  time.Now(),
	}

	d.logger.Debug("suspend.detect.done", "Activity detection completed", map[string]interface{}{
		"cpu_percent": cpuPercent,
		"gpu_percent": gpuPercent,
		"is_idle":     isIdle,
	})

	return status, nil
}
