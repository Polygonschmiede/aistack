//go:build cuda

package metrics

import (
	"testing"

	"aistack/internal/logging"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

func TestGPUCollector_Creation(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	collector := NewGPUCollector(logger)

	if collector == nil {
		t.Fatal("Expected collector to be created")
	}

	if collector.deviceIndex != 0 {
		t.Errorf("Expected device index 0, got %d", collector.deviceIndex)
	}

	if collector.initialized {
		t.Error("Expected collector to not be initialized")
	}
}

func TestGPUCollector_Initialize_Success(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)

	// Setup mock NVML
	mockNVML := newMockNVML()
	mockNVML.DeviceCount = 1
	mockNVML.Devices = []mockDevice{
		{
			Name:              "Test GPU",
			NameReturn:        nvml.SUCCESS,
			UtilizationReturn: nvml.SUCCESS,
			GPUUtil:           50,
			MemUtil:           40,
			MemoryTotal:       8 * 1024 * 1024 * 1024,
			MemoryUsed:        2 * 1024 * 1024 * 1024,
			MemoryInfoReturn:  nvml.SUCCESS,
			PowerUsage:        150000, // 150W in milliwatts
			PowerUsageReturn:  nvml.SUCCESS,
			Temperature:       65,
			TemperatureReturn: nvml.SUCCESS,
		},
	}

	collector := NewGPUCollectorWithNVML(mockNVML, logger)
	err := collector.Initialize()
	if err != nil {
		t.Fatalf("Expected successful initialization, got error: %v", err)
	}

	if !collector.IsInitialized() {
		t.Error("Expected collector to be initialized")
	}
}

func TestGPUCollector_Initialize_NVMLFail(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)

	mockNVML := newMockNVML()
	mockNVML.InitReturn = nvml.ERROR_LIBRARY_NOT_FOUND

	collector := NewGPUCollectorWithNVML(mockNVML, logger)
	err := collector.Initialize()
	if err == nil {
		t.Error("Expected initialization to fail")
	}

	if collector.IsInitialized() {
		t.Error("Expected collector to not be initialized")
	}
}

func TestGPUCollector_Collect_Success(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)

	// Setup mock NVML
	mockNVML := newMockNVML()
	mockNVML.DeviceCount = 1
	mockNVML.Devices = []mockDevice{
		{
			Name:              "Test GPU",
			NameReturn:        nvml.SUCCESS,
			UtilizationReturn: nvml.SUCCESS,
			GPUUtil:           75,
			MemUtil:           60,
			MemoryTotal:       8 * 1024 * 1024 * 1024,
			MemoryUsed:        3 * 1024 * 1024 * 1024,
			MemoryInfoReturn:  nvml.SUCCESS,
			PowerUsage:        200000, // 200W in milliwatts
			PowerUsageReturn:  nvml.SUCCESS,
			Temperature:       72,
			TemperatureReturn: nvml.SUCCESS,
		},
	}

	collector := NewGPUCollectorWithNVML(mockNVML, logger)
	if err := collector.Initialize(); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	defer collector.Shutdown()

	// Collect metrics
	gpuUtil, gpuMemMB, gpuWatts, tempGPU, err := collector.Collect()
	if err != nil {
		t.Fatalf("Expected successful collect, got error: %v", err)
	}

	// Verify metrics
	if gpuUtil == nil || *gpuUtil != 75.0 {
		t.Errorf("Expected GPU util 75%%, got %v", gpuUtil)
	}

	expectedMem := uint64(3072) // 3GB in MB
	if gpuMemMB == nil || *gpuMemMB != expectedMem {
		t.Errorf("Expected GPU mem %d MB, got %v", expectedMem, gpuMemMB)
	}

	if gpuWatts == nil || *gpuWatts != 200.0 {
		t.Errorf("Expected GPU power 200W, got %v", gpuWatts)
	}

	if tempGPU == nil || *tempGPU != 72.0 {
		t.Errorf("Expected GPU temp 72°C, got %v", tempGPU)
	}
}

func TestGPUCollector_Collect_NotInitialized(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	collector := NewGPUCollector(logger)

	_, _, _, _, err := collector.Collect()
	if err == nil {
		t.Error("Expected error when collecting without initialization")
	}
}
