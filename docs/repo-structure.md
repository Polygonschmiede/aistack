# Repository Structure

This document describes the directory structure and organization of the aistack project.

## Directory Layout

```
aistack/
├── cmd/
│   └── aistack/           # Main CLI/TUI entry point
│       └── main.go
├── internal/
│   ├── agent/            # Background agent + idle orchestration
│   ├── config/           # Config loading, defaults, validation
│   ├── configdir/        # OS-specific config/state paths
│   ├── diag/             # Diagnostics + support bundles
│   ├── gpu/              # GPU detection + NVML helpers
│   ├── gpulock/          # Exclusive GPU locking
│   ├── idle/             # Idle engine + suspend executor
│   ├── logging/          # Structured JSON logger
│   ├── metrics/          # CPU/GPU metrics collection
│   ├── models/           # Model inventory + eviction logic
│   ├── secrets/          # Encrypted secret store
│   ├── services/         # Compose orchestration helpers
│   ├── tui/              # Bubble Tea UI
│   └── wol/              # Wake-on-LAN + relay server
├── assets/
│   ├── systemd/          # Agent/timer/tui units
│   ├── tmpfiles.d/       # Permission fixes (e.g., RAPL)
│   ├── logrotate/        # Log rotation policies
│   └── udev/             # Wake-on-LAN & RAPL rules
├── compose/              # Docker Compose templates (Ollama, LocalAI, OpenWebUI, shared)
├── docs/
│   ├── features/         # Feature specs & epics
│   ├── cheat-sheets/     # Quick reference guides
│   ├── reports/          # Audits & postmortems
│   └── repo-structure.md
├── dist/                 # Generated build artifacts
├── go.mod                # Go module definition
├── go.sum                # Go module checksums
├── Makefile              # Build automation
├── README.md             # Project overview
├── CLAUDE.md             # AI assistant instructions
├── AGENTS.md             # Contributor guidelines
└── status.md             # Work session log
```

## Key Modules

### cmd/aistack
CLI entry point hosting all subcommands (`install`, `status`, `agent`, `idle-check`, WoL helpers, diagnostics, etc.). It wires together services, idle logic, and the Bubble Tea TUI.

### internal packages
- **agent** — Supervises idle detection, metrics sampling, and TUI/CLI coordination when running as a systemd service.
- **config / configdir** — Loads YAML configuration, applies defaults, and resolves system/user config paths.
- **services** — Manages Docker Compose stacks (Ollama, Open WebUI, LocalAI), including install/update/remove flows and backend switching.
- **models** — Tracks downloaded models, volume usage, and eviction logic shared between Ollama and LocalAI.
- **metrics** — Collects CPU/GPU utilization, NVML stats, and RAPL-derived power estimates; writes JSONL logs used by idle logic and diagnostics.
- **idle** — Implements the sliding window idle engine, gating reasons, and suspend executor invoked by the timer/CLI.
- **gpulock** — Ensures only one workload owns the GPU at a time; integrates with services and CLI commands.
- **gpu** — Hardware detection helpers (NVML, CUDA toolkit presence, driver validation).
- **wol** — Wake-on-LAN setup, relay HTTP server, and CLI helpers for sending/testing WoL packets.
- **logging** — Structured JSON logger shared by CLI, agent, and diagnostics.
- **diag** — Diagnostic bundle creation (log collection, redaction, zip packaging).
- **secrets** — AES-GCM encrypted credential storage for services and relay endpoints.
- **tui** — Bubble Tea model/view powering the interactive dashboard.

## Build Targets

See `Makefile` for available targets:
- `make build` - Create static binary in dist/
- `make test` - Run unit tests
- `make lint` - Run code quality checks
- `make run` - Execute application locally
- `make clean` - Remove build artifacts

## Testing Strategy

- Unit tests co-located with source files (*_test.go)
- Table-driven tests preferred with table names covering scenario + expectation
- Target: ≥80% coverage for core packages (internal/*); see `AUDIT_REPORT.md` for current numbers
- Run with: `make test`, `make race`, or `go test ./...`

## Static Binary

The binary is built with:
- `CGO_ENABLED=0` - No C dependencies
- `-tags netgo` - Pure Go networking
- `-ldflags "-s -w"` - Strip debug symbols

This produces a fully static binary deployable on any Linux system without dependencies.
