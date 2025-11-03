package services

import (
	"fmt"
	"testing"

	"aistack/internal/logging"
)

func TestManager_RepairService(t *testing.T) {
	tests := []struct {
		name            string
		serviceName     string
		initialHealth   HealthStatus
		finalHealth     HealthStatus
		startError      error
		expectedSuccess bool
		expectedSkipped bool
	}{
		{
			name:            "repair unhealthy service successfully",
			serviceName:     "ollama",
			initialHealth:   HealthRed,
			finalHealth:     HealthGreen,
			expectedSuccess: true,
			expectedSkipped: false,
		},
		{
			name:            "skip repair for healthy service (idempotent)",
			serviceName:     "ollama",
			initialHealth:   HealthGreen,
			finalHealth:     HealthGreen,
			expectedSuccess: true,
			expectedSkipped: true,
		},
		{
			name:            "repair fails - service still unhealthy",
			serviceName:     "ollama",
			initialHealth:   HealthRed,
			finalHealth:     HealthRed,
			expectedSuccess: false,
			expectedSkipped: false,
		},
		{
			name:            "repair fails - service won't start",
			serviceName:     "ollama",
			initialHealth:   HealthRed,
			startError:      fmt.Errorf("failed to start"),
			expectedSuccess: false,
			expectedSkipped: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.NewLogger(logging.LevelInfo)

			// Create mock runtime with dynamic health states
			runtime := &MockRuntime{
				containerStatuses: make(map[string]ServiceStatus),
				startError:        tt.startError,
			}

			// Initial state
			runtime.containerStatuses[tt.serviceName] = ServiceStatus{
				Name:   tt.serviceName,
				State:  getStateFromHealth(tt.initialHealth),
				Health: tt.initialHealth,
			}

			// Create manager
			manager := &Manager{
				runtime:  runtime,
				logger:   logger,
				services: make(map[string]Service),
			}

			// Create service with dynamic health check
			healthCheck := &DynamicMockHealthCheck{
				initialStatus: tt.initialHealth,
				finalStatus:   tt.finalHealth,
				hasRepaired:   false,
			}

			service := NewBaseService(tt.serviceName, "/tmp", healthCheck, []string{"test_volume"}, runtime, logger)
			manager.services[tt.serviceName] = service

			// Perform repair
			result, err := manager.RepairService(tt.serviceName)

			// Verify error handling
			if tt.startError != nil {
				if err == nil {
					t.Error("Expected error when service fails to start")
				}
			}

			// Verify result
			if result.ServiceName != tt.serviceName {
				t.Errorf("Expected service name %s, got %s", tt.serviceName, result.ServiceName)
			}

			if result.Success != tt.expectedSuccess {
				t.Errorf("Expected success=%v, got %v", tt.expectedSuccess, result.Success)
			}

			// Verify skipped reason
			if tt.expectedSkipped {
				if result.SkippedReason == "" {
					t.Error("Expected skipped reason to be set")
				}
			} else {
				if result.SkippedReason != "" {
					t.Errorf("Expected no skipped reason, got: %s", result.SkippedReason)
				}
			}

			// Verify health states
			if result.HealthBefore != tt.initialHealth {
				t.Errorf("Expected initial health %s, got %s", tt.initialHealth, result.HealthBefore)
			}

			if tt.expectedSuccess && !tt.expectedSkipped {
				if result.HealthAfter != HealthGreen {
					t.Errorf("Expected final health green, got %s", result.HealthAfter)
				}
			}
		})
	}
}

func TestManager_RepairService_VolumesPreserved(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)

	runtime := &MockRuntime{
		containerStatuses: map[string]ServiceStatus{
			"ollama": {
				Name:   "ollama",
				State:  "stopped",
				Health: HealthRed,
			},
		},
		RemovedVolumes: make([]string, 0),
	}

	manager := &Manager{
		runtime:  runtime,
		logger:   logger,
		services: make(map[string]Service),
	}

	healthCheck := &DynamicMockHealthCheck{
		initialStatus: HealthRed,
		finalStatus:   HealthGreen,
	}

	volumes := []string{"ollama_data", "ollama_models"}
	service := NewBaseService("ollama", "/tmp", healthCheck, volumes, runtime, logger)
	manager.services["ollama"] = service

	// Perform repair
	_, err := manager.RepairService("ollama")
	if err != nil {
		t.Fatalf("RepairService() error = %v", err)
	}

	// Verify volumes were NOT removed
	if len(runtime.RemovedVolumes) > 0 {
		t.Errorf("Expected no volumes to be removed during repair, but %d were removed: %v",
			len(runtime.RemovedVolumes), runtime.RemovedVolumes)
	}
}

func TestManager_RepairAll(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)

	runtime := &MockRuntime{
		containerStatuses: map[string]ServiceStatus{
			"ollama": {
				Name:   "ollama",
				State:  "stopped",
				Health: HealthRed,
			},
			"openwebui": {
				Name:   "openwebui",
				State:  "running",
				Health: HealthGreen,
			},
			"localai": {
				Name:   "localai",
				State:  "stopped",
				Health: HealthYellow,
			},
		},
	}

	manager := &Manager{
		runtime:  runtime,
		logger:   logger,
		services: make(map[string]Service),
	}

	// Create services with different health states
	services := []struct {
		name          string
		initialHealth HealthStatus
		finalHealth   HealthStatus
	}{
		{"ollama", HealthRed, HealthGreen},
		{"openwebui", HealthGreen, HealthGreen},
		{"localai", HealthYellow, HealthGreen},
	}

	for _, s := range services {
		healthCheck := &DynamicMockHealthCheck{
			initialStatus: s.initialHealth,
			finalStatus:   s.finalHealth,
		}
		service := NewBaseService(s.name, "/tmp", healthCheck, nil, runtime, logger)
		manager.services[s.name] = service
	}

	// Perform repair all
	results, err := manager.RepairAll()
	if err != nil {
		t.Fatalf("RepairAll() error = %v", err)
	}

	// Should repair ollama and localai (unhealthy), skip openwebui (healthy)
	// Expected: 2 repairs (ollama and localai)
	if len(results) != 2 {
		t.Errorf("Expected 2 repair results, got %d", len(results))
	}

	// Verify all repaired services are now successful
	for _, result := range results {
		if !result.Success {
			t.Errorf("Expected service %s to be repaired successfully", result.ServiceName)
		}
	}
}

// DynamicMockHealthCheck simulates health state changing after repair
type DynamicMockHealthCheck struct {
	initialStatus HealthStatus
	finalStatus   HealthStatus
	hasRepaired   bool
}

func (m *DynamicMockHealthCheck) Check() (HealthStatus, error) {
	// First call returns initial status, subsequent calls return final status
	if !m.hasRepaired {
		m.hasRepaired = true
		return m.initialStatus, nil
	}
	return m.finalStatus, nil
}

func getStateFromHealth(health HealthStatus) string {
	if health == HealthGreen {
		return "running"
	}
	return "stopped"
}
