package diag

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"aistack/internal/logging"
)

// Collector gathers diagnostic artifacts
type Collector struct {
	config   *DiagConfig
	redactor *Redactor
	logger   *logging.Logger
}

// NewCollector creates a new diagnostic collector
func NewCollector(config *DiagConfig, logger *logging.Logger) *Collector {
	return &Collector{
		config:   config,
		redactor: NewRedactor(),
		logger:   logger,
	}
}

// CollectLogs gathers all log files from the log directory
func (c *Collector) CollectLogs() (map[string][]byte, error) {
	if !c.config.IncludeLogs {
		return nil, nil
	}

	files := make(map[string][]byte)

	// Check if log directory exists
	if _, err := os.Stat(c.config.LogDir); os.IsNotExist(err) {
		c.logger.Warn("diag.collect.logs.missing", "Log directory not found", map[string]interface{}{
			"path": c.config.LogDir,
		})
		return files, nil
	}

	// Walk log directory
	err := filepath.Walk(c.config.LogDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			c.logger.Warn("diag.collect.logs.walk_error", "Error accessing file", map[string]interface{}{
				"path":  path,
				"error": err.Error(),
			})
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only include .log files
		if filepath.Ext(path) != ".log" {
			return nil
		}

		// Read file
		content, err := os.ReadFile(path)
		if err != nil {
			c.logger.Warn("diag.collect.logs.read_error", "Failed to read log file", map[string]interface{}{
				"path":  path,
				"error": err.Error(),
			})
			return nil // Continue with other files
		}

		// Get relative path for ZIP
		relPath, err := filepath.Rel(c.config.LogDir, path)
		if err != nil {
			relPath = filepath.Base(path)
		}

		files["logs/"+relPath] = content
		return nil
	})

	if err != nil {
		return files, fmt.Errorf("failed to walk log directory: %w", err)
	}

	c.logger.Info("diag.collect.logs.complete", "Log collection complete", map[string]interface{}{
		"file_count": len(files),
	})

	return files, nil
}

// CollectConfig gathers and redacts the configuration file
func (c *Collector) CollectConfig() (map[string][]byte, error) {
	if !c.config.IncludeConfig {
		return nil, nil
	}

	files := make(map[string][]byte)

	// Check if config file exists
	if _, err := os.Stat(c.config.ConfigPath); os.IsNotExist(err) {
		c.logger.Warn("diag.collect.config.missing", "Config file not found", map[string]interface{}{
			"path": c.config.ConfigPath,
		})
		return files, nil
	}

	// Read config file
	content, err := os.ReadFile(c.config.ConfigPath)
	if err != nil {
		c.logger.Error("diag.collect.config.read_error", "Failed to read config file", map[string]interface{}{
			"path":  c.config.ConfigPath,
			"error": err.Error(),
		})
		return files, fmt.Errorf("failed to read config: %w", err)
	}

	// Redact secrets
	redactedContent := c.redactor.RedactFile(string(content))

	files["config/config.yaml"] = []byte(redactedContent)

	c.logger.Info("diag.collect.config.complete", "Config collection complete", map[string]interface{}{
		"redacted": true,
	})

	return files, nil
}

// CollectSystemInfo gathers system and version information
func (c *Collector) CollectSystemInfo() (map[string][]byte, error) {
	files := make(map[string][]byte)

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Create system info
	sysInfo := map[string]interface{}{
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
		"host":            hostname,
		"aistack_version": c.config.Version,
	}

	sysInfoJSON, err := json.MarshalIndent(sysInfo, "", "  ")
	if err != nil {
		return files, fmt.Errorf("failed to marshal system info: %w", err)
	}

	files["system_info.json"] = sysInfoJSON

	c.logger.Info("diag.collect.sysinfo.complete", "System info collection complete", nil)

	return files, nil
}

// CalculateSHA256 computes SHA256 hash of data
func CalculateSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// CalculateFileSHA256 computes SHA256 hash of a file
func CalculateFileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close() // Read-only operation, error can be safely ignored
	}()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
