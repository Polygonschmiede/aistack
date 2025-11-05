package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Verify defaults
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"ContainerRuntime", cfg.ContainerRuntime, "docker"},
		{"Profile", cfg.Profile, "standard-gpu"},
		{"GPULock", cfg.GPULock, true},
		{"CPUIdleThreshold", cfg.Idle.CPUIdleThreshold, 10},
		{"GPUIdleThreshold", cfg.Idle.GPUIdleThreshold, 5},
		{"WindowSeconds", cfg.Idle.WindowSeconds, 300},
		{"IdleTimeoutSeconds", cfg.Idle.IdleTimeoutSeconds, 1800},
		{"BaselineWatts", cfg.PowerEstimation.BaselineWatts, 50.0},
		{"LogLevel", cfg.Logging.Level, "info"},
		{"LogFormat", cfg.Logging.Format, "json"},
		{"KeepCache", cfg.Models.KeepCacheOnUninstall, true},
		{"UpdatesMode", cfg.Updates.Mode, "rolling"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("DefaultConfig().%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestValidation_ValidConfig(t *testing.T) {
	cfg := DefaultConfig()
	errors := cfg.Validate()

	if len(errors) != 0 {
		t.Errorf("Validate() on default config returned errors: %v", errors)
	}
}

func TestValidation_InvalidContainerRuntime(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContainerRuntime = "invalid"

	errors := cfg.Validate()
	if len(errors) == 0 {
		t.Error("Validate() should return error for invalid container_runtime")
	}

	found := false
	for _, err := range errors {
		if err.Path == "container_runtime" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Validate() should return error for container_runtime field")
	}
}

func TestValidation_InvalidProfile(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Profile = "unknown-profile"

	errors := cfg.Validate()
	if len(errors) == 0 {
		t.Error("Validate() should return error for invalid profile")
	}
}

func TestValidation_CPUThresholdOutOfRange(t *testing.T) {
	tests := []struct {
		name      string
		threshold int
	}{
		{"negative", -1},
		{"too high", 101},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Idle.CPUIdleThreshold = tt.threshold

			errors := cfg.Validate()
			if len(errors) == 0 {
				t.Errorf("Validate() should return error for CPU threshold %d", tt.threshold)
			}
		})
	}
}

func TestValidation_GPUThresholdOutOfRange(t *testing.T) {
	tests := []struct {
		name      string
		threshold int
	}{
		{"negative", -1},
		{"too high", 101},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Idle.GPUIdleThreshold = tt.threshold

			errors := cfg.Validate()
			if len(errors) == 0 {
				t.Errorf("Validate() should return error for GPU threshold %d", tt.threshold)
			}
		})
	}
}

func TestValidation_WindowSecondsTooSmall(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Idle.WindowSeconds = 5

	errors := cfg.Validate()
	if len(errors) == 0 {
		t.Error("Validate() should return error for window_seconds < 10")
	}
}

func TestValidation_IdleTimeoutTooSmall(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Idle.IdleTimeoutSeconds = 30

	errors := cfg.Validate()
	if len(errors) == 0 {
		t.Error("Validate() should return error for idle_timeout_seconds < 60")
	}
}

func TestValidation_NegativeBaselineWatts(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PowerEstimation.BaselineWatts = -10

	errors := cfg.Validate()
	if len(errors) == 0 {
		t.Error("Validate() should return error for negative baseline_watts")
	}
}

func TestValidation_InvalidLogLevel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Logging.Level = "trace"

	errors := cfg.Validate()
	if len(errors) == 0 {
		t.Error("Validate() should return error for invalid log level")
	}
}

func TestValidation_InvalidLogFormat(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Logging.Format = "xml"

	errors := cfg.Validate()
	if len(errors) == 0 {
		t.Error("Validate() should return error for invalid log format")
	}
}

func TestValidation_InvalidUpdatesMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Updates.Mode = "automatic"

	errors := cfg.Validate()
	if len(errors) == 0 {
		t.Error("Validate() should return error for invalid updates mode")
	}
}

func TestValidation_InvalidMACAddress(t *testing.T) {
	tests := []struct {
		name string
		mac  string
	}{
		{"too short", "00:00:00:00:00"},
		{"invalid chars", "ZZ:ZZ:ZZ:ZZ:ZZ:ZZ"},
		{"wrong separator", "00.00.00.00.00.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.WoL.MAC = tt.mac

			errors := cfg.Validate()
			if len(errors) == 0 {
				t.Errorf("Validate() should return error for invalid MAC address: %s", tt.mac)
			}
		})
	}
}

func TestValidation_ValidMACAddresses(t *testing.T) {
	tests := []struct {
		name string
		mac  string
	}{
		{"colon-separated", "AA:BB:CC:DD:EE:FF"},
		{"dash-separated", "AA-BB-CC-DD-EE-FF"},
		{"lowercase", "aa:bb:cc:dd:ee:ff"},
		{"default placeholder", "00:00:00:00:00:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.WoL.MAC = tt.mac

			errors := cfg.Validate()
			if len(errors) != 0 {
				t.Errorf("Validate() should not return error for valid MAC address %s: %v", tt.mac, errors)
			}
		})
	}
}

func TestLoadFrom_ValidFile(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
container_runtime: podman
profile: minimal
idle:
  cpu_idle_threshold: 20
  gpu_idle_threshold: 10
logging:
  level: debug
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadFrom(configPath)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	// Verify overrides
	if cfg.ContainerRuntime != "podman" {
		t.Errorf("ContainerRuntime = %s, want podman", cfg.ContainerRuntime)
	}
	if cfg.Profile != "minimal" {
		t.Errorf("Profile = %s, want minimal", cfg.Profile)
	}
	if cfg.Idle.CPUIdleThreshold != 20 {
		t.Errorf("CPUIdleThreshold = %d, want 20", cfg.Idle.CPUIdleThreshold)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("LogLevel = %s, want debug", cfg.Logging.Level)
	}

	// Verify defaults are preserved for unspecified fields
	if cfg.PowerEstimation.BaselineWatts != 50 {
		t.Errorf("BaselineWatts = %f, want 50 (default)", cfg.PowerEstimation.BaselineWatts)
	}
}

func TestLoadFrom_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidContent := `
container_runtime: invalid_runtime
profile: unknown
`
	if err := os.WriteFile(configPath, []byte(invalidContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadFrom(configPath)
	if err == nil {
		t.Error("LoadFrom() should return error for invalid config")
	}
}

func TestLoadFrom_NonexistentFile(t *testing.T) {
	_, err := LoadFrom("/nonexistent/config.yaml")
	if err == nil {
		t.Error("LoadFrom() should return error for nonexistent file")
	}
}

func TestLoadFrom_MalformedYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	malformedContent := `
container_runtime: docker
  invalid_indentation: value
profile: minimal
`
	if err := os.WriteFile(configPath, []byte(malformedContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadFrom(configPath)
	if err == nil {
		t.Error("LoadFrom() should return error for malformed YAML")
	}
}

func TestMergeConfig(t *testing.T) {
	// Start with defaults
	dst := DefaultConfig()

	// Create overlay with partial config
	src := Config{
		ContainerRuntime: "podman",
		Idle: IdleConfig{
			CPUIdleThreshold: 25,
		},
		Logging: LoggingConfig{
			Level: "warn",
		},
	}

	mergeConfig(&dst, &src)

	// Verify overridden values
	if dst.ContainerRuntime != "podman" {
		t.Errorf("ContainerRuntime = %s, want podman", dst.ContainerRuntime)
	}
	if dst.Idle.CPUIdleThreshold != 25 {
		t.Errorf("CPUIdleThreshold = %d, want 25", dst.Idle.CPUIdleThreshold)
	}
	if dst.Logging.Level != "warn" {
		t.Errorf("LogLevel = %s, want warn", dst.Logging.Level)
	}

	// Verify preserved defaults
	if dst.Profile != "standard-gpu" {
		t.Errorf("Profile = %s, want standard-gpu (default)", dst.Profile)
	}
	if dst.Idle.GPUIdleThreshold != 5 {
		t.Errorf("GPUIdleThreshold = %d, want 5 (default)", dst.Idle.GPUIdleThreshold)
	}
	if dst.Logging.Format != "json" {
		t.Errorf("LogFormat = %s, want json (default)", dst.Logging.Format)
	}
}

func TestSystemConfigPath(t *testing.T) {
	path := SystemConfigPath()
	if path == "" {
		t.Error("SystemConfigPath() should not return empty string")
	}
	if filepath.Base(path) != "config.yaml" {
		t.Errorf("SystemConfigPath() basename = %s, want config.yaml", filepath.Base(path))
	}
}

func TestUserConfigPath(t *testing.T) {
	path := UserConfigPath()
	// May be empty if home dir not available
	if path != "" && filepath.Base(path) != "config.yaml" {
		t.Errorf("UserConfigPath() basename = %s, want config.yaml", filepath.Base(path))
	}
}

func TestValidationError_Error(t *testing.T) {
	err := ValidationError{
		Path:    "idle.cpu_idle_threshold",
		Message: "must be between 0 and 100",
	}

	expected := "idle.cpu_idle_threshold: must be between 0 and 100"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %s, want %s", err.Error(), expected)
	}
}

func TestFormatValidationErrors_Single(t *testing.T) {
	errors := []ValidationError{
		{Path: "test.field", Message: "error message"},
	}

	result := formatValidationErrors(errors)
	expected := "test.field: error message"
	if result != expected {
		t.Errorf("formatValidationErrors() = %s, want %s", result, expected)
	}
}

func TestFormatValidationErrors_Multiple(t *testing.T) {
	errors := []ValidationError{
		{Path: "field1", Message: "error 1"},
		{Path: "field2", Message: "error 2"},
	}

	result := formatValidationErrors(errors)
	if result == "" {
		t.Error("formatValidationErrors() should not return empty string for multiple errors")
	}
	// Should contain count
	if len(result) < 10 {
		t.Errorf("formatValidationErrors() result too short: %s", result)
	}
}

func TestFormatValidationErrors_Empty(t *testing.T) {
	result := formatValidationErrors([]ValidationError{})
	if result != "" {
		t.Errorf("formatValidationErrors() = %s, want empty string", result)
	}
}
