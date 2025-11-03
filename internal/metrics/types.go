package metrics

import (
	"time"
)

// MetricsSample represents a single metrics sample
// Data Contract from EP-005: metrics.sample.jsonl
type MetricsSample struct {
	Timestamp time.Time `json:"ts"`
	CPUUtil   *float64  `json:"cpu_util,omitempty"`    // CPU utilization percentage (0-100)
	CPUWatts  *float64  `json:"cpu_w,omitempty"`       // CPU power consumption in watts
	GPUUtil   *float64  `json:"gpu_util,omitempty"`    // GPU utilization percentage (0-100)
	GPUMemMB  *uint64   `json:"gpu_mem,omitempty"`     // GPU memory used in MB
	GPUWatts  *float64  `json:"gpu_w,omitempty"`       // GPU power consumption in watts
	TempCPU   *float64  `json:"temp_cpu,omitempty"`    // CPU temperature in Celsius
	TempGPU   *float64  `json:"temp_gpu,omitempty"`    // GPU temperature in Celsius
	EstTotalW *float64  `json:"est_total_w,omitempty"` // Estimated total power consumption
}

// CPUStats represents CPU statistics from /proc/stat
type CPUStats struct {
	User    uint64
	Nice    uint64
	System  uint64
	Idle    uint64
	IOWait  uint64
	IRQ     uint64
	SoftIRQ uint64
	Steal   uint64
}

// Total returns total CPU time
func (s CPUStats) Total() uint64 {
	return s.User + s.Nice + s.System + s.Idle + s.IOWait + s.IRQ + s.SoftIRQ + s.Steal
}

// Idle returns total idle time
func (s CPUStats) IdleTime() uint64 {
	return s.Idle + s.IOWait
}

// MetricsConfig holds configuration for metrics collection
type MetricsConfig struct {
	SampleInterval time.Duration // How often to collect metrics
	BaselinePowerW float64       // Baseline power consumption (system overhead)
	EnableGPU      bool          // Whether to collect GPU metrics
	EnableCPUPower bool          // Whether to collect CPU power (RAPL)
}

// DefaultConfig returns a default metrics configuration
func DefaultConfig() MetricsConfig {
	return MetricsConfig{
		SampleInterval: 10 * time.Second,
		BaselinePowerW: 50.0, // Conservative baseline for system overhead
		EnableGPU:      true,
		EnableCPUPower: true,
	}
}
