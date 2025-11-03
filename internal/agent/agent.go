// Package agent provides the background service implementation for aistack
// This is a placeholder for EP-002 to satisfy systemd service requirements
package agent

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aistack/internal/logging"
)

// Agent represents the background service
type Agent struct {
	logger    *logging.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	tickRate  time.Duration
	startTime time.Time
}

// NewAgent creates a new agent instance
func NewAgent(logger *logging.Logger) *Agent {
	ctx, cancel := context.WithCancel(context.Background())

	return &Agent{
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
		tickRate:  10 * time.Second, // Default tick rate
		startTime: time.Now(),
	}
}

// Run starts the agent background service
// This is a minimal placeholder implementation for EP-002
func (a *Agent) Run() error {
	a.logger.Info("agent.started", "Agent service started", map[string]interface{}{
		"pid":       os.Getpid(),
		"tick_rate": a.tickRate.String(),
	})

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
			a.logger.Info("agent.context_cancelled", "Agent context cancelled", nil)
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
			// Periodic heartbeat - placeholder for future functionality
			uptime := time.Since(a.startTime)
			a.logger.Debug("agent.heartbeat", "Agent heartbeat", map[string]interface{}{
				"uptime_seconds": uptime.Seconds(),
			})

			// TODO: Future epics will add:
			// - Metrics collection (EP-005)
			// - Container health checks (EP-008-010)
			// - GPU monitoring (EP-004)
		}
	}
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
// This is a placeholder for EP-006 (Idle Engine)
func IdleCheck(logger *logging.Logger) error {
	logger.Info("idle.check_started", "Idle check started", nil)

	// TODO: Implement idle detection in EP-006
	// For now, just log that we were called
	logger.Debug("idle.placeholder", "Idle check is a placeholder", map[string]interface{}{
		"status": "not_implemented",
		"epic":   "EP-006",
	})

	logger.Info("idle.check_completed", "Idle check completed", nil)
	return nil
}

// HealthCheck performs a health check of the agent
func (a *Agent) HealthCheck() error {
	// Check if context is still valid
	select {
	case <-a.ctx.Done():
		return fmt.Errorf("agent context is cancelled")
	default:
		return nil
	}
}
