# Git Pilot

Git Pilot is a Go-based CLI that helps you review local Git changes, generate AI-assisted commit messages with Groq, split commits file-wise or category-wise, and keep a human approval step before every commit and push.

## Features

- Interactive commit workflow with approval before each commit
- AI-generated commit messages with conventional prefixes such as `feat:`, `fix:`, and `refactor:`
- File-wise commit mode
- AI group-wise commit mode for related changes
- Push and pull confirmation flows
- Local repository configuration for Groq model and API key source detection
- Modern terminal UI with structured previews and summaries

## Installation

### Prerequisites

- Go `1.26.2` or newer
- Git
- A Groq API key for AI commit generation

### Build

```bash
git clone https://github.com/mohammadumar-dev/gitpilot.git
cd gitpilot
go build -o gitpilot .
```

### Run without building

```bash
go run .
```

## Configuration

You can provide the Groq API key in either of these ways:

```bash
export GROQ_API_KEY="your-groq-api-key"
```

or:

```bash
go run . config groq-key your-groq-api-key
```

Set or inspect the model:

```bash
go run . config groq-model llama-3.3-70b-versatile
go run . config show
```

Initialize repository-local Git Pilot settings:

```bash
go run . init
```

`init` validates the current Git repository and seeds local Git config values such as `gitpilot.groq-model` and `gitpilot.initialized`.

## Usage

Start interactive mode:

```bash
go run .
```

Common commands:

```bash
go run . status
go run . diff
go run . commit
go run . push
go run . pull
go run . help
```

`push` and `pull` both show a preview and ask for approval before running the underlying Git command. `pull` uses `git pull --ff-only` to avoid implicit merge commits.

### Commit workflow

Run:

```bash
go run . commit
```

Git Pilot will:

1. Inspect changed files.
2. Ask whether to commit file-wise or AI group-wise.
3. Generate one commit message at a time.
4. Show the target files and message preview.
5. Ask for `y/n` approval before the actual commit.
6. Offer a push prompt after successful commits.

Direct modes are also supported:

```bash
go run . commit file main.go
go run . commit group
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
├── main.go      # CLI, Git integration, AI prompts, and terminal UI
├── go.mod       # Go module definition
├── README.md    # Project documentation
├── AGENTS.md    # Contributor guide
└── .gitignore
```

## Development

Useful commands:

```bash
gofmt -w main.go
go build ./...
go run . help
```

## Current Limitations

- AI features require a valid Groq API key.
- `pull` uses `git pull --ff-only` to avoid implicit merge commits.
- The tool currently keeps most logic in `main.go`; future refactors can split command, UI, and AI logic into packages.

## Roadmap

- Spinner/progress feedback for AI calls
- Editable AI-generated commit messages before approval
- Richer Git status breakdowns
- Better non-interactive scripting support

## Contributing

Contributions are welcome. Before opening a pull request:

- run `gofmt -w main.go`
- run `go build ./...`
- test the command flow you changed

See [`AGENTS.md`](./AGENTS.md) for repository-specific contribution guidance.

## License

This project is licensed under the terms in [`LICENSE`](./LICENSE).
