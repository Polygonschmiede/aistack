package metrics

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"aistack/internal/logging"
)

func TestWriter_Write(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	writer := NewWriter(logger)

	tmpFile := "/tmp/test_metrics.jsonl"
	defer os.Remove(tmpFile)

	// Create a sample
	cpuUtil := 50.0
	gpuUtil := 75.0
	sample := MetricsSample{
		Timestamp: time.Now().UTC(),
		CPUUtil:   &cpuUtil,
		GPUUtil:   &gpuUtil,
	}

	// Write sample
	err := writer.Write(sample, tmpFile)
	if err != nil {
		t.Fatalf("Expected successful write, got error: %v", err)
	}

	// Read and verify
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "cpu_util") {
		t.Error("Expected file to contain cpu_util field")
	}

	if !strings.Contains(content, "gpu_util") {
		t.Error("Expected file to contain gpu_util field")
	}

	// Verify it's valid JSON
	var readSample MetricsSample
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) != 1 {
		t.Errorf("Expected 1 line, got %d", len(lines))
	}

	err = json.Unmarshal([]byte(lines[0]), &readSample)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
}

func TestWriter_Write_Multiple(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	writer := NewWriter(logger)

	tmpFile := "/tmp/test_metrics_multi.jsonl"
	defer os.Remove(tmpFile)

	// Write multiple samples
	for i := 0; i < 3; i++ {
		util := float64(i * 10)
		sample := MetricsSample{
			Timestamp: time.Now().UTC(),
			CPUUtil:   &util,
		}

		err := writer.Write(sample, tmpFile)
		if err != nil {
			t.Fatalf("Failed to write sample %d: %v", i, err)
		}
	}

	// Verify we have 3 lines
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}
}
