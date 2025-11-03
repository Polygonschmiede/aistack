package gpu

import (
	"aistack/internal/logging"
	"testing"
)

func TestToolkitDetector_Creation(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	detector := NewToolkitDetector(logger)

	if detector == nil {
		t.Fatal("Expected detector to be created")
	}

	if detector.logger == nil {
		t.Error("Expected logger to be set")
	}
}

func TestToolkitDetector_QuickGPUCheck(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	detector := NewToolkitDetector(logger)

	// This will fail on systems without nvidia-smi, which is expected
	// We just verify the method doesn't panic
	_ = detector.QuickGPUCheck()
}

// Note: DetectContainerToolkit requires actual Docker to test properly
// In a real test environment, this would be an integration test
// For unit testing, we would need to refactor to inject command execution
func TestToolkitDetector_DetectContainerToolkit_Structure(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	detector := NewToolkitDetector(logger)

	// Call the method to ensure it doesn't panic
	// On systems without Docker/GPU, this will return a failed report
	report := detector.DetectContainerToolkit()

	// Verify report structure
	if report.DockerSupport && report.ErrorMessage != "" {
		t.Error("If DockerSupport is true, ErrorMessage should be empty")
	}

	if !report.DockerSupport && report.ToolkitVersion != "" {
		t.Error("If DockerSupport is false, ToolkitVersion should be empty")
	}
}

func TestContainerToolkitReport_Structure(t *testing.T) {
	report := ContainerToolkitReport{
		DockerSupport:  true,
		ToolkitVersion: "1.14.5",
	}

	if !report.DockerSupport {
		t.Error("Expected DockerSupport to be true")
	}

	if report.ToolkitVersion != "1.14.5" {
		t.Errorf("Expected version 1.14.5, got: %s", report.ToolkitVersion)
	}
}

func TestGPUInfo_Structure(t *testing.T) {
	info := GPUInfo{
		Name:     "NVIDIA GeForce RTX 4090",
		UUID:     "GPU-12345678-1234-1234-1234-123456789012",
		MemoryMB: 24576,
		Index:    0,
	}

	if info.Name != "NVIDIA GeForce RTX 4090" {
		t.Errorf("Expected name to match, got: %s", info.Name)
	}

	if info.MemoryMB != 24576 {
		t.Errorf("Expected memory 24576, got: %d", info.MemoryMB)
	}
}
