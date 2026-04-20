# cfl CLI Reference

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
| `-o, --output FORMAT` | Output format (see SKILL.md "Output Representation and Format"): `table` (default), `json`, `plain` |
| `--full` | Inspection-oriented representation (see SKILL.md). Not a content-truncation flag — for `page view` content truncation, use `--no-truncate`. |
| `--raw` | Source-faithful representation (see SKILL.md). Command-specific. |
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
| `cfl page view PAGE_ID --no-truncate` | View full content without truncation |
| `cfl page view PAGE_ID --content-only` | Output content only (no metadata headers); implies `--no-truncate` |
| `cfl page view PAGE_ID --raw` | View raw Confluence storage format (XHTML) instead of markdown |
| `cfl page view PAGE_ID --show-macros` | Show macro placeholders (e.g. `[TOC]`) instead of stripping them |
| `cfl page view PAGE_ID --web` | Open page in browser |
| `cfl page view PAGE_ID -o json` | Full JSON output (body always included in full) |
| `cfl page create --space KEY --title "TEXT"` | Create page (opens editor) |
| `cfl page create --space KEY --title "TEXT" --file content.md` | Create from file |
| `cfl page create --space KEY --title "TEXT" --parent PAGE_ID` | Create as child page |
| `cfl page edit PAGE_ID` | Edit page (opens editor) |
| `cfl page edit PAGE_ID --file content.md` | Update from file |
| `cfl page edit PAGE_ID --title "New Title"` | Update title only |
| `cfl page edit PAGE_ID --parent PAGE_ID` | Move page to new parent |
| `cfl page copy PAGE_ID --title "Copy Title"` | Copy page |
| `cfl page copy PAGE_ID --title "Copy" --space OTHER` | Copy to different space |
| `cfl page delete PAGE_ID` | Delete page (with confirmation) |
| `cfl page delete PAGE_ID --force` | Delete without confirmation |

### Create/Edit Flags

| Flag | Description |
|------|-------------|
| `--space KEY` / `-s` | Space key (required for create) |
| `--title "TEXT"` / `-t` | Page title (required for create) |
| `--file PATH` / `-f` | Read content from file |
| `--parent PAGE_ID` / `-p` | Parent page ID |
| `--legacy` | Use legacy editor format instead of cloud (ADF) |
| `--no-markdown` | Disable markdown conversion (use raw XHTML) |
| `--storage` | Input is Confluence storage format (XHTML); sent via storage representation API regardless of the page's editor type |
| `--editor` | Open interactive editor |

### Page View Flags

| Flag | Description |
|------|-------------|
| `--no-truncate` | Show full content without truncation |
| `--content-only` | Output only page content (no metadata headers); implies `--no-truncate` |
| `--raw` | Raw Confluence storage format (XHTML) |
| `--show-macros` | Show macro placeholders (e.g. `[TOC]`) instead of stripping them |
| `-w, --web` | Open in browser |

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
- Fetch the page with `-o json` (the `content` field holds the raw storage XHTML)
- Modify the XHTML
- Send the modified XHTML back: `cfl page edit PAGE_ID --storage` (reads from stdin, or pass via `--file`)

See ViewPage.md for the JSON output structure and ManagePage.md for a full walkthrough.

Create from stdin:
```bash
echo "# Hello World" | cfl page create -s DEV -t "My Page"
```

## Search

| Command | Description |
|---------|-------------|
| `cfl search "query"` | Full-text search |
| `cfl search "query" --space KEY` | Search within space |
| `cfl search "query" --type page` | Search pages only |
| `cfl search --label TAG` | Filter by label |
| `cfl search --title "TEXT"` | Filter by title |
| `cfl search --cql "CQL_QUERY"` | Raw CQL query |

**Note:** When `--cql` is provided, it takes precedence over the positional `[query]` argument. Don't combine them.

### Search Flags

| Flag | Description |
|------|-------------|
| `--space KEY` / `-s` | Filter by space key |
| `--type TYPE` / `-t` | Content type: `page`, `blogpost`, `attachment`, `comment` |
| `--label TAG` | Filter by label |
| `--title "TEXT"` | Filter by title (contains) |
| `--cql "QUERY"` | Raw CQL query (advanced). Takes precedence over positional query. |
| `--limit N` / `-l` | Max results (default 25) |

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
| `cfl space delete KEY` | Delete space |

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
| `cfl attachment list --page PAGE_ID --limit 50` | List with custom limit (default 25) |
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
- Use `-o json` for machine-readable output (useful for scripting — e.g. extracting IDs from search results or storage-format body from a page)
- Use `-o plain` for plain text
- Use `--no-color` to disable colored output
- Data goes to stdout (pipeable)
