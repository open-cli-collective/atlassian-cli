---
name: Confluence
description: Confluence wiki and knowledge base management — search, create, view, edit pages, manage spaces, attachments. USE WHEN confluence, wiki, knowledge base, runbook, documentation, docs page, wiki search, confluence page, create page, edit page, move page, search confluence, confluence space, view page, page attachments, attach to page, CQL, confluence search.
---

# Confluence

> **Skill expects:** cfl v1.0.36+

Wiki and knowledge base management via the `cfl` CLI tool ([open-cli-collective/atlassian-cli](https://github.com/open-cli-collective/atlassian-cli)).

## Prerequisites

All workflows share these prerequisites. Workflows do not repeat them — check these before entering any workflow.

1. **cfl installed** — verify with `cfl --version`
2. **Auth configured** — run `cfl init` to set up interactively. Verify with `cfl config test` (reports connection + identity). Run `cfl config show` to see the resolved config file path on your system (typical location: `~/.config/cfl/config.yml` on Linux/macOS; `$XDG_CONFIG_HOME/cfl/config.yml` if set).
3. **Environment-variable auth** (alternative to config file) — precedence: `CFL_*` env vars → `ATLASSIAN_*` env vars → config file. Use `ATLASSIAN_URL` / `ATLASSIAN_EMAIL` / `ATLASSIAN_API_TOKEN` for credentials shared with `jtk` (Jira CLI); use `CFL_*` to override per-tool. Bearer auth additionally requires `CFL_CLOUD_ID` (or `ATLASSIAN_CLOUD_ID`) — find the Cloud ID at `https://your-site.atlassian.net/_edge/tenant_info`.
4. **Personal space keys** — in Confluence, space keys that start with `~` are personal spaces (e.g., `~aaron`). The `~` is part of the key, not a shell home-directory shortcut. In shells that expand leading `~` (bash, zsh), quote the key: `cfl space view '~aaron'`.
5. **Defaults for missing inputs** — see the **Defaults & Missing Inputs** section below.

## Defaults & Missing Inputs

When the user's request doesn't specify a required input (space key, page ID, etc.), consult `cfl`'s own defaulting mechanisms before asking. If the input is still missing, ask the user — and suggest the one-time setup that would persist the default.

### Space key

Resolution order `cfl` uses: `--space` flag → `CFL_DEFAULT_SPACE` env var → `default_space` in config file.

If still missing: ask the user. Then suggest they persist it:

- **Env var (per-shell):** `export CFL_DEFAULT_SPACE=KEY` — add to `.bashrc` / `.zshrc` / equivalent to persist
- **Config file:** edit the `default_space` field in the file shown by `cfl config show`

Env var wins over config. Once set, space-scoped commands work without `--space`.

### Page ID, attachment ID, etc.

No defaults exist. Ask the user. Do not guess. If the user provides a Confluence URL, extract the page ID directly from the `/pages/` segment — see **Extracting Page IDs from URLs** below.

## Common Patterns

### Output Representation and Format

`cfl` distinguishes two independent output concerns (per the repo's [Artifact Contract](../../docs/ARTIFACT_CONTRACT.md)):

- **Representation** — what content is shown:
  - `agent` (default) — curated, action-oriented, LLM-optimized
  - `full` (`--full`) — inspection-oriented, additional fields (dates, authors, versions)
  - `raw` (`--raw`) — source-faithful content (e.g., XHTML instead of markdown). Command-specific; only supported where source transformation occurs (currently `page view`).
- **Output format** — how it's rendered: `table` (default), `json` (`-o json`), `plain` (`-o plain`)

They combine freely — e.g., `--full -o json` returns the inspection representation as JSON.

### Extracting Page IDs from URLs

Many workflows (ViewPage, ManagePage, ManageAttachments) and CQL filters (`ancestor=PAGE_ID`) take a numeric `PAGE_ID`. If the user provides a Confluence URL, the page ID is the path segment immediately after `/pages/`:

```
https://INSTANCE.atlassian.net/wiki/spaces/SPACEKEY/pages/PAGE_ID/Page-Title-Slug
                                                        ^^^^^^^^
```

Use that numeric segment as the `PAGE_ID` in any command that takes one.

## Common Errors

| Symptom | Likely Cause | Remedy |
|---------|--------------|--------|
| `unauthorized` / `401` / "invalid credentials" | Missing or expired API token | Run `cfl init` to reconfigure; tokens from https://id.atlassian.com/manage-profile/security/api-tokens |
| `cfl config test` fails after `cfl init` | URL typo, wrong instance, or token scoped to a different product | Re-run `cfl init` and double-check the URL and token |
| `permission denied` on a specific page/space | Account lacks permission on that space | Verify space membership; ask a space admin to grant access |
| `not found` on a valid-looking page ID | Wrong ID, page deleted/archived, or insufficient permission (Confluence may return 404 for unauthorized reads) | Try `cfl search --title "..." --space KEY` to re-locate |
| Page body looks empty or missing structure | Macros stripped by default markdown rendering | Use `cfl page view ID --show-macros` to preserve macro placeholders, or `--raw` for full storage format |
| Edit via markdown loses formatting | Markdown round-trip is lossy for macro-rich pages | Use the storage-format round-trip (fetch via `-o json`, modify the `content` field, send back with `--storage`) — see ManagePage.md |

## Workflow Routing

| Workflow | Trigger | File |
|----------|---------|------|
| **SearchPages** | "search confluence", "find pages", "CQL", "search wiki" | `Workflows/SearchPages.md` |
| **ManagePage** | "create page", "edit page", "update page", "move page", "reparent page", "delete page", "copy page", "rename page" | `Workflows/ManagePage.md` |
| **ViewPage** | "view page", "show page", "read page", "open page" | `Workflows/ViewPage.md` |
| **ManageSpaces** | "list spaces", "create space", "space details", "update space" | `Workflows/ManageSpaces.md` |
| **ManageAttachments** | "attach file", "list attachments", "download attachment", "upload to page" | `Workflows/ManageAttachments.md` |

## Quick Reference

| Operation | Command |
|-----------|---------|
| Search pages | `cfl search "query" --space KEY --type page` |
| Search with CQL | `cfl search --cql "CQL_QUERY"` |
| List pages in space | `cfl page list --space KEY` |
| View page (truncated) | `cfl page view PAGE_ID` |
| View full page content | `cfl page view PAGE_ID --no-truncate` |
| View page content-only (pipe-friendly) | `cfl page view PAGE_ID --content-only` |
| Create page | `cfl page create --space KEY --title "..." --file content.md` |
| Edit page | `cfl page edit PAGE_ID --file content.md` |
| List spaces | `cfl space list` |
| Upload attachment | `cfl attachment upload --page PAGE_ID --file path` |

**Full CLI reference:** load `CliReference.md`

## Examples

**Example 1: Search for documentation**
```
User: "Search confluence for deployment guide"
-> Invokes SearchPages workflow
-> Runs: cfl search "deployment guide" --type page
-> Returns formatted results with page IDs, titles, spaces
```

**Example 2: Create a new page**
```
User: "Create a confluence page in DEV space titled 'API Reference'"
-> Invokes ManagePage workflow
-> Runs: cfl page create --space DEV --title "API Reference"
-> Opens editor or accepts piped content
-> Returns page ID and link
```

**Example 3: View a page**
```
User: "Show me confluence page 12345"
-> Invokes ViewPage workflow
-> Runs: cfl page view 12345
-> Returns page content in markdown format (truncated at 5000 chars by default)
```
