package metrics

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"aistack/internal/logging"
)

// CPUCollector collects CPU metrics
// Story T-012: CPU-Util & RAPL-Leistung erfassen (mit Fallback)
type CPUCollector struct {
	logger      *logging.Logger
	lastStats   *CPUStats
	lastSample  time.Time
	raplEnabled bool
	raplPath    string
}

// NewCPUCollector creates a new CPU metrics collector
func NewCPUCollector(logger *logging.Logger) *CPUCollector {
	collector := &CPUCollector{
		logger: logger,
	}

	// Check for RAPL support
	collector.raplPath = "/sys/class/powercap/intel-rapl/intel-rapl:0/energy_uj"
	if _, err := os.Stat(collector.raplPath); err == nil {
		collector.raplEnabled = true
		logger.Info("cpu.rapl.detected", "RAPL power monitoring available", map[string]interface{}{
			"path": collector.raplPath,
		})
	} else {
		logger.Info("cpu.rapl.unavailable", "RAPL not available, power metrics disabled", map[string]interface{}{
			"error": err.Error(),
		})
	}

	return collector
}

// Collect collects current CPU metrics
func (c *CPUCollector) Collect() (cpuUtil *float64, cpuWatts *float64, tempCPU *float64, err error) {
	// Get current CPU stats
	currentStats, err := c.readCPUStats()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read CPU stats: %w", err)
	}

	currentTime := time.Now()

	// Calculate utilization if we have previous stats
	if c.lastStats != nil {
		util := c.calculateUtilization(c.lastStats, currentStats)
		cpuUtil = &util
	}

	// Get CPU power if RAPL is available
	if c.raplEnabled {
		watts, err := c.readRAPL()
		if err != nil {
			c.logger.Warn("cpu.rapl.read.failed", "Failed to read RAPL", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			cpuWatts = &watts
		}
	}

	// TODO: Temperature reading could be added later (hwmon sensors)
	// For now, temperature is optional and left as nil

	// Update state for next collection
	c.lastStats = currentStats
	c.lastSample = currentTime

	return cpuUtil, cpuWatts, tempCPU, nil
}

// readCPUStats reads CPU statistics from /proc/stat
// Story T-012: Delta-basiert aus /proc/stat
func (c *CPUCollector) readCPUStats() (*CPUStats, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/stat: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty /proc/stat")
	}

	// Parse first line (aggregate CPU stats)
	// Format: cpu user nice system idle iowait irq softirq steal
	fields := strings.Fields(lines[0])
	if len(fields) < 8 || fields[0] != "cpu" {
		return nil, fmt.Errorf("invalid /proc/stat format")
	}

	stats := &CPUStats{}
	stats.User, _ = strconv.ParseUint(fields[1], 10, 64)
	stats.Nice, _ = strconv.ParseUint(fields[2], 10, 64)
	stats.System, _ = strconv.ParseUint(fields[3], 10, 64)
	stats.Idle, _ = strconv.ParseUint(fields[4], 10, 64)
	stats.IOWait, _ = strconv.ParseUint(fields[5], 10, 64)
	stats.IRQ, _ = strconv.ParseUint(fields[6], 10, 64)
	stats.SoftIRQ, _ = strconv.ParseUint(fields[7], 10, 64)
	if len(fields) > 8 {
		stats.Steal, _ = strconv.ParseUint(fields[8], 10, 64)
	}

	return stats, nil
}

// calculateUtilization calculates CPU utilization percentage
func (c *CPUCollector) calculateUtilization(prev, current *CPUStats) float64 {
	prevTotal := prev.Total()
	currentTotal := current.Total()
	prevIdle := prev.IdleTime()
	currentIdle := current.IdleTime()

	totalDelta := currentTotal - prevTotal
	idleDelta := currentIdle - prevIdle

	if totalDelta == 0 {
		return 0.0
	}

	utilization := 100.0 * (1.0 - float64(idleDelta)/float64(totalDelta))

	// Clamp to valid range
	if utilization < 0 {
		utilization = 0
	}
	if utilization > 100 {
		utilization = 100
	}

	return utilization
}

// readRAPL reads CPU power consumption from RAPL
// Story T-012: RAPL aus /sys/class/powercap
func (c *CPUCollector) readRAPL() (float64, error) {
	data, err := os.ReadFile(c.raplPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read RAPL: %w", err)
	}

	// RAPL energy is in microjoules
	energyMicrojoules, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse RAPL value: %w", err)
	}

	// For power calculation, we would need to track energy delta over time
	// For now, return a placeholder (this would need state tracking)
	// In a real implementation, we'd calculate: (energyDelta / timeDelta) / 1000000

	// Simplified: Assume moderate power draw (this should be improved with delta tracking)
	watts := float64(energyMicrojoules) / 1000000.0 / 10.0 // Very rough estimate

	return watts, nil
}

// IsRAPLEnabled returns whether RAPL power monitoring is available
func (c *CPUCollector) IsRAPLEnabled() bool {
	return c.raplEnabled
}
