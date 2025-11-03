package gpu

import (
	"bytes"
	"encoding/json"
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

	support, detail, err := td.detectDockerRuntime()
	if err != nil {
		report.ErrorMessage = detail
		td.logger.Warn("gpu.toolkit.inspect.failed", "Failed to inspect docker runtime", map[string]interface{}{
			"error": err.Error(),
		})
		return report
	}

	if !support {
		report.ErrorMessage = detail
		td.logger.Info("gpu.toolkit.runtime.absent", "NVIDIA runtime not detected", map[string]interface{}{
			"detail": detail,
		})
		return report
	}

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

func (td *ToolkitDetector) detectDockerRuntime() (bool, string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("docker", "info", "--format", "{{json .Runtimes}}")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err == nil {
		runtimes := make(map[string]json.RawMessage)
		if err := json.Unmarshal(stdout.Bytes(), &runtimes); err == nil {
			if _, ok := runtimes["nvidia"]; ok {
				return true, "", nil
			}
		} else {
			td.logger.Warn("gpu.toolkit.runtime.parse_failed", "Failed to parse docker runtime json", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	stdout.Reset()
	stderr.Reset()
	cmd = exec.Command("docker", "info")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return false, fmt.Sprintf("docker info failed: %v", err), err
	}

	infoOutput := stdout.String()
	if strings.Contains(infoOutput, "Runtimes: nvidia") || strings.Contains(infoOutput, "nvidia-container-runtime") {
		return true, "", nil
	}

	return false, "NVIDIA runtime not listed in docker info", nil
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
