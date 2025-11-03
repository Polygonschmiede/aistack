package services

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Runtime represents a container runtime (Docker or Podman)
type Runtime interface {
	// ComposeUp starts services defined in a compose file
	ComposeUp(composeFile string, services ...string) error
	// ComposeDown stops and removes services
	ComposeDown(composeFile string) error
	// IsRunning checks if the runtime is available
	IsRunning() bool
	// CreateNetwork creates a network if it doesn't exist
	CreateNetwork(name string) error
	// CreateVolume creates a volume if it doesn't exist
	CreateVolume(name string) error
	// GetContainerStatus returns the status of a container
	GetContainerStatus(name string) (string, error)
	// PullImage pulls a container image
	PullImage(image string) error
	// GetImageID returns the image ID for a given image name
	GetImageID(image string) (string, error)
	// GetContainerLogs returns logs from a container
	GetContainerLogs(name string, tail int) (string, error)
	// RemoveVolume removes a volume
	RemoveVolume(name string) error
}

// DockerRuntime implements Runtime for Docker
type DockerRuntime struct{}

// NewDockerRuntime creates a new Docker runtime
func NewDockerRuntime() *DockerRuntime {
	return &DockerRuntime{}
}

// IsRunning checks if Docker daemon is running
func (r *DockerRuntime) IsRunning() bool {
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}

// ComposeUp starts services using docker compose
func (r *DockerRuntime) ComposeUp(composeFile string, services ...string) error {
	args := []string{"compose", "-f", composeFile, "up", "-d"}
	args = append(args, services...)

	// #nosec G204 — compose arguments originate from curated templates and service names.
	cmd := exec.Command("docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose up failed: %w, stderr: %s", err, stderr.String())
	}
	return nil
}

// ComposeDown stops and removes services
func (r *DockerRuntime) ComposeDown(composeFile string) error {
	// #nosec G204 — compose arguments originate from curated templates.
	cmd := exec.Command("docker", "compose", "-f", composeFile, "down")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose down failed: %w, stderr: %s", err, stderr.String())
	}
	return nil
}

// CreateNetwork creates a Docker network if it doesn't exist (idempotent)
func (r *DockerRuntime) CreateNetwork(name string) error {
	// Check if network exists
	// #nosec G204 — network name is controlled by application logic.
	checkCmd := exec.Command("docker", "network", "inspect", name)
	if checkCmd.Run() == nil {
		// Network already exists
		return nil
	}

	// Create network
	// #nosec G204 — network name is controlled by application logic.
	cmd := exec.Command("docker", "network", "create", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create network %s: %w, stderr: %s", name, err, stderr.String())
	}
	return nil
}

// CreateVolume creates a Docker volume if it doesn't exist (idempotent)
func (r *DockerRuntime) CreateVolume(name string) error {
	// Check if volume exists
	// #nosec G204 — volume name is controlled by application logic.
	checkCmd := exec.Command("docker", "volume", "inspect", name)
	if checkCmd.Run() == nil {
		// Volume already exists
		return nil
	}

	// Create volume
	// #nosec G204 — volume name is controlled by application logic.
	cmd := exec.Command("docker", "volume", "create", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create volume %s: %w, stderr: %s", name, err, stderr.String())
	}
	return nil
}

// GetContainerStatus returns the status of a container
func (r *DockerRuntime) GetContainerStatus(name string) (string, error) {
	// #nosec G204 — container names originate from predefined service IDs.
	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Status}}", name)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get container status: %w, stderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// PullImage pulls a container image
func (r *DockerRuntime) PullImage(image string) error {
	// #nosec G204 — image name is validated before use
	cmd := exec.Command("docker", "pull", image)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull image %s: %w, stderr: %s", image, err, stderr.String())
	}
	return nil
}

// GetImageID returns the image ID for a given image name
func (r *DockerRuntime) GetImageID(image string) (string, error) {
	// #nosec G204 — image name is validated before use
	cmd := exec.Command("docker", "inspect", "-f", "{{.Id}}", image)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get image ID: %w, stderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetContainerLogs returns logs from a container
func (r *DockerRuntime) GetContainerLogs(name string, tail int) (string, error) {
	args := []string{"logs"}
	if tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", tail))
	}
	args = append(args, name)

	// #nosec G204 — container name is validated before use
	cmd := exec.Command("docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get container logs: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// RemoveVolume removes a volume
func (r *DockerRuntime) RemoveVolume(name string) error {
	// #nosec G204 — volume name is validated before use
	cmd := exec.Command("docker", "volume", "rm", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Volume might not exist or might be in use - log but don't fail
		return fmt.Errorf("failed to remove volume %s: %w, stderr: %s", name, err, stderr.String())
	}
	return nil
}

// DetectRuntime detects and returns the available container runtime
func DetectRuntime() (Runtime, error) {
	// Check for Docker first (default)
	docker := NewDockerRuntime()
	if docker.IsRunning() {
		return docker, nil
	}

	// TODO: Add Podman support in future (EP-003 states Podman is best-effort)

	return nil, fmt.Errorf("no container runtime detected (Docker required)")
}
