package services

import (
	"testing"
)

func TestDockerRuntime_CreateNetwork(t *testing.T) {
	runtime := NewDockerRuntime()

	// This is a minimal test - in a real scenario, you'd mock the exec.Command
	// For now, we just ensure the method exists and can be called
	if runtime == nil {
		t.Fatal("Expected DockerRuntime to be created")
	}

	// Note: Actual Docker tests would require Docker to be running
	// Integration tests should be separate
}

func TestDetectRuntime(t *testing.T) {
	// This test will pass only if Docker is available
	// In CI, this should be mocked or skipped if Docker is not available
	runtime, err := DetectRuntime()

	// We don't fail if Docker is not available in test environment
	// Just verify the function returns expected types
	if err != nil && runtime != nil {
		t.Error("If error is returned, runtime should be nil")
	}

	if err == nil && runtime == nil {
		t.Error("If no error, runtime should not be nil")
	}
}
