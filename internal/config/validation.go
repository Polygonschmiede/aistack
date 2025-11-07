package config

import (
	"fmt"
	"strings"
)

const (
	// Container runtime types
	RuntimeDocker = "docker"
	RuntimePodman = "podman"
)

// Validate checks if the configuration is valid
func (c *Config) Validate() []ValidationError {
	var errors []ValidationError

	// Validate container_runtime
	if c.ContainerRuntime != RuntimeDocker && c.ContainerRuntime != RuntimePodman {
		errors = append(errors, ValidationError{
			Path:    "container_runtime",
			Message: fmt.Sprintf("must be 'docker' or 'podman', got '%s'", c.ContainerRuntime),
		})
	}

	// Validate profile
	validProfiles := []string{"minimal", "standard-gpu", "dev"}
	if !contains(validProfiles, c.Profile) {
		errors = append(errors, ValidationError{
			Path:    "profile",
			Message: fmt.Sprintf("must be one of %v, got '%s'", validProfiles, c.Profile),
		})
	}

	// Validate idle thresholds
	if c.Idle.CPUIdleThreshold < 0 || c.Idle.CPUIdleThreshold > 100 {
		errors = append(errors, ValidationError{
			Path:    "idle.cpu_idle_threshold",
			Message: fmt.Sprintf("must be between 0 and 100, got %d", c.Idle.CPUIdleThreshold),
		})
	}

	if c.Idle.GPUIdleThreshold < 0 || c.Idle.GPUIdleThreshold > 100 {
		errors = append(errors, ValidationError{
			Path:    "idle.gpu_idle_threshold",
			Message: fmt.Sprintf("must be between 0 and 100, got %d", c.Idle.GPUIdleThreshold),
		})
	}

	if c.Idle.WindowSeconds < 10 {
		errors = append(errors, ValidationError{
			Path:    "idle.window_seconds",
			Message: fmt.Sprintf("must be at least 10, got %d", c.Idle.WindowSeconds),
		})
	}

	if c.Idle.IdleTimeoutSeconds < 60 {
		errors = append(errors, ValidationError{
			Path:    "idle.idle_timeout_seconds",
			Message: fmt.Sprintf("must be at least 60, got %d", c.Idle.IdleTimeoutSeconds),
		})
	}

	// Validate power estimation
	if c.PowerEstimation.BaselineWatts < 0 {
		errors = append(errors, ValidationError{
			Path:    "power_estimation.baseline_watts",
			Message: fmt.Sprintf("must be non-negative, got %f", c.PowerEstimation.BaselineWatts),
		})
	}

	// Validate logging level
	validLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLevels, c.Logging.Level) {
		errors = append(errors, ValidationError{
			Path:    "logging.level",
			Message: fmt.Sprintf("must be one of %v, got '%s'", validLevels, c.Logging.Level),
		})
	}

	// Validate logging format
	validFormats := []string{"json", "text"}
	if !contains(validFormats, c.Logging.Format) {
		errors = append(errors, ValidationError{
			Path:    "logging.format",
			Message: fmt.Sprintf("must be one of %v, got '%s'", validFormats, c.Logging.Format),
		})
	}

	// Validate updates mode
	validModes := []string{"rolling", "pinned"}
	if !contains(validModes, c.Updates.Mode) {
		errors = append(errors, ValidationError{
			Path:    "updates.mode",
			Message: fmt.Sprintf("must be one of %v, got '%s'", validModes, c.Updates.Mode),
		})
	}

	// Validate WoL MAC address format (basic check)
	if c.WoL.MAC != "" && c.WoL.MAC != "00:00:00:00:00:00" {
		if !isValidMACFormat(c.WoL.MAC) {
			errors = append(errors, ValidationError{
				Path:    "wol.mac",
				Message: fmt.Sprintf("invalid MAC address format: %s", c.WoL.MAC),
			})
		}
	}

	return errors
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// isValidMACFormat checks if a string matches common MAC address formats
func isValidMACFormat(mac string) bool {
	// Accept colon-separated format (XX:XX:XX:XX:XX:XX)
	if len(mac) == 17 && strings.Count(mac, ":") == 5 {
		parts := strings.Split(mac, ":")
		if len(parts) == 6 {
			for _, part := range parts {
				if len(part) != 2 {
					return false
				}
				for _, c := range part {
					if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
						return false
					}
				}
			}
			return true
		}
	}
	// Accept dash-separated format (XX-XX-XX-XX-XX-XX)
	if len(mac) == 17 && strings.Count(mac, "-") == 5 {
		return isValidMACFormat(strings.ReplaceAll(mac, "-", ":"))
	}
	return false
}
