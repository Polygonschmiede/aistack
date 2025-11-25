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

// GenericRuntime implements Runtime for Docker or Podman
type GenericRuntime struct {
	binary string // "docker" or "podman"
}

// NewGenericRuntime creates a new generic runtime with the specified binary
func NewGenericRuntime(binary string) *GenericRuntime {
	return &GenericRuntime{binary: binary}
}

// IsRunning checks if the runtime daemon is running
func (r *GenericRuntime) IsRunning() bool {
	cmd := exec.Command(r.binary, "info")
	return cmd.Run() == nil
}

// ComposeUp starts services using compose
func (r *GenericRuntime) ComposeUp(composeFile string, services ...string) error {
	args := []string{"compose", "-f", composeFile, "up", "-d"}
	args = append(args, services...)

	// #nosec G204 — compose arguments originate from curated templates and service names.
	cmd := exec.Command(r.binary, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s compose up failed: %w, stderr: %s", r.binary, err, stderr.String())
	}
	return nil
}

// ComposeDown stops and removes services
func (r *GenericRuntime) ComposeDown(composeFile string) error {
	// #nosec G204 — compose arguments originate from curated templates.
	cmd := exec.Command(r.binary, "compose", "-f", composeFile, "down")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s compose down failed: %w, stderr: %s", r.binary, err, stderr.String())
	}
	return nil
}

// CreateNetwork creates a network if it doesn't exist (idempotent)
func (r *GenericRuntime) CreateNetwork(name string) error {
	// Check if network exists
	// #nosec G204 — network name is controlled by application logic.
	checkCmd := exec.Command(r.binary, "network", "inspect", name)
	if checkCmd.Run() == nil {
		// Network already exists
		return nil
	}

	// Create network
	// #nosec G204 — network name is controlled by application logic.
	cmd := exec.Command(r.binary, "network", "create", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create %s network %s: %w, stderr: %s", r.binary, name, err, stderr.String())
	}
	return nil
}

// CreateVolume creates a volume if it doesn't exist (idempotent)
func (r *GenericRuntime) CreateVolume(name string) error {
	// Check if volume exists
	// #nosec G204 — volume name is controlled by application logic.
	checkCmd := exec.Command(r.binary, "volume", "inspect", name)
	if checkCmd.Run() == nil {
		// Volume already exists
		return nil
	}

	// Create volume
	// #nosec G204 — volume name is controlled by application logic.
	cmd := exec.Command(r.binary, "volume", "create", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create %s volume %s: %w, stderr: %s", r.binary, name, err, stderr.String())
	}
	return nil
}

// GetContainerStatus returns the status of a container
func (r *GenericRuntime) GetContainerStatus(name string) (string, error) {
	// #nosec G204 — container names originate from predefined service IDs.
	cmd := exec.Command(r.binary, "inspect", "-f", "{{.State.Status}}", name)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get %s container status: %w, stderr: %s", r.binary, err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// PullImage pulls a container image
func (r *GenericRuntime) PullImage(image string) error {
	// #nosec G204 — image name is validated before use
	cmd := exec.Command(r.binary, "pull", image)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull %s image %s: %w, stderr: %s", r.binary, image, err, stderr.String())
	}
	return nil
}

// GetImageID returns the image ID for a given image name
func (r *GenericRuntime) GetImageID(image string) (string, error) {
	// For Docker: use "inspect -f {{.Id}}"
	// For Podman: use "image inspect -f {{.Id}}"
	args := []string{}
	if r.binary == "podman" {
		args = append(args, "image")
	}
	args = append(args, "inspect", "-f", "{{.Id}}", image)

	// #nosec G204 — image name is validated before use
	cmd := exec.Command(r.binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get %s image ID: %w, stderr: %s", r.binary, err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetContainerLogs returns logs from a container
func (r *GenericRuntime) GetContainerLogs(name string, tail int) (string, error) {
	return fetchContainerLogs(r.binary, r.binary, name, tail)
}

// RemoveVolume removes a volume
func (r *GenericRuntime) RemoveVolume(name string) error {
	// #nosec G204 — volume name is validated before use
	cmd := exec.Command(r.binary, "volume", "rm", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Volume might not exist or might be in use - log but don't fail
		return fmt.Errorf("failed to remove %s volume %s: %w, stderr: %s", r.binary, name, err, stderr.String())
	}
	return nil
}

// RemoveContainer removes a container
func (r *GenericRuntime) RemoveContainer(name string) error {
	// #nosec G204 — container name is validated before use
	cmd := exec.Command(r.binary, "rm", "-f", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Container might not exist - return error for caller to handle
		return fmt.Errorf("failed to remove %s container %s: %w, stderr: %s", r.binary, name, err, stderr.String())
	}
	return nil
}

// TagImage retags an image reference
func (r *GenericRuntime) TagImage(source, target string) error {
	// #nosec G204 — image references are validated before use.
	cmd := exec.Command(r.binary, "tag", source, target)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to tag %s image %s as %s: %w, stderr: %s", r.binary, source, target, err, stderr.String())
	}
	return nil
}

// VolumeExists checks if a volume exists
func (r *GenericRuntime) VolumeExists(name string) (bool, error) {
	// #nosec G204 — volume names are validated before use
	cmd := exec.Command(r.binary, "volume", "inspect", name)
	err := cmd.Run()
	if err != nil {
		// Volume doesn't exist
		return false, nil
	}
	return true, nil
}

// RemoveNetwork removes a network
func (r *GenericRuntime) RemoveNetwork(name string) error {
	// #nosec G204 — network names are validated before use
	cmd := exec.Command(r.binary, "network", "rm", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Ignore error if network doesn't exist
		if strings.Contains(stderr.String(), "not found") || strings.Contains(stderr.String(), "No such network") {
			return nil
		}
		return fmt.Errorf("failed to remove %s network %s: %w, stderr: %s", r.binary, name, err, stderr.String())
	}
	return nil
}

// IsContainerRunning checks if a container is running
func (r *GenericRuntime) IsContainerRunning(name string) (bool, error) {
	status, err := r.GetContainerStatus(name)
	if err != nil {
		return false, nil
	}
	return status == containerStatusRunning, nil
}

// DockerRuntime implements Runtime for Docker
type DockerRuntime struct {
	*GenericRuntime
}

// NewDockerRuntime creates a new Docker runtime
func NewDockerRuntime() *DockerRuntime {
	return &DockerRuntime{
		GenericRuntime: NewGenericRuntime("docker"),
	}
}

// PodmanRuntime implements Runtime for Podman (best-effort support)
type PodmanRuntime struct {
	*GenericRuntime
}

// NewPodmanRuntime creates a new Podman runtime
func NewPodmanRuntime() *PodmanRuntime {
	return &PodmanRuntime{
		GenericRuntime: NewGenericRuntime("podman"),
	}
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
