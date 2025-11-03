# Repository Structure

This document describes the directory structure and organization of the aistack project.

## Directory Layout

```
aistack/
├── cmd/                    # Command-line applications
│   └── aistack/           # Main CLI entry point
│       └── main.go        # Application bootstrap and TUI initialization
│
├── internal/              # Private application code
│   ├── tui/              # Terminal User Interface components
│   │   ├── model.go      # Bubble Tea model (state, update, view)
│   │   └── model_test.go # TUI unit tests
│   │
│   ├── logging/          # Structured logging
│   │   ├── logger.go     # JSON logger implementation
│   │   └── logger_test.go # Logging tests
│   │
│   ├── installer/        # Bootstrap and system setup logic
│   ├── services/         # Container service lifecycle management
│   ├── power/            # Power monitoring and idle detection
│   ├── metrics/          # GPU/CPU metrics collection (NVML, RAPL)
│   ├── diag/             # Diagnostics and health checks
│   └── update/           # Update and rollback mechanisms
│
├── assets/               # Static assets
│   ├── systemd/         # systemd unit files (future)
│   └── udev/            # udev rules (future)
│
├── compose/              # Docker Compose templates (future)
│
├── docs/                 # Documentation
│   ├── features/        # Feature specifications and epics
│   ├── cheat-sheets/    # Quick reference guides
│   └── repo-structure.md # This file
│
├── dist/                 # Build output directory (generated)
│
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
Entry point for the CLI application. Contains main() which:
- Initializes structured logging
- Creates and runs the Bubble Tea TUI
- Logs app.started and app.exited events

### internal/tui
Terminal User Interface built with Bubble Tea and Lip Gloss:
- Keyboard-only navigation (no mouse)
- High-contrast color scheme
- Quit via 'q' or Ctrl+C

### internal/logging
Structured JSON logging to stderr:
- ISO-8601 timestamps
- Event types and payloads
- Configurable log levels (debug, info, warn, error)

### internal/* (future modules)
Placeholder directories for upcoming features:
- **installer**: Bootstrap scripts and system integration
- **services**: Container orchestration (Ollama, Open WebUI, LocalAI)
- **power**: Idle detection and auto-suspend
- **metrics**: GPU/CPU monitoring
- **diag**: Health checks and diagnostics
- **update**: Binary and container updates

## Build Targets

See `Makefile` for available targets:
- `make build` - Create static binary in dist/
- `make test` - Run unit tests
- `make lint` - Run code quality checks
- `make run` - Execute application locally
- `make clean` - Remove build artifacts

## Testing Strategy

- Unit tests co-located with source files (*_test.go)
- Table-driven tests preferred
- Target: ≥80% coverage for core packages (internal/*)
- Run with: `make test` or `go test ./...`

## Static Binary

The binary is built with:
- `CGO_ENABLED=0` - No C dependencies
- `-tags netgo` - Pure Go networking
- `-ldflags "-s -w"` - Strip debug symbols

This produces a fully static binary deployable on any Linux system without dependencies.
