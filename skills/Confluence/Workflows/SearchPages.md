# SearchPages Workflow

Search, find, and filter Confluence pages using full-text search, CQL, or space-based queries.

## Intent-to-Flag Mapping

### Search Method

| User Says | Command | When to Use |
|-----------|---------|-------------|
| "search for", "find pages", custom query | `cfl search "query"` | Full-text search |
| "search in SPACE" | `cfl search "query" --space KEY` | Space-scoped search |
| "find pages with label" | `cfl search --label TAG` | Label-based search |
| "search by title" | `cfl search --title "text"` | Title-based search |
| "CQL", advanced query | `cfl search --cql "CQL"` | Raw CQL query |
| "list pages in SPACE" | `cfl page list --space KEY` | Simple space listing |

**Scope:** Searches are global unless `--space` is provided explicitly. `default_space` is not applied.
Raw `--cql` cannot be combined with a positional query or `--space`, `--type`, `--title`, or `--label`.

### Common Filters (CQL Building Blocks)

| User Says | CQL Fragment |
|-----------|-------------|
| "pages only" | `type=page` |
| "blog posts" | `type=blogpost` |
| "in SPACE" | `space=KEY` |
| "with label TAG" | `label="TAG"` |
| "updated recently", "updated this week" | `lastModified > now('-7d')` |
| "created today" | `created > now('-1d')` |
| "my pages", "pages I created" | `creator=currentUser()` |
| "pages I edited" | `contributor=currentUser()` |
| "descendant pages of PAGE_ID" (all nested levels) | `ancestor=PAGE_ID` |
| "title contains" | `title~"search term"` |

### Combining Filters

Build CQL by combining fragments with `AND`:
```
User: "Find pages I created in DEV space with label 'api'"
-> CQL: type=page AND space=DEV AND creator=currentUser() AND label="api"
```

## Execute

Based on the user's request, construct and run the appropriate command:

```bash
# Full-text search
cfl search "query text" --type page

# Space-scoped search
cfl search "query text" --space KEY --type page

# CQL search
cfl search --cql "type=page AND space=KEY AND lastModified > now('-7d')"

# Simple space listing
cfl page list --space KEY
```

Use a positive `--limit N` to control result count (default 25).

### Scripting / Parsing Output

When the next step depends on extracting a page ID from the results, use `-o plain`. List and search output is TSV with a header row; the first column is `ID`.

```bash
cfl search --title "Release plan" --space DEV -o plain | awk -F '\t' 'NR > 1 { print $1 }'
```

The `--title` filter does substring matching, so multiple pages may be returned — narrow with `--space` when you need a single result.

Avoid pattern-matching against default `table` output — it's human-oriented and layout may change. Use `-o plain` TSV when a downstream step parses the response.

## Output Format

Present results clearly:
- For search results: table with Page ID, Title, Space, Last Modified
- For page listings: table with ID, Title, Status, Version
- If no results: state clearly, suggest broadening the query or checking space key

## Post-Action

After returning results:
1. State the result count
2. If results are large, offer to narrow the query
3. If no results, suggest broadening or adjusting filters
4. Offer to view any specific page from the results

## Search Scope

Omit `--space` only when global search is intended. If the user asks for a
space-scoped search without naming the space, ask for the key; do not substitute
the configured default.
