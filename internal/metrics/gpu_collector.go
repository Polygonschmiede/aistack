package metrics

import (
	"fmt"

	"aistack/internal/gpu"
	"aistack/internal/logging"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

// GPUCollector collects GPU metrics using NVML
// Story T-011: GPU-Metriken sammeln (Util/VRAM/Temp/Power)
type GPUCollector struct {
	logger      *logging.Logger
	nvml        gpu.NVMLInterface
	deviceIndex int
	device      gpu.DeviceInterface
	initialized bool
}

// NewGPUCollector creates a new GPU metrics collector
func NewGPUCollector(logger *logging.Logger) *GPUCollector {
	return &GPUCollector{
		logger:      logger,
		nvml:        gpu.NewRealNVML(),
		deviceIndex: 0, // Use first GPU by default
	}
}

// NewGPUCollectorWithNVML creates a collector with custom NVML (for testing)
func NewGPUCollectorWithNVML(nvmlInterface gpu.NVMLInterface, logger *logging.Logger) *GPUCollector {
	return &GPUCollector{
		logger:      logger,
		nvml:        nvmlInterface,
		deviceIndex: 0,
	}
}

// Initialize initializes the GPU collector
func (g *GPUCollector) Initialize() error {
	ret := g.nvml.Init()
	if ret != nvml.SUCCESS {
		return fmt.Errorf("failed to initialize NVML: %v", nvml.ErrorString(ret))
	}

	// Get handle to first GPU
	device, ret := g.nvml.DeviceGetHandleByIndex(g.deviceIndex)
	if ret != nvml.SUCCESS {
		if shutdownRet := g.nvml.Shutdown(); shutdownRet != nvml.SUCCESS {
			g.logger.Warn("gpu.collector.shutdown.failed", "NVML shutdown reported an error during init", map[string]interface{}{
				"error": nvml.ErrorString(shutdownRet),
			})
		}
		return fmt.Errorf("failed to get GPU device: %v", nvml.ErrorString(ret))
	}

	g.device = device
	g.initialized = true

	g.logger.Info("gpu.collector.initialized", "GPU metrics collector initialized", map[string]interface{}{
		"device_index": g.deviceIndex,
	})

	return nil
}

// Collect collects current GPU metrics
// Story T-011: NVML-Sampling alle 10s, JSONL-Log
func (g *GPUCollector) Collect() (gpuUtil *float64, gpuMemMB *uint64, gpuWatts *float64, tempGPU *float64, err error) {
	if !g.initialized {
		return nil, nil, nil, nil, fmt.Errorf("GPU collector not initialized")
	}

	// Get GPU utilization
	utilization, ret := g.device.GetUtilizationRates()
	if ret == nvml.SUCCESS {
		util := float64(utilization.Gpu)
		gpuUtil = &util

		// Memory utilization is also in utilization struct
		// but we'll get actual memory usage separately
	} else {
		g.logger.Warn("gpu.utilization.failed", "Failed to get GPU utilization", map[string]interface{}{
			"error": nvml.ErrorString(ret),
		})
	}

	// Get memory info
	memInfo, ret := g.device.GetMemoryInfo()
	if ret == nvml.SUCCESS {
		usedMB := memInfo.Used / (1024 * 1024)
		gpuMemMB = &usedMB
	} else {
		g.logger.Warn("gpu.memory.failed", "Failed to get GPU memory", map[string]interface{}{
			"error": nvml.ErrorString(ret),
		})
	}

	// Get power usage
	powerMilliwatts, ret := g.device.GetPowerUsage()
	if ret == nvml.SUCCESS {
		watts := float64(powerMilliwatts) / 1000.0
		gpuWatts = &watts
	} else {
		g.logger.Warn("gpu.power.failed", "Failed to get GPU power", map[string]interface{}{
			"error": nvml.ErrorString(ret),
		})
	}

	// Get temperature
	temp, ret := g.device.GetTemperature(nvml.TEMPERATURE_GPU)
	if ret == nvml.SUCCESS {
		tempFloat := float64(temp)
		tempGPU = &tempFloat
	} else {
		g.logger.Warn("gpu.temperature.failed", "Failed to get GPU temperature", map[string]interface{}{
			"error": nvml.ErrorString(ret),
		})
	}

	return gpuUtil, gpuMemMB, gpuWatts, tempGPU, nil
}

// Shutdown shuts down the GPU collector
func (g *GPUCollector) Shutdown() {
	if g.initialized {
		if shutdownRet := g.nvml.Shutdown(); shutdownRet != nvml.SUCCESS {
			g.logger.Warn("gpu.collector.shutdown.failed", "NVML shutdown reported an error", map[string]interface{}{
				"error": nvml.ErrorString(shutdownRet),
			})
		}
		g.initialized = false
		g.logger.Info("gpu.collector.shutdown", "GPU metrics collector shut down", nil)
	}
}

// IsInitialized returns whether the collector is initialized
func (g *GPUCollector) IsInitialized() bool {
	return g.initialized
}
