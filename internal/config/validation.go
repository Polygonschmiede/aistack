package config

import (
	"fmt"
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

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
