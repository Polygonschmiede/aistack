# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Project Overview

aistack is a Go-based CLI tool for managing AI services (Ollama, Open WebUI, LocalAI) with container orchestration, GPU management, power monitoring, and auto-suspend capabilities. Target platform: Ubuntu 24.04 Linux with optional NVIDIA GPU support.

## Quick Reference

**Complete documentation is in README.md** - refer to it for:
- CLI commands and usage examples
- Operations guide (install, update, backup, monitoring)
- Troubleshooting procedures
- Configuration examples
- Development setup and code style
- Architecture and design decisions

## Development Commands

```bash
# Build & Run
go run .              # Quick test
make build            # Compile binary
make test             # Run tests
make lint             # Code quality checks

# Before committing
go fmt ./...          # Format code (required)
go vet ./...          # Static analysis
make test             # Ensure tests pass
```

## Code Quality Rules

**Linting (strictly enforced in CI)**:
- No variable shadowing (especially `err` - reuse with plain assignment)
- Check ALL returned errors (including `Close`, `Remove`, `ReadAll`)
- Factor shared code into helpers (cyclomatic complexity ≤15)
- Move repeated literals to constants
- Directory permissions must be ≤0750

**Error Handling**:
- Return errors, don't log inside helpers
- Wrap with context: `fmt.Errorf("context: %w", err)`
- Let callers decide how to surface errors

**Testing**:
- Table-driven tests with clear scenario names
- Use `t.Helper()` for shared assertions
- Target: ≥80% coverage for `internal/` packages

## Project Structure

```
aistack/
├── cmd/aistack/       # CLI entry point
├── internal/          # Core packages
│   ├── agent/        # Background agent coordination
│   ├── config/       # Configuration management
│   ├── services/     # Docker Compose lifecycle
│   ├── metrics/      # CPU/GPU metrics collection
│   ├── idle/         # Idle detection + suspend
│   ├── gpu/          # NVIDIA detection + NVML
│   ├── wol/          # Wake-on-LAN
│   └── models/       # Model inventory/eviction
├── compose/          # Service templates
└── README.md         # Complete documentation
```

## Commit Guidelines

Use Conventional Commits:
- `feat:` - New features
- `fix:` - Bug fixes
- `refactor:` - Code restructuring
- `test:` - Test changes
- `docs:` - Documentation updates

## Status Tracking

Record all work sessions in `status.md` with:
- Task description
- Approach taken
- Current status (in progress / completed)
- Date and time (CET)

## Important Notes

- **Single source of truth**: All user/dev documentation is in `README.md`
- **Build targets**: Linux (Ubuntu 24.04) on amd64
- **Static binary**: `CGO_ENABLED=0` for portability
- **Idempotency**: Install/uninstall/repair operations must be idempotent
- **No secrets in docs**: Use placeholders, never real credentials
- **Test coverage**: Core packages (`internal/`) must maintain ≥80% coverage

---

For everything else, see **README.md** - it's the complete guide!
