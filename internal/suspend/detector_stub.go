//go:build !linux

package suspend

import (
	"fmt"
	"time"

	"aistack/internal/logging"
)

// Thresholds for idle detection (stub - same as Linux version)
const (
	CPUIdleThreshold = 10.0 // CPU utilization below 10% = idle
	GPUIdleThreshold = 5.0  // GPU utilization below 5% = idle
)

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

// ActivityStatus represents system activity at a point in time
type ActivityStatus struct {
	IsIdle     bool
	CPUPercent float64
	GPUPercent float64
	Timestamp  time.Time
}

// DetectActivity measures current system activity (stub for non-Linux)
func (d *Detector) DetectActivity() (ActivityStatus, error) {
	return ActivityStatus{}, fmt.Errorf("suspend feature only supported on Linux")
}
