package services

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	// Container status constants
	containerStatusRunning = "running"
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
	// RemoveContainer removes a container
	RemoveContainer(name string) error
	// TagImage retags an image reference (digest or ID) to a target reference
	TagImage(source string, target string) error
	// VolumeExists checks if a volume exists
	VolumeExists(name string) (bool, error)
	// RemoveNetwork removes a network
	RemoveNetwork(name string) error
	// IsContainerRunning checks if a container is running
	IsContainerRunning(name string) (bool, error)
}

func fetchContainerLogs(binary, label, name string, tail int) (string, error) {
	args := []string{"logs"}
	if tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", tail))
	}
	args = append(args, name)

	// #nosec G204 — container identifiers are validated before invocation
	cmd := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get %s logs: %w, stderr: %s", label, err, stderr.String())
	}

	return stdout.String(), nil
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
	return fetchContainerLogs("docker", "container", name, tail)
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

// RemoveContainer removes a container
func (r *DockerRuntime) RemoveContainer(name string) error {
	// #nosec G204 — container name is validated before use
	cmd := exec.Command("docker", "rm", "-f", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Container might not exist - return error for caller to handle
		return fmt.Errorf("failed to remove container %s: %w, stderr: %s", name, err, stderr.String())
	}
	return nil
}

// TagImage retags a Docker image reference
func (r *DockerRuntime) TagImage(source, target string) error {
	// #nosec G204 — image references are validated before use.
	cmd := exec.Command("docker", "tag", source, target)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to tag image %s as %s: %w, stderr: %s", source, target, err, stderr.String())
	}
	return nil
}

// VolumeExists checks if a Docker volume exists
func (r *DockerRuntime) VolumeExists(name string) (bool, error) {
	// #nosec G204 — volume names are validated before use
	cmd := exec.Command("docker", "volume", "inspect", name)
	err := cmd.Run()
	if err != nil {
		// Volume doesn't exist
		return false, nil
	}
	return true, nil
}

// RemoveNetwork removes a Docker network
func (r *DockerRuntime) RemoveNetwork(name string) error {
	// #nosec G204 — network names are validated before use
	cmd := exec.Command("docker", "network", "rm", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Ignore error if network doesn't exist
		if strings.Contains(stderr.String(), "not found") || strings.Contains(stderr.String(), "No such network") {
			return nil
		}
		return fmt.Errorf("failed to remove network %s: %w, stderr: %s", name, err, stderr.String())
	}
	return nil
}

// IsContainerRunning checks if a Docker container is running
func (r *DockerRuntime) IsContainerRunning(name string) (bool, error) {
	status, err := r.GetContainerStatus(name)
	if err != nil {
		return false, nil
	}
	return status == containerStatusRunning, nil
}

// PodmanRuntime implements Runtime for Podman (best-effort support)
type PodmanRuntime struct{}

// NewPodmanRuntime creates a new Podman runtime
func NewPodmanRuntime() *PodmanRuntime {
	return &PodmanRuntime{}
}

// IsRunning checks if Podman is available and responsive
func (r *PodmanRuntime) IsRunning() bool {
	cmd := exec.Command("podman", "info")
	return cmd.Run() == nil
}

// ComposeUp starts services using podman compose
func (r *PodmanRuntime) ComposeUp(composeFile string, services ...string) error {
	args := []string{"compose", "-f", composeFile, "up", "-d"}
	args = append(args, services...)

	// #nosec G204 — compose arguments originate from curated templates and service names.
	cmd := exec.Command("podman", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("podman compose up failed: %w, stderr: %s", err, stderr.String())
	}
	return nil
}

// ComposeDown stops and removes services using podman compose
func (r *PodmanRuntime) ComposeDown(composeFile string) error {
	// #nosec G204 — compose arguments originate from curated templates.
	cmd := exec.Command("podman", "compose", "-f", composeFile, "down")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("podman compose down failed: %w, stderr: %s", err, stderr.String())
	}
	return nil
}

// CreateNetwork ensures a Podman network exists (idempotent)
func (r *PodmanRuntime) CreateNetwork(name string) error {
	checkCmd := exec.Command("podman", "network", "inspect", name)
	if checkCmd.Run() == nil {
		return nil
	}

	cmd := exec.Command("podman", "network", "create", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create podman network %s: %w, stderr: %s", name, err, stderr.String())
	}
	return nil
}

// CreateVolume ensures a Podman volume exists (idempotent)
func (r *PodmanRuntime) CreateVolume(name string) error {
	checkCmd := exec.Command("podman", "volume", "inspect", name)
	if checkCmd.Run() == nil {
		return nil
	}

	cmd := exec.Command("podman", "volume", "create", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create podman volume %s: %w, stderr: %s", name, err, stderr.String())
	}
	return nil
}

// GetContainerStatus returns the container status for Podman
func (r *PodmanRuntime) GetContainerStatus(name string) (string, error) {
	cmd := exec.Command("podman", "inspect", "-f", "{{.State.Status}}", name)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get podman container status: %w, stderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// PullImage pulls an image using Podman
func (r *PodmanRuntime) PullImage(image string) error {
	cmd := exec.Command("podman", "pull", image)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull podman image %s: %w, stderr: %s", image, err, stderr.String())
	}
	return nil
}

// GetImageID returns the image ID for a Podman image reference
func (r *PodmanRuntime) GetImageID(image string) (string, error) {
	cmd := exec.Command("podman", "image", "inspect", "-f", "{{.Id}}", image)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to inspect podman image %s: %w, stderr: %s", image, err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetContainerLogs returns logs from a Podman container
func (r *PodmanRuntime) GetContainerLogs(name string, tail int) (string, error) {
	return fetchContainerLogs("podman", "podman", name, tail)
}

// RemoveVolume removes a Podman volume
func (r *PodmanRuntime) RemoveVolume(name string) error {
	cmd := exec.Command("podman", "volume", "rm", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove podman volume %s: %w, stderr: %s", name, err, stderr.String())
	}

	return nil
}

// RemoveContainer removes a Podman container
func (r *PodmanRuntime) RemoveContainer(name string) error {
	cmd := exec.Command("podman", "rm", "-f", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove podman container %s: %w, stderr: %s", name, err, stderr.String())
	}

	return nil
}

// TagImage retags a Podman image reference
func (r *PodmanRuntime) TagImage(source, target string) error {
	cmd := exec.Command("podman", "tag", source, target)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to tag podman image %s as %s: %w, stderr: %s", source, target, err, stderr.String())
	}

	return nil
}

// VolumeExists checks if a Podman volume exists
func (r *PodmanRuntime) VolumeExists(name string) (bool, error) {
	cmd := exec.Command("podman", "volume", "inspect", name)
	err := cmd.Run()
	if err != nil {
		return false, nil
	}
	return true, nil
}

// RemoveNetwork removes a Podman network
func (r *PodmanRuntime) RemoveNetwork(name string) error {
	cmd := exec.Command("podman", "network", "rm", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), "not found") || strings.Contains(stderr.String(), "No such network") {
			return nil
		}
		return fmt.Errorf("failed to remove podman network %s: %w, stderr: %s", name, err, stderr.String())
	}
	return nil
}

// IsContainerRunning checks if a Podman container is running
func (r *PodmanRuntime) IsContainerRunning(name string) (bool, error) {
	status, err := r.GetContainerStatus(name)
	if err != nil {
		return false, nil
	}
	return status == containerStatusRunning, nil
}

// DetectRuntime detects and returns the available container runtime
func DetectRuntime() (Runtime, error) {
	desired := strings.ToLower(strings.TrimSpace(os.Getenv("AISTACK_RUNTIME")))

	docker := NewDockerRuntime()
	podman := NewPodmanRuntime()

	switch desired {
	case "docker":
		if docker.IsRunning() {
			return docker, nil
		}
		return nil, fmt.Errorf("docker requested via AISTACK_RUNTIME but not available")
	case "podman":
		if podman.IsRunning() {
			return podman, nil
		}
		return nil, fmt.Errorf("podman requested via AISTACK_RUNTIME but not available")
	case "", "auto":
		if docker.IsRunning() {
			return docker, nil
		}
		if podman.IsRunning() {
			return podman, nil
		}
	default:
		return nil, fmt.Errorf("unknown container runtime '%s' (expected docker|podman|auto)", desired)
	}

	return nil, fmt.Errorf("no container runtime detected (Docker or Podman required)")
}
