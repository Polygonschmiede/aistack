package services

import (
	"fmt"
	"time"
)

// RepairResult represents the result of a service repair operation
// Story T-026: Repair-Command für einzelne Services
type RepairResult struct {
	ServiceName   string       `json:"service_name"`
	Success       bool         `json:"success"`
	HealthBefore  HealthStatus `json:"health_before"`
	HealthAfter   HealthStatus `json:"health_after"`
	ErrorMessage  string       `json:"error_message,omitempty"`
	RepairedAt    time.Time    `json:"repaired_at"`
	SkippedReason string       `json:"skipped_reason,omitempty"`
}

// RepairService performs repair on a single service
// Story T-026: Stop → Remove → Recreate (ohne Volume-Löschung), Health-Recheck
//
// Repair workflow:
// 1. Check current health (if green, skip with no-op)
// 2. Stop service (graceful)
// 3. Remove container (volumes preserved)
// 4. Start service (recreate)
// 5. Wait for initialization
// 6. Recheck health
// 7. Return result (success if green, failed otherwise)
func (m *Manager) RepairService(serviceName string) (RepairResult, error) {
	m.logger.Info("service.repair.started", "Starting service repair", map[string]interface{}{
		"service": serviceName,
	})

	result := RepairResult{
		ServiceName: serviceName,
		RepairedAt:  time.Now().UTC(),
	}

	// Get service
	service, err := m.GetService(serviceName)
	if err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Failed to get service: %v", err)
		return result, err
	}

	// Check initial health
	initialStatus, err := service.Status()
	if err != nil {
		result.HealthBefore = HealthRed
		m.logger.Warn("service.repair.initial_status_failed", "Failed to get initial status", map[string]interface{}{
			"service": serviceName,
			"error":   err.Error(),
		})
	} else {
		result.HealthBefore = initialStatus.Health
	}

	// If service is already healthy, skip repair (idempotent)
	if result.HealthBefore == HealthGreen {
		m.logger.Info("service.repair.skipped", "Service is already healthy, repair not needed", map[string]interface{}{
			"service": serviceName,
		})
		result.Success = true
		result.HealthAfter = HealthGreen
		result.SkippedReason = "service already healthy"
		return result, nil
	}

	// Step 1: Stop service
	m.logger.Info("service.repair.stopping", "Stopping service for repair", map[string]interface{}{
		"service": serviceName,
	})

	if err := service.Stop(); err != nil {
		// Log warning but continue - service might already be stopped
		m.logger.Warn("service.repair.stop_error", "Error stopping service (continuing)", map[string]interface{}{
			"service": serviceName,
			"error":   err.Error(),
		})
	}

	// Step 2: Remove container (keep volumes)
	m.logger.Info("service.repair.removing", "Removing service container", map[string]interface{}{
		"service": serviceName,
	})

	containerName := fmt.Sprintf("aistack-%s", serviceName)
	if err := m.runtime.RemoveContainer(containerName); err != nil {
		// Log warning but continue - container might not exist
		m.logger.Warn("service.repair.remove_error", "Error removing container (continuing)", map[string]interface{}{
			"service":   serviceName,
			"container": containerName,
			"error":     err.Error(),
		})
	}

	// Step 3: Start service (recreate)
	m.logger.Info("service.repair.starting", "Starting service", map[string]interface{}{
		"service": serviceName,
	})

	if err := service.Start(); err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Failed to start service: %v", err)
		m.logger.Error("service.repair.failed", "Failed to start service during repair", map[string]interface{}{
			"service": serviceName,
			"error":   err.Error(),
		})
		return result, fmt.Errorf("failed to start service during repair: %w", err)
	}

	// Step 4: Wait for initialization
	m.logger.Info("service.repair.waiting", "Waiting for service initialization", map[string]interface{}{
		"service": serviceName,
		"delay":   "5s",
	})
	time.Sleep(5 * time.Second)

	// Step 5: Recheck health
	m.logger.Info("service.repair.health_check", "Checking service health after repair", map[string]interface{}{
		"service": serviceName,
	})

	finalStatus, err := service.Status()
	if err != nil {
		result.Success = false
		result.HealthAfter = HealthRed
		result.ErrorMessage = fmt.Sprintf("Health check failed after repair: %v", err)
		m.logger.Error("service.repair.health_failed", "Health check failed after repair", map[string]interface{}{
			"service": serviceName,
			"error":   err.Error(),
		})
		return result, fmt.Errorf("health check failed after repair: %w", err)
	}

	result.HealthAfter = finalStatus.Health

	if finalStatus.Health == HealthGreen {
		result.Success = true
		m.logger.Info("service.repair.completed", "Service repair completed successfully", map[string]interface{}{
			"service":       serviceName,
			"health_before": result.HealthBefore,
			"health_after":  result.HealthAfter,
		})
	} else {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Service health is still %s after repair: %s", finalStatus.Health, finalStatus.Message)
		m.logger.Error("service.repair.failed", "Service is not healthy after repair", map[string]interface{}{
			"service":       serviceName,
			"health_before": result.HealthBefore,
			"health_after":  result.HealthAfter,
			"message":       finalStatus.Message,
		})
	}

	return result, nil
}

// RepairAll repairs all services that are not healthy
func (m *Manager) RepairAll() ([]RepairResult, error) {
	m.logger.Info("service.repair_all.started", "Starting repair for all unhealthy services", nil)

	results := make([]RepairResult, 0)

	for _, serviceName := range m.ListServices() {
		service, err := m.GetService(serviceName)
		if err != nil {
			m.logger.Warn("service.repair_all.skip", "Skipping service due to error", map[string]interface{}{
				"service": serviceName,
				"error":   err.Error(),
			})
			continue
		}

		// Check if service needs repair
		status, err := service.Status()
		if err != nil || status.Health != HealthGreen {
			result, err := m.RepairService(serviceName)
			if err != nil {
				m.logger.Warn("service.repair_all.error", "Error repairing service", map[string]interface{}{
					"service": serviceName,
					"error":   err.Error(),
				})
			}
			results = append(results, result)
		} else {
			m.logger.Info("service.repair_all.skip_healthy", "Skipping healthy service", map[string]interface{}{
				"service": serviceName,
			})
		}
	}

	m.logger.Info("service.repair_all.completed", "Repair all completed", map[string]interface{}{
		"repaired_count": len(results),
	})

	return results, nil
}
