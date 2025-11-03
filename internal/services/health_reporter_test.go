package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"aistack/internal/logging"
)

// MockGPUHealthChecker implements GPUHealthChecker for testing
type MockGPUHealthChecker struct {
	shouldPass bool
	message    string
}

func (m *MockGPUHealthChecker) CheckGPU() GPUHealthStatus {
	return GPUHealthStatus{
		OK:      m.shouldPass,
		Message: m.message,
	}
}

func TestHealthReporter_GenerateReport(t *testing.T) {
	tests := []struct {
		name              string
		serviceStatuses   map[string]ServiceStatus
		gpuOK             bool
		gpuMessage        string
		expectedGPUOK     bool
		expectedServCount int
	}{
		{
			name: "all services green, GPU ok",
			serviceStatuses: map[string]ServiceStatus{
				"ollama": {
					Name:   "ollama",
					State:  "running",
					Health: HealthGreen,
				},
			},
			gpuOK:             true,
			gpuMessage:        "1 GPU(s) detected",
			expectedGPUOK:     true,
			expectedServCount: 1,
		},
		{
			name: "service red, GPU ok",
			serviceStatuses: map[string]ServiceStatus{
				"ollama": {
					Name:    "ollama",
					State:   "stopped",
					Health:  HealthRed,
					Message: "Service is down",
				},
			},
			gpuOK:             true,
			gpuMessage:        "1 GPU(s) detected",
			expectedGPUOK:     true,
			expectedServCount: 1,
		},
		{
			name: "service green, GPU fail",
			serviceStatuses: map[string]ServiceStatus{
				"ollama": {
					Name:   "ollama",
					State:  "running",
					Health: HealthGreen,
				},
			},
			gpuOK:             false,
			gpuMessage:        "NVML initialization failed",
			expectedGPUOK:     false,
			expectedServCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.NewLogger(logging.LevelInfo)
			runtime := &MockRuntime{
				containerStatuses: tt.serviceStatuses,
			}

			// Create manager with mock runtime
			manager := &Manager{
				runtime:  runtime,
				logger:   logger,
				services: make(map[string]Service),
			}

			// Add mock service
			for name := range tt.serviceStatuses {
				healthCheck := &MockHealthCheck{status: tt.serviceStatuses[name].Health}
				service := NewBaseService(name, "/tmp", healthCheck, nil, runtime, logger)
				manager.services[name] = service
			}

			// Create mock GPU checker
			gpuChecker := &MockGPUHealthChecker{
				shouldPass: tt.gpuOK,
				message:    tt.gpuMessage,
			}

			reporter := NewHealthReporter(manager, gpuChecker, logger)

			// Generate report
			report, err := reporter.GenerateReport()
			if err != nil {
				t.Fatalf("GenerateReport() error = %v", err)
			}

			// Verify timestamp
			if report.Timestamp.IsZero() {
				t.Error("Expected timestamp to be set")
			}

			// Verify service count
			if len(report.Services) != tt.expectedServCount {
				t.Errorf("Expected %d services, got %d", tt.expectedServCount, len(report.Services))
			}

			// Verify GPU status
			if report.GPU.OK != tt.expectedGPUOK {
				t.Errorf("Expected GPU OK=%v, got %v", tt.expectedGPUOK, report.GPU.OK)
			}

			if report.GPU.Message != tt.gpuMessage {
				t.Errorf("Expected GPU message=%q, got %q", tt.gpuMessage, report.GPU.Message)
			}
		})
	}
}

func TestHealthReporter_SaveReport(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	runtime := &MockRuntime{
		containerStatuses: map[string]ServiceStatus{},
	}

	manager := &Manager{
		runtime:  runtime,
		logger:   logger,
		services: make(map[string]Service),
	}

	gpuChecker := &MockGPUHealthChecker{
		shouldPass: true,
		message:    "GPU OK",
	}

	reporter := NewHealthReporter(manager, gpuChecker, logger)

	// Create temp directory
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "health_report.json")

	// Generate and save report
	report := HealthReport{
		Timestamp: time.Now().UTC(),
		Services: []ServiceHealthStatus{
			{
				Name:   "ollama",
				Health: HealthGreen,
			},
		},
		GPU: GPUHealthStatus{
			OK:      true,
			Message: "GPU OK",
		},
	}

	err := reporter.SaveReport(report, reportPath)
	if err != nil {
		t.Fatalf("SaveReport() error = %v", err)
	}

	// Verify file exists
	if _, err = os.Stat(reportPath); os.IsNotExist(err) {
		t.Fatal("Report file was not created")
	}

	// Read and verify contents
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	var loadedReport HealthReport
	if err := json.Unmarshal(data, &loadedReport); err != nil {
		t.Fatalf("Failed to unmarshal report: %v", err)
	}

	// Verify data
	if len(loadedReport.Services) != 1 {
		t.Errorf("Expected 1 service in loaded report, got %d", len(loadedReport.Services))
	}

	if !loadedReport.GPU.OK {
		t.Error("Expected GPU OK in loaded report")
	}
}

func TestHealthReporter_CheckAllHealthy(t *testing.T) {
	tests := []struct {
		name            string
		serviceStatuses map[string]ServiceStatus
		gpuOK           bool
		expectedHealthy bool
	}{
		{
			name: "all healthy",
			serviceStatuses: map[string]ServiceStatus{
				"ollama": {
					Name:   "ollama",
					State:  "running",
					Health: HealthGreen,
				},
			},
			gpuOK:           true,
			expectedHealthy: true,
		},
		{
			name: "service unhealthy",
			serviceStatuses: map[string]ServiceStatus{
				"ollama": {
					Name:   "ollama",
					State:  "stopped",
					Health: HealthRed,
				},
			},
			gpuOK:           true,
			expectedHealthy: false,
		},
		{
			name: "GPU unhealthy",
			serviceStatuses: map[string]ServiceStatus{
				"ollama": {
					Name:   "ollama",
					State:  "running",
					Health: HealthGreen,
				},
			},
			gpuOK:           false,
			expectedHealthy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.NewLogger(logging.LevelInfo)
			runtime := &MockRuntime{
				containerStatuses: tt.serviceStatuses,
			}

			manager := &Manager{
				runtime:  runtime,
				logger:   logger,
				services: make(map[string]Service),
			}

			for name := range tt.serviceStatuses {
				healthCheck := &MockHealthCheck{status: tt.serviceStatuses[name].Health}
				service := NewBaseService(name, "/tmp", healthCheck, nil, runtime, logger)
				manager.services[name] = service
			}

			gpuChecker := &MockGPUHealthChecker{
				shouldPass: tt.gpuOK,
				message:    "test",
			}

			reporter := NewHealthReporter(manager, gpuChecker, logger)

			healthy, err := reporter.CheckAllHealthy()
			if err != nil {
				t.Fatalf("CheckAllHealthy() error = %v", err)
			}

			if healthy != tt.expectedHealthy {
				t.Errorf("Expected healthy=%v, got %v", tt.expectedHealthy, healthy)
			}
		})
	}
}

// MockHealthCheck implements HealthChecker for testing
type MockHealthCheck struct {
	status HealthStatus
	err    error
}

func (m *MockHealthCheck) Check() (HealthStatus, error) {
	if m.err != nil {
		return HealthRed, m.err
	}
	return m.status, nil
}
