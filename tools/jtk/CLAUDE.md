# CLAUDE.md

This file provides guidance for AI agents working with the jira-ticket-cli codebase.

## Project Overview

jira-ticket-cli is a command-line interface for Jira (Cloud and self-hosted) written in Go. It uses the Cobra framework for commands and provides a public `api/` package that can be imported as a Go library. Output is human+agent-readable text; there is no global `--output` format flag.

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
├── cmd/jtk/main.go              # Entry point - registers commands, calls Execute()
├── api/                          # Public Go library (importable)
│   ├── client.go                # Client struct, New(), HTTP helpers
│   ├── types.go                 # All data types (Issue, Sprint, Board, etc.)
│   ├── errors.go                # Error types: APIError, ErrNotFound
│   ├── issues.go                # Issue CRUD operations
│   ├── projects.go              # Project CRUD operations
│   ├── sprints.go               # Sprint operations
│   ├── boards.go                # Board operations
│   ├── comments.go              # Comment operations
│   ├── transitions.go           # Issue transition operations
│   ├── attachments.go           # Attachment operations
│   ├── automation.go            # Automation rule operations
│   ├── automation_types.go      # Automation API types
│   ├── fields.go                # Field metadata
│   ├── users.go                 # User operations
│   └── search.go                # JQL search
├── internal/
│   ├── cmd/                     # Cobra commands (one package per resource)
│   │   ├── root/                # Root command, Options struct, global flags
│   │   ├── issues/              # issues list, get, create, update, delete, search, assign, fields, field-options, types, move
│   │   ├── fields/              # fields list, create, delete, restore, contexts (list/create/delete), options (list/add/update/delete)
│   │   ├── projects/            # projects list, get, create, update, delete, restore, types
│   │   ├── transitions/         # transitions list, do
│   │   ├── comments/            # comments list, add, delete
│   │   ├── attachments/         # attachments list, add, get, delete
│   │   ├── automation/          # automation list, get, export, create, update, enable, disable
│   │   ├── boards/              # boards list, get
│   │   ├── sprints/             # sprints list, current, issues, add
│   │   ├── users/               # users search
│   │   ├── configcmd/           # config show, test, clear
│   │   ├── me/                  # me (current user info)
│   │   └── completion/          # Shell completion
│   ├── config/                  # JSON config loading
│   ├── present/                 # Output rendering (one file per resource type)
│   ├── version/                 # Build-time version injection via ldflags
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
    NoColor  bool
    Extended bool // --extended: admin/schema/audit fields
    FullText bool // --fulltext: disable body truncation
    IDOnly   bool // --id: primary identifier only; takes precedence over Extended/FullText
    Verbose  bool
}

// Command-specific options embed root options
type listOptions struct {
    *root.Options
    project string
    limit   int
}
```

Helper methods on `Options` handle flag precedence: `IsExtended()` and `IsFullText()` both return false when `IDOnly` is set.

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

### Present Pattern

Output rendering lives in `internal/present/` — one file per resource type (`issue.go`, `sprint.go`, `board.go`, etc.). Each present function accepts the data and an `*root.Options`, then writes formatted text directly. The `View` struct (from `shared/view`) handles table mechanics (tabwriter, color). There is no JSON rendering path except in `automation export`, which writes JSON directly to stdout and bypasses the present layer entirely.

## Testing

- Unit tests in `*_test.go` files alongside source
- Use `shared/testutil` for assertions (no testify)
- Table-driven tests for multiple scenarios
- Use `httptest.NewServer()` to mock API responses

Run tests: `make test`

Coverage report: `make test-cover && open coverage.html`

### Integration Tests
After significant code changes, run through the manual integration test suite in [integration-tests.md](integration-tests.md). These tests verify real-world behavior against a live Jira instance and catch edge cases that unit tests miss.

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

Variables are checked in precedence order (first match wins):

| Setting | Precedence |
|---------|------------|
| URL | `JIRA_URL` → `ATLASSIAN_URL` → config |
| Email | `JIRA_EMAIL` → `ATLASSIAN_EMAIL` → config |
| API Token | `JIRA_API_TOKEN` → `ATLASSIAN_API_TOKEN` → config |
| Auth Method | `JIRA_AUTH_METHOD` → `ATLASSIAN_AUTH_METHOD` → config → `"basic"` |
| Cloud ID | `JIRA_CLOUD_ID` → `ATLASSIAN_CLOUD_ID` → config |

Use `ATLASSIAN_*` for shared credentials across jtk and cfl. Use `JIRA_*` to override per-tool.

> **Note:** `JIRA_DOMAIN` is deprecated but still supported for backwards compatibility.

## Authentication

Two auth methods are supported:

- **Basic Auth** (default): Uses `email:token` against the instance URL. Works with classic (unscoped) API tokens.
- **Bearer Auth**: Uses `Authorization: Bearer <token>` against the `api.atlassian.com` gateway. Required for Atlassian service accounts with scoped API tokens.

Bearer auth routes requests through `https://api.atlassian.com/ex/jira/{cloudId}/rest/api/3/...` and requires a Cloud ID. The `api/client.go` file has parallel constructors: `New()` dispatches to `newBearerClient()` when `AuthMethod == "bearer"`.

> **Scope limitations:** Scoped tokens lack Agile (boards/sprints), Automation, and Dashboard scopes. These commands are unavailable with bearer auth — this is an Atlassian platform limitation.

## Output Standards

Commands produce intentional artifacts, not raw API payloads. See [OUTPUT_SPEC.md](OUTPUT_SPEC.md) for the full example catalog. The rules below govern every command implementation.

### Output modes

| Flag | Purpose |
|------|---------|
| *(none)* | Default: contextually-rich human+agent text. Stable format. |
| `--extended` | Adds admin/schema/audit detail on top of default. Always implies `--fulltext`. |
| `--id` | Emits only the primary identifier. Takes precedence over `--extended` and `--fulltext`. |
| `--fulltext` | Disables truncation of descriptions and comments. |

`automation export` is the only command that emits JSON — it writes directly to stdout and bypasses the global flag system entirely. Every other command produces text.

### List commands: pipe-delimited tables

- Headers in ALL_CAPS; separator ` | ` (space-pipe-space)
- Empty/null values: `-`
- `--extended` adds columns; it does not replace default columns
- When more pages exist, append: `More results available (next: TOKEN)`
- Absence of that line signals a complete result set

### Get / single-entity commands: header + key-value block

- First line: `ID  Name` (two spaces between)
- Attribute lines: `Key: Value   Key: Value` (three spaces between same-line pairs)
- Optional rows (Labels, Components) appear **only when non-empty**
- Description: blank line → `Description:` label → body text, always last
- Body text truncates in default mode; trailer: `[truncated — use --fulltext for complete body]`

### Date formatting

- Default: `YYYY-MM-DD`
- Extended: full ISO 8601 with timezone (`2026-04-16T07:16:24+0000`)
- Missing/not-yet-set: `-`

### Mutations: post-state, not confirmation (except destructive ops)

- **Success output mirrors the `get` output of the affected entity** — the caller sees post-state in one call
- After create: always re-fetch (the Jira API returns incomplete data from the create response)
- Delete / archive / remove: plain confirmation only (`Deleted MON-4820`, `Archived MON-4820`)
- `--id` on any mutation: only the affected entity's identifier

### What `--extended` adds

- Raw IDs alongside human-readable names (account IDs, component IDs, sprint IDs, type IDs)
- Full ISO 8601 timestamps instead of `YYYY-MM-DD`
- Admin fields: watchers, resolution, fix versions, status category, all non-null custom fields
- Available workflow transitions (on issue get)
- Always implies `--fulltext`

### Name/ID resolution

Entity-reference flags (`--assignee`, `--project`, `--board`, `--sprint`, link type arguments) resolve via instance cache:

- Unique match → resolve silently
- Ambiguous → fail, listing all matches with identifiers
- No match + looks like a raw ID → pass through unchanged
- No match + looks like a name → fail with `Try \`jtk refresh <resource>\`` suggestion

### Errors

Plain prose to stderr. No structured format. Ambiguity errors list all matches; unknown-entity errors suggest `jtk refresh`.

## Dependencies

Key dependencies:
- `github.com/spf13/cobra` - CLI framework
- `github.com/fatih/color` - Colored terminal output
- `shared/testutil` - Testing assertions (stdlib-based, no third-party deps)
