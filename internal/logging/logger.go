package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Level represents log severity
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
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
type Logger struct {
	minLevel Level
}

// NewLogger creates a new logger
func NewLogger(minLevel Level) *Logger {
	return &Logger{minLevel: minLevel}
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

	fmt.Fprintln(os.Stderr, string(data))
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
