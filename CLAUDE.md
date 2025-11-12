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
- `internal/secrets/` — NaCl secretbox encrypted secret store.
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

- **CPU Metrics**: `/proc/stat` parsing for utilization, RAPL power measurement via `/sys/class/powercap/intel-rapl/`
- **GPU Metrics**: NVML-based metrics (utilization, memory, power, temperature) with automatic fallback when unavailable
- **Aggregation**: Combines CPU and GPU into `MetricsSample` with total power calculation
- **Data Format**: JSONL (JSON Lines) with timestamp and optional fields (`omitempty`)
- **Testing**: MockNVML for hardware-free testing, graceful degradation on macOS

### Logging & Diagnostics

The logging and diagnostics subsystem provides structured event logging and diagnostic package creation:

- **Logging** (`internal/logging/`): Structured JSON format, level-based filtering, dual output modes (stderr/file), file permissions 0750 for directories, 0640 for files
- **Log Rotation**: Size-based rotation (100M general, 500M metrics), daily rotation with retention (7 days general, 30 days metrics)
- **Diagnostics** (`internal/diag/`): Creates ZIP package with redacted secrets, includes logs/config/system info/manifest with SHA256 checksums
- **CLI**: `aistack diag [--output path] [--no-logs] [--no-config]`

### Idle Engine & Autosuspend

The idle detection subsystem (`internal/idle/`) provides intelligent system suspend based on CPU/GPU activity:

- **Configuration**: Window size, idle timeout, CPU/GPU thresholds, minimum samples, suspend enable flag
- **Sliding Window**: Thread-safe metric collection with time-based pruning, hysteresis to prevent flapping
- **Idle Engine**: Three statuses (warming_up/active/idle), gating reasons (warming_up, below_timeout, high_cpu, high_gpu, inhibit)
- **State Persistence**: JSON format saved to `/var/lib/aistack/idle_state.json` (override with `AISTACK_STATE_DIR`)
- **Suspend Executor**: Multi-stage gate checking, systemd-inhibit detection, dry-run mode, force mode with `--ignore-inhibitors`
- **CLI**: `aistack idle-check [--ignore-inhibitors]`

### Wake-on-LAN

The Wake-on-LAN subsystem (`internal/wol/`) provides remote system wake-up capabilities:

- **WoL Detection**: ethtool-based status detection, enable/disable via ethtool (mode 'g' for magic packet)
- **Magic Packet**: Constructs packet (6 bytes 0xFF + 16x MAC), UDP broadcast on ports 7 and 9
- **CLI**: `aistack wol-check`, `aistack wol-setup <interface>`, `aistack wol-send <mac> [broadcast_ip]`
- **Requirements**: ethtool, root/sudo for config changes, hardware/driver support, network switch forwarding

### Service Update & Rollback

The service update subsystem (`internal/services/updater.go`) provides safe service updates with automatic rollback:

- **Update Plan**: Tracks operations for rollback (ServiceName, OldImageID, NewImage, NewImageID, Status, HealthAfterSwap)
- **Update Workflow**: Pull image → Compare IDs → Restart → Wait 5s → Health check → Rollback if failed
- **Update-All**: Updates all services sequentially (LocalAI → Ollama → OpenWebUI) with independent rollback
- **CLI**: `aistack update <service>`, `aistack update-all`, `aistack logs <service> [lines]`

### Update Policy & Version Locking

The update policy subsystem (`internal/services/versions.go`, `internal/config`) provides deterministic version control:

- **Version Lock**: File-based version pinning (`versions.lock`), format `service:image[@digest]`, search order: `$AISTACK_VERSIONS_LOCK` → `/etc/aistack/versions.lock` → executable dir → cwd
- **ImageReference**: PullRef (with digest/tag) and TagRef (local tag)
- **Update Policy**: `rolling` (default, updates allowed) or `pinned` (updates blocked)
- **Policy Enforcement**: Checks config at start of update commands, blocks updates when pinned
- **CLI**: `aistack versions` (shows lock status and policy), `aistack update <service>`, `aistack update-all`

### Backend Binding

The backend binding subsystem (`internal/services/backend_binding.go`) provides dynamic backend switching for Open WebUI:

- **Backend Types**: BackendOllama (http://aistack-ollama:11434), BackendLocalAI (http://aistack-localai:8080)
- **UIBinding**: State persisted to `/var/lib/aistack/ui_binding.json` with atomic writes
- **Workflow**: Update state → Set OLLAMA_BASE_URL → Restart service → Health check
- **CLI**: `aistack backend <ollama|localai>`

### Service Lifecycle & Volume Management

The service lifecycle subsystem provides full lifecycle management:

- **Service Interface**: Install, Start, Stop, Remove, Update, Status, Logs
- **Volume Preservation**: Default keeps volumes on removal, `--purge` flag removes all data
- **Service Volumes**: ollama_data, openwebui_data, localai_models
- **CLI**: `aistack install/start/stop/remove <service> [--purge]`, `aistack status`

### Uninstall & Purge

The uninstall and purge subsystem (`internal/services/purge.go`) provides complete system cleanup:

- **Uninstall Command**: `aistack uninstall <service> [--purge]` (alias for remove)
- **Purge Manager**: Orchestrates complete cleanup with double confirmation, removes services/volumes/networks/state/configs
- **State Directory**: `/var/lib/aistack` (preserves config.yaml and wol_config.json unless `--remove-configs`)
- **Safety Mechanisms**: Double confirmation, graceful degradation, config preservation, post-purge verification, audit trail
- **CLI**: `aistack purge --all [--remove-configs] [--yes]`

### Health Checks & Repair

The health check and repair subsystem provides comprehensive health monitoring and automated service repair:

- **Health Reporter**: Generates report with service statuses and GPU health, saves to JSON
- **GPU Health Checker**: Performs NVML init/shutdown smoke test
- **Service Repair**: Idempotent repair workflow (check health → stop → remove container → start → wait 5s → health check)
- **CLI**: `aistack health [--save]`, `aistack repair <service>`

### Security & Secrets

The security and secrets subsystem (`internal/secrets/`) provides encrypted storage for sensitive data:

- **Encryption**: NaCl secretbox (authenticated encryption), SHA-256 key derivation, 24-byte random nonces
- **Secret Store**: Encrypted storage in `/var/lib/aistack/secrets/*.enc` (permissions 0600)
- **Passphrase Management**: Auto-generated 64-character hex string (256 bits entropy), persistent, permissions 0600
- **Operations**: StoreSecret, RetrieveSecret, DeleteSecret, ListSecrets

### TUI Architecture

The TUI subsystem (`internal/tui/`) provides an interactive terminal interface built with Bubble Tea:

- **Screen Management**: Menu, Status, Install, Models, Logs, Power, Diagnostics, Settings, Help
- **Keyboard Navigation**: q/Ctrl+C (quit), Esc (back), ↑/↓ or j/k (navigate), screen-specific shortcuts
- **Rendering**: Lip Gloss styling with high-contrast colors (#00d7ff cyan, #ffd700 gold)
- **Integration**: Wraps existing CLI functionality, no business logic duplication
- **CLI**: `aistack` (no args launches TUI), `aistack <subcommand>` (direct CLI execution)

### Configuration Management

The configuration subsystem (`internal/config/`) provides robust YAML-based configuration:

- **Configuration Files**: `/etc/aistack/config.yaml` (system), `~/.aistack/config.yaml` (user overrides), `config.yaml.example` (template)
- **Merge Strategy**: defaults → system config → user config
- **Schema**: container_runtime, profile, gpu_lock, idle.*, power_estimation.baseline_watts, wol.*, logging.*, models.keep_cache_on_uninstall, updates.mode
- **Validation**: Strict schema validation with path-based error reporting, range checks, enum validation
- **CLI**: `aistack config test [path]`

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

## Important Patterns

**Idempotency**: Install, uninstall, and repair operations must be idempotent

**Health Checks**: Multi-stage (port → HTTP → service-specific) with circuit breakers

**GPU Concurrency**: Exclusive lock mechanism to prevent VRAM conflicts between services

**Update Safety**: Atomic swap with health validation and automatic rollback on failure

**Secrets Management**: Local encryption with NaCl secretbox, 0600 permissions

## Testing Strategy

**Unit Tests**: Package-local logic, table-driven, mocked dependencies

**Integration Tests**: Docker-in-Docker for container lifecycle, NVML mocking

**E2E Tests**: VM-based (no real GPU), full bootstrap to service health

**Coverage Target**: ≥80% for core packages (`internal/`)

## CI/CD Pipeline

The CI/CD subsystem provides automated testing, building, and releasing with quality gates:

**CI Workflow** (`.github/workflows/ci.yml`):
- **Lint Job**: golangci-lint with 5m timeout
- **Test Job**: Race detector, ≥80% coverage gate for `internal/`, Codecov integration
- **Build Job**: Static binary (`CGO_ENABLED=0`), artifact upload (30-day retention)

**Release Workflow** (`.github/workflows/release.yml`):
- Triggered on version tags (`v*.*.*`)
- Build with embedded version, checksum generation (SHA256), automated changelog
- GitHub Release with binary, tarball, checksums, release report (365-day retention)

**Quality Gates**:
- Linting must pass (golangci-lint)
- All tests must pass with race detector
- Core packages must maintain ≥80% coverage
- Build must succeed on linux/amd64

**Semantic Versioning**: Format `vMAJOR.MINOR.PATCH`, follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)

## Documentation Structure

### User-Facing Documentation

- `README.md` - Quick start guide, installation, basic commands (goal: services green in ≤10 minutes)
- `docs/OPERATIONS.md` - Operations playbook for administrators (service management, troubleshooting, update/rollback, backup/recovery, performance tuning)
- `docs/POWER_AND_WOL.md` - Power management & Wake-on-LAN guide (idle detection, auto-suspend, WoL configuration, troubleshooting, FAQ)
- `config.yaml.example` - Complete configuration template with comments
- `versions.lock.example` - Version pinning template with examples

### Developer Documentation

- `AGENTS.md` - Contributor guidelines and coding standards
- `CLAUDE.md` - AI-assisted development context (this file)
- `status.md` - Work session log
- `docs/features/epics.md` - Product direction and epic definitions
- `docs/cheat-sheets/` - Quick reference guides (Go, Makefile, networking, etc.)

### Documentation Principles

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
