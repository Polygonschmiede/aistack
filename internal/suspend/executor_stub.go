//go:build !linux

package suspend

import (
	"fmt"

	"aistack/internal/logging"
)

// Executor handles suspend logic and execution (stub for non-Linux)
type Executor struct {
	logger *logging.Logger
}

// NewExecutor creates a new suspend executor
func NewExecutor(logger *logging.Logger, dryRun bool) *Executor {
	return &Executor{
		logger: logger,
	}
}

// CheckAndSuspend is the main entry point for suspend checking (stub for non-Linux)
func (e *Executor) CheckAndSuspend() error {
	return fmt.Errorf("suspend feature only supported on Linux")
}
