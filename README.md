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
- **Optional**: NVIDIA GPU with compatible drivers for GPU workloads
- **Go**: 1.22+ (for development)

## Quick Start

### Development Build

```bash
# Clone the repository
git clone https://github.com/yourusername/aistack.git
cd aistack

# Build
make build

# Run
./dist/aistack
```

### Development Commands

```bash
make help          # Show all available commands
make build         # Build binary
make test          # Run tests
make lint          # Run linters
make run           # Run directly with go run
make coverage      # Generate coverage report
```

## Project Status

Currently implementing foundational epics:

- ✅ **EP-001**: Repository & Tech Baseline (Go + TUI skeleton)
- 🚧 **EP-002**: Bootstrap & System Integration
- 🚧 **EP-003**: Container Runtime & Compose Assets
- 📋 **EP-004**: NVIDIA Stack Detection
- 📋 **EP-005**: Metrics & Sensors

See `docs/features/epics.md` for complete roadmap.

## Architecture

```
aistack/
├── cmd/aistack/           # CLI entry point
├── internal/              # Private application code
│   ├── installer/        # Bootstrap and system setup
│   ├── services/         # Container lifecycle management
│   ├── power/            # Power monitoring and idle detection
│   ├── metrics/          # GPU/CPU metrics collection
│   └── diag/             # Diagnostics and health checks
├── assets/               # systemd units, configs
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
