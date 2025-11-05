package diag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aistack/internal/logging"
)

func TestCollector_CollectLogs(t *testing.T) {
	// Create temp log directory
	tmpDir, err := os.MkdirTemp("", "aistack-diag-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test log files
	logFiles := map[string]string{
		"agent.log":   "log line 1\nlog line 2\n",
		"metrics.log": "metric data\n",
		"app.log":     "application log\n",
	}

	for name, content := range logFiles {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create collector
	config := &DiagConfig{
		LogDir:      tmpDir,
		IncludeLogs: true,
	}
	logger := logging.NewLogger(logging.LevelInfo)
	collector := NewCollector(config, logger)

	// Collect logs
	files, err := collector.CollectLogs()
	if err != nil {
		t.Fatalf("CollectLogs() error = %v", err)
	}

	// Verify all log files were collected
	if len(files) != len(logFiles) {
		t.Errorf("Expected %d files, got %d", len(logFiles), len(files))
	}

	for name, expectedContent := range logFiles {
		key := "logs/" + name
		content, exists := files[key]
		if !exists {
			t.Errorf("File %s not found in collected files", name)
			continue
		}

		if string(content) != expectedContent {
			t.Errorf("File %s content = %q, want %q", name, string(content), expectedContent)
		}
	}
}

func TestCollector_CollectLogs_MissingDirectory(t *testing.T) {
	config := &DiagConfig{
		LogDir:      "/nonexistent/path",
		IncludeLogs: true,
	}
	logger := logging.NewLogger(logging.LevelInfo)
	collector := NewCollector(config, logger)

	// Should not error, just return empty map
	files, err := collector.CollectLogs()
	if err != nil {
		t.Fatalf("CollectLogs() error = %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected empty map, got %d files", len(files))
	}
}

func TestCollector_CollectLogs_Disabled(t *testing.T) {
	config := &DiagConfig{
		LogDir:      "/var/log/aistack",
		IncludeLogs: false,
	}
	logger := logging.NewLogger(logging.LevelInfo)
	collector := NewCollector(config, logger)

	files, err := collector.CollectLogs()
	if err != nil {
		t.Fatalf("CollectLogs() error = %v", err)
	}

	if files != nil {
		t.Error("Expected nil when logs disabled")
	}
}

func TestCollector_CollectConfig(t *testing.T) {
	// Create temp config file
	tmpDir, err := os.MkdirTemp("", "aistack-diag-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `log_level: info
api_key: sk-1234567890abcdef
timeout: 30
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create collector
	config := &DiagConfig{
		ConfigPath:    configPath,
		IncludeConfig: true,
	}
	logger := logging.NewLogger(logging.LevelInfo)
	collector := NewCollector(config, logger)

	// Collect config
	files, err := collector.CollectConfig()
	if err != nil {
		t.Fatalf("CollectConfig() error = %v", err)
	}

	// Verify config was collected
	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(files))
	}

	content, exists := files["config/config.yaml"]
	if !exists {
		t.Fatal("Config file not found")
	}

	contentStr := string(content)

	// Should not contain secret
	if strings.Contains(contentStr, "sk-1234567890abcdef") {
		t.Error("API key was not redacted")
	}

	// Should contain redaction marker
	if !strings.Contains(contentStr, "[REDACTED]") {
		t.Error("Redaction marker not present")
	}

	// Should contain non-sensitive data
	if !strings.Contains(contentStr, "log_level: info") {
		t.Error("Non-sensitive config was modified")
	}
}

func TestCollector_CollectConfig_MissingFile(t *testing.T) {
	config := &DiagConfig{
		ConfigPath:    "/nonexistent/config.yaml",
		IncludeConfig: true,
	}
	logger := logging.NewLogger(logging.LevelInfo)
	collector := NewCollector(config, logger)

	// Should not error, just return empty map
	files, err := collector.CollectConfig()
	if err != nil {
		t.Fatalf("CollectConfig() error = %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected empty map, got %d files", len(files))
	}
}

func TestCollector_CollectSystemInfo(t *testing.T) {
	config := &DiagConfig{
		Version: "0.9.0",
	}
	logger := logging.NewLogger(logging.LevelInfo)
	collector := NewCollector(config, logger)

	files, err := collector.CollectSystemInfo()
	if err != nil {
		t.Fatalf("CollectSystemInfo() error = %v", err)
	}

	// Verify system info file
	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(files))
	}

	content, exists := files["system_info.json"]
	if !exists {
		t.Fatal("system_info.json not found")
	}

	// Verify it's valid JSON
	contentStr := string(content)
	if !strings.Contains(contentStr, "timestamp") {
		t.Error("Timestamp not found in system info")
	}
	if !strings.Contains(contentStr, "0.9.0") {
		t.Error("Version not found in system info")
	}
}

func TestCalculateSHA256(t *testing.T) {
	data := []byte("test content")
	hash := CalculateSHA256(data)

	// Verify hash format (64 hex characters)
	if len(hash) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash))
	}

	// Same data should produce same hash
	hash2 := CalculateSHA256(data)
	if hash != hash2 {
		t.Error("Same data produced different hashes")
	}

	// Different data should produce different hash
	hash3 := CalculateSHA256([]byte("different content"))
	if hash == hash3 {
		t.Error("Different data produced same hash")
	}
}
