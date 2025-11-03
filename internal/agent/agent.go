// Package agent provides the background service implementation for aistack
// This is a placeholder for EP-002 to satisfy systemd service requirements
package agent

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"aistack/internal/idle"
	"aistack/internal/logging"
	"aistack/internal/metrics"
)

// Agent represents the background service
type Agent struct {
	logger               *logging.Logger
	ctx                  context.Context
	cancel               context.CancelFunc
	tickRate             time.Duration
	startTime            time.Time
	metricsCollector     *metrics.Collector
	idleEngine           *idle.Engine
	idleStateManager     *idle.StateManager
	idleExecutor         *idle.Executor
	metricsLogPath       string
	metricsWriteFailed   bool
	inhibitorCheckFailed bool
}

// NewAgent creates a new agent instance
func NewAgent(logger *logging.Logger) *Agent {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize metrics collector
	metricsConfig := metrics.DefaultConfig()
	metricsCollector := metrics.NewCollector(metricsConfig, logger)

	// Initialize idle detection
	idleConfig := idle.DefaultIdleConfig()
	idleEngine := idle.NewEngine(idleConfig, logger)
	idleStateManager := idle.NewStateManager(idleConfig.StateFilePath, logger)
	idleExecutor := idle.NewExecutor(idleConfig, logger)

	metricsLogDir := resolveLogDir(logger)
	metricsLogPath := filepath.Join(metricsLogDir, "metrics.log")

	return &Agent{
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
		tickRate:         10 * time.Second, // Default tick rate
		startTime:        time.Now(),
		metricsCollector: metricsCollector,
		idleEngine:       idleEngine,
		idleStateManager: idleStateManager,
		idleExecutor:     idleExecutor,
		metricsLogPath:   metricsLogPath,
	}
}

// Run starts the agent background service
func (a *Agent) Run() error {
	a.logger.Info("agent.started", "Agent service started", map[string]interface{}{
		"pid":       os.Getpid(),
		"tick_rate": a.tickRate.String(),
	})

	// Initialize metrics collector
	if err := a.metricsCollector.Initialize(); err != nil {
		a.logger.Warn("agent.metrics.init_failed", "Failed to initialize metrics collector", map[string]interface{}{
			"error": err.Error(),
		})
	}
	defer a.metricsCollector.Shutdown()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Ticker for periodic tasks
	ticker := time.NewTicker(a.tickRate)
	defer ticker.Stop()

	// Main event loop
	for {
		select {
		case <-a.ctx.Done():
			a.logger.Info("agent.context_canceled", "Agent context canceled", nil)
			return a.ctx.Err()

		case sig := <-sigChan:
			a.logger.Info("agent.signal_received", "Received signal", map[string]interface{}{
				"signal": sig.String(),
			})

			switch sig {
			case syscall.SIGHUP:
				// Reload configuration
				a.logger.Info("agent.reload", "Reloading configuration", nil)
				// TODO: Implement config reload in future epic
			case syscall.SIGTERM, syscall.SIGINT:
				a.logger.Info("agent.shutdown", "Initiating graceful shutdown", nil)
				return a.Shutdown()
			}

		case <-ticker.C:
			// Periodic heartbeat and metrics collection
			uptime := time.Since(a.startTime)
			a.logger.Debug("agent.heartbeat", "Agent heartbeat", map[string]interface{}{
				"uptime_seconds": uptime.Seconds(),
			})

			// Collect metrics and update idle engine
			a.collectAndProcessMetrics()
		}
	}
}

// collectAndProcessMetrics collects metrics and processes idle state
func (a *Agent) collectAndProcessMetrics() {
	// Collect metrics sample
	sample, err := a.metricsCollector.CollectSample()
	if err != nil {
		a.logger.Warn("agent.metrics.collect_failed", "Failed to collect metrics", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Extract CPU and GPU utilization for idle detection
	cpuUtil := 0.0
	gpuUtil := 0.0

	if sample.CPUUtil != nil {
		cpuUtil = *sample.CPUUtil
	}
	if sample.GPUUtil != nil {
		gpuUtil = *sample.GPUUtil
	}

	// Add metrics to idle engine
	a.idleEngine.AddMetrics(cpuUtil, gpuUtil)

	// Get current idle state
	idleState := a.idleEngine.GetState()

	if a.idleExecutor != nil {
		if hasInhibit, inhibitors, err := a.idleExecutor.ActiveInhibitors(); err != nil {
			if !a.inhibitorCheckFailed {
				a.logger.Warn("agent.inhibitors.check_failed", "Failed to inspect systemd inhibitors", map[string]interface{}{
					"error": err.Error(),
				})
				a.inhibitorCheckFailed = true
			}
		} else {
			if hasInhibit {
				idleState.GatingReasons = addGatingReason(idleState.GatingReasons, idle.GatingReasonInhibit)
				a.logger.Debug("agent.inhibitors.active", "Active inhibitors detected", map[string]interface{}{
					"count": len(inhibitors),
				})
			} else {
				idleState.GatingReasons = removeGatingReason(idleState.GatingReasons, idle.GatingReasonInhibit)
			}
			a.inhibitorCheckFailed = false
		}
	}

	// Persist metrics sample to JSONL log
	if err := a.metricsCollector.WriteSample(sample, a.metricsLogPath); err != nil {
		if !a.metricsWriteFailed {
			a.logger.Warn("agent.metrics.write_failed", "Failed to write metrics sample", map[string]interface{}{
				"error": err.Error(),
				"path":  a.metricsLogPath,
			})
			a.metricsWriteFailed = true
		}
	} else {
		if a.metricsWriteFailed {
			a.logger.Info("agent.metrics.write_recovered", "Metrics logging restored", map[string]interface{}{
				"path": a.metricsLogPath,
			})
		}
		a.metricsWriteFailed = false
	}

	// Save idle state
	if err := a.idleStateManager.Save(idleState); err != nil {
		a.logger.Warn("agent.idle.state_save_failed", "Failed to save idle state", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Log idle state
	a.logger.Debug("agent.idle.state_updated", "Idle state updated", map[string]interface{}{
		"status":         idleState.Status,
		"idle_for_s":     idleState.IdleForSeconds,
		"threshold_s":    idleState.ThresholdSeconds,
		"gating_reasons": idleState.GatingReasons,
	})
}

// resolveLogDir determines a writable log directory, favoring production defaults
func resolveLogDir(logger *logging.Logger) string {
	var candidates []string

	if envDir := os.Getenv("AISTACK_LOG_DIR"); envDir != "" {
		candidates = append(candidates, envDir)
	}

	candidates = append(candidates, "/var/log/aistack")

	for _, dir := range candidates {
		if err := ensureWritableDir(dir); err == nil {
			return dir
		}

		logger.Warn("agent.logdir.unwritable", "Log directory not writable, trying fallback", map[string]interface{}{
			"path": dir,
		})
	}

	fallback := filepath.Join(os.TempDir(), "aistack")
	if err := ensureWritableDir(fallback); err != nil {
		logger.Error("agent.logdir.fallback_failed", "Failed to prepare fallback log directory", map[string]interface{}{
			"path":  fallback,
			"error": err.Error(),
		})
	}
	return fallback
}

func ensureWritableDir(path string) error {
	if err := os.MkdirAll(path, 0o750); err != nil {
		return err
	}

	testFile := filepath.Join(path, ".write-test")
	if err := os.WriteFile(testFile, []byte{}, 0o600); err != nil {
		return err
	}

	return os.Remove(testFile)
}

func addGatingReason(reasons []string, reason string) []string {
	for _, r := range reasons {
		if r == reason {
			return reasons
		}
	}
	return append(reasons, reason)
}

func removeGatingReason(reasons []string, reason string) []string {
	filtered := make([]string, 0, len(reasons))
	for _, r := range reasons {
		if r != reason {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// Shutdown performs graceful shutdown of the agent
func (a *Agent) Shutdown() error {
	a.logger.Info("agent.stopping", "Stopping agent service", nil)

	// Cancel context to stop all goroutines
	a.cancel()

	uptime := time.Since(a.startTime)
	a.logger.Info("agent.stopped", "Agent service stopped", map[string]interface{}{
		"uptime_seconds": uptime.Seconds(),
	})

	return nil
}

// IdleCheck performs a single idle evaluation (for timer-triggered runs)
func IdleCheck(logger *logging.Logger) error {
	logger.Info("idle.check_started", "Idle check started", nil)

	// Load idle configuration
	idleConfig := idle.DefaultIdleConfig()

	// Create state manager and load current state
	stateManager := idle.NewStateManager(idleConfig.StateFilePath, logger)

	state, err := stateManager.Load()
	if err != nil {
		logger.Warn("idle.state_load_failed", "Failed to load idle state", map[string]interface{}{
			"error": err.Error(),
		})
		logger.Info("idle.check_completed", "Idle check completed (no state)", nil)
		return nil
	}

	logger.Info("idle.state_loaded", "Idle state loaded", map[string]interface{}{
		"status":         state.Status,
		"idle_for_s":     state.IdleForSeconds,
		"threshold_s":    state.ThresholdSeconds,
		"gating_reasons": state.GatingReasons,
	})

	// Create idle engine and executor
	idleEngine := idle.NewEngine(idleConfig, logger)
	executor := idle.NewExecutor(idleConfig, logger)

	// Check if we should suspend
	if idleEngine.ShouldSuspend(state) {
		logger.Info("idle.suspend_check", "System should suspend", map[string]interface{}{
			"idle_for_s":  state.IdleForSeconds,
			"threshold_s": state.ThresholdSeconds,
		})

		// Attempt suspend
		if err := executor.Execute(&state); err != nil {
			logger.Error("idle.suspend_failed", "Failed to execute suspend", map[string]interface{}{
				"error": err.Error(),
			})
			if saveErr := stateManager.Save(state); saveErr != nil {
				logger.Warn("idle.state_save_failed", "Failed to persist updated state", map[string]interface{}{
					"error": saveErr.Error(),
				})
			}
			return err
		}

		if err := stateManager.Save(state); err != nil {
			logger.Warn("idle.state_save_failed", "Failed to persist updated state", map[string]interface{}{
				"error": err.Error(),
			})
		}
	} else {
		logger.Info("idle.suspend_skipped", "Suspend not required", map[string]interface{}{
			"status":         state.Status,
			"idle_for_s":     state.IdleForSeconds,
			"threshold_s":    state.ThresholdSeconds,
			"gating_reasons": state.GatingReasons,
		})
	}

	logger.Info("idle.check_completed", "Idle check completed", nil)
	return nil
}

// HealthCheck performs a health check of the agent
func (a *Agent) HealthCheck() error {
	// Check if context is still valid
	select {
	case <-a.ctx.Done():
		return fmt.Errorf("agent context is canceled")
	default:
		return nil
	}
}
