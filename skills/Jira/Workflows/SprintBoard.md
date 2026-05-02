# SprintBoard Workflow

Manage sprints and boards — view active sprint, list sprint issues, add issues to sprints.

**Note:** Sprint and board operations require classic API token auth (not bearer/scoped tokens). If operations fail with auth errors, inform user of this Atlassian limitation.

## Intent-to-Flag Mapping

### Action Selection

| User Says | Command | Required |
|-----------|---------|----------|
| "list boards", "show boards" | `jtk boards list` (optionally `--project KEY` to scope) | Nothing |
| "board details" | `jtk boards get ID_OR_NAME` | Board ID or name |
| "current sprint", "active sprint" | `jtk sprints current --board ID_OR_NAME` | Board ID or name |
| "list sprints" | `jtk sprints list --board ID_OR_NAME` | Board ID or name |
| "sprint issues", "what's in the sprint" | `jtk sprints issues SPRINT_ID_OR_NAME` | Sprint ID or name |
| "what's in our current sprint" (shortcut) | `jtk issues list --project KEY --sprint current` | Project key |
| "add to sprint" | `jtk sprints add SPRINT_ID_OR_NAME PROJ-1 PROJ-2 ...` | Sprint ID/name + issue keys |

## Execute

### List Boards

```bash
# All boards (can be long on large instances)
jtk boards list

# Scope to a project — recommended when you already know the project
jtk boards list --project KEY
```

### Get Active Sprint

```bash
# Get board ID/name from customization or ask user
jtk sprints current --board BOARD_ID_OR_NAME
```

To include the sprint goal in the output, add `--extended`.

### List Sprint Issues

Two paths — pick based on what's known:

**Path A: Board-based (when you have a board ID or name)**
```bash
# Get current sprint first
jtk sprints current --board BOARD_ID_OR_NAME

# Then list issues in that sprint
jtk sprints issues SPRINT_ID_OR_NAME
```

**Path B: Project shortcut (when you only have a project key)**
```bash
jtk issues list --project KEY --sprint current
```

Path B is faster when you already know the project key and don't need the sprint metadata (name, start/end dates).

### Add Issues to Sprint

Issues are **positional arguments**, not a flag:

```bash
# Single issue
jtk sprints add SPRINT_ID_OR_NAME PROJ-123

# Multiple issues
jtk sprints add SPRINT_ID_OR_NAME PROJ-123 PROJ-456 PROJ-789
```

## Output Format

- **Boards:** table with ID, Type, Project, Name (column order: `ID | TYPE | PROJECT | NAME`)
- **Sprint list:** table with ID, State, Start, End, Name (column order: `ID | STATE | START | END | NAME`). Sprint goals require `--extended`
- **Sprint current:** sprint metadata with board reference. Sprint Goal requires `--extended`
- **Sprint issues:** table with Key, Summary, Status, Assignee — grouped by status if possible

## Post-Action

After any action:
1. For sprint listings: state total issue count and breakdown by status
2. For add-to-sprint: confirm which issues were added and to which sprint by name
