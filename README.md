# Git Pilot

Git Pilot is a Go-based CLI that helps you review local Git changes, generate AI-assisted commit messages with Groq, split commits file-wise or category-wise, and keep a human approval step before every commit and push.

## Features

- Secure API key storage via OS keychain (macOS Keychain, Linux libsecret, Windows Credential Manager)
- Interactive commit workflow with approval before each commit
- AI-generated commit messages with conventional prefixes such as `feat:`, `fix:`, and `refactor:`
- File-wise commit mode
- AI group-wise commit mode for related changes
- Push and pull confirmation flows
- PR message generation from commit history
- Modern terminal UI with structured previews and summaries

## Installation

### Homebrew (macOS and Linux)

```sh
brew tap mohammadumar-dev/tap
brew install gitpilot
```

### go install (Go developers)

```sh
go install github.com/mohammadumar-dev/gitpilot@latest
```

### One-line installer (Linux and macOS)

Downloads the correct pre-built binary, verifies SHA256 checksum, and installs to `/usr/local/bin`:

```sh
curl -sSfL https://raw.githubusercontent.com/mohammadumar-dev/gitpilot/main/install.sh | sh
```

### Windows (PowerShell)

Downloads, verifies SHA256 checksum, installs to `%LOCALAPPDATA%\Programs\gitpilot`, and adds it to your user PATH automatically:

```powershell
irm https://raw.githubusercontent.com/mohammadumar-dev/gitpilot/main/install.ps1 | iex
```

### Build from source

```sh
git clone https://github.com/mohammadumar-dev/gitpilot.git
cd gitpilot
make build
```

### Prerequisites

- Git (required at runtime)
- A [Groq API key](https://console.groq.com/) for AI commit generation

## Authentication

Store your Groq API key securely using the `auth` command. The key is stored in the OS keychain where available, falling back to `~/.config/gitpilot/credentials` (chmod 0600).

```sh
gitpilot auth login
```

You will be prompted to paste your key — input is hidden and the key is validated before storing.

```sh
gitpilot auth status    # show which source is active
gitpilot auth logout    # remove stored key
```

The environment variable `GROQ_API_KEY` always takes priority over stored keys (useful for CI/CD).

## Configuration

Set or inspect the Groq model:

```sh
gitpilot config groq-model llama-3.3-70b-versatile
gitpilot config show
```

Initialize repository-local Git Pilot settings:

```sh
gitpilot init
```

`init` validates the current Git repository and seeds local Git config values such as `gitpilot.groq-model` and `gitpilot.initialized`.

## Usage

Start interactive mode:

```sh
gitpilot
```

Run a command directly:

```sh
gitpilot status
gitpilot diff
gitpilot commit
gitpilot push
gitpilot pull
gitpilot pr
gitpilot version
gitpilot help
```

`push` and `pull` both show a preview and ask for approval before running the underlying Git command. `pull` uses `git pull --ff-only` to avoid implicit merge commits.

### Commit workflow

```sh
gitpilot commit
```

Git Pilot will:

1. Inspect changed files.
2. Ask whether to commit file-wise or AI group-wise.
3. Generate one commit message at a time.
4. Show the target files and message preview.
5. Ask for `y/n` approval before the actual commit.
6. Offer a push prompt after successful commits.

Direct modes are also supported:

```sh
gitpilot commit file main.go
gitpilot commit group
```

### PR message generation

```sh
gitpilot pr           # uses origin/main as base
gitpilot pr develop   # uses a custom base branch
```

## Example Flow

```text
gitpilot › commit
Select mode (1/2): 2
Approve commit for auth cleanup? [y/N]: y
Approve commit for docs update? [y/N]: n
Push committed changes now? [y/N]: y
```

## Project Structure

```text
.
├── main.go                        # CLI, Git integration, AI prompts, and terminal UI
├── go.mod                         # Go module definition
├── Makefile                       # Developer workflow (build, fmt, clean, release-dry)
├── .goreleaser.yml                # Cross-platform release configuration
├── install.sh                     # Linux/macOS installer
├── install.ps1                    # Windows PowerShell installer
├── install-test.sh                # Local end-to-end installer test (8 tests, no GitHub needed)
├── README.md
├── AGENTS.md
├── LICENSE
└── .github/
    └── workflows/
        └── release.yml            # Publishes release on git tag push
```

## Development

```sh
make build        # build with version from git describe
make fmt          # run gofmt
make clean        # remove binary
make release-dry  # test GoReleaser locally without publishing
```

Or using Go directly:

```sh
go build -ldflags="-X main.version=dev" -o gitpilot .
go run . help
gofmt -w main.go
```

Test the installer locally without a GitHub release:

```sh
sh install-test.sh
```

## Current Limitations

- AI features require a valid Groq API key.
- `pull` uses `git pull --ff-only` to avoid implicit merge commits.
- The tool currently keeps most logic in `main.go`; future refactors can split command, UI, and AI logic into packages.

## Roadmap

- Spinner/progress feedback for AI calls
- Editable AI-generated commit messages before approval
- Richer Git status breakdowns
- Homebrew tap support
- Better non-interactive scripting support

## Contributing

Contributions are welcome. Before opening a pull request:

- run `make fmt`
- run `go build ./...`
- test the command flow you changed

See [`AGENTS.md`](./AGENTS.md) for repository-specific contribution guidance.

## License

This project is licensed under the terms in [`LICENSE`](./LICENSE).
