package metrics

import (
	"time"

	"aistack/internal/logging"
)

// Collector aggregates metrics from CPU and GPU collectors
type Collector struct {
	logger       *logging.Logger
	config       MetricsConfig
	cpuCollector *CPUCollector
	gpuCollector *GPUCollector
	writer       *Writer
}

// NewCollector creates a new metrics collector
func NewCollector(config MetricsConfig, logger *logging.Logger) *Collector {
	return &Collector{
		logger:       logger,
		config:       config,
		cpuCollector: NewCPUCollector(logger, config.EnableCPUPower),
		writer:       NewWriter(logger),
	}
}

// Initialize initializes the metrics collector
func (c *Collector) Initialize() error {
	c.logger.Info("metrics.collector.init", "Initializing metrics collector", map[string]interface{}{
		"sample_interval": c.config.SampleInterval.String(),
		"enable_gpu":      c.config.EnableGPU,
	})

	// Initialize GPU collector if enabled
	if c.config.EnableGPU {
		c.gpuCollector = NewGPUCollector(c.logger)
		if err := c.gpuCollector.Initialize(); err != nil {
			c.logger.Warn("metrics.gpu.init.failed", "GPU metrics disabled", map[string]interface{}{
				"error": err.Error(),
			})
			c.config.EnableGPU = false // Disable if init fails
		}
	}

	// Disable CPU power metrics if requested or unavailable
	if !c.config.EnableCPUPower && c.cpuCollector != nil {
		c.cpuCollector.EnablePowerMetrics(false)
	}

	return nil
}

// CollectSample collects a single metrics sample
func (c *Collector) CollectSample() (MetricsSample, error) {
	sample := MetricsSample{
		Timestamp: time.Now().UTC(),
	}

	// Collect CPU metrics
	cpuUtil, cpuWatts, tempCPU, err := c.cpuCollector.Collect()
	if err != nil {
		c.logger.Warn("metrics.cpu.collect.failed", "Failed to collect CPU metrics", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		sample.CPUUtil = cpuUtil
		sample.CPUWatts = cpuWatts
		sample.TempCPU = tempCPU
	}

	// Collect GPU metrics if enabled
	if c.config.EnableGPU && c.gpuCollector != nil && c.gpuCollector.IsInitialized() {
		gpuUtil, gpuMemMB, gpuWatts, tempGPU, err := c.gpuCollector.Collect()
		if err != nil {
			c.logger.Warn("metrics.gpu.collect.failed", "Failed to collect GPU metrics", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			sample.GPUUtil = gpuUtil
			sample.GPUMemMB = gpuMemMB
			sample.GPUWatts = gpuWatts
			sample.TempGPU = tempGPU
		}
	}

	// Calculate estimated total power
	estTotal := c.calculateTotalPower(sample)
	sample.EstTotalW = &estTotal

	return sample, nil
}

// calculateTotalPower estimates total system power consumption
func (c *Collector) calculateTotalPower(sample MetricsSample) float64 {
	total := c.config.BaselinePowerW

	if sample.CPUWatts != nil {
		total += *sample.CPUWatts
	}

	if sample.GPUWatts != nil {
		total += *sample.GPUWatts
	}

	return total
}

// WriteSample writes a sample to the metrics log
func (c *Collector) WriteSample(sample MetricsSample, logPath string) error {
	return c.writer.Write(sample, logPath)
}

// Run starts the metrics collection loop
func (c *Collector) Run(logPath string, stopChan <-chan struct{}) error {
	ticker := time.NewTicker(c.config.SampleInterval)
	defer ticker.Stop()

	c.logger.Info("metrics.collector.start", "Metrics collection started", map[string]interface{}{
		"interval": c.config.SampleInterval.String(),
		"log_path": logPath,
	})

	for {
		select {
		case <-ticker.C:
			sample, err := c.CollectSample()
			if err != nil {
				c.logger.Error("metrics.sample.failed", "Failed to collect sample", map[string]interface{}{
					"error": err.Error(),
				})
				continue
			}

			if err := c.WriteSample(sample, logPath); err != nil {
				c.logger.Error("metrics.write.failed", "Failed to write sample", map[string]interface{}{
					"error": err.Error(),
				})
			}

		case <-stopChan:
			c.logger.Info("metrics.collector.stop", "Metrics collection stopped", nil)
			return nil
		}
	}
}

// Shutdown shuts down the collector
func (c *Collector) Shutdown() {
	if c.gpuCollector != nil {
		c.gpuCollector.Shutdown()
	}
}
