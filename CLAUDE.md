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
- Always check returned errors (including cleanup calls like `Close`, `Remove`, `ReadAll`, `Scanln`) so `errcheck` stays clean.
- Factor shared code into helpers and keep cyclomatic complexity ≤15 to satisfy `dupl`/`gocyclo`; repeated literals (e.g. paths, status strings) belong in constants to placate `goconst`.
- Directory creation must use permissions ≤0750 to avoid `gosec` G301 violations.

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

### Current Module Structure

- `cmd/aistack/` — CLI entry point and command wiring (agent, install, services, idle, WoL, diagnostics).
- `internal/agent/` — Background agent runtime coordinating metrics + idle loops.
- `internal/config/` & `internal/configdir/` — Configuration parsing, defaults, and filesystem paths.
- `internal/services/` — Compose lifecycle management (install/update/remove/backends).
- `internal/models/` — Model inventory/index shared between runtimes.
- `internal/metrics/` — CPU/GPU metrics collection, RAPL integration, JSONL writer.
- `internal/idle/` — Sliding window idle engine, gating reasons, suspend executor.
- `internal/gpulock/` — GPU lock orchestration for exclusive workloads.
- `internal/gpu/` — Hardware detection, NVML helpers, toolkit detection.
- `internal/wol/` — Wake-on-LAN setup, relay server, CLI helpers.
- `internal/logging/` — Structured JSON logger (stderr + file).
- `internal/diag/` — Diagnostic bundle creation with redaction.
- `internal/secrets/` — AES-GCM encrypted secret store.
- `internal/tui/` — Bubble Tea model/view for interactive dashboard.

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

### Logging & Diagnostics Architecture

The logging and diagnostics subsystem (`internal/logging/` and `internal/diag/`) provides structured event logging and diagnostic package creation:

**Logging** (`internal/logging/`):
- Structured JSON format with ISO-8601 timestamps
- Level-based filtering: debug, info, warn, error
- Dual output modes: stderr (default) and file-based logging
- `NewLogger(minLevel)`: Creates stderr-based logger
- `NewFileLogger(minLevel, logPath)`: Creates file-based logger with automatic directory creation
- File permissions: 0750 for directories, 0640 for files (gosec compliant)
- Thread-safe with configurable output writers
- Event structure: `{"ts": "2025-11-05T...", "level": "info", "type": "event.type", "message": "...", "payload": {...}}`

**Log Rotation** (`assets/logrotate/aistack`):
- Size-based rotation: 100M for general logs, 500M for metrics
- Daily rotation with retention: 7 days (general), 30 days (metrics)
- Compression with `delaycompress`
- Post-rotation hook: `systemctl reload aistack-agent.service`
- Graceful handling: `missingok`, `notifempty`, `minsize`

**Diagnostics** (`internal/diag/`):
- `aistack diag`: Creates ZIP package with redacted secrets
- Components:
  - `redactor.go`: Secret redaction with regex patterns (API keys, tokens, passwords, env vars, connection strings)
  - `collector.go`: Gathers logs, config, system info
  - `packager.go`: Creates ZIP with manifest and SHA256 checksums
  - `types.go`: Manifest format and configuration

**Diagnostic Package Structure**:
```
aistack-diag-YYYYMMDD-HHMMSS.zip
├── logs/
│   ├── agent.log
│   └── metrics.log
├── config/
│   └── config.yaml (secrets redacted)
├── system_info.json (hostname, version, timestamp)
└── diag_manifest.json (file list with SHA256 checksums)
```

**Manifest Format**:
```json
{
  "timestamp": "2025-11-05T...",
  "host": "ai-server-01",
  "aistack_version": "0.1.0-dev",
  "files": [
    {
      "path": "logs/agent.log",
      "size_bytes": 102400,
      "sha256": "abc123..."
    }
  ]
}
```

**Secret Redaction Patterns**:
- API keys: `api_key: sk-123` → `api_key: [REDACTED]`
- Environment variables: `export API_KEY=xyz` → `export API_KEY=[REDACTED]`
- Bearer tokens: `Authorization: Bearer xyz` → `Authorization: Bearer [REDACTED]`
- Database URLs: `postgres://user:pass@host` → `postgres://user:[REDACTED]@host`

**CLI Commands**:
- `aistack diag`: Create diagnostic package (default path: `aistack-diag-<timestamp>.zip`)
- `aistack diag --output /path/to/diag.zip`: Custom output path
- `aistack diag --no-logs`: Exclude log files
- `aistack diag --no-config`: Exclude configuration

**Event Logging**:
- `diag.collect.logs.complete`: Log collection finished (file count)
- `diag.collect.config.complete`: Config collection with redaction
- `diag.collect.sysinfo.complete`: System info gathered
- `diag.package.start`: Diagnostic package creation started
- `diag.package.complete`: Package created (file count, output path)
- `diag.package.*.error`: Collection/packaging errors (graceful degradation)

**Testing Pattern**:
- Redactor tests: Verify all secret patterns are detected and redacted
- Collector tests: Verify artifact gathering with missing files/directories
- Packager tests: End-to-end ZIP creation with manifest validation
- All tests use temporary directories for isolation

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
  2. Check systemd-inhibit for active locks (unless `--ignore-inhibitors`)
  3. Execute `systemctl suspend`
- Events logged: `power.suspend.requested`, `power.suspend.skipped`, `power.suspend.done`, `power.inhibit.check.skipped`
- Dry-run mode for safe testing without actual suspend
- Inhibitor detection via `systemd-inhibit --list`
- Force mode: `ExecuteWithOptions(state, ignoreInhibitors=true)` bypasses systemd locks
- CLI usage: `aistack idle-check --ignore-inhibitors` (useful for testing with Desktop Environment)

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
- `aistack update <service>`: Update single service with automatic rollback
- `aistack update-all`: Update all services sequentially with independent rollback
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

**Update-All Feature** (Story T-029):
- Updates all services sequentially in order: LocalAI → Ollama → Open WebUI
- Each service update is independent: failure in one service does not affect others
- `UpdateAllResult` tracks: successful, failed, rolled_back, unchanged counts
- Per-service results with health status and error messages
- Exit code 0 if all successful or unchanged, 1 if any failures

**Update-All Workflow**:
```
For each service in [localai, ollama, openwebui]:
  ↓
Update service (with health-gating and rollback)
  ├─ Success → Continue to next service
  ├─ Unchanged → Continue to next service
  ├─ Rollback → Log warning, continue to next service
  └─ Failed → Log error, continue to next service
  ↓
Display summary (totals + per-service results)
  ↓
Exit with status based on overall success
```

**Testing Pattern**:
- Mock manager for unit tests
- Verify correct update order (LocalAI first, OpenWebUI last)
- Verify independent failure handling (all services attempted)
- Verify count consistency (total = successful + failed + rolled_back + unchanged)

### Update Policy & Version Locking Architecture (EP-021)

The update policy subsystem (`internal/services/versions.go`, `internal/config`) provides deterministic version control and update policy enforcement:

**Version Lock** (`versions.lock`):
- File-based version pinning for deterministic deployments
- Format: `service:image[@digest]` (one per line)
- Supports both tags and digests (digests preferred for immutability)
- Comment lines supported (starting with `#`)
- Location search order:
  1. `$AISTACK_VERSIONS_LOCK` environment variable
  2. `/etc/aistack/versions.lock`
  3. Executable directory
  4. Current working directory

**VersionLock Structure**:
- `entries map[string]string`: Service name → image reference mapping
- `path string`: Location of loaded lock file
- `Resolve(serviceName, defaultImage)`: Returns `ImageReference` with PullRef and TagRef
- Graceful fallback: Services not in lock use default images

**ImageReference**:
- `PullRef`: Image reference for docker pull (with digest or tag)
- `TagRef`: Image reference for docker tag (local tag, usually default)
- Digest example: `PullRef=ollama/ollama@sha256:abc123`, `TagRef=ollama/ollama:latest`
- Tag example: `PullRef=ollama/ollama:v0.1.0`, `TagRef=ollama/ollama:latest`

**Update Policy** (`updates.mode` in config):
- `rolling` (default): Updates allowed, uses latest tags or lock file if present
- `pinned`: Updates blocked, services remain at current versions
- Validated in `config/validation.go` (only "rolling" or "pinned" accepted)
- Default: `rolling` for flexibility

**Policy Enforcement**:
- `Manager.checkUpdatePolicy()`: Loads config and validates update policy
- Called at start of `UpdateAllServices()` and in CLI `handleServiceUpdate()`
- When `pinned`: Returns error with clear message to user
- When `rolling`: Allows updates to proceed
- Fail-open: If config can't be loaded, updates are allowed (backwards compatibility)

**Update Blocking Workflow**:
```
User runs: aistack update <service> OR aistack update-all
  ↓
checkUpdatePolicy()
  ↓
Load config.yaml
  ├─ Config load failed → Warn and allow update
  └─ Config loaded successfully
      ↓
      Check updates.mode
      ├─ "rolling" → Allow update
      └─ "pinned" → Block with error message
          ↓
          Error: "updates are disabled: updates.mode is set to 'pinned'"
          ↓
          Exit with code 1
```

**Version Lock Example** (`/etc/aistack/versions.lock`):
```
# Version lock file for aistack
# Format: service:image[@digest|:tag]

# Use digests for deterministic builds
ollama:ollama/ollama@sha256:abc123def456...
openwebui:ghcr.io/open-webui/open-webui@sha256:789012...

# Or use specific tags
localai:quay.io/go-skynet/local-ai:v2.8.0
```

**Configuration Example** (`config.yaml`):
```yaml
updates:
  mode: pinned  # or "rolling" (default)
```

**CLI Commands**:
- `aistack versions`: Display version lock status and update policy
  - Shows current update mode (rolling/pinned)
  - Shows version lock status (active/not found)
  - Lists locked services with their image references
- `aistack update <service>`: Update single service (blocked if pinned)
- `aistack update-all`: Update all services (blocked if pinned)

**Event Logging**:
- `update.policy.check.failed`: Config load failed, allowing updates
- `update.policy.blocked`: Updates blocked by pinned policy
- `update.policy.allowed`: Updates allowed by rolling policy

**Testing Pattern** (`versions_test.go`):
- 13 comprehensive tests covering all scenarios
- Tests for Resolve() with nil lock, tags, digests, missing services
- Tests for loadVersionLock() with valid/invalid files
- Tests for parser error cases (missing colon, empty entries)
- Tests for file location resolution
- Tests for empty files and comment-only files
- All tests use temporary directories for isolation

**Use Cases**:
- **Development**: `rolling` mode for latest features
- **Production**: `pinned` mode + `versions.lock` for stability
- **CI/CD**: Lock file ensures reproducible deployments
- **Testing**: Pin to specific versions for regression testing
- **Rollback**: Update lock file to previous versions

**Version Lock Benefits**:
- **Determinism**: Same lock file = same deployment
- **Auditability**: Git-tracked lock file shows version history
- **Safety**: Prevents unintended updates in production
- **Flexibility**: Per-service version control
- **Digest support**: Immutable image references

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

### Uninstall & Purge Architecture (EP-020)

The uninstall and purge subsystem (`internal/services/purge.go`) provides complete system cleanup with safety mechanisms:

**Uninstall Command**:
- `aistack uninstall <service> [--purge]`: Alias for `remove` command
- Consistent terminology for users familiar with package managers
- Same behavior: Default keeps volumes, `--purge` removes all data

**Purge Manager** (`purge.go`):
- `PurgeManager`: Orchestrates complete system cleanup
- `UninstallLog`: Structured log of removal operations (JSON format)
- `PurgeAll(removeConfigs bool)`: Removes all services, volumes, networks, and optionally configs
- `VerifyClean()`: Post-purge verification with leftover detection
- `SaveUninstallLog()`: Persists operation log for audit trail

**Purge Workflow**:
```
CLI: aistack purge --all [--remove-configs] [--yes]
  ↓
Double Confirmation (unless --yes):
  ├─ First prompt: Type 'yes' to confirm
  └─ Second prompt: Type 'PURGE' to confirm
  ↓
PurgeAll() execution:
  ├─ Remove all services (ollama, openwebui, localai)
  ├─ Remove aistack network
  ├─ Clean state directory (/var/lib/aistack)
  │   ├─ Remove all files
  │   └─ Preserve config.yaml and wol_config.json (unless --remove-configs)
  └─ Remove configs (/etc/aistack) if --remove-configs
  ↓
VerifyClean():
  ├─ Check for running containers
  ├─ Check for remaining volumes
  └─ Check for files in state directory
  ↓
SaveUninstallLog():
  └─ Save to /var/lib/aistack/uninstall_log.json
  ↓
Display results:
  ├─ Removed items count
  ├─ Errors encountered
  └─ Leftovers (if any)
```

**UninstallLog Structure**:
```json
{
  "timestamp": "2025-11-06T10:00:00Z",
  "target": "all",
  "keep_cache": false,
  "removed_items": [
    "service:ollama",
    "service:openwebui",
    "service:localai",
    "network:aistack-net",
    "state:ollama_state.json",
    "state:openwebui_state.json",
    "configs:/etc/aistack"
  ],
  "errors": []
}
```

**Safety Mechanisms**:
- **Double confirmation**: Prevents accidental purge operations
- **Graceful degradation**: Errors logged but don't stop cleanup process
- **Config preservation**: By default, keeps user configurations
- **Post-purge verification**: Detects and reports any leftovers
- **Audit trail**: Detailed JSON log of all operations

**State Directory Cleanup**:
- Default location: `/var/lib/aistack` (override with `AISTACK_STATE_DIR`)
- Files removed by default:
  - Service state files (JSON)
  - Health reports
  - Idle state
  - Update plans
  - UI state
  - Backend binding
- Files preserved (unless `--remove-configs`):
  - `config.yaml`: User configuration
  - `wol_config.json`: Wake-on-LAN settings

**Config Directory Cleanup**:
- Location: `/etc/aistack` (from `configdir.ConfigDir()`)
- Only removed with `--remove-configs` flag
- Safety check: Only removes if path is `/etc/aistack`
- Prevents accidental removal of non-standard config directories

**Runtime Interface Extensions**:
- `VolumeExists(name string)`: Check if volume exists
- `RemoveNetwork(name string)`: Remove Docker/Podman network
- `IsContainerRunning(name string)`: Check container state
- Implemented for both DockerRuntime and PodmanRuntime

**CLI Commands**:
- `aistack uninstall <service> [--purge]`: Remove single service
- `aistack purge --all`: Remove everything with double confirmation
- `aistack purge --all --remove-configs`: Remove everything including configs
- `aistack purge --all --yes`: Skip confirmation prompts (CI/automation)

**Event Logging**:
- `purge.started`: Purge operation initiated
- `purge.service`: Service removal in progress
- `purge.network`: Network removal
- `purge.state_dir`: State directory cleanup
- `purge.state_dir.skip`: File skipped (config preservation)
- `purge.configs`: Config directory removal
- `purge.completed`: Purge operation finished
- `purge.verify`: Verification started
- `purge.log.saved`: Uninstall log saved

**Testing Pattern**:
- `purge_test.go`: Comprehensive test coverage
- Tests use temporary directories (via `AISTACK_STATE_DIR`)
- Table-driven tests for state directory cleanup
- Verification of config preservation logic
- Mock implementations for all Runtime methods
- File permission verification (0640 for logs)

**Use Cases**:
- **Development**: Clean slate between test runs
- **CI/CD**: Reset environment state
- **Troubleshooting**: Complete reinstall
- **Decommissioning**: Remove all traces of aistack
- **Disk space recovery**: Remove all cached models and data

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

### Security & Secrets Architecture

The security and secrets subsystem (`internal/secrets/`) provides encrypted storage for sensitive data:

**Encryption** (`crypto.go`):
- NaCl secretbox (authenticated encryption) from `golang.org/x/crypto/nacl/secretbox`
- Key derivation: SHA-256 hash of passphrase (32 bytes for secretbox)
- Nonce: 24 bytes, randomly generated per encryption
- Encrypted format: nonce (24 bytes) + authenticated ciphertext
- `DeriveKey(passphrase)`: Derives encryption key from passphrase
- `Encrypt(plaintext, key)`: Returns nonce + ciphertext
- `Decrypt(encrypted, key)`: Extracts nonce, verifies & decrypts

**Secret Store** (`store.go`):
- Encrypted secret storage with automatic passphrase management
- Storage: `/var/lib/aistack/secrets/*.enc` (file permissions 0600)
- Passphrase file: `/var/lib/aistack/.passphrase` (permissions 0600)
- Index file: `/var/lib/aistack/secrets/secrets_index.json` (metadata with last_rotated)
- `StoreSecret(name, value)`: Encrypts and stores secret with permission verification
- `RetrieveSecret(name)`: Decrypts and returns secret
- `DeleteSecret(name)`: Removes secret and updates index
- `ListSecrets()`: Returns list of stored secret names

**Passphrase Management**:
- Auto-generation: 64-character hex string (256 bits of entropy)
- Persistent: Same passphrase reused across store instances
- Strict permissions: 0600 on passphrase file
- Location: Configurable via `SecretStoreConfig`

**Secrets Index** (`secrets_index.json`):
```json
{
  "entries": [
    {
      "name": "api-key",
      "last_rotated": "2025-11-05T15:00:00Z"
    }
  ]
}
```

**File Permissions**:
- Secrets directory: 0750
- Secret files (*.enc): 0600
- Passphrase file: 0600
- Index file: 0600
- Automatic verification on store/retrieve

**Security Properties**:
- Authenticated encryption (NaCl secretbox prevents tampering)
- Random nonces (same plaintext encrypts to different ciphertext)
- Key derivation (passphrase → 32-byte key via SHA-256)
- File permissions enforcement (0600 for all sensitive files)
- No secrets in memory after operation completes

**Error Handling**:
- Missing passphrase: Auto-generated on first use
- Wrong permissions: Warning logged, operation continues
- Wrong key: Decryption fails with clear error message
- Corrupted data: Authentication check fails
- Missing secret: Clear "secret not found" error

**Testing Pattern**:
- Crypto tests: Encrypt/decrypt round-trip, wrong key, corrupted data, large data
- Store tests: Store/retrieve, permissions, index updates, persistent passphrase
- All tests use temporary directories for isolation
- 16 comprehensive tests covering all failure modes

**Use Cases**:
- API keys and tokens
- Database passwords
- Service credentials
- Any sensitive configuration data

### TUI Architecture

The TUI subsystem (`internal/tui/`) provides an interactive terminal interface built with Bubble Tea:

**Screen Management** (`types.go`):
- Screen enum: Menu, Status, Install, Models, Logs, Power, Diagnostics, Settings, Help
- MenuItem struct: Key shortcuts, labels, descriptions, target screens
- UIState: Persisted state (current screen, selection, last error)

**Model Structure** (`model.go`):
- Main Model: TUI application state with screen routing
- System state: GPU report, idle state, backend binding
- Screen-specific state:
  - Install: service selection, operation in progress, result messages
  - Logs: service selection, log content (50 lines)
  - Models: provider selection, cached list, stats display
- State persistence: Saves/restores UI state to `/var/lib/aistack/ui_state.json`

**Keyboard Navigation**:
- Global: q/Ctrl+C (quit), Esc (back to menu), ↑/↓ or j/k (navigate)
- Status screen: b (toggle backend), r (refresh)
- Install screen: i (install service), u (uninstall service), r (refresh)
- Logs screen: Enter/Space (view logs), r (refresh)
- Models screen: l (list models), s (show stats), r (refresh)
- Power screen: t (toggle auto-suspend), r (refresh)

**Rendering** (`menu.go`):
- Lip Gloss styling: High-contrast colors (#00d7ff cyan, #ffd700 gold)
- Screen renderers:
  - `renderMenu()`: Main menu with service list
  - `renderStatusScreen()`: GPU, idle, backend status
  - `renderInstallScreen()`: Service management interface
  - `renderLogsScreen()`: Log viewer with service selection
  - `renderModelsScreen()`: Model cache management
  - `renderPowerScreen()`: Power management and idle configuration
  - `renderHelpScreen()`: Keyboard shortcuts reference

**Integration with CLI**:
- TUI wraps existing CLI functionality (no business logic duplication)
- Install/Uninstall: Calls `services.Manager` methods
- Logs: Uses `Service.Logs()` for container output
- Models: Loads state from `models.StateManager`
- Backend switching: Uses `OpenWebUIService.SwitchBackend()`

**Update Flow**:
```
Keyboard Event
  ↓
handleQuitKeys / handleEscapeKey
  ↓
handleMenuNavigationKeys (if menu screen)
  ↓
handleMenuSelectionKey (if menu screen)
  ↓
handleShortcutKeys (number keys)
  ↓
handleStatusScreenKeys / handleInstallScreenKeys / etc.
  ↓
Action methods (installService, loadLogs, listModels, etc.)
  ↓
Update model state
  ↓
saveState (persist to JSON)
  ↓
View() renders updated screen
```

**CLI Commands**:
- `aistack` (no args): Launch interactive TUI
- `aistack <subcommand>`: Direct CLI execution (install, status, logs, models, etc.)

**Event Logging**:
- `tui.state.save_failed`: UI state persistence error
- Service operations logged via existing event types

**Testing Pattern**:
- Unit tests for menu rendering and navigation
- State persistence tests with temporary directories
- UI state loading/saving validation

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

## Configuration Management Architecture (EP-018)

The configuration subsystem (`internal/config/`) provides robust YAML-based configuration with system/user merge and validation:

**Configuration Files**:
- `/etc/aistack/config.yaml` - System-wide configuration
- `~/.aistack/config.yaml` - User-specific overrides
- `config.yaml.example` - Example configuration template

**Merge Strategy**:
- Priority: defaults → system config → user config
- User settings override system settings
- All unspecified values use defaults from `DefaultConfig()`

**Configuration Schema**:
- `container_runtime` - Docker/Podman selection (default: docker)
- `profile` - Minimal/Standard-GPU/Dev (default: standard-gpu)
- `gpu_lock` - Exclusive GPU mutex (default: true)
- `idle.*` - CPU/GPU thresholds, window, timeout
  - `cpu_idle_threshold` - CPU idle % (default: 10)
  - `gpu_idle_threshold` - GPU idle % (default: 5)
  - `window_seconds` - Sliding window (default: 300)
  - `idle_timeout_seconds` - Suspend timeout (default: 1800)
- `power_estimation.baseline_watts` - Power calculation baseline (default: 50)
- `wol.*` - Wake-on-LAN settings
  - `interface` - Network interface (default: eth0)
  - `mac` - MAC address
  - `relay_url` - Optional HTTP relay URL
- `logging.*` - Level and format
  - `level` - debug/info/warn/error (default: info)
  - `format` - json/text (default: json)
- `models.keep_cache_on_uninstall` - Cache retention (default: true)
- `updates.mode` - Rolling vs. pinned (default: rolling)

**Validation** (`validation.go`):
- Strict schema validation with path-based error reporting
- Range checks for thresholds (0-100%)
- Minimum values for timing parameters
- MAC address format validation
- Enum validation for runtime/profile/log level/format/update mode

**CLI Commands**:
- `aistack config test [path]` - Test configuration file for validity
  - Without path: Tests system + user merge
  - With path: Tests specific file
- Exit code 0 if valid, non-zero if validation fails

**Testing Pattern**:
- Table-driven tests for defaults, validation, and merging
- Temporary directories for file-based tests
- 26 comprehensive tests covering all validation rules

**Usage**:
```bash
# Test default configuration
aistack config test

# Test specific file
aistack config test /path/to/config.yaml

# Test example config
aistack config test config.yaml.example
```

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

## CI/CD Pipeline (EP-019)

The CI/CD subsystem provides automated testing, building, and releasing with quality gates:

**CI Workflow** (`.github/workflows/ci.yml`):
- **Lint Job**: golangci-lint with 5m timeout
- **Test Job**:
  - Race detector enabled (`-race`)
  - Coverage gate: ≥80% for `internal/` packages
  - Coverage report generation with `internal_coverage.txt`
  - CI report artifact (`ci_report.json`) with job metadata
  - Codecov integration for coverage tracking
- **Build Job**:
  - Static binary build (`CGO_ENABLED=0`)
  - Artifact upload (30-day retention)
  - Runs only after lint and test pass

**Release Workflow** (`.github/workflows/release.yml`):
- Triggered on version tags (`v*.*.*`)
- Build with version information embedded
- Checksum generation (SHA256)
- Automated changelog from git commits
- GitHub Release creation with:
  - Binary (`aistack`)
  - Tarball (`aistack-linux-amd64.tar.gz`)
  - Checksums for verification
  - Release report artifact (365-day retention)

**CI Report Format** (`ci_report.json`):
```json
{
  "job": "test",
  "status": "success",
  "timestamp": "2025-11-05T17:00:00Z",
  "coverage": {
    "total": 85.2,
    "threshold": 80,
    "passed": true
  },
  "race_detector": "enabled",
  "go_version": "1.22"
}
```

**Coverage Gate Implementation**:
- Extracts coverage for `internal/` packages only
- Calculates average coverage across core packages
- Fails build if below 80% threshold
- Reports detailed per-file coverage in artifact

**Release Process**:
1. Create and push version tag: `git tag v1.0.0 && git push origin v1.0.0`
2. Release workflow automatically:
   - Builds binary with version embedded
   - Generates checksums
   - Creates changelog from commits
   - Publishes GitHub Release with all artifacts

**Quality Gates**:
- Linting must pass (golangci-lint)
- All tests must pass with race detector
- Core packages must maintain ≥80% coverage
- Build must succeed on linux/amd64

**Semantic Versioning**:
- Format: `vMAJOR.MINOR.PATCH`
- Follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
- Conventional commit messages encouraged

**Artifact Retention**:
- CI reports: 90 days
- Coverage reports: 30 days
- Build artifacts: 30 days
- Release reports: 365 days

## Documentation Structure (EP-022)

### User-Facing Documentation

- `README.md` - Quick start guide, installation, basic commands
  - Production installation steps (Ubuntu 24.04)
  - Goal: Services green in ≤10 minutes
  - Development build instructions
  - Troubleshooting quick reference

- `docs/OPERATIONS.md` - Operations playbook for administrators
  - Service management procedures
  - Comprehensive troubleshooting guide
  - Update & rollback workflows
  - Backup & recovery procedures
  - Performance tuning
  - Common error patterns with solutions
  - Emergency procedures

- `docs/POWER_AND_WOL.md` - Power management & Wake-on-LAN guide
  - Idle detection and auto-suspend setup
  - Wake-on-LAN configuration and testing
  - Detailed troubleshooting for suspend/wake issues
  - Configuration tuning recommendations
  - Advanced usage scenarios
  - FAQ section

- `config.yaml.example` - Complete configuration template
  - All supported options with comments
  - Default values documented
  - Location: `/etc/aistack/config.yaml` or `~/.aistack/config.yaml`

- `versions.lock.example` - Version pinning template
  - Digest and tag format examples
  - Usage notes and best practices
  - Location: `/etc/aistack/versions.lock`

### Developer Documentation

- `AGENTS.md` - Contributor guidelines and coding standards
- `CLAUDE.md` - AI-assisted development context (this file)
- `status.md` - Work session log
- `docs/features/epics.md` - Product direction and epic definitions
- `docs/cheat-sheets/` - Quick reference guides (Go, Makefile, networking, etc.)

### Documentation Principles (from EP-022)

- **Pragmatic**: Focus on getting things done, not comprehensive theory
- **Tested**: All commands and procedures have been verified
- **Structured**: Clear sections, table of contents, easy navigation
- **Searchable**: Keywords and error messages included for easy searching
- **Maintained**: Documentation updated alongside code changes
- **No Secrets**: Examples use placeholders, never real credentials

## Commit Guidelines

Follow Conventional Commits format:
- `feat:` - New features
- `fix:` - Bug fixes
- `refactor:` - Code restructuring
- `test:` - Test additions/changes
- `docs:` - Documentation updates

Keep commits scoped to a single concern with passing tests.
