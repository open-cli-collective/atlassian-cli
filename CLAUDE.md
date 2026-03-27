# CLAUDE.md

This file provides guidance for working with the atlassian-cli monorepo.

## Project Overview

atlassian-cli is a Go workspace monorepo containing CLI tools for Atlassian products. It uses `go.work` to manage multiple modules while preserving their independent `go.mod` files.

## Repository Structure

```
atlassian-cli/
├── go.work              # Go workspace file
├── tools/
│   ├── cfl/             # Confluence CLI (full git history preserved)
│   │   ├── go.mod
│   │   ├── cmd/cfl/
│   │   ├── api/
│   │   └── internal/
│   └── jtk/             # Jira CLI (full git history preserved)
│       ├── go.mod
│       ├── cmd/jtk/
│       ├── api/
│       └── internal/
```

## Tools

| Tool | Directory | Description |
|------|-----------|-------------|
| `cfl` | `tools/cfl` | Confluence Cloud CLI - markdown-first page management |
| `jtk` | `tools/jtk` | Jira Cloud CLI - issue, sprint, and board management |

Each tool has its own `CLAUDE.md` with detailed guidance. See:
- `tools/cfl/CLAUDE.md` - Confluence CLI specifics
- `tools/jtk/CLAUDE.md` - Jira CLI specifics

## Quick Commands

```bash
# Using Makefile (recommended)
make build              # Build both tools
make test               # Run all tests
make lint               # Run golangci-lint for both tools
make all                # Build, test, and lint

# Build individual tools to bin/
make build-cfl          # Build bin/cfl
make build-jtk          # Build bin/jtk

# Direct go commands
go build ./tools/cfl/cmd/cfl
go build ./tools/jtk/cmd/jtk
go test ./tools/cfl/...
go test ./tools/jtk/...
go work sync
```

## CI/CD

### CI Workflows

GitHub Actions CI runs on all PRs and pushes to main with **path filtering**:
- Changes to `tools/cfl/**` trigger cfl build/test/lint only
- Changes to `tools/jtk/**` trigger jtk build/test/lint only
- Changes to `shared/**` trigger both (future shared code)

### Release Workflow

Releases are automated with a dual-gate system:

1. **Path gate**: Only Go code changes (`**/*.go`, `go.mod`, `go.sum`) can trigger releases
2. **Commit gate**: Only `feat:` and `fix:` commits create releases

**Tag format**: `{tool}-v{base}.{run}` (e.g., `cfl-v0.9.150`, `jtk-v0.1.75`)

When a release-triggering commit is merged to main:
1. `auto-release-{tool}.yml` creates a tag (e.g., `cfl-v1.0.150`)
2. Tag push triggers `release-{tool}.yml`
3. A temporary semver tag (`v1.0.150`) is created for GoReleaser compatibility
4. GoReleaser builds binaries, creates the GitHub release, and pushes the Homebrew cask
5. The release is re-tagged from `v1.0.150` → `cfl-v1.0.150` and the temporary tag is deleted
6. Chocolatey and Winget workflows publish packages

**Fragile: tag rename and download URLs.** GoReleaser runs *before* the tag rename in step 5. Any GoReleaser-generated download URLs must use `url.template` to hardcode the final tool-prefixed tag — otherwise they'll reference the deleted temporary tag and 404. The `homebrew_casks` sections in `.goreleaser-{tool}.yml` have `url.template` set for this reason. If you add a new packaging integration that uses release download URLs, it must account for the tag rename.

**`jira-ticket-cli` alias cask.** GoReleaser Free doesn't support `alternative_names` for casks, so `jira-ticket-cli.rb` is auto-generated from `jtk.rb` via sed in the `release-jtk.yml` workflow (after the tag rename step).

### Required Secrets

| Secret | Purpose |
|--------|---------|
| `TAP_GITHUB_TOKEN` | Push tags + update Homebrew tap |
| `CHOCOLATEY_API_KEY` | Publish to Chocolatey |
| `WINGET_GITHUB_TOKEN` | Submit to microsoft/winget-pkgs |
| `LINUX_PACKAGES_DISPATCH_TOKEN` | Trigger APT/RPM repo update in open-cli-collective/linux-packages |

### Build Matrix

Each tool builds 6 binaries:
- darwin/amd64, darwin/arm64 (.tar.gz)
- linux/amd64, linux/arm64 (.tar.gz + .deb + .rpm)
- windows/amd64, windows/arm64 (.zip)

## Environment Variables

Both tools support shared Atlassian credentials via `ATLASSIAN_*` environment variables:

| Variable | Description |
|----------|-------------|
| `ATLASSIAN_URL` | Base URL for Atlassian instance |
| `ATLASSIAN_EMAIL` | User email for authentication |
| `ATLASSIAN_API_TOKEN` | API token for authentication |
| `ATLASSIAN_AUTH_METHOD` | `basic` (default) or `bearer` for service accounts |
| `ATLASSIAN_CLOUD_ID` | Cloud ID for bearer auth (gateway URL construction) |

Tool-specific variables (`CFL_*`, `JIRA_*`) take precedence over shared variables. Both tools support Basic Auth (classic tokens, instance URL) and Bearer Auth (scoped tokens, api.atlassian.com gateway).

## Git History

This monorepo was created using `git subtree` to preserve the full commit history of both tools. Use `git log --oneline` to see the complete history from both source repositories.
