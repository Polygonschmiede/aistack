//go:build !cuda

package metrics

import (
	"fmt"

	"aistack/internal/logging"
)

// GPUCollector is a no-op collector used when CUDA/NVML support is disabled.
type GPUCollector struct {
	logger      *logging.Logger
	initialized bool
}

// NewGPUCollector creates a stub GPU collector that records that NVML is unavailable.
func NewGPUCollector(logger *logging.Logger) *GPUCollector {
	return &GPUCollector{logger: logger}
}

// Initialize logs that GPU metrics are disabled.
func (g *GPUCollector) Initialize() error {
	if g.logger != nil {
		g.logger.Info("gpu.collector.disabled", "Skipping GPU metrics collection (built without cuda tag)", nil)
	}
	g.initialized = false
	return fmt.Errorf("gpu metrics disabled: rebuild with -tags cuda")
}

// Collect returns nil values because GPU metrics are unavailable.
func (g *GPUCollector) Collect() (gpuUtil *float64, gpuMemMB *uint64, gpuWatts *float64, tempGPU *float64, err error) {
	return nil, nil, nil, nil, fmt.Errorf("gpu metrics disabled: rebuild with -tags cuda")
}

// Shutdown is a no-op for the stub collector.
func (g *GPUCollector) Shutdown() {}

// IsInitialized always reports false for the stub collector.
func (g *GPUCollector) IsInitialized() bool {
	return false
}
