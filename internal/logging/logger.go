package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Level represents log severity
type Level string

const (
	// LevelDebug indicates fine-grained diagnostic logging.
	LevelDebug Level = "debug"
	// LevelInfo indicates informational logging.
	LevelInfo Level = "info"
	// LevelWarn indicates non-fatal warnings.
	LevelWarn Level = "warn"
	// LevelError indicates error logging requiring attention.
	LevelError Level = "error"
)

// Event represents a structured log event
type Event struct {
	Timestamp string                 `json:"ts"`
	Level     Level                  `json:"level"`
	Type      string                 `json:"type"`
	Message   string                 `json:"message"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}

// Logger provides structured logging
// Story T-027: Extended with file-based output and rotation support
type Logger struct {
	minLevel Level
	output   io.Writer
	logFile  *os.File
}

// NewLogger creates a new logger writing to stderr
func NewLogger(minLevel Level) *Logger {
	return &Logger{
		minLevel: minLevel,
		output:   os.Stderr,
	}
}

// NewFileLogger creates a new logger writing to a file
// Story T-027: File-based logging with automatic directory creation
func NewFileLogger(minLevel Level, logFilePath string) (*Logger, error) {
	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file (append mode, create if not exists)
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &Logger{
		minLevel: minLevel,
		output:   logFile,
		logFile:  logFile,
	}, nil
}

// Close closes the log file if open
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// Log writes a structured log event
func (l *Logger) Log(level Level, eventType, message string, payload map[string]interface{}) {
	if !l.shouldLog(level) {
		return
	}

	event := Event{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Type:      eventType,
		Message:   message,
		Payload:   payload,
	}

	data, err := json.Marshal(event)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal log event: %v\n", err)
		return
	}

	output := l.output
	if output == nil {
		output = os.Stderr
	}

	if _, err := fmt.Fprintln(output, string(data)); err != nil {
		// Best-effort logging: fallback to stderr when the primary writer fails
		if output != os.Stderr {
			fmt.Fprintf(os.Stderr, "Failed to write log event: %v\n", err)
		}
	}
}

// Debug logs a debug-level event
func (l *Logger) Debug(eventType, message string, payload map[string]interface{}) {
	l.Log(LevelDebug, eventType, message, payload)
}

// Info logs an info-level event
func (l *Logger) Info(eventType, message string, payload map[string]interface{}) {
	l.Log(LevelInfo, eventType, message, payload)
}

// Warn logs a warn-level event
func (l *Logger) Warn(eventType, message string, payload map[string]interface{}) {
	l.Log(LevelWarn, eventType, message, payload)
}

// Error logs an error-level event
func (l *Logger) Error(eventType, message string, payload map[string]interface{}) {
	l.Log(LevelError, eventType, message, payload)
}

// shouldLog determines if a log level should be output
func (l *Logger) shouldLog(level Level) bool {
	levels := map[Level]int{
		LevelDebug: 0,
		LevelInfo:  1,
		LevelWarn:  2,
		LevelError: 3,
	}
	return levels[level] >= levels[l.minLevel]
}
