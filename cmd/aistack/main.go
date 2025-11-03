package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"aistack/internal/agent"
	"aistack/internal/logging"
	"aistack/internal/services"
	"aistack/internal/tui"
)

const version = "0.1.0-dev"

func main() {
	// Parse command line arguments
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "agent":
			runAgent()
			return
		case "idle-check":
			runIdleCheck()
			return
		case "install":
			runInstall()
			return
		case "start":
			runServiceCommand("start")
			return
		case "stop":
			runServiceCommand("stop")
			return
		case "status":
			runStatus()
			return
		case "version":
			fmt.Printf("aistack version %s\n", version)
			return
		case "help", "--help", "-h":
			printUsage()
			return
		}
	}

	// Default: run TUI
	runTUI()
}

// runTUI starts the interactive TUI mode
func runTUI() {
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

// runAgent starts the background agent service (for systemd)
func runAgent() {
	// Setup logger for agent mode (structured JSON to journald)
	logger := logging.NewLogger(logging.LevelInfo)

	logger.Info("agent.mode", "Starting in agent mode", map[string]interface{}{
		"version": version,
	})

	// Create and run agent
	agentInstance := agent.NewAgent(logger)

	if err := agentInstance.Run(); err != nil {
		logger.Error("agent.error", "Agent error", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}
}

// runIdleCheck performs a single idle check (for timer-triggered runs)
func runIdleCheck() {
	logger := logging.NewLogger(logging.LevelInfo)

	if err := agent.IdleCheck(logger); err != nil {
		logger.Error("idle.error", "Idle check error", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}
}

// runInstall installs services based on profile or individual service
func runInstall() {
	logger := logging.NewLogger(logging.LevelInfo)

	// Get compose directory (relative to binary or default)
	composeDir := "./compose"

	manager, err := services.NewManager(composeDir, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing service manager: %v\n", err)
		os.Exit(1)
	}

	// Check for --profile flag
	if len(os.Args) > 2 {
		if os.Args[2] == "--profile" && len(os.Args) > 3 {
			profile := os.Args[3]
			fmt.Printf("Installing profile: %s\n", profile)
			if err := manager.InstallProfile(profile); err != nil {
				fmt.Fprintf(os.Stderr, "Error installing profile: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Profile %s installed successfully\n", profile)
			return
		}

		// Install specific service
		serviceName := os.Args[2]
		service, err := manager.GetService(serviceName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Installing service: %s\n", serviceName)
		if err := service.Install(); err != nil {
			fmt.Fprintf(os.Stderr, "Error installing service: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Service %s installed successfully\n", serviceName)
		return
	}

	fmt.Fprintf(os.Stderr, "Usage: aistack install [--profile <profile>|<service>]\n")
	os.Exit(1)
}

// runServiceCommand runs start/stop commands on services
func runServiceCommand(command string) {
	logger := logging.NewLogger(logging.LevelInfo)
	composeDir := "./compose"

	manager, err := services.NewManager(composeDir, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing service manager: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: aistack %s <service>\n", command)
		os.Exit(1)
	}

	serviceName := os.Args[2]
	service, err := manager.GetService(serviceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "start":
		fmt.Printf("Starting service: %s\n", serviceName)
		if err := service.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting service: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Service %s started successfully\n", serviceName)
	case "stop":
		fmt.Printf("Stopping service: %s\n", serviceName)
		if err := service.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "Error stopping service: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Service %s stopped successfully\n", serviceName)
	}
}

// runStatus displays status of all services
func runStatus() {
	logger := logging.NewLogger(logging.LevelInfo)
	composeDir := "./compose"

	manager, err := services.NewManager(composeDir, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing service manager: %v\n", err)
		os.Exit(1)
	}

	statuses, err := manager.StatusAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting status: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Service Status:")
	fmt.Println("---------------")
	for _, status := range statuses {
		fmt.Printf("%-12s  State: %-10s  Health: %s\n",
			status.Name, status.State, status.Health)
	}
}

// printUsage displays usage information
func printUsage() {
	fmt.Printf(`aistack - AI Stack Management Tool (version %s)

Usage:
  aistack                          Start the interactive TUI (default)
  aistack agent                    Run as background agent service
  aistack idle-check               Perform idle evaluation (timer-triggered)
  aistack install --profile <name> Install services from profile (standard-gpu, minimal)
  aistack install <service>        Install a specific service (ollama, openwebui, localai)
  aistack start <service>          Start a service
  aistack stop <service>           Stop a service
  aistack status                   Show status of all services
  aistack version                  Print version information
  aistack help                     Show this help message

For more information, visit: https://github.com/polygonschmiede/aistack
`, version)
}

// parseFlags is a placeholder for future flag parsing
func parseFlags() {
	flag.Parse()
}
