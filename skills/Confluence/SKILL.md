---
name: Confluence
description: Confluence wiki and knowledge base management via cfl CLI — search, create, view, edit pages, manage spaces, attachments. USE WHEN confluence, wiki, knowledge base, runbook, documentation, docs page, wiki search, confluence page, create page, edit page, move page, search confluence, confluence space, view page, page attachments, attach to page, CQL, confluence search.
---

## Customization

**Before executing, check for user customizations at:**
`~/.claude/PAI/USER/SKILLCUSTOMIZATIONS/Confluence/`

If this directory exists, load and apply any PREFERENCES.md, configurations, or resources found there. These override default behavior. If the directory does not exist, proceed with skill defaults.

If PREFERENCES.md exists but is malformed, unreadable, or missing referenced values, treat it the same as "no preferences" and proceed with skill defaults — do not abort the workflow on a broken preferences file.

# Confluence

Wiki and knowledge base management via the `cfl` CLI tool ([open-cli-collective/atlassian-cli](https://github.com/open-cli-collective/atlassian-cli)).

## Prerequisites

All workflows share these prerequisites. Workflows do not repeat them — check these before entering any workflow.

1. **cfl installed** — verify with `cfl --version`
2. **Auth configured** — config file at `~/.config/cfl/config.yml` (run `cfl init` to set up). Verify with `cfl config test` (reports connection + identity).
3. **Personal space keys** — in Confluence, space keys that start with `~` are personal spaces (e.g., `~aaron`). The `~` is part of the key, not a shell home-directory shortcut. In shells that expand leading `~` (bash, zsh), quote the key: `cfl space view '~aaron'`.
4. **Customization** (optional) — `~/.claude/PAI/USER/SKILLCUSTOMIZATIONS/Confluence/PREFERENCES.md` for defaults
   - If no customization AND the request requires a space key not specified by the user: abort with a message explaining how to create PREFERENCES.md (see template in SearchPages.md)

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
