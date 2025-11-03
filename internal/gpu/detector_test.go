package gpu

import (
	"errors"
	"os"
	"testing"

	"aistack/internal/logging"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

const (
	mockDriverVersion = "535.104.05"
)

func TestDetector_DetectGPUs_Success(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)

	// Setup mock NVML
	mockNVML := NewMockNVML()
	mockNVML.DriverVersion = mockDriverVersion
	mockNVML.CudaVersion = 12020 // CUDA 12.2
	mockNVML.DeviceCount = 2

	// Add mock devices
	mockNVML.Devices = []MockDevice{
		{
			Name:             "NVIDIA GeForce RTX 4090",
			NameReturn:       nvml.SUCCESS,
			UUID:             "GPU-12345678-1234-1234-1234-123456789012",
			UUIDReturn:       nvml.SUCCESS,
			MemoryTotal:      24 * 1024 * 1024 * 1024, // 24GB
			MemoryInfoReturn: nvml.SUCCESS,
		},
		{
			Name:             "NVIDIA GeForce RTX 3080",
			NameReturn:       nvml.SUCCESS,
			UUID:             "GPU-87654321-4321-4321-4321-210987654321",
			UUIDReturn:       nvml.SUCCESS,
			MemoryTotal:      10 * 1024 * 1024 * 1024, // 10GB
			MemoryInfoReturn: nvml.SUCCESS,
		},
	}

	detector := NewDetectorWithNVML(mockNVML, logger)
	report := detector.DetectGPUs()

	// Verify report
	if !report.NVMLOk {
		t.Error("Expected NVML to be OK")
	}

	if report.DriverVersion != mockDriverVersion {
		t.Errorf("Expected driver version %s, got: %s", mockDriverVersion, report.DriverVersion)
	}

	if report.CUDAVersion != 12020 {
		t.Errorf("Expected CUDA version 12020, got: %d", report.CUDAVersion)
	}

	if len(report.GPUs) != 2 {
		t.Errorf("Expected 2 GPUs, got: %d", len(report.GPUs))
	}

	// Verify first GPU
	if report.GPUs[0].Name != "NVIDIA GeForce RTX 4090" {
		t.Errorf("Expected GPU 0 name 'NVIDIA GeForce RTX 4090', got: %s", report.GPUs[0].Name)
	}

	if report.GPUs[0].MemoryMB != 24*1024 {
		t.Errorf("Expected GPU 0 memory 24576 MB, got: %d", report.GPUs[0].MemoryMB)
	}
}

func TestDetector_DetectGPUs_InitFailed(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)

	mockNVML := NewMockNVML()
	mockNVML.InitReturn = nvml.ERROR_LIBRARY_NOT_FOUND

	detector := NewDetectorWithNVML(mockNVML, logger)
	report := detector.DetectGPUs()

	if report.NVMLOk {
		t.Error("Expected NVML to be not OK when init fails")
	}

	if report.ErrorMessage == "" {
		t.Error("Expected error message when NVML init fails")
	}

	if len(report.GPUs) != 0 {
		t.Error("Expected no GPUs when NVML init fails")
	}
}

func TestDetector_DetectGPUs_NoDevices(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)

	mockNVML := NewMockNVML()
	mockNVML.DriverVersion = mockDriverVersion
	mockNVML.CudaVersion = 12020
	mockNVML.DeviceCount = 0

	detector := NewDetectorWithNVML(mockNVML, logger)
	report := detector.DetectGPUs()

	if !report.NVMLOk {
		t.Error("Expected NVML to be OK even with no devices")
	}

	if len(report.GPUs) != 0 {
		t.Errorf("Expected 0 GPUs, got: %d", len(report.GPUs))
	}
}

func TestDetector_DetectGPUs_DeviceCountFailed(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)

	mockNVML := NewMockNVML()
	mockNVML.DeviceCountReturn = nvml.ERROR_UNKNOWN

	detector := NewDetectorWithNVML(mockNVML, logger)
	report := detector.DetectGPUs()

	if !report.NVMLOk {
		t.Error("Expected NVML to be OK (init succeeded)")
	}

	if report.ErrorMessage == "" {
		t.Error("Expected error message when device count fails")
	}
}

func TestDetector_SaveReport(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	detector := NewDetector(logger)

	report := GPUReport{
		DriverVersion: mockDriverVersion,
		CUDAVersion:   12020,
		NVMLOk:        true,
		GPUs: []GPUInfo{
			{
				Name:     "Test GPU",
				UUID:     "GPU-test",
				MemoryMB: 8192,
				Index:    0,
			},
		},
	}

	tmpFile := "/tmp/test_gpu_report.json"
	defer os.Remove(tmpFile)

	err := detector.SaveReport(report, tmpFile)
	if err != nil {
		t.Fatalf("Expected no error saving report, got: %v", err)
	}

	// Verify file exists
	if _, statErr := os.Stat(tmpFile); errors.Is(statErr, os.ErrNotExist) {
		t.Error("Expected report file to exist")
	}

	// Verify file content (basic check)
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	content := string(data)
	if len(content) == 0 {
		t.Error("Expected non-empty report file")
	}
}
