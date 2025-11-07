package diag

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aistack/internal/logging"
)

func TestPackager_CreatePackage(t *testing.T) {
	// Create temp directories
	tmpDir, err := os.MkdirTemp("", "aistack-diag-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logDir := filepath.Join(tmpDir, "logs")
	if mkErr := os.MkdirAll(logDir, 0o755); mkErr != nil {
		t.Fatal(mkErr)
	}

	// Create test log files
	testLog := filepath.Join(logDir, "test.log")
	if writeErr := os.WriteFile(testLog, []byte("test log content\n"), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}

	// Create test config
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `log_level: info
api_key: sk-secret123
timeout: 30
`
	if writeErr := os.WriteFile(configPath, []byte(configContent), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}

	// Output path
	outputPath := filepath.Join(tmpDir, "diag.zip")

	// Create packager
	config := &Config{
		LogDir:        logDir,
		ConfigPath:    configPath,
		OutputPath:    outputPath,
		IncludeLogs:   true,
		IncludeConfig: true,
		Version:       "0.9.0-test",
	}
	logger := logging.NewLogger(logging.LevelInfo)
	packager := NewPackager(config, logger)

	// Create package
	zipPath, err := packager.CreatePackage()
	if err != nil {
		t.Fatalf("CreatePackage() error = %v", err)
	}

	if zipPath != outputPath {
		t.Errorf("Expected output path %s, got %s", outputPath, zipPath)
	}

	// Verify ZIP file exists
	if _, statErr := os.Stat(zipPath); os.IsNotExist(statErr) {
		t.Fatal("ZIP file was not created")
	}

	// Open and verify ZIP contents
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("Failed to open ZIP: %v", err)
	}
	defer zipReader.Close()

	// Check for expected files
	expectedFiles := map[string]bool{
		"logs/test.log":      false,
		"config/config.yaml": false,
		"system_info.json":   false,
		"diag_manifest.json": false,
	}

	for _, f := range zipReader.File {
		if _, expected := expectedFiles[f.Name]; expected {
			expectedFiles[f.Name] = true
		}
	}

	// Verify all expected files present
	for name, found := range expectedFiles {
		if !found {
			t.Errorf("Expected file %s not found in ZIP", name)
		}
	}

	// Verify manifest content
	var manifestFile *zip.File
	for _, f := range zipReader.File {
		if f.Name == "diag_manifest.json" {
			manifestFile = f
			break
		}
	}

	if manifestFile == nil {
		t.Fatal("Manifest file not found")
	}

	manifestReader, err := manifestFile.Open()
	if err != nil {
		t.Fatalf("Failed to open manifest: %v", err)
	}
	defer manifestReader.Close()

	var manifest Manifest
	if err := json.NewDecoder(manifestReader).Decode(&manifest); err != nil {
		t.Fatalf("Failed to decode manifest: %v", err)
	}

	// Verify manifest fields
	if manifest.AistackVersion != "0.9.0-test" {
		t.Errorf("Expected version 0.9.0-test, got %s", manifest.AistackVersion)
	}

	if manifest.Timestamp == "" {
		t.Error("Manifest timestamp is empty")
	}

	if manifest.Host == "" {
		t.Error("Manifest host is empty")
	}

	// Should have at least 4 files (logs, config, system_info, manifest itself not counted in Files array)
	if len(manifest.Files) < 3 {
		t.Errorf("Expected at least 3 files in manifest, got %d", len(manifest.Files))
	}

	// Verify config was redacted
	var configFile *zip.File
	for _, f := range zipReader.File {
		if f.Name == "config/config.yaml" {
			configFile = f
			break
		}
	}

	if configFile != nil {
		configReader, err := configFile.Open()
		if err != nil {
			t.Fatalf("Failed to open config: %v", err)
		}
		defer configReader.Close()

		buf := make([]byte, configFile.UncompressedSize64)
		_, err = configReader.Read(buf)
		if err != nil && err.Error() != "EOF" {
			t.Fatalf("Failed to read config: %v", err)
		}

		configStr := string(buf)
		if strings.Contains(configStr, "sk-secret123") {
			t.Error("Secret was not redacted in config")
		}

		if !strings.Contains(configStr, "[REDACTED]") {
			t.Error("Redaction marker not found in config")
		}
	}
}

func TestPackager_CreatePackage_PartialFailure(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "aistack-diag-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	outputPath := filepath.Join(tmpDir, "diag.zip")

	// Config with non-existent paths (should create partial package)
	config := &Config{
		LogDir:        "/nonexistent/logs",
		ConfigPath:    "/nonexistent/config.yaml",
		OutputPath:    outputPath,
		IncludeLogs:   true,
		IncludeConfig: true,
		Version:       "0.9.0-test",
	}
	logger := logging.NewLogger(logging.LevelInfo)
	packager := NewPackager(config, logger)

	// Should still create a package (with at least system_info and manifest)
	zipPath, err := packager.CreatePackage()
	if err != nil {
		t.Fatalf("CreatePackage() should not fail with missing files: %v", err)
	}

	// Verify ZIP file exists
	if _, statErr := os.Stat(zipPath); os.IsNotExist(statErr) {
		t.Fatal("ZIP file was not created")
	}

	// Open and verify ZIP contains at least system_info and manifest
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("Failed to open ZIP: %v", err)
	}
	defer zipReader.Close()

	foundSystemInfo := false
	foundManifest := false

	for _, f := range zipReader.File {
		if f.Name == "system_info.json" {
			foundSystemInfo = true
		}
		if f.Name == "diag_manifest.json" {
			foundManifest = true
		}
	}

	if !foundSystemInfo {
		t.Error("system_info.json should be present even with missing logs/config")
	}

	if !foundManifest {
		t.Error("diag_manifest.json should be present")
	}
}

func TestNewConfig(t *testing.T) {
	config := NewConfig("1.0.0")

	if config.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", config.Version)
	}

	if config.LogDir != "/var/log/aistack" {
		t.Errorf("Expected default log dir /var/log/aistack, got %s", config.LogDir)
	}

	if config.ConfigPath != "/etc/aistack/config.yaml" {
		t.Errorf("Expected default config path /etc/aistack/config.yaml, got %s", config.ConfigPath)
	}

	if !config.IncludeLogs {
		t.Error("Expected IncludeLogs to be true by default")
	}

	if !config.IncludeConfig {
		t.Error("Expected IncludeConfig to be true by default")
	}

	if config.OutputPath == "" {
		t.Error("OutputPath should be auto-generated")
	}

	// Should have timestamp format
	if !strings.HasPrefix(config.OutputPath, "aistack-diag-") {
		t.Errorf("Expected output path to start with 'aistack-diag-', got %s", config.OutputPath)
	}

	if !strings.HasSuffix(config.OutputPath, ".zip") {
		t.Errorf("Expected output path to end with '.zip', got %s", config.OutputPath)
	}
}
