# atlassian-cli Standards Index

This file is an index, not a standalone Go style guide. Shared Open CLI standards live in `cli-common`; atlassian-cli-specific constraints live in this repository's local docs and specs.

## Repo-Local Standards

### Development Guide

Source of truth: https://github.com/open-cli-collective/atlassian-cli/blob/main/docs/development.md
Local convenience copy, if present: `docs/development.md`

Use this for monorepo layout, shared credential behavior, root commands, and links to tool-specific guidance.

### Rendering Architecture

Source of truth: https://github.com/open-cli-collective/atlassian-cli/blob/main/ARCHITECTURE.md
Local convenience copy, if present: `ARCHITECTURE.md`

Use this for the presentation model, renderer boundaries, and output architecture across the workspace.

### Artifact Contract

Source of truth: https://github.com/open-cli-collective/atlassian-cli/blob/main/docs/ARTIFACT_CONTRACT.md
Local convenience copy, if present: `docs/ARTIFACT_CONTRACT.md`

Use this for the shared artifact vocabulary and output modes that apply across `jtk` and `cfl`.

### cfl Development

Source of truth: https://github.com/open-cli-collective/atlassian-cli/blob/main/tools/cfl/docs/development.md
Local convenience copy, if present: `tools/cfl/docs/development.md`

Use this for Confluence-specific command architecture, Markdown/XHTML conversion, and cfl verification commands.

### jtk Development

Source of truth: https://github.com/open-cli-collective/atlassian-cli/blob/main/tools/jtk/docs/development.md
Local convenience copy, if present: `tools/jtk/docs/development.md`

Use this for Jira-specific command architecture, text output behavior, and jtk verification commands.

### jtk Command Surface

Source of truth: https://github.com/open-cli-collective/atlassian-cli/blob/main/tools/jtk/internal/cmd/GUARDRAILS.md
Local convenience copy, if present: `tools/jtk/internal/cmd/GUARDRAILS.md`

Source of truth: https://github.com/open-cli-collective/atlassian-cli/blob/main/tools/jtk/internal/cmd/OUTPUT_SPEC.md
Local convenience copy, if present: `tools/jtk/internal/cmd/OUTPUT_SPEC.md`

Use these for jtk command names, flags, mutation safety, pagination, and output shapes.

## Shared Open CLI Standards

### Repository Shape

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/repo-layout.md
Local convenience copy, if present: `../cli-common/docs/repo-layout.md`

Use this for family-wide repository layout, required files, Makefile target names, lint config, Go version policy, branch settings, and commit hygiene.

### CI

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/ci.md
Local convenience copy, if present: `../cli-common/docs/ci.md`

Use this for shared CI behavior and how repository Makefile targets are consumed by GitHub workflows.

### Command Surface

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/command-surface.md
Local convenience copy, if present: `../cli-common/docs/command-surface.md`

Use this for family-wide command, argument, flag, prompt, alias, and mutation-safety conventions.

### Output and Rendering

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/output-and-rendering.md
Local convenience copy, if present: `../cli-common/docs/output-and-rendering.md`

Use this for family-wide output shape, stream discipline, color, pagination, and presenter boundaries.

### Secrets

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/working-with-secrets.md
Local convenience copy, if present: `../cli-common/docs/working-with-secrets.md`

Use this for credential ingress, keyring storage, migration behavior, redaction, and no-leak tests.

### State

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/working-with-state.md
Local convenience copy, if present: `../cli-common/docs/working-with-state.md`

Use this for config/cache locations, credential references, cache freshness, state migration, and hermetic tests.

### Scriptability

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/scriptability.md
Local convenience copy, if present: `../cli-common/docs/scriptability.md`

Use this for non-interactive setup, env-bridge flags, health checks, OAuth/browser handoff patterns, and stdout/stderr behavior that scripts depend on.

### Release and Distribution

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/release.md
Local convenience copy, if present: `../cli-common/docs/release.md`

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/distribution.md
Local convenience copy, if present: `../cli-common/docs/distribution.md`

Use these for shared release and installation rules. Keep repository-specific workflow implementation in the shared automation source, not in this file.

## Shared Automation

Source of truth: https://github.com/open-cli-collective/.github
Local convenience copy, if present: `../.github`

Use this for shared actions, reusable workflow implementations, and organization-level automation. Policy and conventions live in the shared standards docs above.

## Conflict Resolution

Local atlassian-cli docs define repo-specific constraints. `cli-common` docs define family-wide Open CLI standards. When a rule should apply to every CLI, update the shared source instead of copying the rule here.
