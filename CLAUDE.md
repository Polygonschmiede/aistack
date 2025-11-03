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
