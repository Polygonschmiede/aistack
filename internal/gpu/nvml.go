//go:build cuda

package gpu

import (
	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

// DeviceInterface defines the interface for GPU device operations (for mocking)
type DeviceInterface interface {
	GetName() (string, nvml.Return)
	GetUUID() (string, nvml.Return)
	GetMemoryInfo() (nvml.Memory, nvml.Return)
	GetUtilizationRates() (nvml.Utilization, nvml.Return)
	GetPowerUsage() (uint32, nvml.Return)
	GetTemperature(sensor nvml.TemperatureSensors) (uint32, nvml.Return)
}

// NVMLInterface defines the interface for NVML operations (for mocking)
type NVMLInterface interface {
	Init() nvml.Return
	Shutdown() nvml.Return
	DeviceGetCount() (int, nvml.Return)
	DeviceGetHandleByIndex(index int) (DeviceInterface, nvml.Return)
	SystemGetDriverVersion() (string, nvml.Return)
	SystemGetCudaDriverVersion() (int, nvml.Return)
}

// deviceWrapper wraps nvml.Device to implement DeviceInterface
type deviceWrapper struct {
	device nvml.Device
}

func (w deviceWrapper) GetName() (string, nvml.Return) {
	return w.device.GetName()
}

func (w deviceWrapper) GetUUID() (string, nvml.Return) {
	return w.device.GetUUID()
}

func (w deviceWrapper) GetMemoryInfo() (nvml.Memory, nvml.Return) {
	return w.device.GetMemoryInfo()
}

func (w deviceWrapper) GetUtilizationRates() (nvml.Utilization, nvml.Return) {
	return w.device.GetUtilizationRates()
}

func (w deviceWrapper) GetPowerUsage() (uint32, nvml.Return) {
	return w.device.GetPowerUsage()
}

func (w deviceWrapper) GetTemperature(sensor nvml.TemperatureSensors) (uint32, nvml.Return) {
	return w.device.GetTemperature(sensor)
}

// RealNVML implements NVMLInterface using actual NVML library
type RealNVML struct{}

// NewRealNVML creates a new real NVML instance
func NewRealNVML() *RealNVML {
	return &RealNVML{}
}

// Init initializes NVML
func (r *RealNVML) Init() nvml.Return {
	return nvml.Init()
}

// Shutdown shuts down NVML
func (r *RealNVML) Shutdown() nvml.Return {
	return nvml.Shutdown()
}

// DeviceGetCount returns the number of GPU devices
func (r *RealNVML) DeviceGetCount() (int, nvml.Return) {
	return nvml.DeviceGetCount()
}

// DeviceGetHandleByIndex returns a handle to a GPU device
func (r *RealNVML) DeviceGetHandleByIndex(index int) (DeviceInterface, nvml.Return) {
	device, ret := nvml.DeviceGetHandleByIndex(index)
	if ret != nvml.SUCCESS {
		return nil, ret
	}
	return deviceWrapper{device: device}, ret
}

// SystemGetDriverVersion returns the driver version
func (r *RealNVML) SystemGetDriverVersion() (string, nvml.Return) {
	return nvml.SystemGetDriverVersion()
}

// SystemGetCudaDriverVersion returns the CUDA driver version
func (r *RealNVML) SystemGetCudaDriverVersion() (int, nvml.Return) {
	return nvml.SystemGetCudaDriverVersion()
}
