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
