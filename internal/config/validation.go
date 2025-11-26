package config

import (
	"fmt"
	"regexp"
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
	errors = append(errors, c.validateIdle()...)
	errors = append(errors, c.validatePowerEstimation()...)
	errors = append(errors, c.validateWoL()...)
	errors = append(errors, c.validateLogging()...)
	errors = append(errors, c.validateUpdates()...)

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

func (c *Config) validateIdle() []ValidationError {
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
	if c.PowerEstimation.BaselineWatts < 0 {
		return []ValidationError{{
			Path:    "power_estimation.baseline_watts",
			Message: fmt.Sprintf("must be non-negative, got %f", c.PowerEstimation.BaselineWatts),
		}}
	}
	return nil
}

func (c *Config) validateWoL() []ValidationError {
	// If MAC is default placeholder, it's valid
	if c.WoL.MAC == "00:00:00:00:00:00" || c.WoL.MAC == "" {
		return nil
	}

	// Validate MAC address format (supports : and - separators)
	macPattern := `^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`
	matched, err := regexp.MatchString(macPattern, c.WoL.MAC)
	if err != nil || !matched {
		return []ValidationError{{
			Path:    "wol.mac",
			Message: fmt.Sprintf("invalid MAC address format, got '%s'", c.WoL.MAC),
		}}
	}

	return nil
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
