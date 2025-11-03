package gpu

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"aistack/internal/logging"
)

// ToolkitDetector handles NVIDIA Container Toolkit detection
// Story T-010: NVIDIA Container Toolkit Detection
type ToolkitDetector struct {
	logger *logging.Logger
}

// NewToolkitDetector creates a new toolkit detector
func NewToolkitDetector(logger *logging.Logger) *ToolkitDetector {
	return &ToolkitDetector{
		logger: logger,
	}
}

// DetectContainerToolkit checks if NVIDIA Container Toolkit is available
// Story T-010: Test-Container mit --gpus all dry-run
func (td *ToolkitDetector) DetectContainerToolkit() ContainerToolkitReport {
	td.logger.Info("gpu.toolkit.detect.start", "Starting Container Toolkit detection", nil)

	report := ContainerToolkitReport{
		DockerSupport: false,
	}

	// First, check if docker is available
	if !td.isDockerAvailable() {
		report.ErrorMessage = "Docker is not available"
		td.logger.Warn("gpu.toolkit.docker.unavailable", "Docker not found", nil)
		return report
	}

	// Try to run a test container with --gpus flag
	// This is a dry-run test (using --rm and immediate exit)
	cmd := exec.Command("docker", "run", "--rm", "--gpus", "all", "nvidia/cuda:12.0.0-base-ubuntu22.04", "nvidia-smi", "--version")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		report.ErrorMessage = fmt.Sprintf("GPU support test failed: %v, stderr: %s", err, stderr.String())
		td.logger.Warn("gpu.toolkit.test.failed", "GPU support test failed", map[string]interface{}{
			"error":  err.Error(),
			"stderr": stderr.String(),
		})
		return report
	}

	// If we got here, --gpus is supported
	report.DockerSupport = true

	// Try to extract toolkit version from nvidia-container-toolkit
	version := td.getToolkitVersion()
	if version != "" {
		report.ToolkitVersion = version
	}

	td.logger.Info("gpu.toolkit.detected", "Container Toolkit detected successfully", map[string]interface{}{
		"version": report.ToolkitVersion,
	})

	return report
}

// isDockerAvailable checks if Docker daemon is running
func (td *ToolkitDetector) isDockerAvailable() bool {
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}

// getToolkitVersion attempts to get the NVIDIA Container Toolkit version
func (td *ToolkitDetector) getToolkitVersion() string {
	cmd := exec.Command("nvidia-container-toolkit", "--version")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		// Toolkit CLI might not be in PATH, this is optional
		return ""
	}

	// Parse version from output
	output := stdout.String()
	// Expected format: "NVIDIA Container Toolkit version X.Y.Z"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "version") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				return parts[len(parts)-1]
			}
		}
	}

	return ""
}

// QuickGPUCheck performs a quick GPU availability check without full detection
// This is useful for fast pre-flight checks
func (td *ToolkitDetector) QuickGPUCheck() bool {
	cmd := exec.Command("nvidia-smi")
	return cmd.Run() == nil
}
