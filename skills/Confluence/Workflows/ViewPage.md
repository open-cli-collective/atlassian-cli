# ViewPage Workflow

View Confluence page content and metadata.

## Intent-to-Flag Mapping

### Truncation Rule

See SKILL.md "Output Representation and Format" for artifact breadth and page-body representation.

The default Markdown body is truncated at 5000 chars; `--show-macros` follows the same rule. Exact ADF and XHTML bodies are never truncated. Use `--no-truncate`, or `--content-only` (which implies it), for complete Markdown output.

`--content-only` already implies `--no-truncate`; don't combine them.

### View Mode

| User Says | Command | When to Use |
|-----------|---------|-------------|
| "view page", "show page", "read page" | `cfl page view PAGE_ID` | Default markdown view (subject to truncation) |
| "show full page", "all content", "no truncation" | `cfl page view PAGE_ID --no-truncate` | Full content without truncation |
| "just the content", "content only" | `cfl page view PAGE_ID --content-only` | Content without metadata headers (implies `--no-truncate`) |
| "XHTML", "storage format" | `cfl page view PAGE_ID --body-format xhtml` | Complete exact storage XHTML |
| "ADF", "Atlassian document format" | `cfl page view PAGE_ID --body-format adf` | Complete exact ADF JSON |
| "show macros" | `cfl page view PAGE_ID --show-macros` | Preserve macro placeholders like `[TOC]` (subject to truncation) |
| "open in browser", "open page" | `cfl page view PAGE_ID --web` | Opens in default browser |

### Finding Page IDs

If the user provides a page title instead of ID, search first:
```bash
cfl search --title "Page Title" --type page --space KEY
```

Then use the page ID from the results.

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

# Exact storage XHTML — never truncated
cfl page view PAGE_ID --body-format xhtml

# Exact ADF JSON — never truncated
cfl page view PAGE_ID --body-format adf

# Open in browser
cfl page view PAGE_ID --web
```

### Macro Handling

By default, Confluence macros (TOC, include, status, etc.) are stripped from the markdown output. If the page structure depends on macros, use `--show-macros` to preserve their placeholders (e.g. `[TOC]`) so the structure remains visible.

## Output Format

Present page content clearly:
- Show page title, space, last modified date, and version at the top
- Show the page body in markdown format
- If truncated, note that and offer `--no-truncate` or `--content-only`
- For exact formats, identify ADF JSON or storage XHTML

## Post-Action

After viewing:
1. If content was truncated, mention it and offer `--no-truncate` for full content
2. If the user might want to edit, mention the page ID for reference
3. If page has child pages, note their existence
