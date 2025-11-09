# aistack

> Headless AI service orchestrator with auto-suspend and Wake-on-LAN for power-efficient GPU workloads.

**Status**: Early development (v0.1) - Foundation being built

## Overview

aistack is a Go-based TUI/CLI tool for managing AI services (Ollama, Open WebUI, LocalAI) on Ubuntu 24.04 Linux systems with optional NVIDIA GPU support. It provides:

- **Container Orchestration**: Manage Ollama, Open WebUI, and LocalAI via Docker Compose
- **GPU Management**: NVIDIA GPU detection, metrics, and exclusive locking to prevent VRAM conflicts
- **Power Efficiency**: Automatic idle detection and suspend-to-RAM with Wake-on-LAN support
- **Metrics Collection**: GPU/CPU utilization, temperature, and power consumption tracking
- **TUI Interface**: Keyboard-driven terminal UI built with Bubble Tea (no mouse required)

## Prerequisites

- **OS**: Ubuntu 24.04 LTS (x86_64)
- **Runtime**: Docker (default) or Podman (best-effort support)
- **Build**: Go 1.22+ (required for building from source)
- **Optional**: NVIDIA GPU with compatible drivers for GPU workloads

## Quick Start

### Production Installation (Ubuntu 24.04)

**Goal**: Get all services running (green status) in ≤10 minutes.

**Prerequisites**:
- Fresh Ubuntu 24.04 LTS installation
- Go 1.22+ installed (`sudo apt install golang-go`)
- Docker installed (`sudo apt install docker.io docker-compose-v2`)
- User added to docker group (`sudo usermod -aG docker $USER`, then logout/login)
- For GPU support: NVIDIA drivers installed (`nvidia-smi` should work)

**Step 1: Download and Build**

```bash
# Download source from latest release
wget https://github.com/polygonschmiede/aistack/archive/refs/tags/v0.1.0.tar.gz
tar -xzf v0.1.0.tar.gz
cd aistack-0.1.0

# OR clone from repository
git clone https://github.com/polygonschmiede/aistack.git
cd aistack

# Build binary
make build

# Install system-wide (requires sudo)
sudo ./install.sh

# Verify installation
aistack version
```

> **Note**: Precompiled binaries will be available in future releases. Currently, you need Go 1.22+ installed to build from source.

**Step 2: Install Services**

```bash
# Install with standard GPU profile (Ollama + Open WebUI + LocalAI)
sudo aistack install --profile standard-gpu

# OR install minimal profile (Ollama only)
sudo aistack install --profile minimal

# Check status
aistack status
```

**Step 3: Verify Services**

```bash
# All services should show "green" health
aistack health

# Access services:
# - Ollama API: http://localhost:11434
# - Open WebUI: http://localhost:3000
# - LocalAI API: http://localhost:8080
```

**Troubleshooting**:
- If services show "red": Check `aistack logs <service>` for errors
- If GPU not detected: Run `aistack gpu-check` to verify NVIDIA stack
- If network errors: Ensure Docker network is up: `docker network ls | grep aistack`
- For detailed diagnostics: `aistack diag` (creates ZIP with logs)

See [OPERATIONS.md](docs/OPERATIONS.md) for detailed troubleshooting playbooks.

### Development Build
Install beforehand
```bash
sudo apt update
sudo apt install golang-go make docker
```

```bash
# Clone the repository
git clone https://github.com/polygonschmiede/aistack.git
cd aistack

# Build (no GPU support)
make build

# OR: Build with GPU/CUDA support (requires nvidia-cuda-toolkit)
sudo apt install nvidia-cuda-toolkit
make build-cuda

# Run locally (no installation)
./dist/aistack

# Or run with go
make run
```

**Note**: The `install.sh` script automatically detects NVIDIA GPU and builds with CUDA support if available.

### Development Commands

```bash
make help          # Show all available commands
make build         # Build binary (no GPU support)
make build-cuda    # Build binary with CUDA/GPU support
make test          # Run tests
make lint          # Run linters
make coverage      # Generate coverage report
```

### CLI Commands

```bash
./aistack                          # Start TUI (default)
./aistack agent                    # Run as background agent service
./aistack idle-check               # Perform idle evaluation (timer-triggered)
./aistack install --profile <name> # Install from profile (standard-gpu, minimal)
./aistack install <service>        # Install specific service (ollama, openwebui, localai)
./aistack start <service>          # Start a service
./aistack stop <service>           # Stop a service
./aistack update <service>         # Update service to latest (with rollback)
./aistack backend <ollama|localai> # Switch Open WebUI backend (restarts service)
./aistack logs <service> [lines]   # Show service logs (default: 100 lines)
./aistack remove <service> [--purge] # Remove a service (keeps data by default)
./aistack status                   # Show status of all services
./aistack gpu-check                # Check GPU and NVIDIA stack
./aistack metrics-test             # Test metrics collection (3 samples)
./aistack wol-check                # Check Wake-on-LAN status
./aistack wol-setup <iface>        # Enable Wake-on-LAN (requires root)
./aistack wol-send <mac> [ip]      # Send Wake-on-LAN magic packet
./aistack version                  # Show version
./aistack help                     # Show all commands
```

## Project Status

Currently implementing foundational epics:

- ✅ **EP-001**: Repository & Tech Baseline (Go + TUI skeleton)
- ✅ **EP-002**: Bootstrap & System Integration (install.sh + systemd)
- ✅ **EP-003**: Container Runtime & Compose Assets (Docker Compose)
- ✅ **EP-004**: NVIDIA Stack Detection (NVML integration)
- ✅ **EP-005**: Metrics & Sensors (CPU/GPU/Power monitoring)
- ✅ **EP-006**: Idle Engine & Autosuspend (Sliding window detection)
- ✅ **EP-007**: Wake-on-LAN Setup (WoL detection, magic packet sender)
- ✅ **EP-008**: Ollama Orchestration (Lifecycle + Update/Rollback)
- ✅ **EP-009**: Open WebUI Orchestration (Backend-Switch: Ollama ↔ LocalAI)
- ✅ **EP-010**: LocalAI Orchestration (Lifecycle + Remove with Volume Handling)

See `docs/features/epics.md` for complete roadmap.

## Architecture

```
aistack/
├── cmd/aistack/           # CLI + TUI entry point
├── internal/              # Private application code
│   ├── agent/            # Background agent runtime + idle orchestration
│   ├── config/           # Config loading, validation, defaults
│   ├── configdir/        # Cross-platform config/state path helpers
│   ├── diag/             # Diagnostics + zip-pack generator
│   ├── gpu/              # GPU detection, NVML helpers, toolkit checks
│   ├── gpulock/          # Exclusive GPU lock + health enforcement
│   ├── idle/             # Idle engine, suspend executor, WoL gating
│   ├── logging/          # Structured JSON logger
│   ├── metrics/          # CPU/GPU metrics collectors + JSONL writer
│   ├── models/           # Ollama / LocalAI model inventory + eviction
│   ├── secrets/          # AES-GCM secret storage for service creds
│   ├── services/         # Compose orchestration & lifecycle helpers
│   ├── tui/              # Bubble Tea UI
│   └── wol/              # Wake-on-LAN setup, relay server, CLI cmds
├── assets/               # systemd units, tmpfiles, logrotate, udev
├── compose/              # Docker Compose templates
└── docs/                 # Documentation and guides
```

## Configuration

System-wide config: `/etc/aistack/config.yaml`
User config: `~/.aistack/config.yaml`

Key settings:
- `container_runtime`: Docker or Podman
- `idle.*`: CPU/GPU thresholds, timeout
- `wol.*`: Wake-on-LAN interface and MAC
- `gpu_lock`: Exclusive GPU access control

Environment overrides:
- `AISTACK_COMPOSE_DIR` — absolute or relative path to packaged Compose bundles (defaults to the binary’s `compose/` directory).
- `AISTACK_LOG_DIR` — writable directory for JSON/JSONL output such as `metrics.log` (defaults to `/var/log/aistack`, then falls back to a temp dir when unavailable).
- `AISTACK_STATE_DIR` — idle state persistence root for developer runs; systemd deployments remain rooted at `/var/lib/aistack`.

## Development

See [AGENTS.md](AGENTS.md) for contributor guidelines and [CLAUDE.md](CLAUDE.md) for AI-assisted development context.

**Testing**:
```bash
go test ./...              # Unit tests
go test ./... -race        # Race detector
go test ./... -cover       # Coverage
```

**Code Style**:
- Run `go fmt ./...` before commits
- Follow guidelines in `docs/cheat-sheets/golangbp.md`
- Use conventional commits (`feat:`, `fix:`, `refactor:`)

## Work Log

All development sessions are tracked in `status.md` for continuity and historical context.

## License

[License Type] - See [LICENSE](LICENSE) for details.

## Roadmap

- **v0.1**: Foundation (TUI, Docker integration, basic GPU detection)
- **v0.2**: Service orchestration (Ollama, Open WebUI, LocalAI)
- **v0.3**: Power management (idle detection, auto-suspend, WoL)
- **v0.4**: Model management and caching
- **v1.0**: Production-ready with full epic implementation

---

**Note**: This is a headless/server tool designed for SSH access, not desktop environments. Designed for nerdy early adopters who understand Linux system administration.
