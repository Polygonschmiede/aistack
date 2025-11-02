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

**Logging**: Structured JSON logs under `/var/log/aistack/`, logrotate-managed

**Testing**: Table-driven tests, interfaces for mocking external dependencies

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