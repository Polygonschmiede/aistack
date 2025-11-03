# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

aistack is a Go-based TUI/CLI tool for managing AI services (Ollama, Open WebUI, LocalAI) with container orchestration, GPU management, power monitoring, and auto-suspend capabilities. The project targets Ubuntu 24.04 Linux systems with optional NVIDIA GPU support.

## Development Commands

### Build & Run
- `go run .` - Execute the CLI locally for quick testing
- `go build ./...` - Compile all packages (run before pushing)

### Testing
- `go test ./...` - Run the unit test suite
- `go test ./... -race` - Run tests with race detector for concurrency-sensitive code
- `go test ./... -cover` - Run tests with coverage reporting
- Target: ≥80% coverage for core packages

### Code Quality
- `go fmt ./...` - Format code (required before commits)
- `go vet ./...` - Run static analysis
- `golangci-lint run` - Run comprehensive linting (if configured)
- `go mod tidy` - Clean up and verify dependencies
- Lint strictly forbids variable shadowing (notably re-declaring `err` in inner scopes); reuse existing variables with plain assignment or declare them once to keep `shadow` happy.

## Architecture Overview

### Epic-Based Development Structure
The project follows an epic-based development approach documented in `docs/features/epics.md`. Key epics include:

1. **EP-001**: Repository & Tech Baseline (Go + Bubble Tea TUI)
2. **EP-002**: Bootstrap & System Integration (install.sh + systemd)
3. **EP-003**: Container Runtime & Compose Assets (Docker/Podman)
4. **EP-004**: NVIDIA Stack Detection & Enablement
5. **EP-005**: Metrics & Sensors (GPU/CPU/Temp/Power)
6. **EP-006**: Idle Engine & Autosuspend
7. **EP-007**: Wake-on-LAN Setup & HTTP Relay
8. **EP-008-010**: Service Orchestration (Ollama, Open WebUI, LocalAI)
11. **EP-011**: GPU Lock & Concurrency Control
12. **EP-012**: Model Management & Caching

### Planned Module Structure
The codebase is designed to grow into these modules (currently minimal):

- `cmd/aistack/` - CLI entry point
- `internal/installer/` - Bootstrap and system setup logic
- `internal/services/` - Container service lifecycle management
- `internal/power/` - Power monitoring and idle detection
- `internal/metrics/` - GPU/CPU metrics collection (NVML, RAPL)
- `internal/diag/` - Diagnostics and health checks
- `internal/update/` - Update and rollback mechanisms
- `pkg/` - Reusable packages (if needed)

### Key Technical Decisions

**UI Framework**: Bubble Tea + Lip Gloss for TUI (keyboard-only, no mouse)

**Container Runtime**: Docker (default), Podman (best-effort support)

**GPU Management**: NVML bindings for NVIDIA GPU detection, metrics, and health checks

**Power Management**: systemd integration for suspend/resume, RAPL for CPU power measurement

**Configuration**: YAML-based (`/etc/aistack/config.yaml` + `~/.aistack/config.yaml`)

**Logging**: Structured JSON logs under `/var/log/aistack/` by default (override with `AISTACK_LOG_DIR`), logrotate-managed

**Testing**: Table-driven tests, interfaces for mocking external dependencies

**Metrics Collection**: JSONL-based metrics logging with CPU/GPU sampling and RAPL delta power estimates

### Metrics Architecture

The metrics subsystem (`internal/metrics/`) collects system and GPU metrics for power monitoring and idle detection:

**CPU Metrics** (`cpu_collector.go`):
- `/proc/stat` parsing for CPU utilization (delta-based calculation)
- RAPL power measurement via `/sys/class/powercap/intel-rapl/` with energy delta tracking
- Graceful degradation when RAPL unavailable or disabled (`MetricsConfig.EnableCPUPower`)

**GPU Metrics** (`gpu_collector.go`):
- NVML-based metrics: GPU/Memory utilization, power usage, temperature
- DeviceInterface abstraction for testability
- Initialize/Collect/Shutdown lifecycle with thread-safety
- Automatic fallback when GPU unavailable

**Metrics Aggregation** (`collector.go`):
- Combines CPU and GPU metrics into `MetricsSample`
- Total power calculation: Baseline + CPU + GPU
- Configurable sample interval (default: 10s)
- Run loop with ticker and stop channel for continuous collection

**Data Format** (`writer.go`):
- JSONL (JSON Lines) format for append-only logging
- One JSON object per line with timestamp
- Optional fields (`omitempty`) for unavailable metrics

**Sample Structure**:
```json
{
  "ts": "2025-11-03T10:44:26Z",
  "cpu_util": 45.2,
  "cpu_w": 35.0,
  "gpu_util": 75.0,
  "gpu_mem": 3072,
  "gpu_w": 200.0,
  "temp_gpu": 72.0,
  "est_total_w": 285.0
}
```

**Testing Pattern**:
- MockNVML for GPU/metrics testing without hardware
- Local mock implementations per package (avoids test file exports)
- Table-driven tests for various hardware availability scenarios
- Graceful degradation verified on macOS (no `/proc/stat`, RAPL, NVML)

### Idle Engine & Autosuspend Architecture

The idle detection subsystem (`internal/idle/`) provides intelligent system suspend based on CPU/GPU activity:

**Idle Configuration** (`types.go`):
- `WindowSeconds`: Sliding window size for idle calculation (default: 60s)
- `IdleTimeoutSeconds`: Idle duration before suspend (default: 300s = 5min)
- `CPUThresholdPct`: CPU utilization threshold (default: 10%)
- `GPUThresholdPct`: GPU utilization threshold (default: 5%)
- `MinSamplesRequired`: Minimum samples before decision (default: 6)
- `EnableSuspend`: Enable actual suspend execution (false for dry-run)

**Sliding Window** (`window.go`):
- Thread-safe metric sample collection with time-based pruning
- IsIdle() calculation: System idle when CPU < threshold AND GPU < threshold
- GetIdleDuration() tracks continuous idle time
- Hysteresis: Resets idle duration immediately when activity detected
- Prevents flapping with minimum sample requirements

**Idle Engine** (`engine.go`):
- Consumes CPU/GPU metrics from metrics collector
- Calculates idle state with three statuses: `warming_up`, `active`, `idle`
- Gating reasons prevent premature suspend:
  - `warming_up`: Insufficient samples collected
  - `below_timeout`: Idle but not long enough
  - `high_cpu`: CPU above threshold
  - `high_gpu`: GPU above threshold
  - `inhibit`: systemd inhibitor active
- ShouldSuspend() decision gate checks all conditions

**State Persistence** (`state.go`):
- JSON format saved to `/var/lib/aistack/idle_state.json` by default (override with `AISTACK_STATE_DIR` for developer runs)
- Atomic writes (temp file + rename) for crash safety
- Schema: `{status, idle_for_s, threshold_s, cpu_idle_pct, gpu_idle_pct, gating_reasons, last_update}`
- Used by timer-triggered idle-check for suspend decisions

**Suspend Executor** (`executor.go`):
- Multi-stage gate checking:
  1. Check gating reasons
  2. Check systemd-inhibit for active locks
  3. Execute `systemctl suspend`
- Events logged: `power.suspend.requested`, `power.suspend.skipped`, `power.suspend.done`
- Dry-run mode for safe testing without actual suspend
- Inhibitor detection via `systemd-inhibit --list`

**Agent Integration**:
- Metrics collected every 10s
- CPU/GPU utilization fed to idle engine
- Idle state updated and persisted on each tick
- Timer-triggered `aistack idle-check` evaluates suspend eligibility
- Graceful shutdown preserves idle state across restarts

**Workflow**:
```
Agent Tick (10s)
  ↓
Collect Metrics (CPU%, GPU%)
  ↓
Idle Engine: AddMetrics()
  ↓
Calculate State (warming_up/active/idle)
  ↓
Persist to idle_state.json
  ↓
(Timer triggers idle-check every 10s)
  ↓
Load idle_state.json
  ↓
ShouldSuspend() decision
  ↓
Check inhibitors
  ↓
systemctl suspend (if all gates pass)
```

### Wake-on-LAN Architecture

The Wake-on-LAN subsystem (`internal/wol/`) provides remote system wake-up capabilities:

**WoL Types** (`types.go`):
- `WoLConfig`: Configuration with Interface, MAC, WoLState, BroadcastIP
- `WoLStatus`: Detection result with Supported, Enabled, WoLModes, CurrentMode
- MAC address validation: Regex-based for various formats (XX:XX:XX:XX:XX:XX, XX-XX-XX-XX-XX-XX, XXXXXXXXXXXX)
- `NormalizeMAC()`: Converts to uppercase colon-separated format
- `ParseMAC()`: Converts to net.HardwareAddr with validation
- `GetBroadcastAddr()`: Calculates broadcast IP from interface network

**WoL Detector** (`detector.go`):
- `DetectWoL()`: ethtool-based WoL status detection
- `EnableWoL()/DisableWoL()`: Configure WoL via ethtool (mode 'g' for magic packet, 'd' for disabled)
- `GetDefaultInterface()`: Auto-detect suitable network interface (IPv4, not loopback)
- `parseEthtoolOutput()`: Parse ethtool output for "Supports Wake-on:" and "Wake-on:" lines
- `parseWoLModes()`: Extract WoL modes (p/u/m/b/g/d) from ethtool string
- Graceful degradation when ethtool not available

**WoL Modes**:
- `p`: Wake on PHY activity
- `u`: Wake on unicast messages
- `m`: Wake on multicast messages
- `b`: Wake on broadcast messages
- `g`: Wake on magic packet (most common)
- `d`: Disabled

**Magic Packet Sender** (`magic.go`):
- `buildMagicPacket()`: Constructs magic packet (6 bytes 0xFF + 16x MAC address = 102 bytes)
- `SendMagicPacket()`: UDP broadcast on ports 7 and 9 for maximum compatibility
- `ValidateMagicPacket()`: Verification for testing (header check + repetition validation)
- Dual-port sending: Success if at least one port works
- Default broadcast: 255.255.255.255 (customizable per interface)

**CLI Commands**:
- `aistack wol-check`: Display WoL status for default interface
- `aistack wol-setup <interface>`: Enable WoL on specified interface (requires root)
- `aistack wol-send <mac> [broadcast_ip]`: Send magic packet to MAC address

**Event Logging**:
- `wol.detect.*`: WoL detection events (found, not_found, ethtool_not_found)
- `wol.send.*`: Magic packet sending events (success, port_failed)
- `wol.default_interface.*`: Interface detection events

**Requirements**:
- ethtool required for WoL detection and configuration (Linux only)
- Root/sudo required for WoL configuration changes
- Hardware/driver must support WoL (check BIOS/UEFI settings)
- Network switch must forward broadcast packets

**HTTP Relay** (Optional, Story T-016):
- Not implemented in core
- Would provide HTTP→WoL gateway for remote wake-up via API
- Planned for future enhancement

### Service Update & Rollback Architecture

The service update subsystem (`internal/services/updater.go`) provides safe service updates with automatic rollback:

**Update Plan** (`UpdatePlan`):
- Tracks update operations for rollback capability
- Fields: ServiceName, OldImageID, NewImage, NewImageID, Status, HealthAfterSwap
- Persisted to `/var/lib/aistack/{service}_update_plan.json`
- Status values: pending, completed, rolled_back, failed

**ServiceUpdater**:
- `Update()`: Pull image → Health validation → Swap or Rollback
- Image change detection: Skips restart if image unchanged
- 5-second health check delay after service restart
- Automatic rollback on health check failure

**Update Workflow**:
```
Get current image ID (for rollback tracking)
  ↓
Pull new image
  ↓
Get new image ID
  ↓
Compare IDs (skip if unchanged)
  ↓
Stop service
  ↓
Start service with new image
  ↓
Wait 5s for initialization
  ↓
Health check
  ├─ Green → Mark as completed
  └─ Red → Rollback
      ├─ Stop service
      ├─ Start service (uses old image from cache)
      ├─ Wait 5s
      ├─ Health check
      ├─ Green → Mark as rolled_back
      └─ Red → Mark as failed
```

**Service-Specific Images**:
- Ollama: `ollama/ollama:latest`
- OpenWebUI: `ghcr.io/open-webui/open-webui:main`
- LocalAI: `quay.io/go-skynet/local-ai:latest`

**Runtime Extensions**:
- `PullImage()`: Docker image pull
- `GetImageID()`: Image ID retrieval for comparison
- `GetContainerLogs()`: Log retrieval with tail support
- `RemoveVolume()`: Volume cleanup on service removal

**HealthChecker Interface**:
- Abstracts health checking for testability
- `HealthCheck` struct implements interface
- Allows mock health checks in tests

**CLI Commands**:
- `aistack update <service>`: Update with automatic rollback
- `aistack logs <service> [lines]`: View container logs (default: 100 lines)

**Event Logging**:
- `service.update.start`: Update initiated
- `service.update.pull`: Image pull started
- `service.update.restart`: Service restart
- `service.update.health_check`: Health validation
- `service.update.success`: Update completed
- `service.update.health_failed`: Health check failed
- `service.update.rollback`: Rollback initiated
- `service.update.rollback.success`: Rollback succeeded

### Backend Binding Architecture

The backend binding subsystem (`internal/services/backend_binding.go`) provides dynamic backend switching for Open WebUI:

**Backend Types**:
- `BackendOllama`: Ollama backend (http://aistack-ollama:11434)
- `BackendLocalAI`: LocalAI backend (http://aistack-localai:8080)

**UIBinding Structure**:
- `ActiveBackend`: Currently selected backend (BackendType)
- `URL`: Backend URL for API communication
- JSON serialization for state persistence

**BackendBindingManager**:
- `GetBinding()`: Load current binding or return Ollama default
- `SetBinding(backend BackendType)`: Persist backend selection to JSON
- `SwitchBackend(newBackend BackendType)`: Switch backend and return old backend
- State persisted to `/var/lib/aistack/ui_binding.json`
- Atomic writes for crash safety

**OpenWebUI Service Integration**:
- `SwitchBackend(backend BackendType)`: Changes backend with service restart
- `GetCurrentBackend()`: Query currently active backend
- Workflow: Update state → Set environment variable → Restart service
- Idempotency: Skip restart if backend unchanged

**Backend Switch Workflow**:
```
CLI: aistack backend <ollama|localai>
  ↓
BackendBindingManager.SwitchBackend()
  ↓
Get current binding from ui_binding.json
  ↓
Compare with requested backend
  ├─ Same → Return (no change needed)
  └─ Different → Continue
      ↓
      SetBinding(newBackend)
      ↓
      Persist to ui_binding.json
      ↓
      GetBackendURL(newBackend)
      ↓
      Set OLLAMA_BASE_URL environment
      ↓
      Stop OpenWebUI service
      ↓
      Start OpenWebUI service
      ↓
      Health check via compose
```

**Environment Integration**:
- `OLLAMA_BASE_URL`: Environment variable for docker compose
- Set before service restart to apply backend change
- Compose template uses `${OLLAMA_BASE_URL:-http://aistack-ollama:11434}`

**CLI Commands**:
- `aistack backend <ollama|localai>`: Switch backend with validation
- User-friendly output: Current backend, switch progress, success message
- Error handling: Invalid backend names rejected with clear messages

**Event Logging**:
- `openwebui.backend.switch.start`: Backend switch initiated
- `openwebui.backend.switch.no_change`: Backend already set (idempotent)
- `openwebui.backend.switch.restart`: Service restart for backend change
- `openwebui.backend.switch.success`: Backend switched successfully
- `ui.backend.changed`: Backend binding updated
- `ui.backend.switched`: Backend switched with from/to tracking

**Testing Pattern**:
- All tests use temporary directories for state isolation
- Table-driven tests for backend validation
- Idempotency testing (switch to same backend)
- State persistence verification (JSON format, file creation)

### Service Lifecycle & Volume Management

The service lifecycle subsystem provides full lifecycle management for all services with intelligent volume handling:

**Service Interface**:
- `Install()`: Ensures network, volumes, and starts service
- `Start()`: Starts service via docker compose up
- `Stop()`: Stops service via docker compose down
- `Remove(keepData bool)`: Removes service with optional volume purge
- `Update()`: Updates to latest image with rollback (see Update & Rollback Architecture)
- `Status()`: Returns service state and health
- `Logs(tail int)`: Retrieves container logs

**Remove Workflow**:
```
CLI: aistack remove <service> [--purge]
  ↓
Parse flags: keepData = !purge
  ↓
service.Remove(keepData)
  ↓
Stop service (graceful, errors logged but continue)
  ↓
If keepData = false (--purge specified):
  ├─ For each volume in service.volumes:
  │   └─ runtime.RemoveVolume(volume)
  └─ Log volume removal (warnings on errors)
  ↓
If keepData = true (default):
  └─ Volumes preserved
  ↓
Log service.removed event
```

**Volume Preservation Strategy**:
- **Default behavior**: Volumes are kept when removing a service
- **Rationale**: Data preservation for reinstalls, prevents accidental data loss
- **Purge option**: `--purge` flag explicitly removes all data volumes
- **Use cases**:
  - Remove without purge: Temporary service removal, troubleshooting, upgrades
  - Remove with purge: Complete cleanup, fresh start, disk space recovery

**Service-Specific Volumes**:
- Ollama: `ollama_data` (models and configuration)
- OpenWebUI: `openwebui_data` (user data, conversations)
- LocalAI: `localai_models` (AI models and cache)

**CLI Commands**:
- `aistack install <service>`: Install service with volumes
- `aistack start <service>`: Start service
- `aistack stop <service>`: Stop service (volumes remain)
- `aistack remove <service>`: Remove service (keep volumes)
- `aistack remove <service> --purge`: Remove service and delete all volumes
- `aistack status`: Show status of all services

**Event Logging**:
- `service.install.start`: Installation started
- `service.install.complete`: Installation completed
- `service.start`: Service starting
- `service.started`: Service started successfully
- `service.stop`: Service stopping
- `service.stoped`: Service stopped successfully
- `service.remove`: Service removal initiated (includes keep_data flag)
- `service.removed`: Service removed successfully
- `service.remove.stop_error`: Error during stop (logged but continues)
- `service.remove.volume_error`: Error removing volume (logged but continues)

**Testing Pattern**:
- MockRuntime tracks removed volumes in `RemovedVolumes` slice
- Tests verify volume preservation with keepData=true
- Tests verify volume deletion with keepData=false
- Graceful degradation on errors (logged warnings, no hard failures)

### Health Checks & Repair Architecture

The health check and repair subsystem (`internal/services/health_reporter.go`, `repair.go`) provides comprehensive health monitoring and automated service repair:

**Health Reporter** (Story T-025):
- `HealthReport`: Aggregated health report with timestamp, service statuses, and GPU health
- `ServiceHealthStatus`: Per-service health with name, status (green/yellow/red), and optional message
- `GPUHealthStatus`: GPU smoke test result (NVML init/shutdown check)
- `HealthReporter.GenerateReport()`: Collects health from all services and GPU
- `HealthReporter.SaveReport()`: Persists report to JSON (default: `/var/lib/aistack/health_report.json`)
- `HealthReporter.CheckAllHealthy()`: Boolean check for automation (returns false if any component unhealthy)

**GPU Health Checker**:
- `DefaultGPUHealthChecker`: Performs NVML init/shutdown smoke test
- Returns GPU count on success, error message on failure
- Graceful degradation when GPU not available (reports as not OK, doesn't crash)

**Service Repair** (Story T-026):
- `RepairResult`: Tracks repair operation with before/after health, success status, error messages
- `Manager.RepairService()`: Idempotent repair workflow
  1. Check current health (skip if already green - no-op)
  2. Stop service (graceful, errors logged)
  3. Remove container (volumes preserved)
  4. Start service (recreate with compose)
  5. Wait 5s for initialization
  6. Recheck health (green = success, otherwise fail)
- `Manager.RepairAll()`: Repairs all unhealthy services automatically

**Repair Workflow**:
```
Check health
  ├─ Green → Skip (idempotent)
  └─ Red/Yellow → Continue
      ↓
Stop service (ignore errors)
      ↓
Remove container (ignore errors)
      ↓
Start service
      ↓
Wait 5s
      ↓
Health check
      ├─ Green → Success
      └─ Red → Failed (but service recreated)
```

**Health Report Format** (`health_report.json`):
```json
{
  "timestamp": "2025-11-03T20:00:00Z",
  "services": [
    {
      "name": "ollama",
      "health": "green",
      "message": ""
    }
  ],
  "gpu": {
    "ok": true,
    "message": "2 GPU(s) detected"
  }
}
```

**CLI Commands**:
- `aistack health [--save]`: Generate and display health report, optionally save to JSON
- `aistack repair <service>`: Repair a specific service with health validation

**Event Logging**:
- `health.report.start`: Health report generation started
- `health.report.service`: Service health checked
- `health.report.complete`: Health report generated
- `health.report.save`: Saving report to file
- `health.report.saved`: Report saved successfully
- `health.gpu.check.start`: GPU smoke test started
- `health.gpu.check.success`: GPU smoke test passed
- `health.gpu.check.failed`: GPU smoke test failed
- `service.repair.started`: Repair initiated
- `service.repair.stopping`: Stopping service for repair
- `service.repair.removing`: Removing container
- `service.repair.starting`: Starting service
- `service.repair.waiting`: Waiting for initialization
- `service.repair.health_check`: Rechecking health after repair
- `service.repair.completed`: Repair completed successfully
- `service.repair.failed`: Repair failed (service not healthy)
- `service.repair.skipped`: Repair skipped (already healthy)

**Testing Pattern**:
- MockGPUHealthChecker for GPU simulation
- DynamicMockHealthCheck for state transitions (red → green after repair)
- Table-driven tests for various repair scenarios
- Idempotency testing (repair on already-healthy service)
- Volume preservation verification

## Go Style Guidelines

From `docs/cheat-sheets/golangbp.md`:

**Formatting**:
- Use `gofmt` defaults (tabs for indentation)
- Never hand-format; run `go fmt ./...` before commits
- Prefer early returns over deep nesting

**Naming**:
- Exported: PascalCase
- Unexported: camelCase
- Package names: singular, lowercase, match directory name
- Keep public APIs minimal

**Error Handling**:
- Return errors instead of logging inside helpers
- Wrap with context: `fmt.Errorf("context: %w", err)`
- Let callers decide how to surface issues

**Concurrency**:
- Prefer channels for sharing data
- Use mutexes for shared mutable state when simpler
- Close channels from sender side only
- Use `context.Context` for cancellation
- Use `sync.WaitGroup` for lifecycle management

**Testing**:
- Co-locate as `*_test.go` in same package
- Use table-driven tests
- Use `t.Helper()` for shared assertions
- Mock external dependencies with interfaces

## Status Tracking

**Work Log**: All work sessions must be recorded in `status.md` with:
- Task description
- Approach taken
- Current status (in progress / completed)
- Date and time (CET)

This ensures continuity across sessions and provides a historical record of development decisions.

## Build Targets & Deployment

**Target Platform**: Linux (Ubuntu 24.04) on amd64

**Build Output**: Single static binary (`aistack`) via `CGO_ENABLED=0` for portability

**Deployment**: systemd units (`aistack-agent.service`, `aistack-idle.timer`)

**Bootstrap**: Headless installation via `install.sh` script

## Configuration Schema

Key configuration sections (from EP-018):
- `container_runtime` - Docker/Podman selection
- `profile` - Minimal/Standard-GPU/Dev
- `gpu_lock` - Exclusive GPU mutex
- `idle.*` - CPU/GPU thresholds, window, timeout
- `power_estimation.baseline_watts` - Power calculation baseline
- `wol.*` - Wake-on-LAN settings
- `logging.*` - Level and format
- `models.keep_cache_on_uninstall` - Cache retention
- `updates.mode` - Rolling vs. pinned

## Important Patterns

**Idempotency**: Install, uninstall, and repair operations must be idempotent

**Health Checks**: Multi-stage (port → HTTP → service-specific) with circuit breakers

**GPU Concurrency**: Exclusive lock mechanism to prevent VRAM conflicts between services

**Update Safety**: Atomic swap with health validation and automatic rollback on failure

**Secrets Management**: Local encryption with libsodium secretbox, 600 permissions

## Testing Strategy

**Unit Tests**: Package-local logic, table-driven, mocked dependencies

**Integration Tests**: Docker-in-Docker for container lifecycle, NVML mocking

**E2E Tests**: VM-based (no real GPU), full bootstrap to service health

**Coverage Target**: ≥80% for core packages (`internal/`)

## CI/CD Expectations

Based on EP-019:
- GitHub Actions workflow for lint/test/build
- Race detector enabled
- Coverage gates enforced
- Artifact upload for snapshot builds
- Semantic versioning with conventional commits

## Documentation Structure

- `AGENTS.md` - Contributor guidelines and coding standards
- `status.md` - Work session log
- `docs/features/epics.md` - Product direction and epic definitions
- `docs/cheat-sheets/` - Quick reference guides (Go, Makefile, networking, etc.)

## Commit Guidelines

Follow Conventional Commits format:
- `feat:` - New features
- `fix:` - Bug fixes
- `refactor:` - Code restructuring
- `test:` - Test additions/changes
- `docs:` - Documentation updates

Keep commits scoped to a single concern with passing tests.
