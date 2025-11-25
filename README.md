# aistack

> Headless AI service orchestrator with auto-suspend and Wake-on-LAN for power-efficient GPU workloads.

**Status**: Early development (v0.1) - Foundation complete, ready for testing

---

## Table of Contents

1. [What is aistack?](#what-is-aistack)
2. [Quick Start](#quick-start)
3. [CLI Commands](#cli-commands)
4. [Operations Guide](#operations-guide)
5. [Troubleshooting](#troubleshooting)
6. [Configuration](#configuration)
7. [Development](#development)
8. [Architecture](#architecture)

---

## What is aistack?

aistack manages AI services (Ollama, Open WebUI, LocalAI) on Ubuntu 24.04 with:

- **Smart Power Management**: Auto-suspend when idle, Wake-on-LAN for remote wake-up
- **GPU Management**: NVIDIA GPU detection, metrics, and exclusive locking
- **Service Orchestration**: Docker Compose-based service lifecycle management
- **TUI Interface**: Keyboard-driven terminal UI (no mouse required)
- **Metrics & Monitoring**: Track GPU/CPU utilization, temperature, and power consumption

---

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

# Build
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

# Health check
aistack health
```

### Access Services

- **Ollama API**: http://localhost:11434
- **Open WebUI**: http://localhost:3000
- **LocalAI API**: http://localhost:8080

---

## CLI Commands

### Complete Command Reference

```
aistack - AI Stack Management Tool (version 0.1.0-dev)

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
```

### Common Usage Examples

**Service Management**
```bash
# Install and start Ollama
aistack install ollama
aistack start ollama

# Check logs
aistack logs ollama 50

# Update with automatic rollback on failure
aistack update ollama

# Remove but keep data
aistack remove ollama

# Remove and delete all data
aistack remove ollama --purge
```

**Health & Diagnostics**
```bash
# Check overall health
aistack health

# Check GPU status
aistack gpu-check

# Repair unhealthy service
aistack repair openwebui

# Create diagnostic package
aistack diag --output /tmp/debug.zip
```

**Model Management**
```bash
# List models
aistack models list ollama
aistack models list localai

# Download model (Ollama only)
aistack models download ollama llama2

# Delete model
aistack models delete ollama llama2

# Show cache stats
aistack models stats ollama

# Free space by removing oldest model
aistack models evict-oldest ollama
```

**Backend Switching**
```bash
# Switch Open WebUI to use LocalAI backend
aistack backend localai

# Switch back to Ollama
aistack backend ollama
```

**Power Management**
```bash
# Check suspend status
aistack suspend status

# Disable auto-suspend temporarily
aistack suspend disable

# Re-enable
aistack suspend enable
```

---

## Operations Guide

### Service Lifecycle

**Installation**
```bash
# Profile-based (recommended)
aistack install --profile standard-gpu  # Ollama + OpenWebUI + LocalAI
aistack install --profile minimal       # Ollama only

# Individual services
aistack install ollama
aistack install openwebui
aistack install localai
```

**Starting/Stopping**
```bash
# Start service
aistack start ollama

# Stop service
aistack stop ollama

# Restart (stop + start)
aistack stop ollama && aistack start ollama
```

**Removal**
```bash
# Remove service (keeps volumes)
aistack remove ollama

# Remove service and purge all data
aistack remove ollama --purge

# Complete system purge (double confirmation required)
aistack purge --all
aistack purge --all --remove-configs  # Also removes /etc/aistack
```

### Updates & Rollback

**Update Single Service**
```bash
# Update with automatic health check and rollback
aistack update ollama

# Check if update succeeded
aistack health
aistack logs ollama 50
```

**Update All Services**
```bash
# Sequential update: LocalAI → Ollama → Open WebUI
aistack update-all

# Each service updates independently
# Failure in one doesn't affect others
```

**Version Pinning**

Create `/etc/aistack/versions.lock`:
```
# Pin with digest (reproducible)
ollama:ollama/ollama@sha256:abc123...
openwebui:ghcr.io/open-webui/open-webui@sha256:def456...

# Or use specific tags
localai:quay.io/go-skynet/local-ai:v2.8.0
```

Set update policy in `/etc/aistack/config.yaml`:
```yaml
updates:
  mode: pinned  # or "rolling" (default)
```

Check status:
```bash
aistack versions
```

**Manual Rollback**
```bash
# Stop service
aistack stop ollama

# Remove container (keeps data)
docker rm aistack-ollama

# Update versions.lock to previous version
sudo nano /etc/aistack/versions.lock

# Start service with pinned version
aistack start ollama
```

### Backup & Recovery

**Backup Service Data**
```bash
# Stop service first
aistack stop ollama

# Backup volume
docker run --rm \
  -v ollama_data:/data \
  -v $(pwd):/backup \
  ubuntu tar czf /backup/ollama-backup-$(date +%Y%m%d).tar.gz /data

# Start service
aistack start ollama
```

**Restore Service Data**
```bash
# Stop service
aistack stop ollama

# Remove existing volume (CAUTION!)
docker volume rm ollama_data

# Restore from backup
docker run --rm \
  -v ollama_data:/data \
  -v $(pwd):/backup \
  ubuntu tar xzf /backup/ollama-backup-20250125.tar.gz -C /

# Start service
aistack start ollama
```

**Backup Configuration**
```bash
# Backup all configs
sudo tar czf aistack-config-$(date +%Y%m%d).tar.gz \
  /etc/aistack \
  /var/lib/aistack/*.json

# Restore
sudo tar xzf aistack-config-20250125.tar.gz -C /
```

### Performance Tuning

**GPU Utilization**
```bash
# Monitor GPU
watch -n 1 nvidia-smi

# Check GPU lock status
aistack status | grep -A5 "GPU Lock"

# Force unlock if stuck
aistack gpu-unlock
```

**Idle Detection Tuning**

Edit `/etc/aistack/config.yaml`:
```yaml
idle:
  cpu_idle_threshold: 10      # CPU below 10% = idle
  gpu_idle_threshold: 5       # GPU below 5% = idle
  window_seconds: 300         # 5-minute sliding window
  idle_timeout_seconds: 1800  # Suspend after 30 min idle
```

Test settings:
```bash
aistack suspend status
```

**Model Cache Management**
```bash
# List models
aistack models list ollama

# Show cache stats
aistack models stats ollama

# Delete unused model
aistack models delete ollama old-model-name

# Evict oldest to free space
aistack models evict-oldest ollama
```

### Monitoring & Logging

**View Logs**
```bash
# Service logs
aistack logs ollama 100     # Last 100 lines
aistack logs openwebui      # Default: 100 lines

# System logs (if agent running)
sudo journalctl -u aistack-agent -f

# Metrics logs (JSON format)
tail -f /var/log/aistack/metrics.log | jq .
```

**Create Diagnostic Package**
```bash
# Generate diagnostic ZIP (secrets redacted)
aistack diag

# Custom output path
aistack diag --output /tmp/debug.zip

# Exclude logs or config
aistack diag --no-logs
aistack diag --no-config
```

---

## Troubleshooting

### Service Shows Red Status

**Problem**: `aistack status` shows service unhealthy

**Diagnosis**:
```bash
# Check logs
aistack logs <service> 50

# Check container status
docker ps -a | grep aistack

# Check compose status
docker compose -f /usr/share/aistack/compose/<service>.yaml ps
```

**Fix**:
```bash
# Automatic repair
aistack repair <service>

# OR manual restart
aistack stop <service>
aistack start <service>

# Check for port conflicts
sudo netstat -tulpn | grep <port>
# Ollama: 11434, OpenWebUI: 3000, LocalAI: 8080
```

### GPU Not Detected

**Problem**: `aistack gpu-check` reports no GPU

**Diagnosis**:
```bash
# Check NVIDIA driver
nvidia-smi

# Check Docker GPU runtime
docker run --rm --gpus all nvidia/cuda:12.0.0-base-ubuntu22.04 nvidia-smi

# Check NVML library
ldconfig -p | grep nvidia-ml
```

**Fix**:
```bash
# Install NVIDIA drivers
sudo ubuntu-drivers devices
sudo ubuntu-drivers autoinstall
sudo reboot

# Install nvidia-container-toolkit
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | \
  sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
curl -s -L https://nvidia.github.io/libnvidia-container/$distribution/libnvidia-container.list | \
  sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
  sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list
sudo apt-get update
sudo apt-get install -y nvidia-container-toolkit

# Restart Docker
sudo systemctl restart docker

# Re-check
aistack gpu-check
```

### Port Already in Use

**Problem**: Service fails with "port already allocated"

**Diagnosis**:
```bash
# Check what's using the port
sudo lsof -i :11434    # Ollama
sudo lsof -i :3000     # OpenWebUI
sudo lsof -i :8080     # LocalAI
```

**Fix**:
```bash
# Option 1: Stop conflicting service
sudo systemctl stop <conflicting-service>

# Option 2: Change port in compose file
sudo nano /usr/share/aistack/compose/<service>.yaml
# Change ports: "3001:3000" (external:internal)
```

### Service Update Failed

**Problem**: `aistack update` returns error

**Diagnosis**:
```bash
# Check update plan
cat /var/lib/aistack/<service>_update_plan.json

# Check if rollback occurred
aistack logs <service> 100 | grep -i rollback

# Check update policy
aistack versions
```

**Fix**:
```bash
# If policy is "pinned", change to "rolling"
sudo nano /etc/aistack/config.yaml
# Set: updates.mode: rolling

# Force recreate
aistack stop <service>
aistack remove <service>
aistack install <service>
```

### Out of Disk Space

**Problem**: Services fail with I/O errors

**Diagnosis**:
```bash
# Check disk usage
df -h

# Check Docker usage
docker system df

# Check aistack data
du -sh /var/lib/aistack/volumes/*
```

**Fix**:
```bash
# Clean Docker system
docker system prune -a -f

# Evict oldest models
aistack models evict-oldest ollama
aistack models evict-oldest localai

# Remove unused services
aistack remove <service> --purge
```

### Cannot Connect to Docker

**Problem**: "Cannot connect to Docker daemon"

**Fix**:
```bash
# Add user to docker group
sudo usermod -aG docker $USER
# Logout and login again

# Start Docker if not running
sudo systemctl start docker
sudo systemctl enable docker
```

### Health Check Timeout

**Problem**: Service stuck in "yellow" status

**Cause**: Service taking longer to initialize (especially on first run when downloading models)

**Fix**:
```bash
# Follow logs to see progress
aistack logs <service> -f

# Wait for service to fully start
# Ollama first startup may take 1-5 minutes
```

### System Won't Suspend Despite Being Idle

**Problem**: System idle but not suspending

**Diagnosis**:
```bash
# Check suspend status
aistack suspend status

# Check for systemd inhibitors
systemd-inhibit --list
```

**Fix**:
```bash
# Check if suspend is enabled
aistack suspend status

# Enable if disabled
aistack suspend enable

# Update fix script (if old installation)
cd ~/aistack
git pull
sudo bash fix_suspend.sh
```

### Complete System Reset

**Warning**: This removes ALL aistack services and data

```bash
# Purge everything
aistack purge --all --remove-configs --yes

# Clean Docker system
docker system prune -a -f --volumes

# Verify clean slate
aistack status  # Should show "No services installed"

# Reinstall from scratch
aistack install --profile standard-gpu
```

---

## Configuration

### Configuration Files

- **System config**: `/etc/aistack/config.yaml`
- **User config**: `~/.aistack/config.yaml` (overrides system)
- **Version lock**: `/etc/aistack/versions.lock`
- **State directory**: `/var/lib/aistack/`
- **Log directory**: `/var/log/aistack/`

### Configuration Example

`/etc/aistack/config.yaml`:
```yaml
# Container runtime
container_runtime: docker  # or "podman"

# Installation profile
profile: standard-gpu  # or "minimal"

# GPU lock (prevent VRAM conflicts)
gpu_lock: true

# Idle detection & auto-suspend
idle:
  cpu_idle_threshold: 10         # CPU below 10% = idle
  gpu_idle_threshold: 5          # GPU below 5% = idle
  window_seconds: 300            # 5-minute sliding window
  idle_timeout_seconds: 1800     # Suspend after 30 min idle
  min_samples: 30                # Min samples before suspend
  enable_suspend: true           # Enable auto-suspend

# Power estimation
power_estimation:
  baseline_watts: 150  # Baseline power consumption

# Wake-on-LAN
wol:
  interface: eth0
  mac: "AA:BB:CC:DD:EE:FF"

# Logging
logging:
  level: info         # debug, info, warn, error
  output: file        # file or stderr
  directory: /var/log/aistack

# Model cache
models:
  keep_cache_on_uninstall: true

# Update policy
updates:
  mode: rolling  # or "pinned"
```

### Version Locking

`/etc/aistack/versions.lock`:
```
# Use digests for reproducible deployments
ollama:ollama/ollama@sha256:abc123...
openwebui:ghcr.io/open-webui/open-webui@sha256:def456...

# Or specific tags
localai:quay.io/go-skynet/local-ai:v2.8.0
```

### Testing Configuration

```bash
# Test config validity
aistack config test

# Test specific config file
aistack config test /path/to/config.yaml

# Check version lock status
aistack versions
```

---

## Development

### Build & Test

**Prerequisites**:
- Go 1.22+
- Make
- Docker

**Quick Start**:
```bash
# Clone
git clone https://github.com/polygonschmiede/aistack.git
cd aistack

# Download dependencies
go mod download

# Build
make build

# Run without building
go run .
```

**Build Commands**:
```bash
make build         # Build static binary (dist/aistack)
make run           # Run without building
make test          # Run unit tests
make race          # Run tests with race detector
make coverage      # Generate coverage report (coverage.html)
make lint          # Run linters (gofmt, go vet, golangci-lint)
make clean         # Remove build artifacts
make all           # Full CI workflow (clean, deps, lint, test, build)
```

### Code Style Guidelines

**Formatting**:
- Use `gofmt` defaults (tabs for indentation)
- Run `go fmt ./...` before commits
- Never hand-format code

**Naming**:
- Exported: PascalCase (`NewLogger`, `Model`)
- Unexported: camelCase (`startTime`, `shouldLog`)
- Package names: singular, lowercase (`tui`, `logging`)
- Avoid shadowing variables (especially `err`)

**Error Handling**:
- Return errors instead of logging inside helpers
- Wrap with context: `fmt.Errorf("context: %w", err)`
- Let callers decide how to surface errors
- Check ALL returned errors (including cleanup: `Close`, `Remove`, etc.)

**Example**:
```go
// Good - return error with context
func readConfig() (*Config, error) {
    data, err := os.ReadFile("config.yaml")
    if err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }
    return parseConfig(data)
}

// Bad - logging inside helper
func readConfig() (*Config, error) {
    data, err := os.ReadFile("config.yaml")
    if err != nil {
        log.Error("Failed to read config")  // Don't do this!
        return nil, err
    }
    return parseConfig(data)
}
```

**Testing**:
- Co-locate tests: `file.go` → `file_test.go`
- Use table-driven tests
- Mark test helpers with `t.Helper()`
- Target: ≥80% coverage for `internal/` packages

**Example Test**:
```go
func TestLogger_ShouldLog(t *testing.T) {
    tests := []struct {
        name     string
        minLevel Level
        logLevel Level
        want     bool
    }{
        {"debug logs when min is debug", LevelDebug, LevelDebug, true},
        {"info does not log when min is error", LevelError, LevelInfo, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            logger := NewLogger(tt.minLevel)
            got := logger.shouldLog(tt.logLevel)
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Commit Guidelines

Follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New features
- `fix:` - Bug fixes
- `refactor:` - Code restructuring without behavior changes
- `test:` - Test additions/modifications
- `docs:` - Documentation updates
- `chore:` - Build process or auxiliary tool changes

**Example**:
```
feat: add GPU metrics collection via NVML

Implements basic NVML bindings to collect GPU utilization,
memory usage, temperature, and power consumption.

Relates to EP-005 (Metrics & Sensors)
```

### Pull Requests

1. Create feature branch from `main`
2. Make changes with atomic commits
3. Ensure all tests pass: `make all`
4. Update documentation if needed
5. Submit PR with clear description

### Status Tracking

Record work sessions in `status.md`:
- Date and time (CET)
- Task description
- Approach taken
- Current status (In Arbeit / Abgeschlossen)

---

## Architecture

### Project Structure

```
aistack/
├── cmd/aistack/           # CLI entry point
├── internal/              # Core application
│   ├── agent/            # Background agent + orchestration
│   ├── config/           # Configuration management
│   ├── services/         # Docker Compose lifecycle
│   ├── metrics/          # CPU/GPU metrics collection
│   ├── idle/             # Idle detection + suspend
│   ├── gpu/              # GPU detection + NVML
│   ├── gpulock/          # Exclusive GPU locking
│   ├── wol/              # Wake-on-LAN
│   ├── models/           # Model inventory + eviction
│   ├── tui/              # Terminal UI (Bubble Tea)
│   ├── logging/          # Structured JSON logger
│   ├── diag/             # Diagnostics + support bundles
│   └── secrets/          # Encrypted secret store
├── compose/              # Docker Compose templates
├── assets/               # systemd units, logrotate, udev rules
├── docs/                 # Additional documentation
├── dist/                 # Build artifacts
├── Makefile              # Build automation
├── go.mod                # Go module definition
└── README.md             # This file
```

### Key Design Decisions

**Single Binary**: Static linking (`CGO_ENABLED=0`) for portability

**TUI Framework**: Bubble Tea + Lip Gloss (keyboard-only, no mouse)

**Container Runtime**: Docker (default), Podman (best-effort support)

**GPU Management**: NVML bindings for metrics, advisory locking for exclusive access

**Power Management**: systemd integration for suspend/resume, RAPL for CPU power

**Configuration**: YAML-based with system/user merge (`/etc/aistack` + `~/.aistack`)

**Logging**: Structured JSON to `/var/log/aistack/` (logrotate-managed)

**Metrics**: JSONL format with CPU/GPU sampling, RAPL power estimates

**Secrets**: NaCl secretbox encryption (AES-256-GCM)

### Update Architecture

**Update Workflow**:
1. Pull new image
2. Compare image IDs (skip if same)
3. Restart service with new image
4. Wait 5 seconds
5. Health check
6. Rollback if failed

**Rollback Safety**:
- Automatic rollback on health failure
- Update plan persisted to disk
- Old image ID tracked for restoration
- Volume data never touched

**Version Policy**:
- `rolling`: Updates allowed (default)
- `pinned`: Updates blocked, requires manual version.lock edit

### Health Check Architecture

**Multi-stage Health Checks**:
1. Port check (TCP connection)
2. HTTP check (GET / returns 200)
3. Service-specific check (e.g., Ollama `/api/tags`)

**Repair Flow**:
1. Check current health
2. Stop service
3. Remove container (keep volumes)
4. Recreate container
5. Wait 5 seconds
6. Health check
7. Report success/failure

### Idle Detection Architecture

**Sliding Window**:
- Collects CPU/GPU metrics
- 5-minute window (configurable)
- Time-based pruning

**Idle Engine**:
- Three states: `warming_up`, `active`, `idle`
- Gating reasons: `warming_up`, `below_timeout`, `high_cpu`, `high_gpu`, `inhibit`
- Hysteresis prevents flapping

**Suspend Executor**:
- Multi-stage gate checking
- systemd-inhibit detection
- Dry-run mode for testing
- Force mode with `--ignore-inhibitors`

### Model Management Architecture

**Providers**: Ollama (API-based), LocalAI (filesystem-based)

**Inventory**: JSON state tracking (name, size, last access time)

**Eviction Policy**: LRU (Least Recently Used) - evict oldest

**Cache Preservation**: Optional on uninstall (`models.keep_cache_on_uninstall`)

---

## License

[License Type] - See [LICENSE](LICENSE) for details.

---

## Links

- **GitHub**: https://github.com/polygonschmiede/aistack
- **Issues**: https://github.com/polygonschmiede/aistack/issues
- **Discussions**: https://github.com/polygonschmiede/aistack/discussions

---

**Note**: Designed for headless servers with SSH access, not desktop environments. For nerdy early adopters who understand Linux system administration.
