# cfl CLI Reference

> **Covers:** cfl v1.0.36

Reference for the `cfl` command line tool from [open-cli-collective/atlassian-cli](https://github.com/open-cli-collective/atlassian-cli).

## Authentication

**Config file** (recommended): `~/.config/cfl/config.yml`

Set up interactively:
```bash
cfl init
```

Prompts for: Atlassian instance URL, email, API token (from https://id.atlassian.com/manage-profile/security/api-tokens).

Test connectivity:
```bash
cfl config test
```

## Global Flags

| Flag | Description |
|------|-------------|
| `-o, --output FORMAT` | Output format (see SKILL.md "Output Representation and Format"): `table` (default), `plain` |
| `--full` | Inspection additions on supported page/space views and list/search commands. Unsupported commands reject it. Not a content-truncation flag — use `--no-truncate` for that. |
| `--no-color` | Disable colored output |
| `-c, --config PATH` | Override config file location (default: `~/.config/cfl/config.yml`) |

## Command Structure

```
cfl [resource] [action] [ID] [flags]
```

## Pages

| Command | Description |
|---------|-------------|
| `cfl page list --space KEY` | List pages in space |
| `cfl page view PAGE_ID` | View page content as markdown (truncated at 5000 chars by default) |
| `cfl page view PAGE_ID --full` | Add parent ID, creation time, and author ID metadata |
| `cfl page view PAGE_ID --no-truncate` | View full content without truncation |
| `cfl page view PAGE_ID --content-only` | Output content only (no metadata headers); implies `--no-truncate` |
| `cfl page view PAGE_ID --body-format xhtml` | View exact Confluence storage XHTML |
| `cfl page view PAGE_ID --body-format adf` | View exact ADF JSON |
| `cfl page view PAGE_ID --show-macros` | Show macro placeholders (e.g. `[TOC]`) instead of stripping them |
| `cfl page view PAGE_ID --web` | Open page in browser |
| `cfl page create --space KEY --title "TEXT"` | Create page (opens editor) |
| `cfl page create --space KEY --title "TEXT" --file content.md` | Create from file |
| `cfl page create --space KEY --title "TEXT" --parent PAGE_ID` | Create as child page |
| `cfl page edit PAGE_ID` | Edit page (opens editor) |
| `cfl page edit PAGE_ID --file content.md` | Update from file |
| `cfl page edit PAGE_ID --title "New Title"` | Update title only |
| `cfl page edit PAGE_ID --parent PAGE_ID` | Move page to new parent |
| `cfl page copy PAGE_ID --title "Copy Title"` | Copy page |
| `cfl page copy PAGE_ID --title "Copy" --space OTHER` | Copy to different space |
| `cfl page copy PAGE_ID --title "Copy" --no-attachments` | Copy without attachments |
| `cfl page copy PAGE_ID --title "Copy" --no-labels` | Copy without labels |
| `cfl page delete PAGE_ID` | Delete page (with confirmation) |
| `cfl page delete PAGE_ID --force` | Delete without confirmation |

### Create/Edit Flags

| Flag | Description |
|------|-------------|
| `--space KEY` / `-s` | Space key (required for create) |
| `--title "TEXT"` / `-t` | Page title (required for create) |
| `--file PATH` / `-f` | Read content from file |
| `--parent PAGE_ID` / `-p` | Parent page ID |
| `--body-format markdown\|adf\|xhtml` | Input/editor representation; defaults to Markdown |
| `--legacy` | Convert Markdown to storage XHTML instead of ADF; invalid with ADF/XHTML input |
| `--editor` | Open interactive editor |

### Page View Flags

| Flag | Description |
|------|-------------|
| `--no-truncate` | Show full Markdown content without truncation; exact ADF/XHTML are always complete |
| `--content-only` | Output only page content (no metadata headers); implies `--no-truncate` |
| `--body-format markdown\|adf\|xhtml` | Body representation; ADF and XHTML are emitted exactly and never truncated |
| `--show-macros` | Show macro placeholders (e.g. `[TOC]`) instead of stripping them |
| `-w, --web` | Open in browser |

`--full` composes with `--body-format` and is incompatible with `--content-only` and `--web`.

### Page List Flags

| Flag | Description |
|------|-------------|
| `--space KEY` / `-s` | Space key (required) |
| `--limit N` / `-l` | Max results (default 25) |
| `--status STATUS` | Page status: `current`, `archived`, `trashed` (default `current`) |

### Content Piping & Lossless Round-Trip

Markdown round-trip (lossy — macros and some formatting lost):
```bash
# Edit current content via stdin
cfl page view 12345 --content-only | cfl page edit 12345 --legacy
```

Storage-format round-trip (lossless — preserves macros and all formatting):
- Fetch with `cfl page view PAGE_ID --body-format xhtml --content-only`
- Modify the XHTML
- Send it back with `cfl page edit PAGE_ID --body-format xhtml` (stdin or `--file`)

See ManagePage.md for a full walkthrough.

Create from stdin:
```bash
echo "# Hello World" | cfl page create -s DEV -t "My Page"
```

## Search

| Command | Description |
|---------|-------------|
| `cfl search "query"` | Global full-text search |
| `cfl search "query" --space KEY` | Search within space |
| `cfl search "query" --type page` | Search pages only |
| `cfl search --label TAG` | Filter by label |
| `cfl search --title "TEXT"` | Filter by title |
| `cfl search --cql "CQL_QUERY"` | Raw CQL query |

**Scope:** Search is global unless `--space` is explicit; configured `default_space` is not used.
Raw `--cql` cannot be combined with the positional query or any builder flag.

### Search Flags

| Flag | Description |
|------|-------------|
| `--space KEY` / `-s` | Explicitly filter by space key |
| `--type TYPE` / `-t` | Content type: `page`, `blogpost`, `attachment`, `comment` |
| `--label TAG` | Filter by label |
| `--title "TEXT"` | Filter by title (contains) |
| `--cql "QUERY"` | Raw CQL query; mutually exclusive with query, space, type, title, and label inputs |
| `--limit N` / `-l` | Max results, greater than zero (default 25) |

### Common CQL Patterns

| Intent | CQL |
|--------|-----|
| Recently modified pages | `type=page AND lastModified > now('-7d')` |
| Pages in space | `type=page AND space=KEY` |
| Pages by creator | `type=page AND creator=currentUser()` |
| Pages with label | `type=page AND label="TAG"` |
| Pages modified by me | `type=page AND contributor=currentUser()` |
| Blog posts in space | `type=blogpost AND space=KEY` |
| Ancestor (child pages) | `type=page AND ancestor=PAGE_ID` |
| Title match | `type=page AND title~"search term"` |
| Combined filters | `type=page AND space=DEV AND lastModified > now('-7d') AND label="api"` |

## Spaces

| Command | Description |
|---------|-------------|
| `cfl space list` | List all spaces |
| `cfl space list --type global` | List only global spaces |
| `cfl space list --type personal` | List only personal spaces |
| `cfl space list --cursor CURSOR` | Paginate (use cursor from previous response) |
| `cfl space view KEY` | View space details (alias: `get`) |
| `cfl space create --key KEY --name "NAME"` | Create space |
| `cfl space update KEY --name "NAME"` | Update space name |
| `cfl space update KEY --description "TEXT"` | Update space description |
| `cfl space delete KEY` | Delete space (with confirmation) |
| `cfl space delete KEY --force` | Delete without confirmation |

### Space List Flags

| Flag | Description |
|------|-------------|
| `--type TYPE` / `-t` | Filter by space type: `global`, `personal` |
| `--limit N` / `-l` | Max results (default 25) |
| `--cursor CURSOR` | Pagination cursor for next page |

### Space Create Flags

| Flag | Description |
|------|-------------|
| `--key KEY` / `-k` | Space key (required) |
| `--name "NAME"` / `-n` | Space name (required) |
| `--description "TEXT"` / `-d` | Space description |
| `--type TYPE` / `-t` | Space type: `global`, `personal` (default `global`) |

## Attachments

| Command | Description |
|---------|-------------|
| `cfl attachment list --page PAGE_ID` | List attachments on page |
| `cfl attachment list --page PAGE_ID --limit 50` | List with positive custom limit (default 25) |
| `cfl attachment list --page PAGE_ID --unused` | List orphaned attachments (not referenced in page content) |
| `cfl attachment upload --page PAGE_ID --file PATH` | Upload attachment |
| `cfl attachment upload --page PAGE_ID --file PATH -m "comment"` | Upload with comment |
| `cfl attachment download ATT_ID` | Download (uses original filename) |
| `cfl attachment download ATT_ID -O filename` | Download to specific filename |
| `cfl attachment download ATT_ID --force` | Overwrite existing file without warning |
| `cfl attachment delete ATT_ID` | Delete attachment (with confirmation) |
| `cfl attachment delete ATT_ID --force` | Delete without confirmation |

## Output

- Default representation: `agent`; default format: `table`. See SKILL.md "Output Representation and Format" for the full model.
- Use `-o plain` (TSV) for machine-readable output; JSON output was removed — parse the stable plain/table columns, or use `--body-format adf|xhtml` for exact page body representations
- Use `--no-color` to disable colored output
- Data goes to stdout (pipeable)
