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
	logger          *logging.Logger
	lastStats       *CPUStats
	lastSample      time.Time
	raplEnabled     bool
	raplPath        string
	enablePower     bool
	lastEnergyMicro uint64
	lastEnergyTime  time.Time
	hasEnergySample bool
}

// NewCPUCollector creates a new CPU metrics collector
func NewCPUCollector(logger *logging.Logger, enablePower bool) *CPUCollector {
	collector := &CPUCollector{
		logger:      logger,
		enablePower: enablePower,
	}

	if enablePower {
		collector.detectRAPL()
	}

	return collector
}

// detectRAPL checks the filesystem for RAPL support
func (c *CPUCollector) detectRAPL() {
	c.raplPath = "/sys/class/powercap/intel-rapl/intel-rapl:0/energy_uj"
	if _, err := os.Stat(c.raplPath); err == nil {
		c.raplEnabled = true
		c.logger.Info("cpu.rapl.detected", "RAPL power monitoring available", map[string]interface{}{
			"path": c.raplPath,
		})
	} else {
		c.raplEnabled = false
		c.logger.Info("cpu.rapl.unavailable", "RAPL not available, power metrics disabled", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// EnablePowerMetrics toggles RAPL power collection at runtime
func (c *CPUCollector) EnablePowerMetrics(enable bool) {
	c.enablePower = enable
	if enable {
		c.detectRAPL()
	}
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

	// Get CPU power if enabled and RAPL available
	if c.enablePower && c.raplEnabled {
		watts, err := c.readRAPLWatts(currentTime)
		if err != nil {
			c.logger.Warn("cpu.rapl.read.failed", "Failed to read RAPL", map[string]interface{}{
				"error": err.Error(),
			})
		} else if watts != nil {
			cpuWatts = watts
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

	parse := func(value string) (uint64, error) {
		parsedValue, parseErr := strconv.ParseUint(value, 10, 64)
		if parseErr != nil {
			return 0, fmt.Errorf("invalid cpu stat value %q: %w", value, parseErr)
		}
		return parsedValue, nil
	}

	if stats.User, err = parse(fields[1]); err != nil {
		return nil, err
	}
	if stats.Nice, err = parse(fields[2]); err != nil {
		return nil, err
	}
	if stats.System, err = parse(fields[3]); err != nil {
		return nil, err
	}
	if stats.Idle, err = parse(fields[4]); err != nil {
		return nil, err
	}
	if stats.IOWait, err = parse(fields[5]); err != nil {
		return nil, err
	}
	if stats.IRQ, err = parse(fields[6]); err != nil {
		return nil, err
	}
	if stats.SoftIRQ, err = parse(fields[7]); err != nil {
		return nil, err
	}
	if len(fields) > 8 {
		value, parseErr := parse(fields[8])
		if parseErr != nil {
			return nil, parseErr
		}
		stats.Steal = value
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
func (c *CPUCollector) readRAPLWatts(now time.Time) (*float64, error) {
	data, err := os.ReadFile(c.raplPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read RAPL: %w", err)
	}

	energyMicrojoules, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RAPL value: %w", err)
	}

	if !c.hasEnergySample {
		c.lastEnergyMicro = energyMicrojoules
		c.lastEnergyTime = now
		c.hasEnergySample = true
		return nil, nil
	}

	// Handle counter rollover
	var deltaEnergy uint64
	if energyMicrojoules >= c.lastEnergyMicro {
		deltaEnergy = energyMicrojoules - c.lastEnergyMicro
	} else {
		// Counter wrapped; reset tracking to avoid bogus negative values
		c.lastEnergyMicro = energyMicrojoules
		c.lastEnergyTime = now
		return nil, nil
	}

	elapsed := now.Sub(c.lastEnergyTime)
	if elapsed <= 0 {
		return nil, nil
	}

	watts := float64(deltaEnergy) / 1_000_000.0 / elapsed.Seconds()

	c.lastEnergyMicro = energyMicrojoules
	c.lastEnergyTime = now

	return &watts, nil
}

// IsRAPLEnabled returns whether RAPL power monitoring is available
func (c *CPUCollector) IsRAPLEnabled() bool {
	return c.raplEnabled
}
