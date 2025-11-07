package diag

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"aistack/internal/logging"
)

// Packager creates diagnostic ZIP packages
type Packager struct {
	config    *Config
	collector *Collector
	logger    *logging.Logger
}

// NewPackager creates a new diagnostic packager
func NewPackager(config *Config, logger *logging.Logger) *Packager {
	return &Packager{
		config:    config,
		collector: NewCollector(config, logger),
		logger:    logger,
	}
}

// CreatePackage creates a complete diagnostic package
func (p *Packager) CreatePackage() (string, error) {
	p.logger.Info("diag.package.start", "Creating diagnostic package", map[string]interface{}{
		"output": p.config.OutputPath,
	})

	// Collect all artifacts
	allFiles := make(map[string][]byte)

	// Collect logs
	logs, err := p.collector.CollectLogs()
	if err != nil {
		p.logger.Error("diag.package.logs_error", "Failed to collect logs", map[string]interface{}{
			"error": err.Error(),
		})
		// Continue with partial package
	}
	for path, content := range logs {
		allFiles[path] = content
	}

	// Collect config
	config, err := p.collector.CollectConfig()
	if err != nil {
		p.logger.Error("diag.package.config_error", "Failed to collect config", map[string]interface{}{
			"error": err.Error(),
		})
		// Continue with partial package
	}
	for path, content := range config {
		allFiles[path] = content
	}

	// Collect system info
	sysInfo, err := p.collector.CollectSystemInfo()
	if err != nil {
		p.logger.Error("diag.package.sysinfo_error", "Failed to collect system info", map[string]interface{}{
			"error": err.Error(),
		})
		// Continue with partial package
	}
	for path, content := range sysInfo {
		allFiles[path] = content
	}

	// Create manifest
	manifest, err := p.createManifest(allFiles)
	if err != nil {
		return "", fmt.Errorf("failed to create manifest: %w", err)
	}

	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal manifest: %w", err)
	}
	allFiles["diag_manifest.json"] = manifestJSON

	// Create ZIP file
	if err := p.createZIP(allFiles); err != nil {
		return "", fmt.Errorf("failed to create ZIP: %w", err)
	}

	p.logger.Info("diag.package.complete", "Diagnostic package created", map[string]interface{}{
		"output":     p.config.OutputPath,
		"file_count": len(allFiles),
	})

	return p.config.OutputPath, nil
}

// createManifest generates the diagnostic manifest
func (p *Packager) createManifest(files map[string][]byte) (*Manifest, error) {
	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	manifest := &Manifest{
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Host:           hostname,
		AistackVersion: p.config.Version,
		Files:          make([]ManifestFile, 0, len(files)),
	}

	// Add file entries
	for path, content := range files {
		manifestFile := ManifestFile{
			Path:      path,
			SizeBytes: int64(len(content)),
			SHA256:    CalculateSHA256(content),
		}
		manifest.Files = append(manifest.Files, manifestFile)
	}

	return manifest, nil
}

// createZIP creates the ZIP archive
func (p *Packager) createZIP(files map[string][]byte) error {
	// Create output file
	zipFile, err := os.Create(p.config.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if closeErr := zipFile.Close(); closeErr != nil {
			p.logger.Warn("diag.package.zipfile.close_error", "Failed to close ZIP file", map[string]interface{}{
				"error": closeErr.Error(),
			})
		}
	}()

	// Create ZIP writer
	zipWriter := zip.NewWriter(zipFile)
	defer func() {
		if closeErr := zipWriter.Close(); closeErr != nil {
			p.logger.Error("diag.package.zip.close_error", "Failed to close ZIP writer", map[string]interface{}{
				"error": closeErr.Error(),
			})
		}
	}()

	// Add all files to ZIP
	for path, content := range files {
		writer, err := zipWriter.Create(path)
		if err != nil {
			p.logger.Warn("diag.package.zip.file_error", "Failed to add file to ZIP", map[string]interface{}{
				"path":  path,
				"error": err.Error(),
			})
			continue
		}

		if _, err := writer.Write(content); err != nil {
			p.logger.Warn("diag.package.zip.write_error", "Failed to write file to ZIP", map[string]interface{}{
				"path":  path,
				"error": err.Error(),
			})
			continue
		}
	}

	return nil
}
