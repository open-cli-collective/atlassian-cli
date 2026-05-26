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
6. The release workflow's `trigger-publish` job dispatches `chocolatey-publish-{tool}.yml` and `winget-publish-{tool}.yml` via `gh workflow run` (manual `workflow_dispatch` is retained as a fallback)

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

## Shared credential store

The **API token is stored in the OS keyring** (macOS Keychain / Linux
Secret Service / Windows Credential Manager, or an opt-in encrypted-file
backend) — never in a plaintext file. Only **non-secret** config lives
in the shared OS-native config dir (`~/Library/Application
Support/atlassian-cli/config.yml` on macOS, `%APPDATA%\atlassian-cli\
config.yml` on Windows, `~/.config/atlassian-cli/config.yml` on Linux;
mode 0600), written by `cfl init` and `jtk init`:

```yaml
default:
  url: https://acme.atlassian.net   # base URL; cfl appends /wiki on read
  email: u@example.com
  auth_method: basic                # or "bearer"
  cloud_id: <id>                    # required for bearer
cfl:
  default_space: SPACE              # cfl-only defaults
  output_format: table              # cfl-only: table | json | plain
jtk:
  default_project: PROJ             # jtk-only defaults
```

Note: there is **no `api_token:` field** — the secret is in the keyring.
Per §2.2 (MON-5328) the `cfl`/`jtk` sections carry **only** non-secret
per-tool defaults (`default_space`, `default_project`,
`output_format`); they may **not** override connection fields
(`url`/`email`/`auth_method`/`cloud_id`). Connection is single-sourced
from `default` (env still overrides at runtime). A pre-MON-5328 file
with per-tool connection fields is read once by the migration; if those
diverge from `default`, `init` fails loud (naming every source + field,
no value) instead of precedence-picking.

Keyring bundle: fixed ref `atlassian-cli/default`, exactly one key
`api_token` shared by cfl and jtk (Secret-Handling Standard §1.11.10:
one key per logical credential — there are no per-tool override keys).
Backend selection happens at three layers (precedence high → low):
the `--backend` flag (root persistent, available on every command), the
`ATLASSIAN_CLI_KEYRING_BACKEND` env var, the `keyring.backend` config
key (jtk: `~/.config/jira-ticket-cli/config.json`; cfl:
`~/.config/cfl/config.yml`), and finally the OS auto-default. Valid
values: `keychain`, `wincred`, `secret-service`, `file`, `memory`
(`credstore.ValidBackendNames()` is the source of truth). Leave all
three unset to auto-select the OS keyring. The file-backend passphrase
comes from `ATLASSIAN_CLI_KEYRING_PASSPHRASE` or a no-echo TTY prompt. **The file
backend cannot prompt non-interactively:** any non-TTY invocation (CI, a
piped token) MUST pre-set `ATLASSIAN_CLI_KEYRING_PASSPHRASE`, and the
passphrase can never share stdin with a piped token.

**Token resolution precedence (highest wins):**

1. Tool-specific env (`CFL_API_TOKEN` / `JIRA_API_TOKEN`)
2. `ATLASSIAN_API_TOKEN` env
3. Keyring shared `api_token`

Non-secret fields keep their previous precedence (env → shared store
override → shared default → legacy file).

**One-time auto-migration:** the first API/`test`/`init` invocation that
actually opens the keyring moves any pre-existing plaintext token (shared
`config.yml` *or* a legacy per-tool file) — and any deprecated per-tool
keyring key left by an older build (B3 upgrade path) — into the single
`api_token` and scrubs the plaintext in place, printing a one-line
notice. If the collected sources hold **more than one distinct token**
the migration fails loud, names every source (never the value), and
mutates nothing — it never precedence-picks a secret winner. Legacy
non-secret files keep working; init still detects/reconciles them. **Caveat:** when an
API-token env var is set it wins outright and the keyring is not opened,
so migration is deferred until an invocation actually needs the keyring
(`init`/`set-credential`, or a command run without the env var). A user
who *permanently* exports the token via env therefore keeps the plaintext
file until they run `init`/`set-credential` — env is an explicit runtime
override, so a read path is never forced to mutate disk behind it.

**Non-interactive ingress:** `cfl set-credential` / `jtk set-credential`
read a token from stdin or `--from-env VAR` and store it in the keyring
(never echoed). `config show` reports token presence + source + keyring
backend only (never the value). `config clear` removes the single shared
`api_token` (warning that the sibling tool loses access too, since both
resolve the same key); `config clear --all` removes the whole bundle
(including any deprecated per-tool keys) plus the non-secret config file.

## Git History

This monorepo was created using `git subtree` to preserve the full commit history of both tools. Use `git log --oneline` to see the complete history from both source repositories.
