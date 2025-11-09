# Build and Test Guide

This guide covers building, testing, and verifying the aistack project.

## Prerequisites

- Go 1.22 or later
- Make (for build automation)
- golangci-lint (optional, for comprehensive linting)

### Installing Go

On Ubuntu 24.04:
```bash
sudo apt update
sudo apt install -y golang-1.22
```

On macOS:
```bash
brew install go
```

Verify installation:
```bash
go version
```

## Quick Start

```bash
# Clone the repository
git clone https://github.com/polygonschmiede/aistack.git
cd aistack

# Download dependencies
go mod download

# Build the binary
make build

# Run the application
./dist/aistack
```

## Build Commands

### Standard Build
```bash
make build
```
Creates a static binary at `dist/aistack`.

Build flags:
- `CGO_ENABLED=0` - No C dependencies
- `-tags netgo` - Pure Go networking
- `-ldflags "-s -w"` - Strip debug symbols for smaller binary

### Run Without Building
```bash
make run
# or directly:
go run ./cmd/aistack
```

### Clean Build Artifacts
```bash
make clean
```

## Testing

### Run All Tests
```bash
make test
# or:
go test ./...
```

### Run Tests with Race Detector
```bash
make race
# or:
go test ./... -race
```

### Generate Coverage Report
```bash
make coverage
```
This creates `coverage.html` which you can open in a browser.

### Run Tests for Specific Package
```bash
go test ./internal/tui -v
go test ./internal/logging -v
```

## Code Quality

### Format Code
```bash
make fmt
# or:
go fmt ./...
```

### Run Static Analysis
```bash
make vet
# or:
go vet ./...
```

### Run All Linters
```bash
make lint
```
This runs `gofmt`, `go vet`, and `golangci-lint` (if installed).

### Install golangci-lint
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Dependency Management

### Add a New Dependency
```bash
# Import in code, then:
go mod tidy
```

### Update Dependencies
```bash
go get -u ./...
go mod tidy
```

### Vendor Dependencies (Optional)
```bash
go mod vendor
```

## Verification (Definition of Done)

### EP-001 Story T-001: Repository Skeleton & Build Pipeline

**DoD 1**: Given a fresh clone, When `make build` runs, Then a static `aistack` binary is created.

```bash
# Clean slate
make clean

# Build
make build

# Verify
ls -lh dist/aistack
file dist/aistack  # Should show "statically linked"
```

**DoD 2**: Given the repo, When `make test` runs, Then all tests pass with exit code 0.

```bash
make test
echo $?  # Should be 0
```

**DoD 3**: Given the repo, When `make lint` runs, Then no lint errors.

```bash
make lint
echo $?  # Should be 0
```

### EP-001 Story T-002: TUI Bootstrap

**DoD 1**: Given `aistack` started, Then an empty frame with title "aistack" appears.

```bash
./dist/aistack
# Should show TUI with "aistack" title
```

**DoD 2**: Given TUI visible, When pressing 'q', Then program exits with code 0.

```bash
./dist/aistack
# Press 'q'
echo $?  # Should be 0
```

**DoD 3**: Given start/exit, Then `app.started` and `app.exited` are logged.

```bash
./dist/aistack 2>&1 | grep "app.started"
./dist/aistack 2>&1 | grep "app.exited"
```

## Continuous Integration

The project includes a Makefile target for full CI workflow:

```bash
make all
```

This runs:
1. `make clean` - Remove old artifacts
2. `make deps` - Download dependencies
3. `make lint` - Code quality checks
4. `make test` - Unit tests
5. `make build` - Create binary

## Troubleshooting

### "go: command not found"
Ensure Go is installed and in your PATH:
```bash
export PATH=$PATH:/usr/local/go/bin
```

### "package not found" errors
Run dependency download:
```bash
go mod download
go mod tidy
```

### Tests failing on fresh clone
Ensure you've downloaded dependencies:
```bash
go mod download
```

### Build fails with "CGO_ENABLED=0"
This is expected for static builds. If you need CGO (e.g., for SQLite), modify the Makefile build flags temporarily.

## Next Steps

After successful build:
1. Review [docs/repo-structure.md](repo-structure.md) for codebase organization
2. Read [docs/styleguide.md](styleguide.md) for coding conventions
3. Check [docs/features/epics.md](features/epics.md) for upcoming features
4. See [CONTRIBUTING.md](../CONTRIBUTING.md) for contribution guidelines
