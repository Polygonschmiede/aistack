# aistack

> Headless AI service orchestrator with auto-suspend and Wake-on-LAN for power-efficient GPU workloads.

**Status**: Early development (v0.1) - Foundation complete, ready for testing

## What is aistack?

aistack manages AI services (Ollama, Open WebUI, LocalAI) on Ubuntu 24.04 with:

- **Smart Power Management**: Auto-suspend when idle, Wake-on-LAN for remote wake-up
- **GPU Management**: NVIDIA GPU detection, metrics, and exclusive locking
- **Service Orchestration**: Docker Compose-based service lifecycle management
- **TUI Interface**: Keyboard-driven terminal UI (no mouse required)
- **Metrics & Monitoring**: Track GPU/CPU utilization, temperature, and power consumption

## Quick Start

### Prerequisites

- Ubuntu 24.04 LTS (x86_64)
- Docker (or Podman)
- Go 1.22+ (for building from source)
- Optional: NVIDIA GPU with drivers

### Installation

```bash
# Clone repository
git clone https://github.com/polygonschmiede/aistack.git
cd aistack

# Build (auto-detects CUDA)
make build

# Install system-wide
sudo ./install.sh

# Verify
aistack version
```

### Install Services

```bash
# Install all services (Ollama + Open WebUI + LocalAI)
sudo aistack install --profile standard-gpu

# OR minimal (Ollama only)
sudo aistack install --profile minimal

# Check status
aistack status

# All services should show green
aistack health
```

### Access Services

- **Ollama API**: http://localhost:11434
- **Open WebUI**: http://localhost:3000
- **LocalAI API**: http://localhost:8080

## Usage

### TUI (Interactive Mode)

```bash
aistack                    # Launch interactive TUI
```

Navigate with arrow keys or j/k, press numbers for shortcuts, q to quit.

### Common Commands

**Service Management:**
```bash
aistack status                        # Show all services
aistack install <service>             # Install service
aistack start/stop <service>          # Start/stop service
aistack remove <service>              # Remove (keeps data)
aistack remove <service> --purge      # Remove with data
aistack update <service>              # Update with auto-rollback
aistack logs <service> [lines]        # View logs
```

**System Management:**
```bash
aistack health [--save]               # Health check
aistack repair <service>              # Repair unhealthy service
aistack backend <ollama|localai>      # Switch Open WebUI backend
aistack gpu-check                     # Check GPU status
aistack diag                          # Create diagnostic ZIP
```

**Power Management:**
```bash
aistack agent                         # Run as background service
aistack idle-check                    # Manual idle check
aistack wol-check                     # Check Wake-on-LAN
aistack wol-setup <interface>         # Enable Wake-on-LAN
aistack wol-send <mac> [ip]           # Send magic packet
```

**Configuration:**
```bash
aistack config test [path]            # Validate config
aistack versions                      # Show version lock status
```

See `aistack help` for all commands.

## Configuration

System config: `/etc/aistack/config.yaml`
User config: `~/.aistack/config.yaml`

Example:
```yaml
container_runtime: docker
profile: standard-gpu
gpu_lock: true

idle:
  cpu_idle_threshold: 10
  gpu_idle_threshold: 5
  idle_timeout_seconds: 1800

wol:
  interface: eth0
  mac: "AA:BB:CC:DD:EE:FF"

updates:
  mode: rolling  # or "pinned"
```

See `config.yaml.example` for all options.

### Version Locking

Pin service versions in `/etc/aistack/versions.lock`:

```
# Use digests for reproducible deployments
ollama:ollama/ollama@sha256:abc123...
openwebui:ghcr.io/open-webui/open-webui@sha256:def456...

# Or specific tags
localai:quay.io/go-skynet/local-ai:v2.8.0
```

## Development

### Build & Test

```bash
make build         # Build (auto-detects CUDA)
make build-no-cuda # Force build without CUDA
make test          # Run tests
make race          # Race detector
make coverage      # Coverage report
make lint          # Linters
make run           # Run without building
```

### Project Structure

```
aistack/
├── cmd/aistack/           # CLI entry point
├── internal/              # Core application
│   ├── agent/            # Background agent + orchestration
│   ├── config/           # Configuration
│   ├── services/         # Docker Compose lifecycle
│   ├── metrics/          # CPU/GPU metrics collection
│   ├── idle/             # Idle detection + suspend
│   ├── gpu/              # GPU detection + NVML
│   ├── wol/              # Wake-on-LAN
│   ├── tui/              # Terminal UI
│   └── ...               # (diag, secrets, logging, etc.)
├── compose/              # Docker Compose templates
└── docs/                 # Documentation
```

### Code Style

- Run `go fmt ./...` before commits
- Use conventional commits: `feat:`, `fix:`, `refactor:`, `docs:`
- See [AGENTS.md](AGENTS.md) for contributor guidelines
- See [CLAUDE.md](CLAUDE.md) for AI development context

## Documentation

- **[OPERATIONS.md](docs/OPERATIONS.md)**: Operations guide for administrators
- **[POWER_AND_WOL.md](docs/POWER_AND_WOL.md)**: Power management & Wake-on-LAN guide
- **[docs/features/epics.md](docs/features/epics.md)**: Feature roadmap

## Troubleshooting

**Services show red:**
```bash
aistack logs <service>     # Check logs
aistack repair <service>   # Attempt repair
```

**GPU not detected:**
```bash
aistack gpu-check          # Verify NVIDIA stack
nvidia-smi                 # Check drivers
```

**Network issues:**
```bash
docker network ls | grep aistack   # Check network
aistack diag                       # Generate diagnostic package
```

See [OPERATIONS.md](docs/OPERATIONS.md) for detailed troubleshooting.

## Roadmap

- ✅ Foundation (TUI, Docker integration, GPU detection)
- ✅ Service orchestration (Ollama, Open WebUI, LocalAI)
- ✅ Power management (idle detection, auto-suspend, WoL)
- 🚧 Model management and caching
- 🚧 Advanced monitoring and alerting
- 🎯 v1.0: Production-ready release

## License

[License Type] - See [LICENSE](LICENSE) for details.

---

**Note**: Designed for headless servers with SSH access, not desktop environments. For nerdy early adopters who understand Linux system administration.
