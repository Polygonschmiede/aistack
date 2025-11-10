# Suspend Flow - Complete Technical Documentation

This document traces the complete execution flow of the auto-suspend feature from installation to execution.

## Table of Contents

1. [Installation Flow](#installation-flow)
2. [Agent Startup](#agent-startup)
3. [Metrics Collection Loop](#metrics-collection-loop)
4. [State Persistence](#state-persistence)
5. [Timer-Triggered Idle Check](#timer-triggered-idle-check)
6. [Suspend Execution](#suspend-execution)
7. [File Locations](#file-locations)
8. [Environment Variables](#environment-variables)
9. [Debugging Guide](#debugging-guide)

---

## Installation Flow

### 1. `install.sh` Execution

**File:** `install.sh`

**Key Steps:**
```bash
# Line 197: Build binary with auto-detection
build_aistack_binary() {
    (cd "$script_dir" && make build)
}

# Line 223: Install CLI binary
install_cli_binary() {
    install -m 0755 "$source" /usr/local/bin/aistack
}

# Line 257: Create config (sets defaults)
ensure_config_defaults() {
    cat > /etc/aistack/config.yaml <<CONFIG
idle:
  window_seconds: 60
  idle_timeout_seconds: 300
  cpu_threshold_pct: 10.0
  gpu_threshold_pct: 5.0
  enable_suspend: true
CONFIG
}

# Line 358: Create aistack user
create_aistack_user() {
    useradd -r -s /bin/false -d /var/lib/aistack aistack
    usermod -aG docker aistack
}

# Line 376: Create directories
create_directories() {
    mkdir -p /var/lib/aistack
    mkdir -p /var/log/aistack
    mkdir -p /etc/aistack
    chown -R aistack:aistack /var/lib/aistack
    chown -R aistack:aistack /var/log/aistack
}

# Line 401: Deploy systemd units
deploy_systemd_units() {
    # Stop existing services
    systemctl stop aistack-agent.service
    systemctl stop aistack-idle.timer

    # Copy unit files
    cp -f "$systemd_source"/*.service /etc/systemd/system/
    cp -f "$systemd_source"/*.timer /etc/systemd/system/

    # Reload systemd
    systemctl daemon-reload

    # Enable and start
    systemctl enable aistack-agent.service
    systemctl start aistack-agent.service
    systemctl enable aistack-idle.timer
    systemctl start aistack-idle.timer
}
```

**Result:**
- Binary installed: `/usr/local/bin/aistack`
- Config created: `/etc/aistack/config.yaml`
- User created: `aistack` (uid: system user, groups: aistack, docker)
- Directories created: `/var/lib/aistack`, `/var/log/aistack`, `/etc/aistack`
- Services installed: `aistack-agent.service`, `aistack-idle.service`, `aistack-idle.timer`

---

## Agent Startup

### 2. systemd Service Start

**File:** `assets/systemd/aistack-agent.service`

```ini
[Service]
Type=simple
User=aistack
Group=aistack
ExecStart=/usr/local/bin/aistack agent
WorkingDirectory=/var/lib/aistack

# Environment Variables
Environment="AISTACK_CONFIG=/etc/aistack/config.yaml"
Environment="AISTACK_LOG_DIR=/var/log/aistack"
Environment="AISTACK_STATE_DIR=/var/lib/aistack"

# Security
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/aistack /var/log/aistack
```

**Key Points:**
- Runs as user `aistack` (NOT root)
- Working directory: `/var/lib/aistack`
- Environment variables set by systemd
- Has write access ONLY to `/var/lib/aistack` and `/var/log/aistack`

### 3. Agent Initialization

**File:** `cmd/aistack/agent.go` → `handleAgent()`

```go
// Line 14
func handleAgent(cmd *cobra.Command, args []string) error {
    // Create logger
    logger := logging.NewFileLogger(
        logging.LevelInfo,
        filepath.Join(resolveLogDir(), "agent.log"),
    )

    // Create agent
    agent := agent.NewAgent(logger)

    // Run agent (blocks until shutdown)
    return agent.Run()
}
```

**File:** `internal/agent/agent.go` → `NewAgent()`

```go
// Line 35
func NewAgent(logger *logging.Logger) *Agent {
    // Load idle configuration (reads AISTACK_STATE_DIR env var)
    idleConfig := idle.DefaultIdleConfig()

    // Create idle engine
    idleEngine := idle.NewEngine(idleConfig, logger)

    // Create state manager with path from config
    idleStateManager := idle.NewStateManager(idleConfig.StateFilePath, logger)

    // Create executor
    idleExecutor := idle.NewExecutor(idleConfig, logger)

    return &Agent{
        logger:           logger,
        ctx:              ctx,
        cancel:           cancel,
        tickRate:         10 * time.Second,
        idleEngine:       idleEngine,
        idleStateManager: idleStateManager,
        idleExecutor:     idleExecutor,
    }
}
```

**State File Path Resolution:**

**File:** `internal/idle/types.go` → `DefaultIdleConfig()`

```go
// Line 34
func DefaultIdleConfig() IdleConfig {
    stateDir := "/var/lib/aistack"

    // Check AISTACK_STATE_DIR environment variable
    if envDir := os.Getenv("AISTACK_STATE_DIR"); envDir != "" {
        stateDir = envDir
    } else if os.Geteuid() != 0 {
        // NOT ROOT: Fall back to user home directory
        if home, err := os.UserHomeDir(); err == nil {
            stateDir = filepath.Join(home, ".local", "state", "aistack")
        } else {
            stateDir = filepath.Join(os.TempDir(), "aistack")
        }
    }

    return IdleConfig{
        WindowSeconds:      60,
        IdleTimeoutSeconds: 300,
        CPUThresholdPct:    10.0,
        GPUThresholdPct:    5.0,
        MinSamplesRequired: 6,
        EnableSuspend:      true,
        StateFilePath:      filepath.Join(stateDir, "idle_state.json"),
    }
}
```

**CRITICAL BUG:** If `AISTACK_STATE_DIR` is NOT set in systemd service, and user is NOT root (which is the case - user is `aistack`), then state file goes to:
- `/home/aistack/.local/state/aistack/idle_state.json`

Instead of:
- `/var/lib/aistack/idle_state.json`

### 4. Agent Run Loop

**File:** `internal/agent/agent.go` → `Run()`

```go
// Line 66
func (a *Agent) Run() error {
    a.logger.Info("agent.started", "Agent service started", map[string]interface{}{
        "pid":       os.Getpid(),
        "tick_rate": a.tickRate.String(),
    })

    // Initialize metrics collector
    a.metricsCollector.Initialize()
    defer a.metricsCollector.Shutdown()

    // Setup signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

    // Ticker for periodic tasks (10 seconds)
    ticker := time.NewTicker(a.tickRate)
    defer ticker.Stop()

    // Main event loop
    for {
        select {
        case <-a.ctx.Done():
            return a.ctx.Err()

        case sig := <-sigChan:
            if sig == syscall.SIGTERM || sig == syscall.SIGINT {
                return a.Shutdown()
            }

        case <-ticker.C:
            // Every 10 seconds: collect metrics and update state
            a.collectAndProcessMetrics()
        }
    }
}
```

---

## Metrics Collection Loop

### 5. Metrics Collection (Every 10 Seconds)

**File:** `internal/agent/agent.go` → `collectAndProcessMetrics()`

```go
// Line 124
func (a *Agent) collectAndProcessMetrics() {
    // 1. Collect metrics sample (CPU, GPU)
    sample, err := a.metricsCollector.CollectSample()
    if err != nil {
        a.logger.Warn("agent.metrics.collect_failed", "Failed to collect metrics", ...)
        return  // EARLY RETURN ON ERROR
    }

    // 2. Extract CPU and GPU utilization
    cpuUtil := 0.0
    gpuUtil := 0.0
    if sample.CPUUtil != nil {
        cpuUtil = *sample.CPUUtil
    }
    if sample.GPUUtil != nil {
        gpuUtil = *sample.GPUUtil
    }

    // 3. Add metrics to idle engine (sliding window)
    a.idleEngine.AddMetrics(cpuUtil, gpuUtil)

    // 4. Get current idle state
    idleState := a.idleEngine.GetState()

    // 5. Check for systemd inhibitors
    if a.idleExecutor != nil {
        if hasInhibit, inhibitors, err := a.idleExecutor.ActiveInhibitors(); err != nil {
            // Log warning but continue
            a.logger.Warn("agent.inhibitors.check_failed", ...)
        } else {
            if hasInhibit {
                idleState.GatingReasons = addGatingReason(idleState.GatingReasons, idle.GatingReasonInhibit)
            } else {
                idleState.GatingReasons = removeGatingReason(idleState.GatingReasons, idle.GatingReasonInhibit)
            }
        }
    }

    // 6. Persist metrics sample to JSONL log
    if err := a.metricsCollector.WriteSample(sample, a.metricsLogPath); err != nil {
        a.logger.Warn("agent.metrics.write_failed", ...)
    }

    // 7. Save idle state
    if err := a.idleStateManager.Save(idleState); err != nil {
        a.logger.Warn("agent.idle.state_save_failed", "Failed to save idle state", ...)
    } else {
        a.logger.Info("agent.idle.state_saved", "Idle state saved successfully", ...)
    }
}
```

**Metrics Collection:**

**File:** `internal/metrics/collector.go` → `CollectSample()`

```go
// Line 47
func (c *Collector) CollectSample() (MetricsSample, error) {
    sample := MetricsSample{
        Timestamp: time.Now(),
    }

    // Collect CPU metrics
    if cpuUtil, cpuPower, err := c.cpuCollector.Collect(); err == nil {
        sample.CPUUtil = &cpuUtil
        if cpuPower > 0 {
            sample.CPUPower = &cpuPower
        }
    } else {
        // Log warning but continue (CPU metrics optional)
        c.logger.Warn("cpu.collect.failed", ...)
    }

    // Collect GPU metrics
    if c.config.EnableGPU {
        if gpuMetrics, err := c.gpuCollector.Collect(); err == nil {
            sample.GPUUtil = &gpuMetrics.Utilization
            sample.GPUMemory = &gpuMetrics.MemoryUsedMB
            sample.GPUPower = &gpuMetrics.PowerW
            sample.GPUTemp = &gpuMetrics.TempC
        } else {
            // Log warning but continue (GPU metrics optional)
            c.logger.Warn("gpu.collect.failed", ...)
        }
    }

    return sample, nil
}
```

**CPU Collection:**

**File:** `internal/metrics/cpu_collector.go` → `Collect()`

```go
// Line 34
func (c *CPUCollector) Collect() (util float64, power float64, err error) {
    // 1. Read /proc/stat for CPU utilization
    cpuUtil, err := c.readCPUUtil()
    if err != nil {
        return 0, 0, fmt.Errorf("failed to read CPU util: %w", err)
    }

    // 2. Read RAPL for power (optional)
    if c.config.EnableCPUPower {
        if cpuPower, err := c.readRAPL(); err == nil {
            return cpuUtil, cpuPower, nil
        } else {
            // Log warning but continue without power
            c.logger.Warn("cpu.rapl.read.failed", ...)
        }
    }

    return cpuUtil, 0, nil
}
```

---

## State Persistence

### 6. State Save

**File:** `internal/idle/state.go` → `Save()`

```go
// Line 26
func (sm *StateManager) Save(state IdleState) error {
    // 1. Ensure directory exists
    dir := filepath.Dir(sm.filePath)
    if err := os.MkdirAll(dir, 0o750); err != nil {
        return fmt.Errorf("failed to create directory: %w", err)
    }

    // 2. Marshal to JSON
    data, err := json.MarshalIndent(state, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal state: %w", err)
    }

    // 3. Write to file atomically (write to temp, then rename)
    tempPath := sm.filePath + ".tmp"
    if err := os.WriteFile(tempPath, data, 0o600); err != nil {
        return fmt.Errorf("failed to write temp file: %w", err)
    }

    if err := os.Rename(tempPath, sm.filePath); err != nil {
        return fmt.Errorf("failed to rename temp file: %w", err)
    }

    sm.logger.Debug("idle.state.saved", "Idle state saved", ...)

    return nil
}
```

**State File Format:**
```json
{
  "status": "idle",
  "idle_for_s": 250,
  "threshold_s": 300,
  "cpu_idle_pct": 95.2,
  "gpu_idle_pct": 98.5,
  "gating_reasons": ["below_timeout", "inhibit"],
  "last_update": "2025-11-10T14:50:09Z"
}
```

---

## Timer-Triggered Idle Check

### 7. Timer Configuration

**File:** `assets/systemd/aistack-idle.timer`

```ini
[Unit]
Description=aistack Idle Check Timer

[Timer]
OnBootSec=60s
OnUnitActiveSec=10s

[Install]
WantedBy=timers.target
```

**Trigger Schedule:**
- First run: 60 seconds after boot
- Subsequent runs: Every 10 seconds after previous run completes

### 8. Idle Check Service

**File:** `assets/systemd/aistack-idle.service`

```ini
[Service]
Type=oneshot
User=aistack
Group=aistack
ExecStart=/usr/local/bin/aistack idle-check --ignore-inhibitors
WorkingDirectory=/var/lib/aistack

Environment="AISTACK_CONFIG=/etc/aistack/config.yaml"
Environment="AISTACK_LOG_DIR=/var/log/aistack"
Environment="AISTACK_STATE_DIR=/var/lib/aistack"
```

### 9. Idle Check Command

**File:** `cmd/aistack/idle.go` → `handleIdleCheck()`

```go
// Line 14
func handleIdleCheck(cmd *cobra.Command, args []string) error {
    // Get --ignore-inhibitors flag
    ignoreInhibitors, _ := cmd.Flags().GetBool("ignore-inhibitors")

    // Create logger
    logger := logging.NewFileLogger(
        logging.LevelInfo,
        filepath.Join(resolveLogDir(), "idle-check.log"),
    )

    // Call agent.IdleCheck
    return agent.IdleCheck(logger, ignoreInhibitors)
}
```

**File:** `internal/agent/agent.go` → `IdleCheck()`

```go
// Line 284
func IdleCheck(logger *logging.Logger, ignoreInhibitors bool) error {
    logger.Info("idle.check_started", "Idle check started", ...)

    // 1. Load idle configuration
    idleConfig := idle.DefaultIdleConfig()

    // 2. Create state manager and load current state
    stateManager := idle.NewStateManager(idleConfig.StateFilePath, logger)
    state, err := stateManager.Load()

    // 3. Remove inhibit gating reason if ignore flag set
    if ignoreInhibitors {
        logger.Info("idle.ignore_inhibitors", "Removing inhibit gating reason", ...)
        state.GatingReasons = removeGatingReason(state.GatingReasons, idle.GatingReasonInhibit)
    }

    if err != nil {
        logger.Warn("idle.state_load_failed", "Failed to load idle state", ...)
        return nil  // EARLY RETURN - NO STATE FILE
    }

    logger.Info("idle.state_loaded", "Idle state loaded successfully", ...)

    // 4. Create idle engine and executor
    idleEngine := idle.NewEngine(idleConfig, logger)
    executor := idle.NewExecutor(idleConfig, logger)

    // 5. Check if we should suspend
    shouldSuspend := idleEngine.ShouldSuspend(state)
    logger.Info("idle.should_suspend_check", "Checked if system should suspend", ...)

    if shouldSuspend {
        logger.Info("idle.suspend_check_passed", "System should suspend", ...)

        // 6. Attempt suspend
        if err := executor.ExecuteWithOptions(&state, ignoreInhibitors); err != nil {
            logger.Error("idle.suspend_failed", "Failed to execute suspend", ...)
            stateManager.Save(state)  // Save updated state
            return err
        }

        logger.Info("idle.suspend_success", "Suspend executed successfully", ...)
        stateManager.Save(state)  // Save updated state
    } else {
        logger.Info("idle.suspend_skipped", "Suspend not required", ...)
    }

    return nil
}
```

### 10. Should Suspend Check

**File:** `internal/idle/engine.go` → `ShouldSuspend()`

```go
// Line 72
func (e *Engine) ShouldSuspend(state IdleState) bool {
    // 1. Check if idle
    if state.Status != StatusIdle {
        return false
    }

    // 2. Check if idle long enough
    if state.IdleForSeconds < state.ThresholdSeconds {
        return false
    }

    // 3. Check for gating reasons
    if len(state.GatingReasons) > 0 {
        return false
    }

    return true
}
```

---

## Suspend Execution

### 11. Execute Suspend

**File:** `internal/idle/executor.go` → `ExecuteWithOptions()`

```go
// Line 30
func (e *Executor) ExecuteWithOptions(state *IdleState, ignoreInhibitors bool) error {
    e.logger.Info("power.suspend.execute_start", "Starting suspend execution", ...)

    // 1. Filter inhibit gating reason if ignore flag set
    if ignoreInhibitors {
        filtered := make([]string, 0, len(state.GatingReasons))
        for _, r := range state.GatingReasons {
            if r != GatingReasonInhibit {
                filtered = append(filtered, r)
            }
        }
        state.GatingReasons = filtered
    }

    // 2. Check if any gating reasons remain
    if len(state.GatingReasons) > 0 {
        e.logger.Warn("power.suspend.blocked_by_gating", "Suspend blocked", ...)
        return fmt.Errorf("suspend blocked by gating reasons: %s", ...)
    }

    e.logger.Info("power.suspend.gating_check_passed", "No gating reasons", ...)

    // 3. Check if suspend is enabled
    if !e.config.EnableSuspend {
        e.logger.Warn("power.suspend.disabled", "Suspend disabled (dry-run)", ...)
        return nil
    }

    e.logger.Info("power.suspend.config_check_passed", "Suspend enabled", ...)

    // 4. Check for inhibitors (unless explicitly ignored)
    if !ignoreInhibitors {
        e.logger.Info("power.suspend.check_inhibitors", "Checking systemd inhibitors", ...)
        hasInhibit, inhibitors, err := e.checkInhibitors()
        if err != nil {
            e.logger.Warn("power.inhibit.check.failed", "Failed to check inhibitors", ...)
            // Continue anyway
        }

        if hasInhibit {
            e.logger.Warn("power.suspend.blocked_by_inhibitors", "Suspend blocked", ...)
            state.GatingReasons = append(state.GatingReasons, GatingReasonInhibit)
            return fmt.Errorf("suspend blocked by inhibitors: %s", ...)
        }
    } else {
        e.logger.Info("power.inhibit.check.skipped", "Skipping inhibitor check", ...)
    }

    // 5. All gates passed - request suspend
    e.logger.Info("power.suspend.all_checks_passed", "Executing suspend", ...)

    // 6. Execute systemctl suspend
    if err := e.executeSuspend(); err != nil {
        e.logger.Error("power.suspend.failed", "Failed to execute suspend", ...)
        return fmt.Errorf("failed to execute suspend: %w", err)
    }

    e.logger.Info("power.suspend.done", "Suspend command executed", ...)
    return nil
}
```

### 12. Check Inhibitors

**File:** `internal/idle/executor.go` → `checkInhibitors()`

```go
// Line 153
func (e *Executor) checkInhibitors() (bool, []string, error) {
    // Run systemd-inhibit --list
    cmd := exec.Command("systemd-inhibit", "--list", "--no-pager", "--no-legend")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return false, nil, err
    }

    e.logger.Info("power.inhibit.command_output", "systemd-inhibit raw output", ...)

    lines := strings.Split(strings.TrimSpace(string(output)), "\n")
    inhibitors := make([]string, 0)

    // Parse output for sleep/shutdown inhibitors
    for i, line := range lines {
        if strings.Contains(line, "sleep") || strings.Contains(line, "shutdown") {
            fields := strings.Fields(line)
            if len(fields) > 0 {
                inhibitors = append(inhibitors, fields[0])
            }
        }
    }

    hasInhibit := len(inhibitors) > 0
    return hasInhibit, inhibitors, nil
}
```

### 13. Execute Suspend Command

**File:** `internal/idle/executor.go` → `executeSuspend()`

```go
// Line 213
func (e *Executor) executeSuspend() error {
    e.logger.Info("power.suspend.command_start", "Executing systemctl suspend", ...)

    cmd := exec.Command("systemctl", "suspend")
    output, err := cmd.CombinedOutput()

    e.logger.Info("power.suspend.command_result", "systemctl suspend completed", ...)

    if err != nil {
        e.logger.Error("power.suspend.command_error", "systemctl suspend failed", ...)
        return fmt.Errorf("systemctl suspend failed: %w (output: %s)", err, string(output))
    }

    return nil
}
```

---

## File Locations

### Configuration Files
- **Config:** `/etc/aistack/config.yaml`
- **Versions Lock:** `/etc/aistack/versions.lock` (optional)

### State Files
- **Idle State:** `/var/lib/aistack/idle_state.json` (if `AISTACK_STATE_DIR` is set correctly)
- **Idle State (WRONG):** `/home/aistack/.local/state/aistack/idle_state.json` (if env var missing)
- **UI State:** `/var/lib/aistack/ui_state.json`
- **GPU Lock:** `/var/lib/aistack/gpu_lock.json`
- **Backend Binding:** `/var/lib/aistack/ui_binding.json`
- **Update Plans:** `/var/lib/aistack/{service}_update_plan.json`

### Log Files
- **Agent Log:** `/var/log/aistack/agent.log`
- **Idle Check Log:** `/var/log/aistack/idle-check.log`
- **Metrics Log:** `/var/log/aistack/metrics.log` (JSONL format)

### Systemd Files
- **Agent Service:** `/etc/systemd/system/aistack-agent.service`
- **Idle Service:** `/etc/systemd/system/aistack-idle.service`
- **Idle Timer:** `/etc/systemd/system/aistack-idle.timer`

### Binary
- **CLI Binary:** `/usr/local/bin/aistack`

---

## Environment Variables

### Set by systemd Service

**Agent Service:**
```ini
Environment="AISTACK_CONFIG=/etc/aistack/config.yaml"
Environment="AISTACK_LOG_DIR=/var/log/aistack"
Environment="AISTACK_STATE_DIR=/var/lib/aistack"
```

**Idle Service:**
```ini
Environment="AISTACK_CONFIG=/etc/aistack/config.yaml"
Environment="AISTACK_LOG_DIR=/var/log/aistack"
Environment="AISTACK_STATE_DIR=/var/lib/aistack"
```

### Read by Code

**`internal/idle/types.go` → `DefaultIdleConfig()`:**
```go
stateDir := "/var/lib/aistack"
if envDir := os.Getenv("AISTACK_STATE_DIR"); envDir != "" {
    stateDir = envDir
}
```

**`internal/logging/logger.go` → `NewFileLogger()`:**
```go
logDir := "/var/log/aistack"
if envDir := os.Getenv("AISTACK_LOG_DIR"); envDir != "" {
    logDir = envDir
}
```

---

## Debugging Guide

### Check if State File Exists

```bash
# Check expected location (CORRECT)
sudo ls -la /var/lib/aistack/idle_state.json

# Check wrong location (if env var missing)
sudo ls -la /home/aistack/.local/state/aistack/idle_state.json
```

### Verify Environment Variables in Service

```bash
# Show environment of running agent
sudo systemctl show aistack-agent.service | grep Environment

# Expected output:
# Environment=AISTACK_CONFIG=/etc/aistack/config.yaml AISTACK_LOG_DIR=/var/log/aistack AISTACK_STATE_DIR=/var/lib/aistack
```

### Check Agent Logs

```bash
# Watch agent logs in real-time
sudo journalctl -u aistack-agent -f

# Look for:
# - agent.idle.state_saved (SUCCESS)
# - agent.idle.state_save_failed (ERROR)
```

### Check Idle Check Logs

```bash
# Watch idle check logs
sudo journalctl -u aistack-idle -f

# Look for:
# - idle.check_started
# - idle.state_loaded (SUCCESS) or idle.state_load_failed (ERROR)
# - idle.should_suspend_check
# - idle.suspend_check_passed or idle.suspend_skipped
```

### Manually Trigger Idle Check

```bash
# Run idle check manually (as aistack user)
sudo -u aistack /usr/local/bin/aistack idle-check --ignore-inhibitors

# Or as root (for debugging)
sudo /usr/local/bin/aistack idle-check --ignore-inhibitors
```

### Check Timer Status

```bash
# Check if timer is running
sudo systemctl status aistack-idle.timer

# Check next trigger time
sudo systemctl list-timers aistack-idle.timer
```

### Check Systemd Inhibitors

```bash
# List active inhibitors
systemd-inhibit --list

# Expected output:
# ModemManager, UPower, Unattended Upgrades Shutdown
```

### Check Metrics Collection

```bash
# Check if metrics log is being written
sudo ls -lah /var/log/aistack/metrics.log

# Tail metrics log (JSONL format)
sudo tail -f /var/log/aistack/metrics.log
```

### Check Service Permissions

```bash
# Check who owns state directory
ls -ld /var/lib/aistack

# Expected: drwxr-xr-x aistack aistack

# Check if aistack user can write to state directory
sudo -u aistack touch /var/lib/aistack/test.txt
sudo -u aistack rm /var/lib/aistack/test.txt
```

### Full State Check

```bash
# Check all relevant locations
echo "=== State File ==="
sudo ls -la /var/lib/aistack/idle_state.json
echo ""
echo "=== Wrong Location (if env var missing) ==="
sudo ls -la /home/aistack/.local/state/aistack/idle_state.json
echo ""
echo "=== Metrics Log ==="
sudo ls -lah /var/log/aistack/metrics.log
echo ""
echo "=== Agent Status ==="
sudo systemctl status aistack-agent --no-pager
echo ""
echo "=== Timer Status ==="
sudo systemctl status aistack-idle.timer --no-pager
echo ""
echo "=== Environment Variables ==="
sudo systemctl show aistack-agent.service | grep Environment
```

---

## Known Issues

### Issue 1: State File in Wrong Location

**Symptom:** Logs show `agent.idle.state_saved` but file doesn't exist at `/var/lib/aistack/idle_state.json`

**Cause:** `AISTACK_STATE_DIR` environment variable not set in systemd service

**Location:** File created at `/home/aistack/.local/state/aistack/idle_state.json` instead

**Fix:** Add `Environment="AISTACK_STATE_DIR=/var/lib/aistack"` to systemd service and reload

### Issue 2: RAPL Permission Denied

**Symptom:** `cpu.rapl.read.failed: permission denied`

**Cause:** RAPL energy counters not readable by non-root user

**Impact:** Non-critical - only affects CPU power measurement, not suspend functionality

**Fix:** Apply tmpfiles.d config: `sudo systemd-tmpfiles --create /etc/tmpfiles.d/aistack-rapl.conf`

### Issue 3: Idle Check Can't Find State File

**Symptom:** `idle.state_load_failed: state file not found`

**Cause:** Agent hasn't run yet to create initial state file, or state file in wrong location

**Fix:**
1. Wait for agent to run (10 seconds)
2. Check if `AISTACK_STATE_DIR` is set correctly in systemd service
3. Check if state file exists in expected location

### Issue 4: Timer Not Running

**Symptom:** No `idle.check_started` logs in journalctl

**Cause:** Timer service not started or enabled

**Fix:**
```bash
sudo systemctl enable aistack-idle.timer
sudo systemctl start aistack-idle.timer
sudo systemctl list-timers aistack-idle.timer
```

---

## Summary

The suspend flow is:

1. **Installation**: `install.sh` → installs binary, creates user, deploys systemd services
2. **Agent Startup**: systemd starts `aistack-agent.service` → runs as user `aistack` → reads `AISTACK_STATE_DIR` env var
3. **Metrics Loop**: Every 10s → collect CPU/GPU metrics → update idle state → save to `idle_state.json`
4. **Timer Trigger**: Every 10s → timer triggers `aistack-idle.service` → runs `aistack idle-check`
5. **Idle Check**: Load state → check if should suspend → execute suspend if conditions met
6. **Suspend**: `systemctl suspend` → system goes to sleep

**Critical Path:**
- `AISTACK_STATE_DIR` MUST be set in systemd service
- Agent must have write permission to state directory
- State file must exist before idle-check runs
- All gating reasons must be clear for suspend to execute
