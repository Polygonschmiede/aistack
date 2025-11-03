package services

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"aistack/internal/gpu"
	"aistack/internal/logging"
)

// HealthReport represents the aggregated health report for all services and GPU
// Story T-025: Health-Reporter (Services + GPU Smoke)
// Data Contract from EP-014: health_report.json
type HealthReport struct {
	Timestamp time.Time             `json:"timestamp"`
	Services  []ServiceHealthStatus `json:"services"`
	GPU       GPUHealthStatus       `json:"gpu"`
}

// ServiceHealthStatus represents health status of a single service
type ServiceHealthStatus struct {
	Name    string       `json:"name"`
	Health  HealthStatus `json:"health"`
	Message string       `json:"message,omitempty"`
}

// GPUHealthStatus represents GPU health status
type GPUHealthStatus struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

// HealthReporter aggregates health checks across services and GPU
type HealthReporter struct {
	manager    *Manager
	gpuChecker GPUHealthChecker
	logger     *logging.Logger
}

// GPUHealthChecker performs GPU smoke tests
type GPUHealthChecker interface {
	CheckGPU() GPUHealthStatus
}

// DefaultGPUHealthChecker implements GPUHealthChecker using NVML
type DefaultGPUHealthChecker struct {
	detector *gpu.Detector
	logger   *logging.Logger
}

// NewDefaultGPUHealthChecker creates a new default GPU health checker
func NewDefaultGPUHealthChecker(logger *logging.Logger) *DefaultGPUHealthChecker {
	return &DefaultGPUHealthChecker{
		detector: gpu.NewDetector(logger),
		logger:   logger,
	}
}

// CheckGPU performs a GPU smoke test (NVML init/shutdown check)
// Story T-025: GPU-Schnelltest
func (c *DefaultGPUHealthChecker) CheckGPU() GPUHealthStatus {
	c.logger.Info("health.gpu.check.start", "Starting GPU smoke test", nil)

	report := c.detector.DetectGPUs()

	if !report.NVMLOk {
		c.logger.Warn("health.gpu.check.failed", "GPU smoke test failed", map[string]interface{}{
			"error": report.ErrorMessage,
		})
		return GPUHealthStatus{
			OK:      false,
			Message: report.ErrorMessage,
		}
	}

	c.logger.Info("health.gpu.check.success", "GPU smoke test passed", map[string]interface{}{
		"gpu_count": len(report.GPUs),
	})

	return GPUHealthStatus{
		OK:      true,
		Message: fmt.Sprintf("%d GPU(s) detected", len(report.GPUs)),
	}
}

// NewHealthReporter creates a new health reporter
func NewHealthReporter(manager *Manager, gpuChecker GPUHealthChecker, logger *logging.Logger) *HealthReporter {
	if gpuChecker == nil {
		gpuChecker = NewDefaultGPUHealthChecker(logger)
	}

	return &HealthReporter{
		manager:    manager,
		gpuChecker: gpuChecker,
		logger:     logger,
	}
}

// GenerateReport generates a comprehensive health report
// Story T-025: HTTP/Port-Probes, GPU-Schnelltest, aggregierter Report
func (r *HealthReporter) GenerateReport() (HealthReport, error) {
	r.logger.Info("health.report.start", "Generating health report", nil)

	report := HealthReport{
		Timestamp: time.Now().UTC(),
		Services:  make([]ServiceHealthStatus, 0),
	}

	// Check all services
	for _, serviceName := range r.manager.ListServices() {
		service, err := r.manager.GetService(serviceName)
		if err != nil {
			r.logger.Warn("health.report.service.error", "Failed to get service", map[string]interface{}{
				"service": serviceName,
				"error":   err.Error(),
			})
			continue
		}

		serviceHealth := ServiceHealthStatus{
			Name: serviceName,
		}

		// Get service status
		status, err := service.Status()
		if err != nil {
			serviceHealth.Health = HealthRed
			serviceHealth.Message = fmt.Sprintf("Status check failed: %v", err)
		} else {
			serviceHealth.Health = status.Health
			serviceHealth.Message = status.Message
		}

		report.Services = append(report.Services, serviceHealth)

		r.logger.Info("health.report.service", "Service health checked", map[string]interface{}{
			"service": serviceName,
			"health":  serviceHealth.Health,
		})
	}

	// Check GPU
	report.GPU = r.gpuChecker.CheckGPU()

	r.logger.Info("health.report.complete", "Health report generated", map[string]interface{}{
		"service_count": len(report.Services),
		"gpu_ok":        report.GPU.OK,
	})

	return report, nil
}

// SaveReport saves the health report to a JSON file
func (r *HealthReporter) SaveReport(report HealthReport, filepath string) error {
	r.logger.Info("health.report.save", "Saving health report", map[string]interface{}{
		"path": filepath,
	})

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal health report: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write health report: %w", err)
	}

	r.logger.Info("health.report.saved", "Health report saved successfully", map[string]interface{}{
		"path": filepath,
	})

	return nil
}

// CheckAllHealthy returns true if all services and GPU are healthy
func (r *HealthReporter) CheckAllHealthy() (bool, error) {
	report, err := r.GenerateReport()
	if err != nil {
		return false, err
	}

	// Check GPU
	if !report.GPU.OK {
		return false, nil
	}

	// Check all services
	for _, service := range report.Services {
		if service.Health != HealthGreen {
			return false, nil
		}
	}

	return true, nil
}
