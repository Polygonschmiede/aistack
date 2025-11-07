package config

import (
	"fmt"
	"strings"
)

const (
	// RuntimeDocker identifies the Docker container runtime option.
	RuntimeDocker = "docker"
	// RuntimePodman identifies the Podman container runtime option.
	RuntimePodman = "podman"
)

// Validate checks if the configuration is valid
func (c *Config) Validate() []ValidationError {
	var errors []ValidationError

	errors = append(errors, c.validateContainerRuntime()...)
	errors = append(errors, c.validateProfile()...)
	errors = append(errors, c.validateIdleSettings()...)
	errors = append(errors, c.validatePowerEstimation()...)
	errors = append(errors, c.validateLogging()...)
	errors = append(errors, c.validateUpdates()...)
	errors = append(errors, c.validateWOL()...)

	return errors
}

func (c *Config) validateContainerRuntime() []ValidationError {
	if c.ContainerRuntime == RuntimeDocker || c.ContainerRuntime == RuntimePodman {
		return nil
	}

	return []ValidationError{{
		Path:    "container_runtime",
		Message: fmt.Sprintf("must be '%s' or '%s', got '%s'", RuntimeDocker, RuntimePodman, c.ContainerRuntime),
	}}
}

func (c *Config) validateProfile() []ValidationError {
	validProfiles := []string{"minimal", "standard-gpu", "dev"}
	if contains(validProfiles, c.Profile) {
		return nil
	}

	return []ValidationError{{
		Path:    "profile",
		Message: fmt.Sprintf("must be one of %v, got '%s'", validProfiles, c.Profile),
	}}
}

func (c *Config) validateIdleSettings() []ValidationError {
	var errors []ValidationError

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

	return errors
}

func (c *Config) validatePowerEstimation() []ValidationError {
	if c.PowerEstimation.BaselineWatts >= 0 {
		return nil
	}

	return []ValidationError{{
		Path:    "power_estimation.baseline_watts",
		Message: fmt.Sprintf("must be non-negative, got %f", c.PowerEstimation.BaselineWatts),
	}}
}

func (c *Config) validateLogging() []ValidationError {
	var errors []ValidationError
	validLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLevels, c.Logging.Level) {
		errors = append(errors, ValidationError{
			Path:    "logging.level",
			Message: fmt.Sprintf("must be one of %v, got '%s'", validLevels, c.Logging.Level),
		})
	}

	validFormats := []string{"json", "text"}
	if !contains(validFormats, c.Logging.Format) {
		errors = append(errors, ValidationError{
			Path:    "logging.format",
			Message: fmt.Sprintf("must be one of %v, got '%s'", validFormats, c.Logging.Format),
		})
	}

	return errors
}

func (c *Config) validateUpdates() []ValidationError {
	validModes := []string{"rolling", "pinned"}
	if contains(validModes, c.Updates.Mode) {
		return nil
	}

	return []ValidationError{{
		Path:    "updates.mode",
		Message: fmt.Sprintf("must be one of %v, got '%s'", validModes, c.Updates.Mode),
	}}
}

func (c *Config) validateWOL() []ValidationError {
	if c.WoL.MAC == "" || c.WoL.MAC == "00:00:00:00:00:00" {
		return nil
	}

	if isValidMACFormat(c.WoL.MAC) {
		return nil
	}

	return []ValidationError{{
		Path:    "wol.mac",
		Message: fmt.Sprintf("invalid MAC address format: %s", c.WoL.MAC),
	}}
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
