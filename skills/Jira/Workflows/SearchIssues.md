# SearchIssues Workflow

Search, list, and filter Jira issues using JQL or project-based queries.

## Intent-to-Flag Mapping

### Search Method

| User Says | Command | When to Use |
|-----------|---------|-------------|
| "search for", "find", "JQL", custom query | `jtk issues search --jql "JQL"` | Flexible JQL-based search |
| "list issues in PROJECT" | `jtk issues list --project KEY` | Simple project listing |
| "in current sprint" | `jtk issues list --project KEY --sprint current` | Current-sprint shortcut |
| "get ISSUE-123", "show me ISSUE-123" | `jtk issues get PROJ-123` | Single issue details |

**Note:** `jtk issues search` **requires** the `--jql` flag — a bare positional query is not accepted.

### Common Filters (JQL Building Blocks)

| User Says | JQL Fragment |
|-----------|-------------|
| "my issues", "assigned to me" | `assignee = currentUser()` |
| "unassigned" | `assignee is EMPTY` |
| "bugs", "bug type" | `type = Bug` |
| "stories" | `type = Story` |
| "tasks" | `type = Task` |
| "high priority" | `priority = High` |
| "critical" | `priority = Critical` |
| "open", "not done" | `status != Done` |
| "in progress" | `status = "In Progress"` |
| "updated recently", "updated this week" | `updated >= -7d` |
| "created today" | `created >= startOfDay()` |
| "overdue" | `duedate < now() AND status != Done` |
| "in current sprint" | `sprint in openSprints()` |

### Combining Filters

Build JQL by combining fragments with `AND`:
```
User: "Find my open bugs in PROJ"
→ JQL: project = PROJ AND assignee = currentUser() AND type = Bug AND status != Done
```

## Execute

Based on the user's request, construct and run the appropriate command:

```bash
# JQL search
jtk issues search --jql "PROJECT_AND_FILTER_JQL"

# Project listing
jtk issues list --project KEY

# Current sprint listing
jtk issues list --project KEY --sprint current

# Single issue
jtk issues get PROJ-123
```

### Result Sizing & Field Control

Both `search` and `list` support the same paging/field flags:

| Flag | Description |
|------|-------------|
| `--max N` / `-m N` | Maximum results (default 25; auto-paginates) |
| `--next-page-token TOKEN` | Resume from previous page token |
| `--all-fields` | Include all fields (descriptions, etc.) |
| `--fields summary,status,customfield_10005` | Comma-separated list of specific fields |

Examples:
```bash
# Get up to 200 results
jtk issues search --jql "project = PROJ" --max 200

# Get specific custom fields (e.g. Story Points)
jtk issues search --jql "project = PROJ" --fields summary,status,customfield_10005

# Include full descriptions
jtk issues search --jql "project = PROJ" --all-fields
```

## Output Format

Present results clearly:
- For lists: table format with Key, Summary, Status, Assignee, Priority
- For single issues: full detail view
- If no results: state clearly, suggest broadening the query

### Scripting / Parsing Output

When the next step depends on extracting an identifier from the result (e.g. feeding an issue key into another command), use the global `--id` flag so the CLI emits only the primary identifier — no decoration to strip:

```bash
# Get all keys in a project as a flat list
jtk issues list --project KEY --id

# Get the single issue key this search would match
jtk issues search --jql "summary ~ \"login bug\"" --max 1 --id

# Feed keys into another command
jtk issues list --project KEY --sprint current --id | xargs -I{} jtk issues get {}
```

Use `--id` whenever a downstream step parses the output. Avoid trying to pattern-match against the default table/plain output — it's human-oriented and may change formatting.

## Post-Action

After returning results:
1. State the result count
2. If results are large, offer to narrow the query
3. If no results, suggest broadening or adjusting filters

## Setting a Default Project

If the user routinely works in the same project and hasn't told you which one, ask — then suggest they make the default stick:

- **Env var:** `export JIRA_DEFAULT_PROJECT=PROJ` (per-shell; add to `.bashrc` / `.zshrc` / etc. to persist)
- **Config file:** edit the `default_project` field in the file shown by `jtk config show` (typical paths: `~/.config/jira-ticket-cli/config.json` on Linux, `~/Library/Application Support/jira-ticket-cli/config.json` on macOS)

Env var wins over config file. Once set, `jtk issues list` without `--project` will use the default.
