# Repository Guidelines

## Project Structure & Module Organization
This repository is a single-binary Go CLI. [`main.go`](/home/mohammad-umar/Documents/GitHub/gitpilot/main.go) currently contains command dispatch, terminal UI helpers, Git integration, Groq API calls, and commit-planning logic. Module metadata lives in [`go.mod`](/home/mohammad-umar/Documents/GitHub/gitpilot/go.mod); top-level docs live in [`README.md`](/home/mohammad-umar/Documents/GitHub/gitpilot/README.md). Keep new behavior consistent with the existing approval-driven commit and push flow.

## Build, Test, and Development Commands
Use standard Go commands from the repo root:

- `go run .` starts the interactive CLI.
- `go run . help` prints current command usage.
- `go run . init` initializes repo-local Git Pilot settings.
- `go run . pull` previews and confirms `git pull --ff-only`.
- `go build -o gitpilot .` builds the binary.
- `gofmt -w main.go` formats the source.

If Go cache writes fail in restricted environments, use `GOCACHE=/tmp/gocache go build ./...`.

## Coding Style & Naming Conventions
Follow standard Go formatting and naming: tabs, `PascalCase` for exported identifiers, `camelCase` for internal helpers. Prefer focused helpers for UI (`printPanel`, `printSection`), Git operations, and AI prompts instead of growing one large command function further.

Terminal UX matters here. Preserve the modern CLI style: structured panels, concise status messages, and explicit confirmation prompts before any mutating Git action.

## Testing Guidelines
There is no automated test suite yet. Validate changes with targeted manual runs such as:

- `go run . help`
- `go run . diff`
- `go run . init`
- `go run . pull`
- `go run . push`

For commit-flow changes, test both missing-key behavior and an approval/cancel path. Add `*_test.go` files for pure helpers where practical.

## Configuration Notes
Groq settings are read from `GROQ_API_KEY`, `GROQ_MODEL`, or local Git config keys `gitpilot.groq-api-key` and `gitpilot.groq-model`. `init` also writes `gitpilot.initialized`. Avoid changing these keys without updating the CLI and README together.

## Commit & Pull Request Guidelines
Use short imperative commit subjects, preferably conventional prefixes like `feat:`, `fix:`, or `refactor:`. PRs should include the user-facing behavior change, the commands used for verification, and screenshots or terminal snippets when the CLI output changed.
