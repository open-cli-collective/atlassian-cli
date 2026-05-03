# Contributing to jira-ticket-cli

Thank you for your interest in contributing to jira-ticket-cli!

## Development Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/open-cli-collective/jira-ticket-cli.git
   cd jira-ticket-cli
   ```

2. Install dependencies:
   ```bash
   make deps
   ```

3. Build and run:
   ```bash
   make build
   ./bin/jtk --version
   ```

## Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-cover
```

## Code Style

- Run `gofmt` and `goimports` before committing
- Run the linter: `make lint`
- Follow Go conventions and idioms

## Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new feature
fix: fix a bug
docs: update documentation
test: add tests
refactor: refactor code
ci: update CI configuration
chore: maintenance tasks
```

Examples:
```
feat: add sprint filtering to issues list
fix: handle empty API response in boards list
docs: update installation instructions
```

## Adding or modifying commands

Two normative specs govern every command in this CLI. Read whichever applies before opening a PR — reviewers will check against them:

- [internal/cmd/GUARDRAILS.md](internal/cmd/GUARDRAILS.md) — command surface contract: verb language, flag aliases, pagination defaults, positional-vs-flag rule, mutation safety, boolean conventions.
- [internal/cmd/OUTPUT_SPEC.md](internal/cmd/OUTPUT_SPEC.md) — output contract: list/get/mutation shapes, output-mode flags, date formatting, error rules.

These docs are the single source of truth. If you find a rule that needs to change, update the spec — don't work around it.

## Pull Request Process

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Make your changes
4. Run tests: `make test`
5. Run linter: `make lint`
6. Commit with a conventional commit message
7. Push and create a pull request

## Project Structure

```
jira-ticket-cli/
├── cmd/jtk/              # Entry point
├── api/                  # Jira API client
├── internal/
│   ├── cmd/              # Command implementations
│   │   ├── boards/       # boards commands
│   │   ├── comments/     # comments commands
│   │   ├── completion/   # shell completion
│   │   ├── configcmd/    # config commands
│   │   ├── issues/       # issues commands
│   │   ├── me/           # me command
│   │   ├── root/         # root command
│   │   ├── sprints/      # sprints commands
│   │   └── transitions/  # transitions commands
│   ├── config/           # Configuration management
│   ├── exitcode/         # Exit code definitions
│   ├── version/          # Version info
│   └── view/             # Output formatting
└── .github/              # GitHub workflows and templates
```

## Questions?

Open an issue or start a discussion on GitHub.
