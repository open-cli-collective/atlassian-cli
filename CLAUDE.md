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

## CI

GitHub Actions CI runs on all PRs and pushes to main:
- **build-and-test**: Verifies `go.work`, builds both binaries, runs all tests
- **lint**: Runs golangci-lint v2 for both tools

## Environment Variables

Both tools support shared Atlassian credentials via `ATLASSIAN_*` environment variables:

| Variable | Description |
|----------|-------------|
| `ATLASSIAN_URL` | Base URL for Atlassian instance |
| `ATLASSIAN_EMAIL` | User email for authentication |
| `ATLASSIAN_API_TOKEN` | API token for authentication |

Tool-specific variables (`CFL_*`, `JIRA_*`) take precedence over shared variables.

## Git History

This monorepo was created using `git subtree` to preserve the full commit history of both tools. Use `git log --oneline` to see the complete history from both source repositories.
