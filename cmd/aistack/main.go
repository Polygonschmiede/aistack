package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"aistack/internal/agent"
	"aistack/internal/gpu"
	"aistack/internal/gpulock"
	"aistack/internal/logging"
	"aistack/internal/metrics"
	"aistack/internal/models"
	"aistack/internal/services"
	"aistack/internal/tui"
	"aistack/internal/wol"
	"aistack/internal/wol/relay"
)

const version = "0.1.0-dev"

func main() {
	if len(os.Args) <= 1 {
		runTUI()
		return
	}

	command := strings.ToLower(os.Args[1])
	if handler, ok := commandHandlers()[command]; ok {
		handler()
		return
	}

	fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
	printUsage()
	os.Exit(1)
}

func commandHandlers() map[string]func() {
	return map[string]func(){
		"agent":        runAgent,
		"idle-check":   runIdleCheck,
		"install":      runInstall,
		"start":        func() { runServiceCommand("start") },
		"stop":         func() { runServiceCommand("stop") },
		"status":       runStatus,
		"update":       func() { runServiceCommand("update") },
		"logs":         func() { runServiceCommand("logs") },
		"remove":       runRemove,
		"backend":      runBackendSwitch,
		"gpu-check":    runGPUCheck,
		"gpu-unlock":   runGPUUnlock,
		"metrics-test": runMetricsTest,
		"wol-check":    runWoLCheck,
		"wol-setup":    runWoLSetup,
		"wol-send":     runWoLSend,
		"wol-apply":    runWoLApply,
		"wol-relay":    runWoLRelay,
		"models":       runModels,
		"health":       runHealth,
		"repair":       func() { runServiceCommand("repair") },
		"version":      runVersion,
		"help":         printUsage,
		"--help":       printUsage,
		"-h":           printUsage,
	}
}

func runVersion() {
	fmt.Printf("aistack version %s\n", version)
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

	composeDir := resolveComposeDir()

	// Create and run the TUI
	p := tea.NewProgram(tui.NewModel(logger, composeDir))

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
	composeDir := resolveComposeDir()

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
	composeDir := resolveComposeDir()

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
	case "update":
		fmt.Printf("Updating service: %s\n", serviceName)
		fmt.Println("This will pull the latest image and restart the service.")
		fmt.Println("Health checks will be performed and rollback will occur on failure.")
		fmt.Println()
		if err := service.Update(); err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Update failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n✓ Service %s updated successfully\n", serviceName)
	case "logs":
		// Default to last 100 lines if no tail parameter specified
		tail := 100
		if len(os.Args) >= 4 {
			if _, err := fmt.Sscanf(os.Args[3], "%d", &tail); err != nil {
				fmt.Fprintf(os.Stderr, "Invalid tail count: %s\n", os.Args[3])
				os.Exit(1)
			}
		}
		fmt.Printf("=== Logs for %s (last %d lines) ===\n\n", serviceName, tail)
		logs, err := service.Logs(tail)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting logs: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(logs)
	case "repair":
		// Story T-026: Repair-Command für einzelne Services
		fmt.Printf("Repairing service: %s\n", serviceName)
		fmt.Println("This will stop, remove, and recreate the service.")
		fmt.Println("Volumes will be preserved. Health checks will validate the repair.")
		fmt.Println()

		result, err := manager.RepairService(serviceName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Repair failed: %v\n", err)
			os.Exit(1)
		}

		// Display result
		fmt.Println()
		fmt.Println("=== Repair Result ===")
		fmt.Printf("Service: %s\n", result.ServiceName)
		fmt.Printf("Health Before: %s\n", result.HealthBefore)
		fmt.Printf("Health After:  %s\n", result.HealthAfter)

		if result.SkippedReason != "" {
			fmt.Printf("\nℹ  Skipped: %s\n", result.SkippedReason)
		}

		if result.Success {
			fmt.Printf("\n✓ Service %s repaired successfully\n", serviceName)
		} else {
			fmt.Fprintf(os.Stderr, "\n❌ Repair completed but service is not healthy\n")
			if result.ErrorMessage != "" {
				fmt.Fprintf(os.Stderr, "   Error: %s\n", result.ErrorMessage)
			}
			os.Exit(1)
		}
	}
}

// runRemove removes a service (optionally purging data volumes)
// Story T-020: LocalAI Lifecycle Commands
func runRemove() {
	logger := logging.NewLogger(logging.LevelInfo)
	composeDir := resolveComposeDir()

	manager, err := services.NewManager(composeDir, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing service manager: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) < 3 {
		fmt.Println("Usage: aistack remove <service> [--purge]")
		fmt.Println()
		fmt.Println("Removes a service. Data volumes are kept by default.")
		fmt.Println("Use --purge to also remove data volumes.")
		os.Exit(1)
	}

	serviceName := os.Args[2]
	service, err := manager.GetService(serviceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Println("Valid services: ollama, openwebui, localai")
		os.Exit(1)
	}

	// Check for --purge flag
	purge := false
	if len(os.Args) >= 4 && os.Args[3] == "--purge" {
		purge = true
	}

	keepData := !purge

	if purge {
		fmt.Printf("Removing service %s and purging all data volumes...\n", serviceName)
		fmt.Println("⚠️  Warning: This will permanently delete all data!")
	} else {
		fmt.Printf("Removing service %s (keeping data volumes)...\n", serviceName)
	}

	if err := service.Remove(keepData); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Remove failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Service %s removed successfully\n", serviceName)
	if keepData {
		fmt.Println("  Data volumes were preserved. Use --purge to remove them.")
	} else {
		fmt.Println("  All data volumes were purged.")
	}
}

// runStatus displays status of all services
func runStatus() {
	logger := logging.NewLogger(logging.LevelInfo)
	composeDir := resolveComposeDir()

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

// runHealth generates a comprehensive health report
// Story T-025: Health-Reporter (Services + GPU Smoke)
func runHealth() {
	logger := logging.NewLogger(logging.LevelInfo)
	composeDir := resolveComposeDir()

	manager, err := services.NewManager(composeDir, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing service manager: %v\n", err)
		os.Exit(1)
	}

	// Create health reporter
	reporter := services.NewHealthReporter(manager, nil, logger)

	// Generate report
	fmt.Println("Generating health report...")
	report, err := reporter.GenerateReport()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating health report: %v\n", err)
		os.Exit(1)
	}

	// Display report
	fmt.Println()
	fmt.Println("=== Health Report ===")
	fmt.Printf("Timestamp: %s\n", report.Timestamp.Format(time.RFC3339))
	fmt.Println()

	fmt.Println("Services:")
	for _, service := range report.Services {
		icon := getHealthIcon(service.Health)
		fmt.Printf("  %s %-12s  Health: %s", icon, service.Name, service.Health)
		if service.Message != "" {
			fmt.Printf(" (%s)", service.Message)
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("GPU:")
	if report.GPU.OK {
		fmt.Printf("  ✓ Status: OK")
		if report.GPU.Message != "" {
			fmt.Printf(" (%s)", report.GPU.Message)
		}
		fmt.Println()
	} else {
		fmt.Printf("  ✗ Status: FAILED")
		if report.GPU.Message != "" {
			fmt.Printf(" (%s)", report.GPU.Message)
		}
		fmt.Println()
	}

	// Save report if requested
	if len(os.Args) > 2 && os.Args[2] == "--save" {
		reportPath := "/var/lib/aistack/health_report.json"
		if os.Getenv("AISTACK_STATE_DIR") != "" {
			reportPath = filepath.Join(os.Getenv("AISTACK_STATE_DIR"), "health_report.json")
		}

		if err := reporter.SaveReport(report, reportPath); err != nil {
			fmt.Fprintf(os.Stderr, "\nWarning: Failed to save report: %v\n", err)
		} else {
			fmt.Printf("\nReport saved to: %s\n", reportPath)
		}
	}

	// Exit with error if not all healthy
	if !report.GPU.OK {
		os.Exit(1)
	}
	for _, service := range report.Services {
		if service.Health != services.HealthGreen {
			os.Exit(1)
		}
	}
}

func getHealthIcon(health services.HealthStatus) string {
	switch health {
	case services.HealthGreen:
		return "✓"
	case services.HealthYellow:
		return "⚠"
	case services.HealthRed:
		return "✗"
	default:
		return "?"
	}
}

// runGPUCheck performs GPU detection and displays results
func runGPUCheck() {
	logger := logging.NewLogger(logging.LevelInfo)

	fmt.Println("Checking GPU and NVIDIA Stack...")
	fmt.Println()

	// Detect GPUs via NVML
	detector := gpu.NewDetector(logger)
	gpuReport := detector.DetectGPUs()

	// Display GPU report
	fmt.Println("=== GPU Detection Report ===")
	if !gpuReport.NVMLOk {
		fmt.Printf("❌ NVML Status: FAILED\n")
		fmt.Printf("   Error: %s\n", gpuReport.ErrorMessage)
		fmt.Println()
		fmt.Println("💡 Hint: Install NVIDIA drivers to enable GPU support")
		fmt.Println("   https://docs.nvidia.com/datacenter/tesla/tesla-installation-notes/")
	} else {
		fmt.Printf("✓ NVML Status: OK\n")
		fmt.Printf("  Driver Version: %s\n", gpuReport.DriverVersion)
		fmt.Printf("  CUDA Version: %d\n", gpuReport.CUDAVersion)
		fmt.Printf("  GPU Count: %d\n", len(gpuReport.GPUs))
		fmt.Println()

		for _, gpu := range gpuReport.GPUs {
			fmt.Printf("  GPU %d:\n", gpu.Index)
			fmt.Printf("    Name: %s\n", gpu.Name)
			fmt.Printf("    UUID: %s\n", gpu.UUID)
			fmt.Printf("    Memory: %d MB\n", gpu.MemoryMB)
		}
	}

	fmt.Println()

	// Detect Container Toolkit
	toolkitDetector := gpu.NewToolkitDetector(logger)
	toolkitReport := toolkitDetector.DetectContainerToolkit()

	fmt.Println("=== NVIDIA Container Toolkit ===")
	if !toolkitReport.DockerSupport {
		fmt.Printf("❌ Docker GPU Support: NOT AVAILABLE\n")
		if toolkitReport.ErrorMessage != "" {
			fmt.Printf("   Error: %s\n", toolkitReport.ErrorMessage)
		}
		fmt.Println()
		fmt.Println("💡 Hint: Install NVIDIA Container Toolkit to enable GPU in containers")
		fmt.Println("   https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html")
	} else {
		fmt.Printf("✓ Docker GPU Support: AVAILABLE\n")
		if toolkitReport.ToolkitVersion != "" {
			fmt.Printf("  Toolkit Version: %s\n", toolkitReport.ToolkitVersion)
		}
	}

	fmt.Println()

	// Save detailed report if requested
	if len(os.Args) > 2 && os.Args[2] == "--save" {
		reportPath := "/tmp/gpu_report.json"
		if err := detector.SaveReport(gpuReport, reportPath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save report: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Detailed report saved to: %s\n", reportPath)
	}
}

// runMetricsTest performs a test metrics collection
func runMetricsTest() {
	logger := logging.NewLogger(logging.LevelInfo)

	fmt.Println("Testing metrics collection...")
	fmt.Println()

	// Create collector with default config
	config := metrics.DefaultConfig()
	collector := metrics.NewCollector(config, logger)

	// Initialize
	if err := collector.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize metrics collector: %v\n", err)
		os.Exit(1)
	}
	defer collector.Shutdown()

	fmt.Println("=== Metrics Collection Test ===")
	fmt.Println("Collecting 3 samples with 5-second interval...")
	fmt.Println()

	// Collect 3 samples
	for i := 0; i < 3; i++ {
		sample, err := collector.CollectSample()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to collect sample: %v\n", err)
			continue
		}

		// Display sample
		fmt.Printf("Sample %d (at %s):\n", i+1, sample.Timestamp.Format(time.RFC3339))
		if sample.CPUUtil != nil {
			fmt.Printf("  CPU Utilization: %.2f%%\n", *sample.CPUUtil)
		}
		if sample.CPUWatts != nil {
			fmt.Printf("  CPU Power: %.2f W\n", *sample.CPUWatts)
		}
		if sample.GPUUtil != nil {
			fmt.Printf("  GPU Utilization: %.2f%%\n", *sample.GPUUtil)
		}
		if sample.GPUMemMB != nil {
			fmt.Printf("  GPU Memory: %d MB\n", *sample.GPUMemMB)
		}
		if sample.GPUWatts != nil {
			fmt.Printf("  GPU Power: %.2f W\n", *sample.GPUWatts)
		}
		if sample.TempGPU != nil {
			fmt.Printf("  GPU Temperature: %.1f°C\n", *sample.TempGPU)
		}
		if sample.EstTotalW != nil {
			fmt.Printf("  Estimated Total Power: %.2f W\n", *sample.EstTotalW)
		}
		fmt.Println()

		// Write to temp file
		tmpFile := "/tmp/aistack_metrics_test.jsonl"
		if err := collector.WriteSample(sample, tmpFile); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to write sample: %v\n", err)
		}

		// Wait before next sample (except for last one)
		if i < 2 {
			time.Sleep(5 * time.Second)
		}
	}

	fmt.Println("✓ Metrics test completed")
	fmt.Println("Sample data written to: /tmp/aistack_metrics_test.jsonl")
}

// runGPUUnlock forcibly removes the GPU lock
// Story T-021: GPU-Mutex (Dateisperre + Lease)
func runGPUUnlock() {
	logger := logging.NewLogger(logging.LevelInfo)

	// Get state directory from env or use default
	stateDir := os.Getenv("AISTACK_STATE_DIR")
	if stateDir == "" {
		stateDir = "/var/lib/aistack"
	}

	manager := gpulock.NewManager(stateDir, logger)

	// Check current lock status
	status, err := manager.GetStatus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to get GPU lock status: %v\n", err)
		os.Exit(1)
	}

	if status.Holder == gpulock.HolderNone {
		fmt.Println("GPU is not locked.")
		return
	}

	// Display lock information
	fmt.Printf("Current GPU lock holder: %s\n", status.Holder)
	fmt.Printf("Lock acquired: %s\n", status.SinceTS.Format(time.RFC3339))
	fmt.Printf("Age: %s\n", time.Since(status.SinceTS).Round(time.Second))
	fmt.Println()

	// Warn user
	fmt.Println("⚠️  Warning: Force unlocking the GPU may cause issues if the service is still using it.")
	fmt.Println()
	fmt.Print("Are you sure you want to force unlock? (yes/no): ")

	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to read response: %v\n", err)
		os.Exit(1)
	}

	if strings.ToLower(response) != "yes" {
		fmt.Println("Aborted.")
		return
	}

	// Force unlock
	if err := manager.ForceUnlock(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to force unlock GPU: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ GPU lock forcibly removed")
	fmt.Println()
	fmt.Println("You can now start another GPU-intensive service.")
}

// runWoLCheck checks Wake-on-LAN status for network interfaces
func runWoLCheck() {
	logger := logging.NewLogger(logging.LevelInfo)

	fmt.Println("Checking Wake-on-LAN configuration...")
	fmt.Println()

	detector := wol.NewDetector(logger)

	// Try to get default interface
	iface, err := detector.GetDefaultInterface()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to detect default network interface: %v\n", err)
		fmt.Println()
		fmt.Println("💡 Hint: Specify interface manually with 'aistack wol-setup <interface>'")
		os.Exit(1)
	}

	fmt.Printf("Checking interface: %s\n", iface)
	fmt.Println()

	// Detect WoL status
	status := detector.DetectWoL(iface)

	// Display results
	fmt.Println("=== Wake-on-LAN Status ===")
	fmt.Printf("Interface: %s\n", status.Interface)
	fmt.Printf("MAC Address: %s\n", status.MAC)
	fmt.Println()

	if status.ErrorMessage != "" {
		fmt.Printf("❌ Error: %s\n", status.ErrorMessage)
		fmt.Println()
		if !status.Supported {
			fmt.Println("💡 Hint: Wake-on-LAN may not be supported by your hardware/driver")
			fmt.Println("   or ethtool may not be installed (apt-get install ethtool)")
		}
		os.Exit(1)
	}

	if status.Supported {
		fmt.Printf("✓ WoL Supported: Yes\n")
		fmt.Printf("  Available modes: %v\n", status.WoLModes)
	} else {
		fmt.Printf("❌ WoL Supported: No\n")
	}

	fmt.Printf("  Current mode: %s\n", status.CurrentMode)

	if status.Enabled {
		fmt.Printf("✓ WoL Status: ENABLED\n")
		fmt.Println()
		fmt.Println("Your system can be woken via Wake-on-LAN magic packets.")
		fmt.Printf("To send a test packet: aistack wol-send %s\n", status.MAC)
	} else {
		fmt.Printf("❌ WoL Status: DISABLED\n")
		fmt.Println()
		fmt.Printf("To enable Wake-on-LAN: sudo aistack wol-setup %s\n", iface)
	}

	fmt.Println()
	fmt.Println("⚠️  Note: BIOS/UEFI WoL settings are outside the scope of this tool.")
	fmt.Println("   Ensure 'Wake on LAN' is enabled in your system BIOS/UEFI.")

	if cfg, err := wol.LoadConfig(); err == nil {
		fmt.Println()
		fmt.Printf("Persisted config: %s\n", wol.ConfigPath())
		fmt.Printf("  Stored interface: %s (mode: %s)\n", cfg.Interface, cfg.WoLState)
	} else if !errors.Is(err, os.ErrNotExist) {
		logger.Warn("wol.config.read_failed", "Failed to read persisted WoL config", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// runWoLSetup enables Wake-on-LAN on a specified interface
func runWoLSetup() {
	logger := logging.NewLogger(logging.LevelInfo)

	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: aistack wol-setup <interface>\n")
		fmt.Fprintf(os.Stderr, "Example: aistack wol-setup eth0\n")
		os.Exit(1)
	}

	iface := os.Args[2]

	fmt.Printf("Setting up Wake-on-LAN on interface: %s\n", iface)
	fmt.Println()

	detector := wol.NewDetector(logger)

	// Check current status
	status := detector.DetectWoL(iface)
	if status.ErrorMessage != "" {
		fmt.Fprintf(os.Stderr, "❌ Error: %s\n", status.ErrorMessage)
		os.Exit(1)
	}

	if !status.Supported {
		fmt.Fprintf(os.Stderr, "❌ Wake-on-LAN is not supported on interface %s\n", iface)
		os.Exit(1)
	}

	if status.Enabled {
		fmt.Printf("✓ Wake-on-LAN is already enabled on %s\n", iface)
		fmt.Printf("  Current mode: %s\n", status.CurrentMode)
		fmt.Printf("  MAC Address: %s\n", status.MAC)

		broadcast, bErr := wol.GetBroadcastAddr(iface)
		if bErr != nil {
			logger.Warn("wol.broadcast.lookup_failed", "Failed to resolve broadcast address", map[string]interface{}{
				"interface": iface,
				"error":     bErr.Error(),
			})
		}

		cfg := wol.WoLConfig{
			Interface:   iface,
			MAC:         status.MAC,
			WoLState:    status.CurrentMode,
			BroadcastIP: broadcast,
		}

		if err := wol.SaveConfig(cfg); err != nil {
			logger.Warn("wol.config.save_failed", "Failed to persist WoL config", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			fmt.Printf("  Persisted config: %s\n", wol.ConfigPath())
		}

		return
	}

	// Enable WoL
	fmt.Printf("Enabling Wake-on-LAN (requires root privileges)...\n")
	if err := detector.EnableWoL(iface); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to enable WoL: %v\n", err)
		fmt.Println()
		fmt.Println("💡 Hint: This command requires root privileges")
		fmt.Printf("   Try: sudo aistack wol-setup %s\n", iface)
		os.Exit(1)
	}

	// Verify
	status = detector.DetectWoL(iface)
	if status.Enabled {
		fmt.Printf("✓ Wake-on-LAN successfully enabled on %s\n", iface)
		fmt.Printf("  Mode: %s (magic packet)\n", status.CurrentMode)
		fmt.Printf("  MAC Address: %s\n", status.MAC)

		broadcast, bErr := wol.GetBroadcastAddr(iface)
		if bErr != nil {
			logger.Warn("wol.broadcast.lookup_failed", "Failed to resolve broadcast address", map[string]interface{}{
				"interface": iface,
				"error":     bErr.Error(),
			})
		}

		cfg := wol.WoLConfig{
			Interface:   iface,
			MAC:         status.MAC,
			WoLState:    status.CurrentMode,
			BroadcastIP: broadcast,
		}

		if err := wol.SaveConfig(cfg); err != nil {
			logger.Warn("wol.config.save_failed", "Failed to persist WoL config", map[string]interface{}{
				"error": err.Error(),
			})
			fmt.Println()
			fmt.Println("⚠️  WoL enabled but config persistence failed — see logs for details")
		} else {
			fmt.Println()
			fmt.Printf("Config persisted to: %s\n", wol.ConfigPath())
			fmt.Println("A udev rule will reapply this setting on interface events.")
		}
	} else {
		fmt.Fprintf(os.Stderr, "❌ WoL enable command succeeded but verification failed\n")
		os.Exit(1)
	}
}

// runWoLSend sends a Wake-on-LAN magic packet
func runWoLSend() {
	logger := logging.NewLogger(logging.LevelInfo)

	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: aistack wol-send <mac-address> [broadcast-ip]\n")
		fmt.Fprintf(os.Stderr, "Example: aistack wol-send AA:BB:CC:DD:EE:FF\n")
		fmt.Fprintf(os.Stderr, "         aistack wol-send AA:BB:CC:DD:EE:FF 192.168.1.255\n")
		os.Exit(1)
	}

	targetMAC := os.Args[2]
	broadcastIP := ""

	if len(os.Args) > 3 {
		broadcastIP = os.Args[3]
	}

	// Validate MAC address
	if err := wol.ValidateMAC(targetMAC); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Invalid MAC address: %v\n", err)
		os.Exit(1)
	}

	normalized, err := wol.NormalizeMAC(targetMAC)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to normalize MAC address: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Sending Wake-on-LAN magic packet...\n")
	fmt.Printf("  Target MAC: %s\n", normalized)

	if broadcastIP != "" {
		fmt.Printf("  Broadcast IP: %s\n", broadcastIP)
	} else {
		fmt.Printf("  Broadcast IP: 255.255.255.255 (default)\n")
	}

	fmt.Println()

	sender := wol.NewSender(logger)
	if err := sender.SendMagicPacket(targetMAC, broadcastIP); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to send magic packet: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Magic packet sent successfully\n")
	fmt.Println()
	fmt.Println("The target system should wake up if:")
	fmt.Println("  1. Wake-on-LAN is enabled in BIOS/UEFI")
	fmt.Println("  2. Wake-on-LAN is enabled in the OS (ethtool)")
	fmt.Println("  3. The system is connected to power")
	fmt.Println("  4. The network switch supports broadcast packets")
}

// runWoLApply reapplies persisted WoL configuration (used by udev/systemd)
func runWoLApply() {
	logger := logging.NewLogger(logging.LevelInfo)

	cfg, err := wol.LoadConfig()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Info("wol.apply.config_missing", "No persisted WoL config found", nil)
			return
		}
		fmt.Fprintf(os.Stderr, "❌ Failed to load WoL config: %v\n", err)
		os.Exit(1)
	}

	var targetIface string
	if len(os.Args) > 2 {
		if os.Args[2] == "--interface" && len(os.Args) > 3 {
			targetIface = os.Args[3]
		} else {
			targetIface = os.Args[2]
		}
	}

	if targetIface != "" && targetIface != cfg.Interface {
		logger.Info("wol.apply.skip_interface", "Configured interface does not match trigger", map[string]interface{}{
			"configured": cfg.Interface,
			"requested":  targetIface,
		})
		return
	}

	detector := wol.NewDetector(logger)
	if err := detector.ApplyConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to apply WoL config: %v\n", err)
		os.Exit(1)
	}

	logger.Info("wol.apply.success", "WoL configuration applied", map[string]interface{}{
		"interface": cfg.Interface,
		"mode":      cfg.WoLState,
	})
}

func runWoLRelay() {
	fs := flag.NewFlagSet("wol-relay", flag.ExitOnError)
	listen := fs.String("listen", ":8081", "host:port to listen on")
	key := fs.String("key", "", "shared secret key (or set AISTACK_WOL_RELAY_KEY)")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to parse flags: %v\n", err)
		os.Exit(1)
	}

	relayKey := *key
	if relayKey == "" {
		relayKey = os.Getenv("AISTACK_WOL_RELAY_KEY")
	}

	if relayKey == "" {
		fmt.Fprintf(os.Stderr, "Usage: aistack wol-relay [--listen :8081] --key <shared-secret>\n")
		os.Exit(1)
	}

	logger := logging.NewLogger(logging.LevelInfo)
	server := relay.NewServer(*listen, relayKey, logger)

	if err := server.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Wol relay stopped: %v\n", err)
		os.Exit(1)
	}
}

// runBackendSwitch handles backend switching for Open WebUI
// Story T-019: Backend-Switch (Ollama ↔ LocalAI)
func runBackendSwitch() {
	logger := logging.NewLogger(logging.LevelInfo)
	composeDir := resolveComposeDir()

	// Parse command line arguments
	if len(os.Args) < 3 {
		fmt.Println("Usage: aistack backend <ollama|localai>")
		fmt.Println()
		fmt.Println("Switches the Open WebUI backend between Ollama and LocalAI.")
		fmt.Println("The service will be restarted to apply the change.")
		os.Exit(1)
	}

	backendArg := os.Args[2]
	var backend services.BackendType

	switch backendArg {
	case "ollama":
		backend = services.BackendOllama
	case "localai":
		backend = services.BackendLocalAI
	default:
		fmt.Fprintf(os.Stderr, "❌ Invalid backend: %s\n", backendArg)
		fmt.Println("Valid backends: ollama, localai")
		os.Exit(1)
	}

	// Create manager and get OpenWebUI service
	manager, err := services.NewManager(composeDir, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing service manager: %v\n", err)
		os.Exit(1)
	}

	service, err := manager.GetService("openwebui")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting Open WebUI service: %v\n", err)
		os.Exit(1)
	}

	// Cast to OpenWebUIService
	openwebuiService, ok := service.(*services.OpenWebUIService)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: service is not an OpenWebUIService\n")
		os.Exit(1)
	}

	// Get current backend
	currentBackend, err := openwebuiService.GetCurrentBackend()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current backend: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Current backend: %s\n", currentBackend)

	if currentBackend == backend {
		fmt.Printf("✓ Backend already set to %s (no change needed)\n", backend)
		return
	}

	fmt.Printf("Switching backend from %s to %s...\n", currentBackend, backend)
	fmt.Println("This will restart the Open WebUI service.")
	fmt.Println()

	// Switch backend
	if err := openwebuiService.SwitchBackend(backend); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Backend switch failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Backend switched to %s successfully\n", backend)
	fmt.Println()
	fmt.Println("Open WebUI is now connected to the new backend.")
	fmt.Println("Access it at: http://localhost:3000")
}

// runModels handles model management commands
// Story T-022, T-023: Model management & caching
func runModels() {
	if len(os.Args) < 3 {
		printModelsUsage()
		os.Exit(1)
	}

	subcommand := strings.ToLower(os.Args[2])

	switch subcommand {
	case "list":
		runModelsList()
	case "download":
		runModelsDownload()
	case "delete":
		runModelsDelete()
	case "stats":
		runModelsStats()
	case "evict-oldest":
		runModelsEvictOldest()
	default:
		fmt.Fprintf(os.Stderr, "Unknown models subcommand: %s\n\n", subcommand)
		printModelsUsage()
		os.Exit(1)
	}
}

// printModelsUsage displays model management usage
func printModelsUsage() {
	fmt.Println("Model Management Commands:")
	fmt.Println()
	fmt.Println("  aistack models list <provider>           List all models for provider (ollama, localai)")
	fmt.Println("  aistack models download <provider> <name> Download a model")
	fmt.Println("  aistack models delete <provider> <name>   Delete a model")
	fmt.Println("  aistack models stats <provider>           Show cache statistics")
	fmt.Println("  aistack models evict-oldest <provider>    Remove oldest model to free space")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  aistack models list ollama")
	fmt.Println("  aistack models download ollama qwen2:7b-instruct-q4")
	fmt.Println("  aistack models stats ollama")
	fmt.Println("  aistack models evict-oldest localai")
}

// runModelsList lists all models for a provider
func runModelsList() {
	logger := logging.NewLogger(logging.LevelInfo)

	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "Usage: aistack models list <provider>\n")
		fmt.Fprintf(os.Stderr, "Example: aistack models list ollama\n")
		os.Exit(1)
	}

	providerStr := strings.ToLower(os.Args[3])
	provider := models.Provider(providerStr)

	if !provider.IsValid() {
		fmt.Fprintf(os.Stderr, "❌ Invalid provider: %s (must be ollama or localai)\n", providerStr)
		os.Exit(1)
	}

	stateDir := getStateDir()

	var modelsList []models.ModelInfo
	var err error

	switch provider {
	case models.ProviderOllama:
		manager := models.NewOllamaManager(stateDir, logger)
		if err := manager.SyncState(); err != nil {
			logger.Warn("models.list.sync_failed", "Failed to sync state", map[string]interface{}{
				"error": err.Error(),
			})
		}
		modelsList, err = manager.List()
	case models.ProviderLocalAI:
		// LocalAI models directory is typically in the volume
		modelsPath := "/var/lib/aistack/volumes/localai_models"
		manager := models.NewLocalAIManager(stateDir, modelsPath, logger)
		if err := manager.SyncState(); err != nil {
			logger.Warn("models.list.sync_failed", "Failed to sync state", map[string]interface{}{
				"error": err.Error(),
			})
		}
		modelsList, err = manager.List()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to list models: %v\n", err)
		os.Exit(1)
	}

	if len(modelsList) == 0 {
		fmt.Printf("No models found for %s\n", provider)
		return
	}

	fmt.Printf("Models for %s (%d total):\n\n", provider, len(modelsList))
	fmt.Printf("%-40s %12s %20s\n", "NAME", "SIZE", "LAST USED")
	fmt.Println(strings.Repeat("-", 75))

	for _, model := range modelsList {
		sizeStr := formatBytes(model.Size)
		lastUsedStr := model.LastUsed.Format("2006-01-02 15:04")
		fmt.Printf("%-40s %12s %20s\n", model.Name, sizeStr, lastUsedStr)
	}
}

// runModelsDownload downloads a model
func runModelsDownload() {
	logger := logging.NewLogger(logging.LevelInfo)

	if len(os.Args) < 5 {
		fmt.Fprintf(os.Stderr, "Usage: aistack models download <provider> <model-name>\n")
		fmt.Fprintf(os.Stderr, "Example: aistack models download ollama qwen2:7b-instruct-q4\n")
		os.Exit(1)
	}

	providerStr := strings.ToLower(os.Args[3])
	provider := models.Provider(providerStr)
	modelName := os.Args[4]

	if !provider.IsValid() {
		fmt.Fprintf(os.Stderr, "❌ Invalid provider: %s (must be ollama or localai)\n", providerStr)
		os.Exit(1)
	}

	if provider != models.ProviderOllama {
		fmt.Fprintf(os.Stderr, "❌ Model download is currently only supported for Ollama\n")
		fmt.Fprintf(os.Stderr, "   LocalAI models must be manually placed in the models directory\n")
		os.Exit(1)
	}

	stateDir := getStateDir()
	manager := models.NewOllamaManager(stateDir, logger)

	fmt.Printf("Downloading model: %s\n", modelName)
	fmt.Printf("Provider: %s\n\n", provider)

	// Create progress channel
	progressChan := make(chan models.DownloadProgress, 10)
	done := make(chan error, 1)

	// Start download in goroutine
	go func() {
		done <- manager.Download(modelName, progressChan)
	}()

	// Display progress
	lastPercentage := -1.0
	for {
		select {
		case progress := <-progressChan:
			switch progress.Status {
			case "started":
				fmt.Println("Download started...")
			case "progress":
				if progress.Percentage != lastPercentage {
					if progress.TotalBytes > 0 {
						fmt.Printf("\rProgress: %.1f%% (%s / %s)",
							progress.Percentage,
							formatBytes(progress.BytesDownloaded),
							formatBytes(progress.TotalBytes))
					} else {
						fmt.Printf("\rDownloaded: %s", formatBytes(progress.BytesDownloaded))
					}
					lastPercentage = progress.Percentage
				}
			case "completed":
				fmt.Printf("\r✓ Download completed successfully\n")
			case "failed":
				fmt.Printf("\n❌ Download failed: %s\n", progress.Error)
			}
		case err := <-done:
			if err != nil {
				fmt.Fprintf(os.Stderr, "\n❌ Download failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println()
			fmt.Printf("Model %s is now available for use\n", modelName)
			return
		}
	}
}

// runModelsDelete deletes a model
func runModelsDelete() {
	logger := logging.NewLogger(logging.LevelInfo)

	if len(os.Args) < 5 {
		fmt.Fprintf(os.Stderr, "Usage: aistack models delete <provider> <model-name>\n")
		fmt.Fprintf(os.Stderr, "Example: aistack models delete ollama qwen2:7b-instruct-q4\n")
		os.Exit(1)
	}

	providerStr := strings.ToLower(os.Args[3])
	provider := models.Provider(providerStr)
	modelName := os.Args[4]

	if !provider.IsValid() {
		fmt.Fprintf(os.Stderr, "❌ Invalid provider: %s (must be ollama or localai)\n", providerStr)
		os.Exit(1)
	}

	stateDir := getStateDir()

	fmt.Printf("⚠️  Warning: This will permanently delete model: %s\n", modelName)
	fmt.Printf("Provider: %s\n", provider)
	fmt.Print("Are you sure? (yes/no): ")

	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "yes" {
		fmt.Println("Aborted.")
		return
	}

	var err error
	switch provider {
	case models.ProviderOllama:
		manager := models.NewOllamaManager(stateDir, logger)
		err = manager.Delete(modelName)
	case models.ProviderLocalAI:
		modelsPath := "/var/lib/aistack/volumes/localai_models"
		manager := models.NewLocalAIManager(stateDir, modelsPath, logger)
		err = manager.Delete(modelName)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to delete model: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Model %s deleted successfully\n", modelName)
}

// runModelsStats shows cache statistics
func runModelsStats() {
	logger := logging.NewLogger(logging.LevelInfo)

	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "Usage: aistack models stats <provider>\n")
		fmt.Fprintf(os.Stderr, "Example: aistack models stats ollama\n")
		os.Exit(1)
	}

	providerStr := strings.ToLower(os.Args[3])
	provider := models.Provider(providerStr)

	if !provider.IsValid() {
		fmt.Fprintf(os.Stderr, "❌ Invalid provider: %s (must be ollama or localai)\n", providerStr)
		os.Exit(1)
	}

	stateDir := getStateDir()

	var stats *models.CacheStats
	var err error

	switch provider {
	case models.ProviderOllama:
		manager := models.NewOllamaManager(stateDir, logger)
		stats, err = manager.GetStats()
	case models.ProviderLocalAI:
		modelsPath := "/var/lib/aistack/volumes/localai_models"
		manager := models.NewLocalAIManager(stateDir, modelsPath, logger)
		stats, err = manager.GetStats()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to get stats: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Cache Statistics for %s:\n\n", provider)
	fmt.Printf("  Total Models: %d\n", stats.ModelCount)
	fmt.Printf("  Total Size:   %s\n", formatBytes(stats.TotalSize))

	if stats.OldestModel != nil {
		fmt.Printf("\nOldest Model:\n")
		fmt.Printf("  Name:       %s\n", stats.OldestModel.Name)
		fmt.Printf("  Size:       %s\n", formatBytes(stats.OldestModel.Size))
		fmt.Printf("  Last Used:  %s\n", stats.OldestModel.LastUsed.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Age:        %s\n", time.Since(stats.OldestModel.LastUsed).Round(time.Second))
	}
}

// runModelsEvictOldest evicts the oldest model
func runModelsEvictOldest() {
	logger := logging.NewLogger(logging.LevelInfo)

	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "Usage: aistack models evict-oldest <provider>\n")
		fmt.Fprintf(os.Stderr, "Example: aistack models evict-oldest ollama\n")
		os.Exit(1)
	}

	providerStr := strings.ToLower(os.Args[3])
	provider := models.Provider(providerStr)

	if !provider.IsValid() {
		fmt.Fprintf(os.Stderr, "❌ Invalid provider: %s (must be ollama or localai)\n", providerStr)
		os.Exit(1)
	}

	stateDir := getStateDir()

	var evicted *models.ModelInfo
	var err error

	switch provider {
	case models.ProviderOllama:
		manager := models.NewOllamaManager(stateDir, logger)
		evicted, err = manager.EvictOldest()
	case models.ProviderLocalAI:
		modelsPath := "/var/lib/aistack/volumes/localai_models"
		manager := models.NewLocalAIManager(stateDir, modelsPath, logger)
		evicted, err = manager.EvictOldest()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to evict oldest model: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Evicted oldest model: %s\n", evicted.Name)
	fmt.Printf("  Size freed:  %s\n", formatBytes(evicted.Size))
	fmt.Printf("  Last used:   %s (%s ago)\n",
		evicted.LastUsed.Format("2006-01-02 15:04"),
		time.Since(evicted.LastUsed).Round(time.Second))
}

// formatBytes formats bytes to human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// getStateDir returns the state directory
func getStateDir() string {
	stateDir := os.Getenv("AISTACK_STATE_DIR")
	if stateDir == "" {
		stateDir = "/var/lib/aistack"
	}
	return stateDir
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
  aistack update <service>         Update a service to latest version (with rollback)
  aistack logs <service> [lines]   Show service logs (default: 100 lines)
  aistack remove <service> [--purge] Remove a service (keeps data by default)
  aistack backend <ollama|localai> Switch Open WebUI backend (restarts service)
  aistack status                   Show status of all services
  aistack health [--save]          Generate comprehensive health report (services + GPU)
  aistack repair <service>         Repair a service (stop → remove → recreate with health check)
  aistack gpu-check [--save]       Check GPU and NVIDIA stack availability
  aistack gpu-unlock               Force unlock GPU mutex (recovery)
  aistack metrics-test             Test metrics collection (CPU/GPU)
  aistack wol-check                Check Wake-on-LAN status
  aistack wol-setup <interface>    Enable Wake-on-LAN on interface (requires root)
  aistack wol-send <mac> [ip]      Send Wake-on-LAN magic packet
  aistack wol-apply [interface]    Reapply persisted WoL configuration (for udev/systemd)
  aistack wol-relay [flags]        Start HTTP→WoL relay (use --key or AISTACK_WOL_RELAY_KEY)
  aistack models <subcommand>      Model management (list, download, delete, stats, evict-oldest)
  aistack version                  Print version information
  aistack help                     Show this help message

Model Management:
  aistack models list <provider>           List all models (ollama, localai)
  aistack models download <provider> <name> Download a model (ollama only)
  aistack models delete <provider> <name>   Delete a model
  aistack models stats <provider>           Show cache statistics
  aistack models evict-oldest <provider>    Remove oldest model to free space

For more information, visit: https://github.com/polygonschmiede/aistack
`, version)
}

func resolveComposeDir() string {
	if envDir := os.Getenv("AISTACK_COMPOSE_DIR"); envDir != "" {
		if abs, err := filepath.Abs(envDir); err == nil {
			if dirExists(abs) {
				return abs
			}
		}
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates := []string{
			filepath.Join(exeDir, "compose"),
			filepath.Join(exeDir, "..", "share", "aistack", "compose"),
		}

		for _, candidate := range candidates {
			if abs, err := filepath.Abs(candidate); err == nil && dirExists(abs) {
				return abs
			}
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		legacy := filepath.Join(cwd, "compose")
		if dirExists(legacy) {
			return legacy
		}
	}

	// Fallback to relative path; downstream code will surface a detailed error
	return "./compose"
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
