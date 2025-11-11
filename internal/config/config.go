package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"aistack/internal/configdir"
)

const (
	systemConfigFile = "config.yaml"
	userConfigDir    = ".aistack"
	userConfigFile   = "config.yaml"
)

// Load loads and merges configuration from system and user files
// Priority: defaults < system config < user config
func Load() (Config, error) {
	// Start with defaults
	cfg := DefaultConfig()

	// Try to load system config
	systemPath := filepath.Join(configdir.ConfigDir(), systemConfigFile)
	if err := mergeConfigFile(&cfg, systemPath); err != nil {
		if !os.IsNotExist(err) {
			return cfg, fmt.Errorf("failed to load system config: %w", err)
		}
		// System config not existing is OK, continue with defaults
	}

	// Try to load user config
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userPath := filepath.Join(homeDir, userConfigDir, userConfigFile)
		if err := mergeConfigFile(&cfg, userPath); err != nil {
			if !os.IsNotExist(err) {
				return cfg, fmt.Errorf("failed to load user config: %w", err)
			}
			// User config not existing is OK
		}
	}

	// Validate the merged configuration
	if validationErrors := cfg.Validate(); len(validationErrors) > 0 {
		return cfg, fmt.Errorf("config.validation.error: %v", formatValidationErrors(validationErrors))
	}

	return cfg, nil
}

// LoadFrom loads configuration from a specific file path
func LoadFrom(path string) (Config, error) {
	cfg := DefaultConfig()
	if err := mergeConfigFile(&cfg, path); err != nil {
		return cfg, fmt.Errorf("failed to load config from %s: %w", path, err)
	}

	// Validate
	if validationErrors := cfg.Validate(); len(validationErrors) > 0 {
		return cfg, fmt.Errorf("config.validation.error: %v", formatValidationErrors(validationErrors))
	}

	return cfg, nil
}

// mergeConfigFile reads a YAML file and merges it into the existing config
func mergeConfigFile(cfg *Config, path string) error {
	data, err := os.ReadFile(filepath.Clean(path)) // #nosec G304 -- path is constructed from trusted sources
	if err != nil {
		return err
	}

	// Parse YAML
	var overlay Config
	if err := yaml.Unmarshal(data, &overlay); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Merge non-zero values from overlay into cfg
	mergeConfig(cfg, &overlay)

	return nil
}

// mergeConfig merges non-zero values from src into dst
func mergeConfig(dst, src *Config) {
	// Simple field merging - only overwrite if source has non-zero value
	if src.ContainerRuntime != "" {
		dst.ContainerRuntime = src.ContainerRuntime
	}
	if src.Profile != "" {
		dst.Profile = src.Profile
	}
	// GPULock is a bool, so we need special handling
	// We assume that if it's explicitly set in YAML, it will be parsed
	// For now, we'll just overwrite it
	dst.GPULock = src.GPULock

	// Merge logging config
	if src.Logging.Level != "" {
		dst.Logging.Level = src.Logging.Level
	}
	if src.Logging.Format != "" {
		dst.Logging.Format = src.Logging.Format
	}

	// Merge models config
	dst.Models.KeepCacheOnUninstall = src.Models.KeepCacheOnUninstall

	// Merge updates config
	if src.Updates.Mode != "" {
		dst.Updates.Mode = src.Updates.Mode
	}
}

// formatValidationErrors formats validation errors for display
func formatValidationErrors(errors []ValidationError) string {
	if len(errors) == 0 {
		return ""
	}
	if len(errors) == 1 {
		return errors[0].Error()
	}
	result := fmt.Sprintf("%d validation errors:\n", len(errors))
	for _, err := range errors {
		result += "  - " + err.Error() + "\n"
	}
	return result
}

// SystemConfigPath returns the path to the system configuration file
func SystemConfigPath() string {
	return filepath.Join(configdir.ConfigDir(), systemConfigFile)
}

// UserConfigPath returns the path to the user configuration file
func UserConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, userConfigDir, userConfigFile)
}
