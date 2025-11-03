//go:build !cuda

package gpu

import "aistack/internal/logging"

// Detector provides a no-op GPU detector when NVML is unavailable.
type Detector struct {
	logger *logging.Logger
}

// NewDetector creates a GPU detector that skips NVML when CUDA support is disabled.
func NewDetector(logger *logging.Logger) *Detector {
	return &Detector{logger: logger}
}

// NewDetectorWithNVML is provided for API compatibility; NVML is ignored when CUDA is disabled.
func NewDetectorWithNVML(_ NVMLInterface, logger *logging.Logger) *Detector {
	return NewDetector(logger)
}

// DetectGPUs returns a report indicating that NVML is unavailable in the current build.
func (d *Detector) DetectGPUs() GPUReport {
	if d.logger != nil {
		d.logger.Info("gpu.detect.disabled", "Skipping NVML detection (built without cuda tag)", nil)
	}

	return GPUReport{
		GPUs:         []GPUInfo{},
		NVMLOk:       false,
		ErrorMessage: "NVML disabled: rebuild with -tags cuda",
	}
}

// SaveReport persists a GPU report to disk.
func (d *Detector) SaveReport(report GPUReport, filepath string) error {
	return saveReportToFile(d.logger, report, filepath)
}
