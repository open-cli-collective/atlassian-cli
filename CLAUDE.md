# CLAUDE.md

This file provides guidance for AI agents working with the jira-ticket-cli codebase.

## Project Overview

jira-ticket-cli is a command-line interface for Jira (Cloud and self-hosted) written in Go. It uses the Cobra framework for commands and provides a public `api/` package that can be imported as a Go library. The CLI supports multiple output formats (table, JSON, plain).

## Quick Commands

```bash
# Build
make build

# Run tests
make test

# Run tests with coverage
make test-cover

# Lint
make lint

# Format and verify
make tidy

# Install locally
make install

# Clean build artifacts
make clean
```

## Architecture

```
jira-ticket-cli/
├── cmd/jtk/main.go  # Entry point - registers commands, calls Execute()
├── api/                          # Public Go library (importable)
│   ├── client.go                # Client struct, New(), HTTP helpers
│   ├── types.go                 # All data types (Issue, Sprint, Board, etc.)
│   ├── errors.go                # Error types: APIError, ErrNotFound
│   ├── issues.go                # Issue CRUD operations
│   ├── sprints.go               # Sprint operations
│   ├── boards.go                # Board operations
│   ├── comments.go              # Comment operations
│   ├── transitions.go           # Issue transition operations
│   ├── fields.go                # Field metadata
│   ├── users.go                 # User operations
│   └── search.go                # JQL search
├── internal/
│   ├── cmd/                     # Cobra commands (one package per resource)
│   │   ├── root/                # Root command, Options struct, global flags
│   │   ├── issues/              # issues list, get, create, update, search, assign
│   │   ├── transitions/         # transitions list, do
│   │   ├── comments/            # comments list, add
│   │   ├── boards/              # boards list, get
│   │   ├── sprints/             # sprints list, current, issues
│   │   ├── configcmd/           # config set
│   │   ├── me/                  # me (current user info)
│   │   └── completion/          # Shell completion
│   ├── config/                  # JSON config loading
│   ├── version/                 # Build-time version injection via ldflags
│   ├── view/                    # Output formatting (table, JSON, plain)
│   └── exitcode/                # Exit code constants
├── Makefile                     # Build, test, lint targets
└── go.mod                       # Module: github.com/open-cli-collective/jira-ticket-cli
```

## Key Patterns

### Options Struct Pattern

Commands use an Options struct for dependency injection:

```go
// Root options (global flags)
type Options struct {
    Output  string
    NoColor bool
}

// Command-specific options embed root options
type listOptions struct {
    *root.Options
    project string
    limit   int
}
```

### Register Pattern

Each command package exports a Register function:

```go
func Register(rootCmd *cobra.Command, opts *root.Options) {
    cmd := &cobra.Command{
        Use:   "issues",
        Short: "Manage Jira issues",
    }
    cmd.AddCommand(newListCmd(opts))
    cmd.AddCommand(newGetCmd(opts))
    rootCmd.AddCommand(cmd)
}
```

### View Pattern

Use the View struct for formatted output:

```go
v := view.New(opts.Output, opts.NoColor)

// Table output
headers := []string{"KEY", "SUMMARY", "STATUS"}
rows := [][]string{{"PROJ-123", "Fix bug", "In Progress"}}
v.Table(headers, rows)

// JSON output
v.JSON(data)
```

## Testing

- Unit tests in `*_test.go` files alongside source
- Use `testify/assert` for assertions
- Table-driven tests for multiple scenarios
- Use `httptest.NewServer()` to mock API responses

Run tests: `make test`

Coverage report: `make test-cover && open coverage.html`

## Commit Conventions

Use conventional commits:

```
type(scope): description

feat(issues): add bulk update command
fix(sprints): handle empty sprint list
docs(readme): add JQL examples
```

| Prefix | Purpose | Triggers Release? |
|--------|---------|-------------------|
| `feat:` | New features | Yes |
| `fix:` | Bug fixes | Yes |
| `docs:` | Documentation only | No |
| `test:` | Adding/updating tests | No |
| `refactor:` | Code changes that don't fix bugs or add features | No |
| `chore:` | Maintenance tasks | No |
| `ci:` | CI/CD changes | No |

## CI & Release Workflow

Releases are automated with a dual-gate system to avoid unnecessary releases:

**Gate 1 - Path filter:** Only triggers when Go code changes (`**.go`, `go.mod`, `go.sum`)
**Gate 2 - Commit prefix:** Only `feat:` and `fix:` commits create releases

This means:
- `feat: add command` + Go files changed → release
- `fix: handle edge case` + Go files changed → release
- `docs:`, `ci:`, `test:`, `refactor:` → no release
- Changes only to docs, packaging, workflows → no release

**After merging a release-triggering PR:** The workflow creates a tag, which triggers GoReleaser to build binaries and publish to Homebrew. Chocolatey and Winget require manual workflow dispatch.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `JIRA_URL` | Full Jira URL (e.g., `https://mycompany.atlassian.net` or `https://jira.internal.corp.com`) |
| `JIRA_EMAIL` | Your Atlassian email |
| `JIRA_API_TOKEN` | Your API token |

> **Note:** `JIRA_DOMAIN` is deprecated but still supported for backwards compatibility.

## Dependencies

Key dependencies:
- `github.com/spf13/cobra` - CLI framework
- `github.com/fatih/color` - Colored terminal output
- `github.com/stretchr/testify` - Testing assertions
