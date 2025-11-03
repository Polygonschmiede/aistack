package services

import (
	"aistack/internal/logging"
	"testing"
)

// MockRuntime is a mock implementation of Runtime for testing
type MockRuntime struct {
	networks          map[string]bool
	volumes           map[string]bool
	RemovedVolumes    []string // Track removed volumes for testing
	RemovedContainers []string // Track removed containers for testing
	isRunning         bool
	imageID           string
	newImageID        string
	ImageID           string                   // Exposed for test setup
	containerStatuses map[string]ServiceStatus // For dynamic container status
	startError        error                    // Simulate start failures
}

func NewMockRuntime() *MockRuntime {
	return &MockRuntime{
		networks:          make(map[string]bool),
		volumes:           make(map[string]bool),
		RemovedVolumes:    make([]string, 0),
		RemovedContainers: make([]string, 0),
		isRunning:         true,
		imageID:           "sha256:mock123",
		newImageID:        "sha256:mock456",
		ImageID:           "sha256:mock123",
		containerStatuses: make(map[string]ServiceStatus),
	}
}

func (m *MockRuntime) IsRunning() bool {
	return m.isRunning
}

func (m *MockRuntime) CreateNetwork(name string) error {
	m.networks[name] = true
	return nil
}

func (m *MockRuntime) CreateVolume(name string) error {
	m.volumes[name] = true
	return nil
}

func (m *MockRuntime) ComposeUp(composeFile string, services ...string) error {
	if m.startError != nil {
		return m.startError
	}
	return nil
}

func (m *MockRuntime) ComposeDown(composeFile string) error {
	return nil
}

func (m *MockRuntime) GetContainerStatus(name string) (string, error) {
	// Check if we have a custom status for this container
	if status, ok := m.containerStatuses[name]; ok {
		return status.State, nil
	}
	// Default to running for backward compatibility
	return "running", nil
}

func (m *MockRuntime) PullImage(image string) error {
	// Simulate image pull by updating imageID to newImageID
	m.imageID = m.newImageID
	return nil
}

func (m *MockRuntime) GetImageID(image string) (string, error) {
	return m.imageID, nil
}

func (m *MockRuntime) GetContainerLogs(name string, tail int) (string, error) {
	return "mock log output\nline 2\nline 3", nil
}

func (m *MockRuntime) RemoveVolume(name string) error {
	m.RemovedVolumes = append(m.RemovedVolumes, name)
	delete(m.volumes, name)
	return nil
}

func (m *MockRuntime) RemoveContainer(name string) error {
	m.RemovedContainers = append(m.RemovedContainers, name)
	return nil
}

func (m *MockRuntime) TagImage(source, target string) error {
	m.imageID = source
	return nil
}

func TestNetworkManager_EnsureNetwork(t *testing.T) {
	runtime := NewMockRuntime()
	logger := logging.NewLogger(logging.LevelInfo)
	nm := NewNetworkManager(runtime, logger)

	err := nm.EnsureNetwork()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !runtime.networks[AistackNetwork] {
		t.Error("Expected aistack-net to be created")
	}

	// Test idempotency - calling again should not fail
	err = nm.EnsureNetwork()
	if err != nil {
		t.Fatalf("Expected idempotent call to succeed, got: %v", err)
	}
}

func TestNetworkManager_EnsureVolumes(t *testing.T) {
	runtime := NewMockRuntime()
	logger := logging.NewLogger(logging.LevelInfo)
	nm := NewNetworkManager(runtime, logger)

	volumes := []string{"vol1", "vol2", "vol3"}
	err := nm.EnsureVolumes(volumes)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	for _, vol := range volumes {
		if !runtime.volumes[vol] {
			t.Errorf("Expected volume %s to be created", vol)
		}
	}
}
