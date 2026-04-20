# ViewPage Workflow

View Confluence page content and metadata.

## Intent-to-Flag Mapping

### Truncation Rule

See SKILL.md "Output Representation and Format" for the global representation (`agent`/`full`/`raw`) and format (`table`/`json`/`plain`) concepts.

The default markdown body is truncated at 5000 chars. This applies to the default view, `--raw`, and `--show-macros`. To get the full body, combine any of these with `--no-truncate`, or use `--content-only` (which implies `--no-truncate`). `-o json` is always full — no truncation, regardless of representation or flags.

`--content-only` already implies `--no-truncate`; don't combine them.

### View Mode

| User Says | Command | When to Use |
|-----------|---------|-------------|
| "view page", "show page", "read page" | `cfl page view PAGE_ID` | Default markdown view (subject to truncation) |
| "show full page", "all content", "no truncation" | `cfl page view PAGE_ID --no-truncate` | Full content without truncation |
| "just the content", "content only" | `cfl page view PAGE_ID --content-only` | Content without metadata headers (implies `--no-truncate`) |
| "raw format", "XHTML", "storage format" | `cfl page view PAGE_ID --raw` | Raw Confluence storage format (subject to truncation) |
| "show macros" | `cfl page view PAGE_ID --show-macros` | Preserve macro placeholders like `[TOC]` (subject to truncation) |
| "open in browser", "open page" | `cfl page view PAGE_ID --web` | Opens in default browser |
| "page as JSON" | `cfl page view PAGE_ID -o json` | Full JSON output (body always included in full — no truncation) |

### Finding Page IDs

If the user provides a page title instead of ID, search first:
```bash
cfl search --title "Page Title" --type page --space KEY
```

Then use the page ID from the results. For scripted extraction, add `-o json` — see SearchPages.md for the output structure.

If the user provides a Confluence URL instead of a page ID, see "Extracting Page IDs from URLs" in SKILL.md.

## Execute

```bash
# Standard view (markdown, truncated at 5000 chars)
cfl page view PAGE_ID

# Full content (no truncation)
cfl page view PAGE_ID --no-truncate

# Content only (for piping or clean reading; implies --no-truncate)
cfl page view PAGE_ID --content-only

# Preserve macros that would otherwise be stripped
cfl page view PAGE_ID --show-macros

# Raw storage format (XHTML) — also subject to default truncation
cfl page view PAGE_ID --raw

# Full raw storage format (no truncation)
cfl page view PAGE_ID --raw --no-truncate

# Open in browser
cfl page view PAGE_ID --web
```

### Macro Handling

By default, Confluence macros (TOC, include, status, etc.) are stripped from the markdown output. If the page structure depends on macros, use `--show-macros` to preserve their placeholders (e.g. `[TOC]`) so the structure remains visible.

## JSON Output Structure

With `-o json`, the output has this structure:

```json
{
  "id": "...",
  "title": "...",
  "spaceId": "...",
  "spaceKey": "...",
  "parentId": "...",
  "content": "..."
}
```

The `content` field holds the full storage-format XHTML with no truncation, regardless of `--no-truncate`. Version and timestamp fields are not included in this output — use the default table view if you need those.

## Output Format

Present page content clearly:
- Show page title, space, last modified date, and version at the top
- Show the page body in markdown format
- If truncated, note that and offer `--no-truncate` or `--content-only`
- For raw format, note it's XHTML storage format

## Post-Action

After viewing:
1. If content was truncated, mention it and offer `--no-truncate` for full content
2. If the user might want to edit, mention the page ID for reference
3. If page has child pages, note their existence
