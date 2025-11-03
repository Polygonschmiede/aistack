//go:build cuda

package gpu

import (
	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

// MockNVML is a mock implementation of NVMLInterface for testing
type MockNVML struct {
	InitReturn                   nvml.Return
	ShutdownReturn               nvml.Return
	DeviceCount                  int
	DeviceCountReturn            nvml.Return
	DriverVersion                string
	DriverVersionReturn          nvml.Return
	CudaVersion                  int
	CudaVersionReturn            nvml.Return
	Devices                      []MockDevice
	DeviceGetHandleByIndexReturn nvml.Return
}

// MockDevice represents a mock GPU device
type MockDevice struct {
	Name              string
	NameReturn        nvml.Return
	UUID              string
	UUIDReturn        nvml.Return
	MemoryTotal       uint64
	MemoryUsed        uint64
	MemoryInfoReturn  nvml.Return
	GPUUtil           uint32
	MemUtil           uint32
	UtilizationReturn nvml.Return
	PowerUsage        uint32
	PowerUsageReturn  nvml.Return
	Temperature       uint32
	TemperatureReturn nvml.Return
}

// NewMockNVML creates a new mock NVML instance
func NewMockNVML() *MockNVML {
	return &MockNVML{
		InitReturn:                   nvml.SUCCESS,
		ShutdownReturn:               nvml.SUCCESS,
		DeviceCountReturn:            nvml.SUCCESS,
		DriverVersionReturn:          nvml.SUCCESS,
		CudaVersionReturn:            nvml.SUCCESS,
		DeviceGetHandleByIndexReturn: nvml.SUCCESS,
		Devices:                      make([]MockDevice, 0),
	}
}

// Init mocks NVML initialization
func (m *MockNVML) Init() nvml.Return {
	return m.InitReturn
}

// Shutdown mocks NVML shutdown
func (m *MockNVML) Shutdown() nvml.Return {
	return m.ShutdownReturn
}

// DeviceGetCount mocks getting device count
func (m *MockNVML) DeviceGetCount() (int, nvml.Return) {
	return m.DeviceCount, m.DeviceCountReturn
}

// DeviceGetHandleByIndex mocks getting device handle
func (m *MockNVML) DeviceGetHandleByIndex(index int) (DeviceInterface, nvml.Return) {
	if index < 0 || index >= len(m.Devices) {
		return nil, nvml.ERROR_INVALID_ARGUMENT
	}
	return mockDeviceImpl{device: &m.Devices[index]}, m.DeviceGetHandleByIndexReturn
}

// SystemGetDriverVersion mocks getting driver version
func (m *MockNVML) SystemGetDriverVersion() (string, nvml.Return) {
	return m.DriverVersion, m.DriverVersionReturn
}

// SystemGetCudaDriverVersion mocks getting CUDA version
func (m *MockNVML) SystemGetCudaDriverVersion() (int, nvml.Return) {
	return m.CudaVersion, m.CudaVersionReturn
}

// mockDeviceImpl implements nvml.Device interface for testing
type mockDeviceImpl struct {
	device *MockDevice
}

// GetName returns the mock device name
func (m mockDeviceImpl) GetName() (string, nvml.Return) {
	return m.device.Name, m.device.NameReturn
}

// GetUUID returns the mock device UUID
func (m mockDeviceImpl) GetUUID() (string, nvml.Return) {
	return m.device.UUID, m.device.UUIDReturn
}

// GetMemoryInfo returns the mock memory info
func (m mockDeviceImpl) GetMemoryInfo() (nvml.Memory, nvml.Return) {
	return nvml.Memory{
		Total: m.device.MemoryTotal,
		Used:  m.device.MemoryUsed,
	}, m.device.MemoryInfoReturn
}

// GetUtilizationRates returns the mock utilization rates
func (m mockDeviceImpl) GetUtilizationRates() (nvml.Utilization, nvml.Return) {
	return nvml.Utilization{
		Gpu:    m.device.GPUUtil,
		Memory: m.device.MemUtil,
	}, m.device.UtilizationReturn
}

// GetPowerUsage returns the mock power usage
func (m mockDeviceImpl) GetPowerUsage() (uint32, nvml.Return) {
	return m.device.PowerUsage, m.device.PowerUsageReturn
}

// GetTemperature returns the mock temperature
func (m mockDeviceImpl) GetTemperature(sensor nvml.TemperatureSensors) (uint32, nvml.Return) {
	return m.device.Temperature, m.device.TemperatureReturn
}
