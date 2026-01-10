# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

cfl is a command-line interface for Atlassian Confluence Cloud written in Go. It provides markdown-first page management with automatic conversion between Markdown and Confluence's XHTML storage format.

## Commands

```bash
# Build
make build                    # Build to bin/cfl with version info from git

# Test
make test                     # Run all tests with race detection
make test-cover               # Tests with coverage (generates coverage.html)
go test -v -race ./api/...    # Run tests for a specific package

# Lint & Format
make lint                     # Run golangci-lint
make fmt                      # Format with gofmt and goimports

# Development
make run ARGS="page list"     # Run CLI directly without building
```

## Architecture

```
cmd/cfl/main.go          → Entry point, creates root command
api/                     → Confluence REST API client (pages, spaces, attachments)
internal/cmd/            → Cobra command implementations
  root/                  → Root command with global flags
  page/                  → page list|view|create|edit|delete
  space/                 → space list
  attachment/            → attachment list|upload|download
  init/                  → Configuration wizard
internal/config/         → YAML config loading with env var overrides
internal/view/           → Output formatting (table/json/plain)
pkg/md/                  → Bidirectional Markdown ↔ XHTML conversion
```

**Data Flow:** Commands load config from `~/.config/cfl/config.yml` → instantiate `api.Client` → call API methods → format output via `view.Renderer`

## Key Patterns

- **Command factories:** `NewCmd{Name}() *cobra.Command` in each command file
- **Options structs:** Commands collect flags into `*Options` structs before execution
- **Run functions:** `run{Action}(opts *Options) error` contains command logic
- **Import ordering:** Standard library, external deps, then `github.com/rianjs/confluence-cli/...` (enforced by goimports)

## Markdown Conversion

The `pkg/md` package handles format conversion:
- `converter.go`: Markdown → XHTML (uses goldmark)
- `from_html.go`: XHTML → Markdown (uses html-to-markdown)

Format auto-detection: `.md` files → markdown, `.html/.xhtml` → storage format, stdin/editor → markdown by default.

## Testing Philosophy

### Goals
- **Safety**: Destructive operations (delete, overwrite) must be tested
- **Recoverability**: Network failures, malformed responses shouldn't corrupt state
- **Pleasant UX**: Clear error messages, graceful degradation

### What We Test (Priority Order)
1. Security-sensitive paths (path traversal, credential handling)
2. Destructive operations (delete confirmations)
3. API client behavior (auth, errors, edge cases)
4. Data transformations (markdown ↔ Confluence HTML)

### Go Testing Idioms

**HTTP mocking**: Use `httptest.NewServer()` - no interface needed
```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // verify request, return mock response
}))
client := api.NewClient(server.URL, "test@example.com", "token")
```

**Injectable stdin for confirmations**: Use `io.Reader` parameter
```go
type deleteOptions struct {
    stdin io.Reader  // injectable for testing
}
// Test: opts.stdin = strings.NewReader("y\n")
```

**Consumer-defined interfaces**: Define small interfaces at point of use
```go
// In the command file, not a central interfaces package
type pageAPI interface {
    GetPage(ctx context.Context, id string) (*api.Page, error)
    DeletePage(ctx context.Context, id string) error
}
```

**Temp directories**: Use `t.TempDir()` for file operations

### What NOT to Do
- No giant interface packages
- No DI frameworks (wire, dig)
- No reflection-based mocking unless necessary
- Don't mock what you can test with httptest

### Test Organization
- `*_test.go` next to implementation
- `testdata/` for JSON fixtures
- Table-driven tests with `t.Run()`
- Use `github.com/stretchr/testify/assert` and `require`

## Undocumented Constants

| Constant | Value | Location |
|----------|-------|----------|
| API timeout | 30s | `api/client.go:16` |
| Init verify timeout | 10s | `internal/cmd/init/init.go:166` |
| Config permissions | 0600 | `internal/config/config.go` |

## Release Workflow

Releases are automated via release-please. When PRs merge to main with conventional commits:
- `feat:` → minor version bump
- `fix:` → patch version bump
- `BREAKING CHANGE:` or `feat!:` → major version bump

**Before merging a PR:** Run `/release-notes` to generate release notes and update the PR description.

**After merging:** release-please creates a Release PR. Merging that PR triggers the full release (GitHub Release + Homebrew tap update).
