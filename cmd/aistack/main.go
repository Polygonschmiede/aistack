package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"aistack/internal/agent"
	"aistack/internal/gpu"
	"aistack/internal/logging"
	"aistack/internal/metrics"
	"aistack/internal/services"
	"aistack/internal/tui"
	"aistack/internal/wol"
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
		case "gpu-check":
			runGPUCheck()
			return
		case "metrics-test":
			runMetricsTest()
			return
		case "wol-check":
			runWoLCheck()
			return
		case "wol-setup":
			runWoLSetup()
			return
		case "wol-send":
			runWoLSend()
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
		fmt.Println()
		fmt.Println("⚠️  Note: This setting may not persist across reboots.")
		fmt.Println("   Consider adding a udev rule or systemd service to make it permanent.")
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

	normalized, _ := wol.NormalizeMAC(targetMAC)

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
  aistack gpu-check [--save]       Check GPU and NVIDIA stack availability
  aistack metrics-test             Test metrics collection (CPU/GPU)
  aistack wol-check                Check Wake-on-LAN status
  aistack wol-setup <interface>    Enable Wake-on-LAN on interface (requires root)
  aistack wol-send <mac> [ip]      Send Wake-on-LAN magic packet
  aistack version                  Print version information
  aistack help                     Show this help message

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
