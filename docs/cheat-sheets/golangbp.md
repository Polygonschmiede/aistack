# Go Best Practices

This guide condenses the patterns we follow when writing Go inside `aistack`. Start here before touching any Go code and keep the official [Effective Go](https://go.dev/doc/effective_go) close for deeper guidance.

## Style & Formatting
- Run `gofmt` (or `go fmt ./...` at package scope) before every commit; never hand-tune alignment.
- Use tabs for indentation, keep lines readable, and prefer early returns to deep nesting.
- Comments immediately above exported identifiers form the package documentation; write full sentences that start with the identifier name.

## Naming & Visibility
- Package names stay short, lowercase, and match their directory (e.g. `internal/metrics` ⇒ package `metrics`).
- Exported types, funcs, and methods use PascalCase; unexported items use camelCase.
- Keep API surfaces minimal—prefer returning interfaces defined by the consumer rather than exposing broad structs.

## Project Layout
- CLI entry points live in `cmd/`; shared logic belongs under `internal/` or `pkg/` when it needs to be reusable.
- Group platform-specific code with build tags rather than scatter conditional logic.
- Keep configuration parsing, I/O, and business logic separated to simplify testing.

## Testing Patterns
- Co-locate tests as `*_test.go` files in the same package.
- Favour table-driven tests and helper functions like `t.Helper()` for shared assertions.
- Run `go test ./... -race` for concurrency-sensitive changes and capture coverage with `-cover`.

## Error Handling
- Return errors instead of logging inside helpers; let callers decide how to surface issues.
- Wrap contextual details using `fmt.Errorf("context: %w", err)` to preserve the original cause.
- Use sentinel errors or custom types sparingly—prefer interfaces for behaviour.

## Concurrency
- Prefer channels to share data, but fall back to mutexes when sharing mutable state is simpler.
- Close channels from the sender side only, and guard goroutines with `context.Context` for cancellation.
- Use `sync.WaitGroup` for lifecycle management; avoid goroutine leaks by always signalling completion.

## Tooling & Checks
- Run `go vet ./...` and `golangci-lint run` (see `docs/cheat-sheets/`) before pushing.
- Keep dependencies tidy with `go mod tidy` and record any manual replacements in `status.md`.
- Profile hot paths with `go test -run=NONE -bench=. -benchmem` when performance matters.

## Further References
- `docs/cheat-sheets/` provides quick reminders for Makefiles, networking, and shell workflows.
- Update this guide whenever we adopt new conventions so future work stays consistent.
