package idle

import (
	"sync"
	"time"
)

// SlidingWindow maintains a time-based sliding window of metric samples
type SlidingWindow struct {
	mu           sync.RWMutex
	samples      []MetricSample
	windowSize   time.Duration
	minSamples   int
	lastIdleTime time.Time
	idleDuration time.Duration
}

// NewSlidingWindow creates a new sliding window
func NewSlidingWindow(windowSeconds int, minSamples int) *SlidingWindow {
	return &SlidingWindow{
		samples:      make([]MetricSample, 0),
		windowSize:   time.Duration(windowSeconds) * time.Second,
		minSamples:   minSamples,
		lastIdleTime: time.Time{},
		idleDuration: 0,
	}
}

// AddSample adds a new metric sample to the window
func (w *SlidingWindow) AddSample(sample MetricSample) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Add new sample
	w.samples = append(w.samples, sample)

	// Remove samples outside the window
	cutoff := time.Now().Add(-w.windowSize)
	validSamples := make([]MetricSample, 0)
	for _, s := range w.samples {
		if s.Timestamp.After(cutoff) {
			validSamples = append(validSamples, s)
		}
	}
	w.samples = validSamples
}

// IsIdle checks if the system is currently idle based on thresholds
func (w *SlidingWindow) IsIdle(cpuThreshold, gpuThreshold float64) (bool, float64, float64) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if len(w.samples) < w.minSamples {
		return false, 0, 0
	}

	// Calculate average utilization over the window
	var cpuSum, gpuSum float64
	for _, s := range w.samples {
		cpuSum += s.CPUUtil
		gpuSum += s.GPUUtil
	}

	cpuAvg := cpuSum / float64(len(w.samples))
	gpuAvg := gpuSum / float64(len(w.samples))

	// System is idle if both CPU and GPU are below their thresholds
	idle := cpuAvg < cpuThreshold && gpuAvg < gpuThreshold

	return idle, cpuAvg, gpuAvg
}

// HasEnoughSamples checks if we have enough samples to make a decision
func (w *SlidingWindow) HasEnoughSamples() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.samples) >= w.minSamples
}

// GetIdleDuration returns how long the system has been continuously idle
func (w *SlidingWindow) GetIdleDuration(cpuThreshold, gpuThreshold float64) time.Duration {
	w.mu.Lock()
	defer w.mu.Unlock()

	idle, _, _ := w.isIdleLocked(cpuThreshold, gpuThreshold)

	now := time.Now()
	if idle {
		// System is currently idle
		if w.lastIdleTime.IsZero() {
			// Just became idle
			w.lastIdleTime = now
			w.idleDuration = 0
		} else {
			// Continue being idle
			w.idleDuration = now.Sub(w.lastIdleTime)
		}
	} else {
		// System is active - reset idle tracking
		w.lastIdleTime = time.Time{}
		w.idleDuration = 0
	}

	return w.idleDuration
}

// isIdleLocked is the internal idle check without locking (caller must hold lock)
func (w *SlidingWindow) isIdleLocked(cpuThreshold, gpuThreshold float64) (bool, float64, float64) {
	if len(w.samples) < w.minSamples {
		return false, 0, 0
	}

	var cpuSum, gpuSum float64
	for _, s := range w.samples {
		cpuSum += s.CPUUtil
		gpuSum += s.GPUUtil
	}

	cpuAvg := cpuSum / float64(len(w.samples))
	gpuAvg := gpuSum / float64(len(w.samples))

	idle := cpuAvg < cpuThreshold && gpuAvg < gpuThreshold

	return idle, cpuAvg, gpuAvg
}

// Reset clears all samples and resets idle tracking
func (w *SlidingWindow) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.samples = make([]MetricSample, 0)
	w.lastIdleTime = time.Time{}
	w.idleDuration = 0
}

// SampleCount returns the current number of samples in the window
func (w *SlidingWindow) SampleCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.samples)
}
