# cfl Development Guide

This is the repo-local source for working on `cfl`, the Confluence CLI inside the atlassian-cli monorepo.

## Repository

Binary: `cfl`
Module: `github.com/open-cli-collective/confluence-cli`
Entrypoint: `cmd/cfl/main.go`
Shared module replacement: `github.com/open-cli-collective/atlassian-go => ../../shared`

## Repo-Local Sources

### Monorepo Guide

Source of truth: https://github.com/open-cli-collective/atlassian-cli/blob/main/docs/development.md
Local convenience copy, if present: `../../../docs/development.md`

### Rendering Architecture

Source of truth: https://github.com/open-cli-collective/atlassian-cli/blob/main/ARCHITECTURE.md
Local convenience copy, if present: `../../../ARCHITECTURE.md`

### Artifact Contract

Source of truth: https://github.com/open-cli-collective/atlassian-cli/blob/main/docs/ARTIFACT_CONTRACT.md
Local convenience copy, if present: `../../../docs/ARTIFACT_CONTRACT.md`

## Shared Sources

### Shared Open CLI Standards

Source of truth: https://github.com/open-cli-collective/cli-common/tree/main/docs
Local convenience copy, if present: `../../../../cli-common/docs`

### Shared Automation

Source of truth: https://github.com/open-cli-collective/.github
Local convenience copy, if present: `../../../../.github`

## Quick Commands

```bash
make build
make test
make test-cover
make test-short
make lint
make fmt
make tidy
make check
make run ARGS="page list"
```

`make build` writes `bin/cfl` from `./cmd/cfl`. `make test` runs the test suite with the race detector. `make check` runs tidy, lint, test, and build for the tool module.

## Architecture

`api/` contains the Confluence REST API client. `internal/cmd/` contains Cobra command implementations, including `root`, `page`, `space`, `attachment`, `init`, and `me`. `internal/config` owns YAML config loading with environment overrides. `pkg/md` owns bidirectional Markdown and Confluence storage XHTML conversion.

Commands should load config, instantiate the API client, call API methods, then render intentional output artifacts. Keep presentation boundaries aligned with the root rendering architecture and artifact contract.

## Markdown Conversion

`pkg/md` is the stable conversion package.

- `ToConfluenceStorage` converts Markdown to Confluence storage XHTML.
- `FromConfluenceStorage` converts Confluence storage XHTML to Markdown.
- `FromConfluenceStorageWithOptions` applies conversion options.

Add new Confluence macros through `MacroRegistry` in `macro.go`; the tokenizer, parser, and renderer components are macro-agnostic. Wiki-link syntax such as `[[Page Title]]` and `[[SPACE:Page Title]]` is implemented in `wikilink.go`.

## Auth and Config

`cfl` participates in the shared Atlassian credential/config model described by the monorepo guide. `ATLASSIAN_*` variables apply across both tools; `CFL_*` variables override for cfl. The cfl-specific config section carries non-secret defaults such as `default_space` and `output_format`.

Basic auth uses an instance URL plus email and token. Bearer auth routes through `api.atlassian.com` and requires a cloud ID. `cfl init` and `cfl me` verify against Confluence's current-user endpoint.

## Output

`cfl` is markdown-first for page content. Page view output should provide action-oriented default artifacts, with richer inspection available through explicit flags where implemented. JSON is reserved for control-plane envelopes documented by the shared standards.

## Testing Notes

Use `httptest.NewServer` for API client behavior, `t.TempDir` for file operations, injectable readers for confirmations, and `shared/testutil` for assertions. Keep tests next to the implementation and use `testdata/` for fixtures.
