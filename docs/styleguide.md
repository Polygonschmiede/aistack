# Style Guide

This document defines coding standards and conventions for the aistack project.

## Logging Levels

Use structured logging with appropriate levels:

### debug
- Detailed diagnostic information
- Function entry/exit traces
- Variable values during processing
- Use sparingly; high volume

### info
- Normal operational events
- Service started/stopped
- Configuration loaded
- State transitions
- Default level for production

### warn
- Unusual but handled conditions
- Deprecated feature usage
- Resource approaching limits
- Fallback behavior triggered

### error
- Error conditions requiring attention
- Failed operations
- Exceptions caught and handled
- Should include context and error details

## Error Handling Principles

### 1. Return Errors, Don't Log Them
```go
// Good
func readConfig() (*Config, error) {
    data, err := os.ReadFile("config.yaml")
    if err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }
    return parseConfig(data)
}

// Bad - logging inside helpers
func readConfig() (*Config, error) {
    data, err := os.ReadFile("config.yaml")
    if err != nil {
        log.Error("Failed to read config") // Don't log here!
        return nil, err
    }
    return parseConfig(data)
}
```

Let callers decide how to surface errors (log, display in TUI, return to user).

### 2. Wrap Errors with Context
```go
// Good - adds context at each layer
if err := deployService(name); err != nil {
    return fmt.Errorf("deploy service %q: %w", name, err)
}

// Bad - loses context
if err := deployService(name); err != nil {
    return err
}
```

Use `%w` to maintain error chain for `errors.Is()` and `errors.As()`.

### 3. Handle Errors at Appropriate Level
- Low-level functions: return errors with context
- High-level orchestrators: log and/or display to user
- Main/CLI layer: convert to exit codes and user messages

### 4. Validate Early
```go
// Good - fail fast
func NewConfig(path string) (*Config, error) {
    if path == "" {
        return nil, errors.New("config path required")
    }
    // ... rest of function
}

// Bad - deep nesting
func NewConfig(path string) (*Config, error) {
    if path != "" {
        // ... many lines of code
    } else {
        return nil, errors.New("config path required")
    }
}
```

### 5. Use Custom Error Types for Specific Handling
```go
type NotFoundError struct {
    Resource string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s not found", e.Resource)
}

// Caller can check error type
if errors.As(err, &NotFoundError{}) {
    // Handle not-found specifically
}
```

## Logging Format

All logs must be structured JSON with these fields:

```json
{
  "ts": "2024-01-15T14:30:00Z",
  "level": "info",
  "type": "app.started",
  "message": "Application started",
  "payload": {
    "version": "0.1.0",
    "config": "/etc/aistack/config.yaml"
  }
}
```

### Required Fields
- `ts`: ISO-8601 UTC timestamp
- `level`: One of debug/info/warn/error
- `type`: Event type (dot-separated namespace)
- `message`: Human-readable description

### Optional Fields
- `payload`: Event-specific structured data

### Event Type Naming
Use namespaced event types:
- `app.started`, `app.exited`
- `service.ollama.started`, `service.ollama.health.degraded`
- `gpu.lock.acquired`, `gpu.lock.released`
- `power.suspend.requested`, `power.suspend.done`

## Code Formatting

Follow standard Go conventions:
- Run `gofmt` / `go fmt ./...` before commits
- Use `golangci-lint` for additional checks
- Tabs for indentation (Go default)
- No line length limit (but be reasonable)

## Naming Conventions

- **Exported**: PascalCase (`Model`, `NewLogger`)
- **Unexported**: camelCase (`startTime`, `shouldLog`)
- **Packages**: lowercase, singular (`tui`, `logging`)
- **Interfaces**: noun or adjective (`Reader`, `Loggable`)

## Testing Conventions

- Co-locate tests: `file.go` → `file_test.go`
- Use table-driven tests for multiple cases
- Mark test helpers with `t.Helper()`
- Use meaningful test names: `TestFeature_Scenario`

Example:
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

## Import Organization

Group imports in this order:
1. Standard library
2. External dependencies
3. Internal packages

Separate groups with blank lines:
```go
import (
    "fmt"
    "os"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"

    "aistack/internal/logging"
    "aistack/internal/tui"
)
```

## Comments

- Package comments: describe purpose and usage
- Exported functions: document behavior and edge cases
- Unexported functions: only if complex logic requires explanation
- Avoid obvious comments that restate code

```go
// Good
// parseConfig unmarshals YAML and validates required fields.
// Returns error if file is malformed or validation fails.
func parseConfig(data []byte) (*Config, error) { ... }

// Bad - obvious
// readConfig reads config
func readConfig() (*Config, error) { ... }
```
