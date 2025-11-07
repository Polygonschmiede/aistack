package logging

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger(LevelInfo)

	if logger == nil {
		t.Fatal("Expected non-nil logger")
	}

	if logger.minLevel != LevelInfo {
		t.Errorf("Expected minLevel to be %s, got %s", LevelInfo, logger.minLevel)
	}
}

func TestLogger_ShouldLog(t *testing.T) {
	tests := []struct {
		name     string
		minLevel Level
		logLevel Level
		want     bool
	}{
		{"debug logs when min is debug", LevelDebug, LevelDebug, true},
		{"info logs when min is debug", LevelDebug, LevelInfo, true},
		{"error logs when min is debug", LevelDebug, LevelError, true},
		{"debug does not log when min is info", LevelInfo, LevelDebug, false},
		{"info logs when min is info", LevelInfo, LevelInfo, true},
		{"error logs when min is info", LevelInfo, LevelError, true},
		{"info does not log when min is error", LevelError, LevelInfo, false},
		{"error logs when min is error", LevelError, LevelError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.minLevel)
			got := logger.shouldLog(tt.logLevel)
			if got != tt.want {
				t.Errorf("shouldLog() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogger_Log(t *testing.T) {
	// Redirect stderr to capture logs
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger := NewLogger(LevelInfo)
	payload := map[string]interface{}{
		"key": "value",
		"num": 42,
	}

	logger.Log(LevelInfo, "test.event", "Test message", payload)

	// Restore stderr and read output
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Parse JSON output
	var event Event
	if err := json.Unmarshal([]byte(output), &event); err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v\nOutput: %s", err, output)
	}

	// Validate event fields
	if event.Level != LevelInfo {
		t.Errorf("Expected level %s, got %s", LevelInfo, event.Level)
	}

	if event.Type != "test.event" {
		t.Errorf("Expected type 'test.event', got %s", event.Type)
	}

	if event.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got %s", event.Message)
	}

	if event.Payload["key"] != "value" {
		t.Errorf("Expected payload key 'key' to be 'value', got %v", event.Payload["key"])
	}

	// Timestamp should be present
	if event.Timestamp == "" {
		t.Error("Expected timestamp to be set")
	}
}

func TestLogger_Info(t *testing.T) {
	// Redirect stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger := NewLogger(LevelInfo)
	logger.Info("test.info", "Info message", nil)

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "test.info") {
		t.Errorf("Expected output to contain 'test.info', got: %s", output)
	}

	if !strings.Contains(output, "Info message") {
		t.Errorf("Expected output to contain 'Info message', got: %s", output)
	}
}

func TestLogger_Error(t *testing.T) {
	// Redirect stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger := NewLogger(LevelError)
	logger.Error("test.error", "Error message", map[string]interface{}{"code": 500})

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "test.error") {
		t.Errorf("Expected output to contain 'test.error', got: %s", output)
	}

	if !strings.Contains(output, "Error message") {
		t.Errorf("Expected output to contain 'Error message', got: %s", output)
	}

	if !strings.Contains(output, "500") {
		t.Errorf("Expected output to contain '500', got: %s", output)
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	// Create logger with Warn level
	logger := NewLogger(LevelWarn)

	// Redirect stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// This should be filtered out
	logger.Info("test.filtered", "Should not appear", nil)

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Output should be empty since Info is below Warn
	if strings.TrimSpace(output) != "" {
		t.Errorf("Expected no output for filtered log, got: %s", output)
	}
}

// Story T-027: File-based logging tests
func TestNewFileLogger(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-log-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := tmpDir + "/test.log"
	logger, err := NewFileLogger(LevelInfo, logPath)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	defer logger.Close()

	// Verify file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestNewFileLogger_CreatesDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-log-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Use nested directory that doesn't exist
	logPath := tmpDir + "/logs/app/test.log"
	logger, err := NewFileLogger(LevelInfo, logPath)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	defer logger.Close()

	// Verify file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestFileLogger_WritesJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-log-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := tmpDir + "/test.log"
	logger, err := NewFileLogger(LevelInfo, logPath)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}

	// Write log event
	logger.Info("test.event", "Test message", map[string]interface{}{
		"key": "value",
	})

	// Close to flush
	if closeErr := logger.Close(); closeErr != nil {
		t.Fatalf("Failed to close logger: %v", closeErr)
	}

	// Read log file
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Verify JSON format
	var event Event
	if err := json.Unmarshal(content, &event); err != nil {
		t.Fatalf("Log content is not valid JSON: %v", err)
	}

	// Verify fields
	if event.Level != LevelInfo {
		t.Errorf("Expected level %s, got %s", LevelInfo, event.Level)
	}
	if event.Type != "test.event" {
		t.Errorf("Expected type 'test.event', got %s", event.Type)
	}
	if event.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got %s", event.Message)
	}
}

func TestFileLogger_LevelFiltering(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-log-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := tmpDir + "/test.log"
	logger, err := NewFileLogger(LevelWarn, logPath)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	defer logger.Close()

	// Write events at different levels
	logger.Debug("test.debug", "Debug message", nil)
	logger.Info("test.info", "Info message", nil)
	logger.Warn("test.warn", "Warn message", nil)
	logger.Error("test.error", "Error message", nil)

	if closeErr := logger.Close(); closeErr != nil {
		t.Fatalf("Failed to close logger: %v", closeErr)
	}

	// Read log file
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)

	// Debug and Info should be filtered out
	if strings.Contains(contentStr, "test.debug") {
		t.Error("Debug event was logged despite LevelWarn filter")
	}
	if strings.Contains(contentStr, "test.info") {
		t.Error("Info event was logged despite LevelWarn filter")
	}

	// Warn and Error should be present
	if !strings.Contains(contentStr, "test.warn") {
		t.Error("Warn event was not logged")
	}
	if !strings.Contains(contentStr, "test.error") {
		t.Error("Error event was not logged")
	}
}

func TestFileLogger_Append(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-log-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := tmpDir + "/test.log"

	// Create first logger and write event
	logger1, err := NewFileLogger(LevelInfo, logPath)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	logger1.Info("test.first", "First message", nil)
	if closeErr := logger1.Close(); closeErr != nil {
		t.Fatalf("Failed to close logger: %v", closeErr)
	}

	// Create second logger and write event
	logger2, err := NewFileLogger(LevelInfo, logPath)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	logger2.Info("test.second", "Second message", nil)
	if closeErr := logger2.Close(); closeErr != nil {
		t.Fatalf("Failed to close logger: %v", closeErr)
	}

	// Read log file
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)

	// Both events should be present
	if !strings.Contains(contentStr, "test.first") {
		t.Error("First event was not found")
	}
	if !strings.Contains(contentStr, "test.second") {
		t.Error("Second event was not appended")
	}
}
