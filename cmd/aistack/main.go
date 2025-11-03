package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"aistack/internal/logging"
	"aistack/internal/tui"
)

const version = "0.1.0-dev"

func main() {
	// Initialize logger
	logger := logging.NewLogger(logging.LevelInfo)

	// Log app.started event
	startTime := time.Now()
	logger.Info("app.started", "Application started", map[string]interface{}{
		"version": version,
		"ts":      startTime.UTC().Format(time.RFC3339),
	})

	// Create and run the TUI
	p := tea.NewProgram(tui.NewModel())

	// Run the program and capture exit reason
	finalModel, err := p.Run()
	exitReason := "normal"

	if err != nil {
		exitReason = "error"
		logger.Error("app.error", "Application error", map[string]interface{}{
			"error": err.Error(),
		})
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}

	// Ensure we got our model type back
	_ = finalModel

	// Log app.exited event
	logger.Info("app.exited", "Application exited", map[string]interface{}{
		"ts":     time.Now().UTC().Format(time.RFC3339),
		"reason": exitReason,
	})
}
