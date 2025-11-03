package gpu

import (
	"encoding/json"
	"fmt"
	"os"

	"aistack/internal/logging"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

// Detector handles GPU detection and reporting
type Detector struct {
	nvml   NVMLInterface
	logger *logging.Logger
}

// NewDetector creates a new GPU detector
func NewDetector(logger *logging.Logger) *Detector {
	return &Detector{
		nvml:   NewRealNVML(),
		logger: logger,
	}
}

// NewDetectorWithNVML creates a detector with a custom NVML interface (for testing)
func NewDetectorWithNVML(nvmlInterface NVMLInterface, logger *logging.Logger) *Detector {
	return &Detector{
		nvml:   nvmlInterface,
		logger: logger,
	}
}

// DetectGPUs performs GPU detection and returns a report
// Story T-009: GPU-Erkennung & NVML-Probe
func (d *Detector) DetectGPUs() GPUReport {
	d.logger.Info("gpu.detect.start", "Starting GPU detection", nil)

	report := GPUReport{
		GPUs: make([]GPUInfo, 0),
	}

	// Initialize NVML
	ret := d.nvml.Init()
	if ret != nvml.SUCCESS {
		report.NVMLOk = false
		report.ErrorMessage = fmt.Sprintf("Failed to initialize NVML: %v", nvml.ErrorString(ret))
		d.logger.Warn("gpu.nvml.init.failed", "NVML initialization failed", map[string]interface{}{
			"error": report.ErrorMessage,
		})
		return report
	}
	defer d.nvml.Shutdown()

	report.NVMLOk = true

	// Get driver version
	driverVersion, ret := d.nvml.SystemGetDriverVersion()
	if ret != nvml.SUCCESS {
		d.logger.Warn("gpu.driver.version.failed", "Failed to get driver version", map[string]interface{}{
			"error": nvml.ErrorString(ret),
		})
	} else {
		report.DriverVersion = driverVersion
	}

	// Get CUDA version
	cudaVersion, ret := d.nvml.SystemGetCudaDriverVersion()
	if ret != nvml.SUCCESS {
		d.logger.Warn("gpu.cuda.version.failed", "Failed to get CUDA version", map[string]interface{}{
			"error": nvml.ErrorString(ret),
		})
	} else {
		report.CUDAVersion = cudaVersion
	}

	// Get device count
	count, ret := d.nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		report.ErrorMessage = fmt.Sprintf("Failed to get device count: %v", nvml.ErrorString(ret))
		d.logger.Error("gpu.device.count.failed", "Failed to get GPU count", map[string]interface{}{
			"error": report.ErrorMessage,
		})
		return report
	}

	d.logger.Info("gpu.device.count", "Found GPU devices", map[string]interface{}{
		"count": count,
	})

	// Iterate through devices
	for i := 0; i < count; i++ {
		device, ret := d.nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			d.logger.Warn("gpu.device.handle.failed", "Failed to get device handle", map[string]interface{}{
				"index": i,
				"error": nvml.ErrorString(ret),
			})
			continue
		}

		gpuInfo := GPUInfo{
			Index: i,
		}

		// Get device name
		name, ret := device.GetName()
		if ret == nvml.SUCCESS {
			gpuInfo.Name = name
		}

		// Get device UUID
		uuid, ret := device.GetUUID()
		if ret == nvml.SUCCESS {
			gpuInfo.UUID = uuid
		}

		// Get memory info
		memInfo, ret := device.GetMemoryInfo()
		if ret == nvml.SUCCESS {
			gpuInfo.MemoryMB = memInfo.Total / (1024 * 1024) // Convert bytes to MB
		}

		report.GPUs = append(report.GPUs, gpuInfo)

		d.logger.Info("gpu.device.detected", "GPU device detected", map[string]interface{}{
			"index":     i,
			"name":      gpuInfo.Name,
			"uuid":      gpuInfo.UUID,
			"memory_mb": gpuInfo.MemoryMB,
		})
	}

	return report
}

// SaveReport saves the GPU report to a JSON file
func (d *Detector) SaveReport(report GPUReport, filepath string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	d.logger.Info("gpu.report.saved", "GPU report saved", map[string]interface{}{
		"filepath": filepath,
	})

	return nil
}
