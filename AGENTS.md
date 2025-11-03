# Repository Guidelines

## Project Structure & Module Organization
- `cmd/aistack/main.go` provides the CLI entry point; grow feature packages under `internal/` or `pkg/` as functionality expands.
- `go.mod` defines the module name and pinned dependencies; update it with `go mod tidy` whenever imports change.
- `docs/features/` captures product direction (`features.md`, `epics.md`); sync engineering plans with these docs before large changes.

## Build, Test, and Development Commands
- `go run .` — execute the CLI locally for quick smoke checks.
- `go build ./...` — ensure every package compiles; run before pushing.
- `go test ./...` — execute the unit test suite; add `-race` for concurrency-sensitive work.
- `go fmt ./...` and `go vet ./...` — enforce formatting and static checks; run both prior to review.

## Coding Style & Naming Conventions
- Rely on `gofmt` defaults (tabs for indentation, newline at EOF); never hand-format.
- Use PascalCase for exported identifiers and camelCase for package-local names; keep package names singular and lowercase.
- Group related logic into focused packages; avoid cyclic imports and keep public APIs minimal.
- Favor clear guard clauses over deeply nested conditionals; document non-obvious business rules with brief comments.
- Avoid shadowing existing variables (especially `err`); if an `err` is already in scope, reuse it with `err = call()` or declare it once before loops instead of `if err := ...`—the `shadow` linter runs in CI.
- Check every returned error, even from cleanup helpers (e.g. `Close`, `Remove`, `io.ReadAll`, `fmt.Scanln`); propagate or log them so `errcheck` stays green.
- Break up functions before they exceed 15 branches; factor shared logic into helpers to satisfy `gocyclo` and avoid duplication (`dupl`).
- Pull repeated literals (paths/status strings) into constants to keep `goconst` happy and centralize maintenance.
- When creating directories, stick to permissions ≤0750 to appease `gosec` checks.

## Testing Guidelines
- Use Go’s standard `testing` package with table-driven tests for input/output coverage.
- Place test files alongside code as `*_test.go`; mirror the package under test.
- Aim for meaningful assertions that cover error branches and boundary inputs; add regression tests for every reported defect.
- Capture external dependencies behind interfaces to allow lightweight fakes in tests.

## Commit & Pull Request Guidelines
- No shared Git history is present; follow Conventional Commits (`feat:`, `fix:`, `refactor:`) to aid changelog automation.
- Keep commits scoped to a single concern with passing tests; include rationale in the body when behavior shifts.
- Pull requests should link the relevant doc or issue, outline testing performed, and attach screenshots or logs for user-visible changes.
- Request feedback early for cross-cutting updates and note any follow-up tasks explicitly.

## Status Tracking & Knowledge Sources
- Record every work session in `status.md` as soon as you start; each entry must state the assigned task, current progress, and how the task was finished before you leave the branch.
- Review `docs/cheat-sheets/golangbp.md` prior to Go changes to stay aligned with the local best-practices summary.
- Leverage the curated references in `docs/cheat-sheets/` whenever you need quick reminders on tooling, networking, or shell workflows.
- When new processes or lessons emerge, append them to `status.md` and cross-link supporting guides so future contributors can follow the same path.
