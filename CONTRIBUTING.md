# Contributing to aistack

Thank you for your interest in contributing to aistack!

## Development Setup

1. Clone the repository
2. Ensure Go 1.22+ is installed
3. Run `make deps` to download dependencies
4. Run `make test` to verify your setup

## Before You Commit

1. **Format your code**: `make fmt`
2. **Run linters**: `make lint`
3. **Run tests**: `make test`
4. **Check race conditions**: `make race` (for concurrency-related changes)

## Commit Guidelines

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New features
- `fix:` - Bug fixes
- `refactor:` - Code restructuring without behavior changes
- `test:` - Test additions or modifications
- `docs:` - Documentation updates
- `chore:` - Build process or auxiliary tool changes

**Example**:
```
feat: add GPU metrics collection via NVML

Implements basic NVML bindings to collect GPU utilization,
memory usage, temperature, and power consumption.

Relates to EP-005 (Metrics & Sensors)
```

## Coding Standards

See [AGENTS.md](AGENTS.md) and `docs/cheat-sheets/golangbp.md` for detailed guidelines:

- Use `gofmt` for formatting (tabs, not spaces)
- Follow standard Go project layout (`cmd/`, `internal/`, `pkg/`)
- Write table-driven tests
- Return errors, don't log inside helpers
- Use interfaces for testability
- Keep exported APIs minimal

## Status Tracking

Record your work session in `status.md` following the existing format:
- Date and time (CET)
- Task description (Aufgabe)
- Approach taken (Vorgehen)
- Current status (Status: In Arbeit / Abgeschlossen)

## Pull Requests

1. Create a feature branch from `main`
2. Make your changes with atomic commits
3. Ensure all tests pass: `make all`
4. Update documentation if needed
5. Submit PR with clear description linking to relevant epic/story

## Testing

- **Unit tests**: Test individual functions and packages
- **Integration tests**: Test component interactions
- **Coverage target**: ≥80% for core packages (`internal/`)

## Epic-Based Development

This project follows epic-based development. See `docs/features/epics.md` for:
- Current epic status
- Story definitions with acceptance criteria
- Architecture decisions and contracts

Before starting significant work, review the relevant epic to understand:
- Dependencies between stories
- Data contracts and API designs
- Acceptance criteria and test plans

## Questions?

- Check `docs/` for guides and references
- Review existing code for patterns
- Ask in issues or discussions

Thank you for contributing!
