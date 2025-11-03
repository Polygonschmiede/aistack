package wol

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"aistack/internal/configdir"
)

const wolConfigFilename = "wol_config.json"

// ConfigPath returns the absolute path to the wol_config.json file
func ConfigPath() string {
	return filepath.Join(configdir.ConfigDir(), wolConfigFilename)
}

// SaveConfig persists the WoL configuration to disk
func SaveConfig(cfg WoLConfig) error {
	path := filepath.Clean(ConfigPath())
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("failed to create WoL config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal WoL config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write WoL config: %w", err)
	}

	return nil
}

// LoadConfig loads the WoL configuration from disk
func LoadConfig() (WoLConfig, error) {
	path := filepath.Clean(ConfigPath())
	data, err := os.ReadFile(path) // #nosec G304 -- path is derived from ConfigDir and not user-controlled
	if err != nil {
		return WoLConfig{}, err
	}

	var cfg WoLConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return WoLConfig{}, fmt.Errorf("failed to decode WoL config: %w", err)
	}

	return cfg, nil
}
