# Repository Guidelines

## Project Structure & Module Organization

This repository is a single-binary Go CLI. [`main.go`](./main.go) contains command dispatch, terminal UI helpers, Git integration, Groq API calls, credential storage, and commit-planning logic. Module metadata lives in [`go.mod`](./go.mod). Distribution is handled by [`Makefile`](./Makefile), [`.goreleaser.yml`](./.goreleaser.yml), and [`.github/workflows/release.yml`](./.github/workflows/release.yml).

Keep new behavior consistent with the existing approval-driven commit and push flow.

## Build & Development Commands

```sh
make build        # build binary with version from git describe
make fmt          # run gofmt on main.go
make clean        # remove local binary
make release-dry  # test GoReleaser build locally without publishing
```

Or with Go directly:

```sh
go build -ldflags="-X main.version=dev" -o gitpilot .
go run . help
gofmt -w main.go
go build ./...
```

If Go cache writes fail in restricted environments, use `GOCACHE=/tmp/gocache go build ./...`.

## Coding Style & Naming Conventions

Follow standard Go formatting and naming: tabs, `PascalCase` for exported identifiers, `camelCase` for internal helpers. Prefer focused helpers for UI (`printPanel`, `printSection`), Git operations, and AI prompts instead of growing one large command function further.

Terminal UX matters here. Preserve the modern CLI style: structured panels, concise status messages, and explicit confirmation prompts before any mutating Git action.

## Authentication & Credential Storage

API key resolution order (highest to lowest priority):

1. `GROQ_API_KEY` environment variable — for CI/CD
2. OS keychain — macOS Keychain (`security` CLI), Linux GNOME Keyring (`secret-tool`), Windows Credential Manager (PowerShell)
3. `~/.config/gitpilot/credentials` — chmod 0600 fallback file

The `auth` command manages keys:

```sh
gitpilot auth login    # hidden input, validates key, stores in keychain or credentials file
gitpilot auth logout   # removes from keychain and credentials file
gitpilot auth status   # shows active source and masked key
```

Do not store the API key in git config (`gitpilot.groq-api-key` is deprecated). Never log or print the key in full — use `maskKey()` for display.

## Configuration Notes

- `GROQ_MODEL` env var or `gitpilot.groq-model` git config key sets the model (default: `llama-3.3-70b-versatile`)
- `gitpilot init` writes `gitpilot.initialized` and `gitpilot.groq-model` to local git config
- `gitpilot config show` displays active key source and model

## Testing Guidelines

There is no automated test suite yet. Validate changes with targeted manual runs:

```sh
gitpilot version
gitpilot auth login
gitpilot auth status
gitpilot auth logout
gitpilot help
gitpilot diff
gitpilot init
gitpilot pull
gitpilot push
```

Test the installer end-to-end without a GitHub release:

```sh
sh install-test.sh
```

For commit-flow changes, test both missing-key behavior and an approval/cancel path. Add `*_test.go` files for pure helpers where practical.

## Release Process

Releases are fully automated. On every pushed tag matching `v*`, GitHub Actions runs GoReleaser which builds 5 binaries (linux/darwin amd64+arm64, windows amd64), generates `checksums.txt`, and publishes a GitHub Release.

```sh
git tag v1.x.x
git push origin v1.x.x
```

To test the release pipeline locally without publishing:

```sh
make release-dry
```

## Commit & Pull Request Guidelines

Use short imperative commit subjects with conventional prefixes: `feat:`, `fix:`, `refactor:`, `chore:`, `docs:`, `ci:`. PRs should include the user-facing behavior change, the commands used for verification, and terminal snippets when CLI output changed.
