package logging

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"testing"
)

type logEvent struct {
	Level   Level                  `json:"level"`
	Type    string                 `json:"type"`
	Message string                 `json:"message"`
	Payload map[string]interface{} `json:"payload"`
}

func captureStderr(t *testing.T, fn func()) []logEvent {
	t.Helper()
	original := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	os.Stderr = original

	reader := bufio.NewReader(r)
	var events []logEvent
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			var event logEvent
			if jerr := json.Unmarshal(line, &event); jerr != nil {
				t.Fatalf("failed to unmarshal log line %q: %v", line, jerr)
			}
			events = append(events, event)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("error reading stderr: %v", err)
		}
	}
	return events
}

func TestLoggerInfoProducesEvent(t *testing.T) {
	logger := NewLogger(LevelInfo)
	events := captureStderr(t, func() {
		logger.Info("app.started", "Application started", map[string]interface{}{"version": "test"})
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Level != LevelInfo {
		t.Fatalf("expected level %q, got %q", LevelInfo, events[0].Level)
	}

	if events[0].Type != "app.started" {
		t.Fatalf("unexpected event type: %s", events[0].Type)
	}

	if events[0].Message != "Application started" {
		t.Fatalf("unexpected message: %s", events[0].Message)
	}

	if got := events[0].Payload["version"]; got != "test" {
		t.Fatalf("unexpected payload value: %v", got)
	}
}

func TestLoggerRespectsMinimumLevel(t *testing.T) {
	logger := NewLogger(LevelError)
	events := captureStderr(t, func() {
		logger.Info("ignored", "should not log", nil)
		logger.Error("app.error", "boom", nil)
	})

	if len(events) != 1 {
		t.Fatalf("expected only error event, got %d", len(events))
	}

	if events[0].Level != LevelError {
		t.Fatalf("expected level %q, got %q", LevelError, events[0].Level)
	}
}
