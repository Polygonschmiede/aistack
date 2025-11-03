package gpu

import (
	"encoding/json"
	"fmt"
	"os"

	"aistack/internal/logging"
)

func saveReportToFile(logger *logging.Logger, report GPUReport, filepath string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	if logger != nil {
		logger.Info("gpu.report.saved", "GPU report saved", map[string]interface{}{
			"filepath": filepath,
		})
	}

	return nil
}
