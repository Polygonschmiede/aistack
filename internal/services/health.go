package services

import (
	"fmt"
	"net/http"
	"time"
)

// HealthStatus represents the health status of a service
type HealthStatus string

const (
	HealthGreen  HealthStatus = "green"
	HealthYellow HealthStatus = "yellow"
	HealthRed    HealthStatus = "red"
)

// HealthChecker is an interface for performing health checks
type HealthChecker interface {
	Check() (HealthStatus, error)
}

// HealthCheck represents a health check configuration
type HealthCheck struct {
	URL            string
	Timeout        time.Duration
	ExpectedStatus int
}

// DefaultHealthCheck returns a default health check configuration
func DefaultHealthCheck(url string) HealthCheck {
	return HealthCheck{
		URL:            url,
		Timeout:        10 * time.Second,
		ExpectedStatus: http.StatusOK,
	}
}

// Check performs the health check
func (hc HealthCheck) Check() (HealthStatus, error) {
	client := &http.Client{
		Timeout: hc.Timeout,
	}

	resp, err := client.Get(hc.URL)
	if err != nil {
		return HealthRed, fmt.Errorf("health check failed: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			// Nothing to log here; best-effort close.
		}
	}()

	if resp.StatusCode != hc.ExpectedStatus {
		return HealthYellow, fmt.Errorf("unexpected status code: got %d, want %d", resp.StatusCode, hc.ExpectedStatus)
	}

	return HealthGreen, nil
}

// CheckWithRetries performs health check with retries
func (hc HealthCheck) CheckWithRetries(maxRetries int, retryDelay time.Duration) (HealthStatus, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		status, err := hc.Check()
		if err == nil && status == HealthGreen {
			return HealthGreen, nil
		}

		lastErr = err
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	return HealthRed, fmt.Errorf("health check failed after %d retries: %w", maxRetries, lastErr)
}
