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
- **Import ordering:** Standard library, external deps, then `github.com/open-cli-collective/confluence-cli/...` (enforced by goimports)

## Markdown Conversion

The `pkg/md` package handles bidirectional format conversion with macro support:

**Public API (stable):**
- `ToConfluenceStorage(markdown []byte) (string, error)` - Markdown → XHTML
- `FromConfluenceStorage(html string) (string, error)` - XHTML → Markdown
- `FromConfluenceStorageWithOptions(html string, opts ConvertOptions) (string, error)`

**Internal Architecture:**
```
converter.go      → Main entry point, coordinates preprocessing/postprocessing
from_html.go      → XHTML→MD coordination, placeholder management
macro.go          → MacroNode, MacroType, MacroRegistry (data model)
tokens.go         → BracketToken, XMLToken (token definitions)
tokenizer_*.go    → TokenizeBrackets(), TokenizeConfluenceXML()
parser_*.go       → ParseBracketMacros(), ParseConfluenceXML()
render.go         → RenderMacroToXML(), RenderMacroToBracket()
```

**Adding New Macros:** Add one entry to `MacroRegistry` in `macro.go`. The tokenizer/parser/render components are macro-agnostic.

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

### Integration Tests
After significant code changes, run through the manual integration test suite in [integration-tests.md](integration-tests.md). These tests verify real-world behavior against a live Confluence instance and catch edge cases that unit tests miss.

## Environment Variables

Variables are checked in precedence order (first match wins):

| Setting | Precedence |
|---------|------------|
| URL | `CFL_URL` → `ATLASSIAN_URL` → config |
| Email | `CFL_EMAIL` → `ATLASSIAN_EMAIL` → config |
| API Token | `CFL_API_TOKEN` → `ATLASSIAN_API_TOKEN` → config |
| Default Space | `CFL_DEFAULT_SPACE` → config |

Use `ATLASSIAN_*` for shared credentials across cfl and jtk. Use `CFL_*` to override per-tool.

## Undocumented Constants

| Constant | Value | Location |
|----------|-------|----------|
| API timeout | 30s | `api/client.go:16` |
| Init verify timeout | 10s | `internal/cmd/init/init.go:166` |
| Config permissions | 0600 | `internal/config/config.go` |

## Issue & PR Workflow

### Issues as Backlog
GitHub Issues serve as the project backlog. Use labels to categorize:
- `bug` - Something isn't working
- `enhancement` - New feature or request

### Creating Issues
When discovering bugs or planning features:
1. Create a GitHub issue with clear reproduction steps (for bugs) or use case (for features)
2. Reference the issue number in related PRs

### PR Workflow
1. Create a branch from updated `main`: `git checkout -b fix/issue-description`
2. Make changes, write tests
3. Run `make test` to verify
4. **For new features:** Update README.md with usage examples and flag documentation
5. Commit with conventional commit message referencing the issue:
   ```
   fix: description of fix

   Fixes #123
   ```
6. Push and create PR referencing the issue in the body
7. After merge, the issue will auto-close if using "Fixes #N" syntax

### README as Living Documentation
The README documents the complete CLI surface area. When adding features:
- Add command examples under the appropriate section
- Document all flags and options
- Include practical use cases
- Keep examples copy-paste ready

### Conventional Commits
Use these prefixes for commit messages:
- `fix:` - Bug fixes (patch version bump)
- `feat:` - New features (minor version bump)
- `docs:` - Documentation only
- `test:` - Adding/updating tests
- `refactor:` - Code changes that don't fix bugs or add features
- `BREAKING CHANGE:` or `feat!:` - Breaking changes (major version bump)

## CI & Release Workflow

**CI** skips Go build/test/lint jobs when only non-Go files change (docs, packaging, workflows). Jobs show as "Skipped" rather than not running, so branch protection still works.

**Releases** are automated with a dual-gate system to avoid unnecessary releases:

**Gate 1 - Path filter:** Only triggers when Go code changes (`**.go`, `go.mod`, `go.sum`)
**Gate 2 - Commit prefix:** Only `feat:` and `fix:` commits create releases

This means:
- ✅ `feat: add command` + Go files changed → release
- ✅ `fix: handle edge case` + Go files changed → release
- ❌ `docs:`, `ci:`, `test:`, `refactor:` → no release
- ❌ Changes only to docs, packaging, workflows → no release

**Before merging a PR:** Run `/release-notes` to generate release notes and update the PR description.

**After merging a release-triggering PR:** The workflow creates a tag, which triggers GoReleaser to build binaries and publish to Homebrew and Chocolatey.

## Packaging

Distribution packages are in `packaging/`:

```
packaging/
├── chocolatey/              # Windows Chocolatey package
│   ├── confluence-cli.nuspec
│   ├── tools/chocolateyInstall.ps1
│   ├── tools/chocolateyUninstall.ps1
│   └── README.md            # Publishing instructions
├── winget/                  # Windows Winget manifests
│   ├── OpenCLICollective.cfl.yaml
│   ├── OpenCLICollective.cfl.installer.yaml
│   ├── OpenCLICollective.cfl.locale.en-US.yaml
│   └── README.md            # Publishing instructions
└── homebrew/
    └── README.md            # Points to GoReleaser config
```

- **Homebrew**: Automated via GoReleaser, published to [open-cli-collective/homebrew-tap](https://github.com/open-cli-collective/homebrew-tap)
- **Chocolatey**: Automated via release workflow, requires `CHOCOLATEY_API_KEY` secret
- **Winget**: Automated via release workflow, requires `WINGET_GITHUB_TOKEN` secret (PAT with `public_repo` scope)
