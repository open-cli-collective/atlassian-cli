# atlassian-cli Development Guide

This is the repo-local source for working on the atlassian-cli monorepo. It contains repository-specific facts only; shared Open CLI standards and automation live in the linked source repositories.

## Repository

The repository is a Go workspace managed by `go.work`.

| Path | Module | Purpose |
| --- | --- | --- |
| `shared` | `github.com/open-cli-collective/atlassian-go` | Shared Atlassian libraries and cross-tool helpers |
| `tools/cfl` | `github.com/open-cli-collective/confluence-cli` | Confluence CLI binary `cfl` |
| `tools/jtk` | `github.com/open-cli-collective/jira-ticket-cli` | Jira CLI binary `jtk` |

Both tool modules replace `github.com/open-cli-collective/atlassian-go` with `../../shared`.

## Repo-Local Sources

### Standards Index

Source of truth: https://github.com/open-cli-collective/atlassian-cli/blob/main/STANDARDS.md
Local convenience copy, if present: `../STANDARDS.md`

### Rendering Architecture

Source of truth: https://github.com/open-cli-collective/atlassian-cli/blob/main/ARCHITECTURE.md
Local convenience copy, if present: `../ARCHITECTURE.md`

### Artifact Contract

Source of truth: https://github.com/open-cli-collective/atlassian-cli/blob/main/docs/ARTIFACT_CONTRACT.md
Local convenience copy, if present: `ARTIFACT_CONTRACT.md`

### cfl Development

Source of truth: https://github.com/open-cli-collective/atlassian-cli/blob/main/tools/cfl/docs/development.md
Local convenience copy, if present: `../tools/cfl/docs/development.md`

### jtk Development

Source of truth: https://github.com/open-cli-collective/atlassian-cli/blob/main/tools/jtk/docs/development.md
Local convenience copy, if present: `../tools/jtk/docs/development.md`

### jtk Command Surface

Source of truth: https://github.com/open-cli-collective/atlassian-cli/blob/main/tools/jtk/internal/cmd/GUARDRAILS.md
Local convenience copy, if present: `../tools/jtk/internal/cmd/GUARDRAILS.md`

Source of truth: https://github.com/open-cli-collective/atlassian-cli/blob/main/tools/jtk/internal/cmd/OUTPUT_SPEC.md
Local convenience copy, if present: `../tools/jtk/internal/cmd/OUTPUT_SPEC.md`

### cfl Output Contract

Source of truth: https://github.com/open-cli-collective/atlassian-cli/blob/main/tools/cfl/internal/cmd/OUTPUT_SPEC.md
Local convenience copy, if present: `../tools/cfl/internal/cmd/OUTPUT_SPEC.md`

### cfl Presenter Migration

Source of truth: https://github.com/open-cli-collective/atlassian-cli/blob/main/tools/cfl/internal/present/README.md
Local convenience copy, if present: `../tools/cfl/internal/present/README.md`

## Shared Sources

### Shared Open CLI Standards

Source of truth: https://github.com/open-cli-collective/cli-common/tree/main/docs
Local convenience copy, if present: `../../cli-common/docs`

Relevant shared docs:

| Document | Use for |
| --- | --- |
| `repo-layout.md` | Repository layout, required files, Makefile target names, lint config, Go version policy, branch settings, commit hygiene |
| `ci.md` | Shared CI behavior and how workflows consume Makefile targets |
| `command-surface.md` | Family-wide command, argument, flag, prompt, alias, and mutation-safety conventions |
| `output-and-rendering.md` | Output shape, stream discipline, color, pagination, and presenter boundaries |
| `working-with-secrets.md` | Credential ingress, keyring storage, migration behavior, redaction, no-leak testing |
| `working-with-state.md` | Config/cache locations, credential references, cache freshness, state migration, hermetic tests |
| `scriptability.md` | Non-interactive setup, env-bridge flags, health checks, and stdout/stderr behavior |
| `release.md` and `distribution.md` | Shared release and installation behavior |

### Shared Automation

Source of truth: https://github.com/open-cli-collective/.github
Local convenience copy, if present: `../../.github`

## Quick Commands

```bash
make build
make test
make lint
make tidy
make check
make build-cfl
make build-jtk
```

`make check` is the root sanity target: tidy, lint, race tests, and build across `shared`, `tools/cfl`, and `tools/jtk`. Tool-level Makefiles also provide their own `check` targets.

## Shared Atlassian Credentials and State

Both tools use the shared Atlassian credential/config model.

Non-secret shared config lives in the OS-native user config directory under `atlassian-cli/config.yml`, such as `~/Library/Application Support/atlassian-cli/config.yml` on macOS, `%APPDATA%\atlassian-cli\config.yml` on Windows, or `~/.config/atlassian-cli/config.yml` on Linux. Connection fields live in the `default` section. Tool sections carry non-secret tool defaults only.

The API token lives in the OS keyring through `cli-common/credstore`. The shared bundle ref is `atlassian-cli/default`, and the single logical key is `api_token`. `ATLASSIAN_*` environment variables apply across both tools; `CFL_*` and `JIRA_*` variables are tool-specific overrides.

Use the shared `cli-common` secrets, state, and scriptability docs for the family-wide rules. This repo-local guide records the current atlassian-cli shape.

## Output Architecture

The repository separates domain values, presentation models, renderers, and command orchestration. Use `ARCHITECTURE.md` for that boundary and `docs/ARTIFACT_CONTRACT.md` for artifact vocabulary.

`jtk` is text-first and keeps its command-surface and output contracts in `tools/jtk/internal/cmd/GUARDRAILS.md` and `tools/jtk/internal/cmd/OUTPUT_SPEC.md`.

`cfl` is Confluence-focused and markdown-first. Its page content flow converts between Markdown and Confluence storage XHTML through `tools/cfl/pkg/md`.
Its target text-output contract lives in `tools/cfl/internal/cmd/OUTPUT_SPEC.md`; cfl-specific presenter migration guidance lives in `tools/cfl/internal/present/README.md`.

## Tool Guides

Use `tools/cfl/docs/development.md` before changing Confluence behavior, Markdown/XHTML conversion, or cfl command output.

Use `tools/jtk/docs/development.md` before changing Jira behavior, command/output flags, cache-backed resolution, or jtk presenter behavior.
