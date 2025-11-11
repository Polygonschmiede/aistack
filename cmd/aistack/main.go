package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"aistack/internal/config"
	"aistack/internal/diag"
	"aistack/internal/gpu"
	"aistack/internal/gpulock"
	"aistack/internal/logging"
	"aistack/internal/models"
	"aistack/internal/services"
	"aistack/internal/suspend"
	"aistack/internal/tui"
)

const (
	version           = "0.1.0-dev"
	localAIModelsPath = "/var/lib/aistack/volumes/localai_models"
	confirmationYes   = "yes"
)

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
		"install":    runInstall,
		"start":      func() { runServiceCommand("start") },
		"stop":       func() { runServiceCommand("stop") },
		"status":     runStatus,
		"update":     func() { runServiceCommand("update") },
		"update-all": runUpdateAll,
		"logs":       func() { runServiceCommand("logs") },
		"remove":     runRemove,
		"uninstall":  runRemove, // Alias for remove
		"purge":      runPurge,
		"backend":    runBackendSwitch,
		"config":     runConfig,
		"gpu-check":  runGPUCheck,
		"gpu-unlock": runGPUUnlock,
		"models":     runModels,
		"health":     runHealth,
		"repair":     func() { runServiceCommand("repair") },
		"diag":       runDiag,
		"versions":   runVersions,
		"version":    runVersion,
		"suspend":    runSuspend,
		"help":       printUsage,
		"--help":     printUsage,
		"-h":         printUsage,
	}
}

func runVersion() {
	fmt.Printf("aistack version %s\n", version)
}

// runVersions displays version lock status and update policy (Story T-035)
func runVersions() {
	fmt.Println("=== Version Lock & Update Policy ===")
	fmt.Println()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load configuration: %v\n", err)
		fmt.Println("Update Mode: unknown (using default: rolling)")
	} else {
		fmt.Printf("Update Mode: %s\n", cfg.Updates.Mode)
		if cfg.Updates.Mode == "pinned" {
			fmt.Println("  ⚠ Updates are DISABLED (change to 'rolling' to allow updates)")
		} else {
			fmt.Println("  ✓ Updates are ALLOWED")
		}
	}
	fmt.Println()

	// Display version lock status
	fmt.Println("Version Lock Status:")

	// Try to locate versions.lock file
	lockPath := locateVersionsLockFile()
	if lockPath == "" {
		fmt.Println("  Status: NOT FOUND")
		fmt.Println("  All services will use latest stable tags (rolling updates)")
	} else {
		fmt.Printf("  Status: ACTIVE\n")
		fmt.Printf("  Location: %s\n", lockPath)
		fmt.Println()
		fmt.Println("  Locked Services:")

		// Read and display lock file contents
		displayVersionLockContents(lockPath)
	}
}

// locateVersionsLockFile tries to find versions.lock using same logic as loadVersionLock
func locateVersionsLockFile() string {
	// Check environment variable first
	if envPath := strings.TrimSpace(os.Getenv("AISTACK_VERSIONS_LOCK")); envPath != "" {
		if abs, err := filepath.Abs(envPath); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}

	// Check config directory
	configCandidate := filepath.Join("/etc/aistack", "versions.lock")
	if _, err := os.Stat(configCandidate); err == nil {
		return configCandidate
	}

	// Check executable directory
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates := []string{
			filepath.Join(exeDir, "versions.lock"),
			filepath.Join(exeDir, "..", "share", "aistack", "versions.lock"),
		}
		for _, candidate := range candidates {
			if abs, err := filepath.Abs(candidate); err == nil {
				if _, err := os.Stat(abs); err == nil {
					return abs
				}
			}
		}
	}

	// Check current working directory
	if cwd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(cwd, "versions.lock")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

// displayVersionLockContents reads and displays the version lock file
func displayVersionLockContents(path string) {
	file, err := os.Open(filepath.Clean(path)) // #nosec G304 -- path is from controlled locations
	if err != nil {
		fmt.Fprintf(os.Stderr, "    Error reading lock file: %v\n", err)
		return
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close file: %v\n", cerr)
		}
	}()

	content, err := io.ReadAll(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "    Error reading lock file: %v\n", err)
		return
	}

	lines := strings.Split(string(content), "\n")
	hasEntries := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fmt.Printf("    %s\n", line)
		hasEntries = true
	}

	if !hasEntries {
		fmt.Println("    (empty lock file)")
	}
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

	if err := executeServiceAction(command, serviceName, service, manager, os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func executeServiceAction(command, serviceName string, service services.Service, manager *services.Manager, extraArgs []string) error {
	switch command {
	case "start":
		return handleServiceStart(serviceName, service)
	case "stop":
		return handleServiceStop(serviceName, service)
	case "update":
		return handleServiceUpdate(serviceName, service)
	case "logs":
		return handleServiceLogs(serviceName, service, extraArgs)
	case "repair":
		return handleServiceRepair(serviceName, manager)
	default:
		return fmt.Errorf("unknown service command: %s", command)
	}
}

func handleServiceStart(serviceName string, service services.Service) error {
	fmt.Printf("Starting service: %s\n", serviceName)
	if err := service.Start(); err != nil {
		return fmt.Errorf("Error starting service: %w", err)
	}
	fmt.Printf("Service %s started successfully\n", serviceName)
	return nil
}

func handleServiceStop(serviceName string, service services.Service) error {
	fmt.Printf("Stopping service: %s\n", serviceName)
	if err := service.Stop(); err != nil {
		return fmt.Errorf("Error stopping service: %w", err)
	}
	fmt.Printf("Service %s stopped successfully\n", serviceName)
	return nil
}

func handleServiceUpdate(serviceName string, service services.Service) error {
	// Check update policy before proceeding (Story T-035)
	cfg, err := config.Load()
	if err != nil {
		// Warn but allow update if config can't be loaded (backwards compatibility)
		fmt.Fprintf(os.Stderr, "Warning: Could not load config, proceeding with update: %v\n", err)
	} else if cfg.Updates.Mode == "pinned" {
		return fmt.Errorf("updates are disabled: updates.mode is set to 'pinned' in configuration\nChange to 'rolling' in config.yaml to allow updates")
	}

	fmt.Printf("Updating service: %s\n", serviceName)
	fmt.Println("This will pull the latest image and restart the service.")
	fmt.Println("Health checks will be performed and rollback will occur on failure.")
	fmt.Println()
	if err := service.Update(); err != nil {
		return fmt.Errorf("\n❌ Update failed: %w", err)
	}
	fmt.Printf("\n✓ Service %s updated successfully\n", serviceName)
	return nil
}

func handleServiceLogs(serviceName string, service services.Service, extraArgs []string) error {
	tail := 100
	if len(extraArgs) > 0 {
		if _, err := fmt.Sscanf(extraArgs[0], "%d", &tail); err != nil {
			return fmt.Errorf("Invalid tail count: %s", extraArgs[0])
		}
	}
	fmt.Printf("=== Logs for %s (last %d lines) ===\n\n", serviceName, tail)
	logs, err := service.Logs(tail)
	if err != nil {
		return fmt.Errorf("Error getting logs: %w", err)
	}
	fmt.Print(logs)
	return nil
}

func handleServiceRepair(serviceName string, manager *services.Manager) error {
	fmt.Printf("Repairing service: %s\n", serviceName)
	fmt.Println("This will stop, remove, and recreate the service.")
	fmt.Println("Volumes will be preserved. Health checks will validate the repair.")
	fmt.Println()

	result, err := manager.RepairService(serviceName)
	if err != nil {
		return fmt.Errorf("\n❌ Repair failed: %w", err)
	}

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
		return nil
	}

	message := "\n❌ Repair completed but service is not healthy"
	if result.ErrorMessage != "" {
		message += "\n   Error: " + result.ErrorMessage
	}
	return errors.New(message)
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

// runPurge performs a complete system purge
func runPurge() {
	logger := logging.NewLogger(logging.LevelInfo)
	composeDir := resolveComposeDir()
	options := parsePurgeOptions(os.Args[2:])
	requirePurgeFlags(options)
	confirmPurge(options)

	fmt.Println()
	fmt.Println("Starting full system purge...")

	manager, err := services.NewManager(composeDir, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing service manager: %v\n", err)
		os.Exit(1)
	}

	purgeManager := services.NewPurgeManager(manager, logger)
	log, err := purgeManager.PurgeAll(options.removeConfigs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Purge failed: %v\n", err)
		os.Exit(1)
	}

	displayPurgeResults(log)

	fmt.Println()
	fmt.Println("Verifying cleanup...")
	isClean, leftovers := purgeManager.VerifyClean()
	reportCleanupStatus(isClean, leftovers)
	saveUninstallLog(purgeManager, logger, log)

	if len(log.Errors) > 0 || !isClean {
		os.Exit(1)
	}
}

type purgeOptions struct {
	all           bool
	removeConfigs bool
	skipConfirm   bool
}

func parsePurgeOptions(args []string) purgeOptions {
	options := purgeOptions{}
	for _, arg := range args {
		switch arg {
		case "--all":
			options.all = true
		case "--remove-configs":
			options.removeConfigs = true
		case "--yes", "-y":
			options.skipConfirm = true
		}
	}
	return options
}

func requirePurgeFlags(options purgeOptions) {
	if options.all {
		return
	}
	printPurgeUsage()
	os.Exit(1)
}

func printPurgeUsage() {
	fmt.Fprintf(os.Stderr, "Usage: aistack purge --all [--remove-configs] [--yes]\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Purges all services, volumes, and optionally configuration files.\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	fmt.Fprintf(os.Stderr, "  --all             Remove all services and data (required)\n")
	fmt.Fprintf(os.Stderr, "  --remove-configs  Also remove /etc/aistack configuration\n")
	fmt.Fprintf(os.Stderr, "  --yes, -y         Skip confirmation prompts\n")
}

func confirmPurge(options purgeOptions) {
	if options.skipConfirm {
		return
	}

	fmt.Println("⚠️  WARNING: This will permanently delete:")
	fmt.Println("  - All services (Ollama, Open WebUI, LocalAI)")
	fmt.Println("  - All data volumes (models, conversations, caches)")
	fmt.Println("  - State directory (/var/lib/aistack)")
	if options.removeConfigs {
		fmt.Println("  - Configuration directory (/etc/aistack)")
	}
	fmt.Println()
	fmt.Print("Type 'yes' to confirm: ")

	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		exitPurgeCanceled()
	}

	if response != confirmationYes {
		exitPurgeCanceled()
	}

	fmt.Println()
	fmt.Print("Are you absolutely sure? Type 'PURGE' to proceed: ")
	if _, err := fmt.Scanln(&response); err != nil {
		exitPurgeCanceled()
	}

	if response != "PURGE" {
		exitPurgeCanceled()
	}
}

func exitPurgeCanceled() {
	fmt.Fprintf(os.Stderr, "\nPurge canceled\n")
	os.Exit(1)
}

func displayPurgeResults(log *services.UninstallLog) {
	fmt.Println()
	fmt.Printf("Purge completed. Removed %d items:\n", len(log.RemovedItems))
	for _, item := range log.RemovedItems {
		fmt.Printf("  - %s\n", item)
	}

	if len(log.Errors) == 0 {
		return
	}

	fmt.Println()
	fmt.Printf("⚠️  Encountered %d errors:\n", len(log.Errors))
	for _, errMsg := range log.Errors {
		fmt.Printf("  - %s\n", errMsg)
	}
}

func reportCleanupStatus(isClean bool, leftovers []string) {
	if isClean {
		fmt.Println("✓ System is clean. All aistack components removed.")
		return
	}

	fmt.Printf("⚠️  Found %d leftover items:\n", len(leftovers))
	for _, item := range leftovers {
		fmt.Printf("  - %s\n", item)
	}
}

func saveUninstallLog(purgeManager *services.PurgeManager, logger *logging.Logger, log *services.UninstallLog) {
	logPath := "/tmp/aistack_uninstall_log.json"
	if err := purgeManager.SaveUninstallLog(log, logPath); err != nil {
		logger.Warn("purge.log.save_failed", "Failed to save uninstall log", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	fmt.Println()
	fmt.Printf("Uninstall log saved to: %s\n", logPath)
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

// runDiag creates a diagnostic package
// Story T-028: Diagnosepaket/ZIP mit Redaction
func runDiag() {
	logger := logging.NewLogger(logging.LevelInfo)

	// Create default config
	config := diag.NewConfig(version)

	// Parse command line options for custom paths
	if len(os.Args) > 2 {
		for i := 2; i < len(os.Args); i++ {
			arg := os.Args[i]
			if arg == "--output" && i+1 < len(os.Args) {
				config.OutputPath = os.Args[i+1]
				i++
			} else if arg == "--no-logs" {
				config.IncludeLogs = false
			} else if arg == "--no-config" {
				config.IncludeConfig = false
			}
		}
	}

	fmt.Println("Creating diagnostic package...")
	fmt.Printf("  Version: %s\n", config.Version)
	fmt.Printf("  Logs: %v\n", config.IncludeLogs)
	fmt.Printf("  Config: %v\n", config.IncludeConfig)
	fmt.Println()

	// Create packager and generate package
	packager := diag.NewPackager(config, logger)
	zipPath, err := packager.CreatePackage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create diagnostic package: %v\n", err)
		os.Exit(1)
	}

	// Get file size
	fileInfo, err := os.Stat(zipPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Package created but failed to get file info: %v\n", err)
		fmt.Printf("✓ Diagnostic package created: %s\n", zipPath)
		return
	}

	fmt.Printf("✓ Diagnostic package created successfully\n")
	fmt.Printf("  Path: %s\n", zipPath)
	fmt.Printf("  Size: %s\n", formatBytes(fileInfo.Size()))
	fmt.Println()
	fmt.Println("The package contains:")
	fmt.Println("  • System information and version details")
	if config.IncludeLogs {
		fmt.Println("  • Application logs (from /var/log/aistack)")
	}
	if config.IncludeConfig {
		fmt.Println("  • Configuration files (secrets redacted)")
	}
	fmt.Println("  • Manifest with file checksums (diag_manifest.json)")
	fmt.Println()
	fmt.Println("You can share this package for troubleshooting.")
	fmt.Println("All sensitive data has been redacted.")
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

	if strings.ToLower(response) != confirmationYes {
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

// runUpdateAll updates all services sequentially with health-gating
// Story T-029: Container-Update "all" mit Health-Gate
func runUpdateAll() {
	logger := logging.NewLogger(logging.LevelInfo)
	composeDir := resolveComposeDir()

	fmt.Println("Updating all services (LocalAI → Ollama → Open WebUI)...")
	fmt.Println()

	// Create service manager
	manager, err := services.NewManager(composeDir, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to initialize service manager: %v\n", err)
		os.Exit(1)
	}

	// Run update all
	result, err := manager.UpdateAllServices()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Update all failed: %v\n", err)
		os.Exit(1)
	}

	// Display summary
	fmt.Println("=== Update Summary ===")
	fmt.Printf("Total Services: %d\n", result.TotalServices)
	fmt.Printf("✓ Successful: %d\n", result.SuccessfulCount)
	fmt.Printf("⚠ Unchanged: %d\n", result.UnchangedCount)
	fmt.Printf("⟲ Rolled Back: %d\n", result.RolledBackCount)
	fmt.Printf("❌ Failed: %d\n", result.FailedCount)
	fmt.Println()

	// Display per-service results
	fmt.Println("=== Service Results ===")
	services := []string{"localai", "ollama", "openwebui"}
	for _, serviceName := range services {
		if res, exists := result.ServiceResults[serviceName]; exists {
			icon := getUpdateStatusIcon(res)
			status := getUpdateStatusText(res)
			fmt.Printf("%s %s: %s (health: %s)\n", icon, serviceName, status, res.Health)
			if res.ErrorMessage != "" && !res.Success {
				fmt.Printf("  Error: %s\n", res.ErrorMessage)
			}
		}
	}
	fmt.Println()

	// Exit with appropriate code
	if result.FailedCount > 0 {
		fmt.Println("⚠ Some services failed to update. Check logs for details.")
		os.Exit(1)
	}

	if result.SuccessfulCount > 0 {
		fmt.Println("✓ All services updated successfully")
	} else if result.UnchangedCount == result.TotalServices {
		fmt.Println("✓ All services are up to date (no changes)")
	}
}

func getUpdateStatusIcon(res services.UpdateResult) string {
	if res.Success {
		if res.Changed {
			return "✓"
		}
		return "○"
	}
	if res.RolledBack {
		return "⟲"
	}
	return "❌"
}

func getUpdateStatusText(res services.UpdateResult) string {
	if res.Success {
		if res.Changed {
			return "updated successfully"
		}
		return "unchanged (no update needed)"
	}
	if res.RolledBack {
		return "rolled back (health check failed)"
	}
	return "failed"
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

// runConfig performs configuration file validation
// Story T-031 (EP-018): Configuration management
func runConfig() {
	logger := logging.NewLogger(logging.LevelInfo)

	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: aistack config <subcommand>\n")
		fmt.Fprintf(os.Stderr, "Subcommands:\n")
		fmt.Fprintf(os.Stderr, "  test [path]  Test configuration file for validity\n")
		os.Exit(1)
	}

	subcommand := strings.ToLower(os.Args[2])

	switch subcommand {
	case "test":
		runConfigTest(logger)
	default:
		fmt.Fprintf(os.Stderr, "Unknown config subcommand: %s\n", subcommand)
		fmt.Fprintf(os.Stderr, "Valid subcommands: test\n")
		os.Exit(1)
	}
}

// runConfigTest validates configuration file(s)
func runConfigTest(logger *logging.Logger) {
	var cfg config.Config
	var configErr error

	// Check if specific path provided
	if len(os.Args) > 3 {
		path := os.Args[3]
		fmt.Printf("Testing configuration file: %s\n", path)
		cfg, configErr = config.LoadFrom(path)
	} else {
		// Test default system/user merge
		fmt.Println("Testing configuration (system + user merge):")
		systemPath := config.SystemConfigPath()
		userPath := config.UserConfigPath()
		fmt.Printf("  System config: %s\n", systemPath)
		if userPath != "" {
			fmt.Printf("  User config:   %s\n", userPath)
		}
		fmt.Println()

		cfg, configErr = config.Load()
	}

	// Check for errors
	if configErr != nil {
		fmt.Fprintf(os.Stderr, "❌ Configuration validation FAILED:\n")
		fmt.Fprintf(os.Stderr, "   %v\n", configErr)

		logger.Error("config.validation.error", "Configuration validation failed", map[string]interface{}{
			"error": configErr.Error(),
		})
		os.Exit(1)
	}

	// Display configuration summary
	fmt.Println("✓ Configuration is VALID")
	fmt.Println()
	fmt.Println("Configuration Summary:")
	fmt.Printf("  Container Runtime:    %s\n", cfg.ContainerRuntime)
	fmt.Printf("  Profile:              %s\n", cfg.Profile)
	fmt.Printf("  GPU Lock:             %t\n", cfg.GPULock)
	fmt.Printf("  Log Level:            %s\n", cfg.Logging.Level)
	fmt.Printf("  Log Format:           %s\n", cfg.Logging.Format)
	fmt.Printf("  Keep Cache:           %t\n", cfg.Models.KeepCacheOnUninstall)
	fmt.Printf("  Updates Mode:         %s\n", cfg.Updates.Mode)

	logger.Info("config.validation.ok", "Configuration validation passed", map[string]interface{}{
		"profile": cfg.Profile,
		"runtime": cfg.ContainerRuntime,
	})
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
		if syncErr := manager.SyncState(); syncErr != nil {
			logger.Warn("models.list.sync_failed", "Failed to sync state", map[string]interface{}{
				"error": syncErr.Error(),
			})
		}
		modelsList, err = manager.List()
	case models.ProviderLocalAI:
		modelsPath := localAIModelsPath
		manager := models.NewLocalAIManager(stateDir, modelsPath, logger)
		if syncErr := manager.SyncState(); syncErr != nil {
			logger.Warn("models.list.sync_failed", "Failed to sync state", map[string]interface{}{
				"error": syncErr.Error(),
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
	if _, scanErr := fmt.Scanln(&response); scanErr != nil && !errors.Is(scanErr, io.EOF) {
		fmt.Fprintf(os.Stderr, "Failed to read confirmation: %v\n", scanErr)
		os.Exit(1)
	}

	if strings.ToLower(response) != confirmationYes {
		fmt.Println("Aborted.")
		return
	}

	var err error
	switch provider {
	case models.ProviderOllama:
		manager := models.NewOllamaManager(stateDir, logger)
		err = manager.Delete(modelName)
	case models.ProviderLocalAI:
		modelsPath := localAIModelsPath
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
		modelsPath := localAIModelsPath
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
		modelsPath := localAIModelsPath
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
  aistack install --profile <name> Install services from profile (standard-gpu, minimal)
  aistack install <service>        Install a specific service (ollama, openwebui, localai)
  aistack start <service>          Start a service
  aistack stop <service>           Stop a service
  aistack update <service>         Update a service to latest version (with rollback)
  aistack update-all               Update all services sequentially (LocalAI → Ollama → OpenWebUI)
  aistack logs <service> [lines]   Show service logs (default: 100 lines)
  aistack remove <service> [--purge] Remove a service (keeps data by default)
  aistack uninstall <service> [--purge] Alias for remove
  aistack purge --all [--remove-configs] [--yes] Remove all services and data (requires double confirmation)
  aistack backend <ollama|localai> Switch Open WebUI backend (restarts service)
  aistack status                   Show status of all services
  aistack health [--save]          Generate comprehensive health report (services + GPU)
  aistack repair <service>         Repair a service (stop → remove → recreate with health check)
  aistack config test [path]       Test configuration file for validity (defaults to system/user configs)
  aistack gpu-check [--save]       Check GPU and NVIDIA stack availability
  aistack gpu-unlock               Force unlock GPU mutex (recovery)
  aistack models <subcommand>      Model management (list, download, delete, stats, evict-oldest)
  aistack diag [--output path] [--no-logs] [--no-config]  Create diagnostic package (ZIP with logs, config, manifest)
  aistack versions                 Show version lock status and update policy (rolling/pinned)
  aistack suspend <subcommand>     Auto-suspend management (enable, disable, status)
  aistack version                  Print version information
  aistack help                     Show this help message

Model Management:
  aistack models list <provider>           List all models (ollama, localai)
  aistack models download <provider> <name> Download a model (ollama only)
  aistack models delete <provider> <name>   Delete a model
  aistack models stats <provider>           Show cache statistics
  aistack models evict-oldest <provider>    Remove oldest model to free space

Suspend Management:
  aistack suspend enable                   Enable auto-suspend (default)
  aistack suspend disable                  Disable auto-suspend
  aistack suspend status                   Show suspend status and configuration

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

// ========================================
// Suspend Command Handlers
// ========================================

// runSuspend handles the suspend command and subcommands
func runSuspend() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: aistack suspend <enable|disable|status|check>\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Subcommands:\n")
		fmt.Fprintf(os.Stderr, "  enable   Enable auto-suspend (default)\n")
		fmt.Fprintf(os.Stderr, "  disable  Disable auto-suspend\n")
		fmt.Fprintf(os.Stderr, "  status   Show suspend status\n")
		fmt.Fprintf(os.Stderr, "  check    Check and execute suspend if idle (internal use)\n")
		os.Exit(1)
	}

	subcommand := strings.ToLower(os.Args[2])

	switch subcommand {
	case "enable":
		runSuspendEnable()
	case "disable":
		runSuspendDisable()
	case "status":
		runSuspendStatus()
	case "check":
		runSuspendCheck()
	default:
		fmt.Fprintf(os.Stderr, "Unknown suspend subcommand: %s\n", subcommand)
		fmt.Fprintf(os.Stderr, "Valid subcommands: enable, disable, status, check\n")
		os.Exit(1)
	}
}

// runSuspendEnable enables auto-suspend
func runSuspendEnable() {
	logger := logging.NewLogger(logging.LevelInfo)
	manager := suspend.NewManager(logger)

	if err := manager.Enable(); err != nil {
		fmt.Fprintf(os.Stderr, "Error enabling auto-suspend: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Auto-suspend enabled")
	fmt.Println()
	fmt.Println("The system will suspend after 5 minutes of idle time.")
	fmt.Println("Idle = CPU < 10% AND GPU < 5%")
}

// runSuspendDisable disables auto-suspend
func runSuspendDisable() {
	logger := logging.NewLogger(logging.LevelInfo)
	manager := suspend.NewManager(logger)

	if err := manager.Disable(); err != nil {
		fmt.Fprintf(os.Stderr, "Error disabling auto-suspend: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Auto-suspend disabled")
	fmt.Println()
	fmt.Println("The system will NOT automatically suspend.")
}

// runSuspendStatus shows current suspend status
func runSuspendStatus() {
	logger := logging.NewLogger(logging.LevelInfo)
	manager := suspend.NewManager(logger)

	state, err := manager.LoadState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading suspend state: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Auto-Suspend Status ===")
	fmt.Println()

	// Status
	if state.Enabled {
		fmt.Println("Status: ENABLED ✓")
	} else {
		fmt.Println("Status: DISABLED ✗")
	}

	// Configuration
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Printf("  Idle Timeout:    %d seconds (5 minutes)\n", suspend.IdleTimeoutSeconds)
	fmt.Printf("  CPU Threshold:   %.1f%%\n", suspend.CPUIdleThreshold)
	fmt.Printf("  GPU Threshold:   %.1f%%\n", suspend.GPUIdleThreshold)

	// Activity status
	fmt.Println()
	lastActive := time.Unix(state.LastActiveTimestamp, 0)
	idleDuration := time.Since(lastActive)

	fmt.Println("Activity:")
	fmt.Printf("  Last Active:     %s\n", lastActive.Format(time.RFC3339))
	fmt.Printf("  Idle Duration:   %s\n", formatDuration(idleDuration))

	if state.Enabled {
		remaining := time.Duration(suspend.IdleTimeoutSeconds)*time.Second - idleDuration
		if remaining > 0 {
			fmt.Printf("  Time to Suspend: %s\n", formatDuration(remaining))
		} else {
			fmt.Println("  Time to Suspend: READY (will suspend on next check)")
		}
	}
}

// runSuspendCheck performs suspend check (called by systemd timer)
func runSuspendCheck() {
	logger := logging.NewLogger(logging.LevelInfo)
	executor := suspend.NewExecutor(logger, false) // dryRun=false for production

	if err := executor.CheckAndSuspend(); err != nil {
		fmt.Fprintf(os.Stderr, "Error during suspend check: %v\n", err)
		os.Exit(1)
	}
}

// formatDuration formats a duration in human-readable format
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
