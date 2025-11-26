//go:build linux

package suspend

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

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
