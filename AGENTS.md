# Repository Guidelines

## Project Structure & Module Organization
This repository is a small Go CLI application. The current entrypoint is [`main.go`](/home/mohammad-umar/Documents/GitHub/gitpilot/main.go), which contains command dispatch, interactive mode, and Git command helpers. Module metadata lives in [`go.mod`](/home/mohammad-umar/Documents/GitHub/gitpilot/go.mod). The top-level [`README.md`](/home/mohammad-umar/Documents/GitHub/gitpilot/README.md) is minimal and should be updated when user-facing behavior changes.

If the project grows, keep CLI wiring in `main.go` thin and move command logic into focused packages such as `internal/commands` or `internal/git`.

## Build, Test, and Development Commands
Use standard Go tooling from the repository root:

- `go run .` runs the interactive CLI locally.
- `go run . status` runs a single command without entering interactive mode.
- `go build -o gitpilot .` builds the binary.
- `go test ./...` runs all tests once test files are added.
- `gofmt -w .` formats Go source files in place.

Run `gofmt` before opening a pull request.

## Coding Style & Naming Conventions
Follow default Go conventions: tabs for indentation, exported names in `PascalCase`, unexported helpers in `camelCase`, and short package names. Keep functions focused; `main.go` currently mixes CLI and Git logic, so new work should prefer small helpers or new packages rather than longer switch blocks.

Use Go’s standard formatter instead of manual alignment. Prefer descriptive command names such as `executeStatus` and data types such as `FileChange`.

## Testing Guidelines
There is no test suite yet. Add table-driven unit tests in `*_test.go` files alongside the code they cover. Use Go’s `testing` package and run `go test ./...` before submitting changes.

For Git command execution, prefer tests around helper functions and error handling rather than fragile end-to-end shell assumptions.

## Commit & Pull Request Guidelines
Recent commit messages use imperative mood and concise summaries, for example: `Add initial implementation of Git Pilot CLI with command handling` and `Refactor command handling to support interactive mode and improve user prompts`.

Keep commits focused and descriptive. Pull requests should include:

- A short description of the behavioral change.
- Any manual test steps used, such as `go run .` or `go build -o gitpilot .`.
- Terminal output or screenshots only when CLI behavior or prompts changed.
