package diag

import "time"

// Manifest represents the diagnostic package manifest
type Manifest struct {
	Timestamp      string         `json:"timestamp"`
	Host           string         `json:"host"`
	AistackVersion string         `json:"aistack_version"`
	Files          []ManifestFile `json:"files"`
}

// ManifestFile represents a file in the diagnostic package
type ManifestFile struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

// Config configures diagnostic collection
type Config struct {
	LogDir        string
	ConfigPath    string
	OutputPath    string
	IncludeLogs   bool
	IncludeConfig bool
	Version       string
}

// NewConfig creates a default diagnostic config
func NewConfig(version string) *Config {
	return &Config{
		LogDir:        "/var/log/aistack",
		ConfigPath:    "/etc/aistack/config.yaml",
		OutputPath:    generateOutputPath(),
		IncludeLogs:   true,
		IncludeConfig: true,
		Version:       version,
	}
}

func generateOutputPath() string {
	timestamp := time.Now().UTC().Format("20060102-150405")
	return "aistack-diag-" + timestamp + ".zip"
}
