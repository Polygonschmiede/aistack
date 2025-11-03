package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"aistack/internal/logging"
)

// Writer handles writing metrics samples to JSONL format
type Writer struct {
	logger *logging.Logger
}

// NewWriter creates a new metrics writer
func NewWriter(logger *logging.Logger) *Writer {
	return &Writer{
		logger: logger,
	}
}

// Write writes a metrics sample to a JSONL file
// Story T-011: JSONL-Log
func (w *Writer) Write(sample MetricsSample, logPath string) error {
	// Marshal sample to JSON
	data, err := json.Marshal(sample)
	if err != nil {
		return fmt.Errorf("failed to marshal sample: %w", err)
	}

	// Append newline for JSONL format
	data = append(data, '\n')

	cleanPath := filepath.Clean(logPath)

	// Open file in append mode (create if not exists)
	// #nosec G304 — log path is controlled by configuration and cleaned above.
	file, err := os.OpenFile(cleanPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open metrics log: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			w.logger.Warn("metrics.writer.close_failed", "Failed to close metrics log", map[string]interface{}{
				"error": cerr.Error(),
				"path":  cleanPath,
			})
		}
	}()

	// Write sample
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write sample: %w", err)
	}

	return nil
}
