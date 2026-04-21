# SprintBoard Workflow

Manage sprints and boards — view active sprint, list sprint issues, add issues to sprints.

**Note:** Sprint and board operations require classic API token auth (not bearer/scoped tokens). If operations fail with auth errors, inform user of this Atlassian limitation.

## Intent-to-Flag Mapping

### Action Selection

| User Says | Command | Required |
|-----------|---------|----------|
| "list boards", "show boards" | `jtk boards list` (optionally `--project KEY` to scope) | Nothing |
| "board details" | `jtk boards get ID` | Board ID |
| "current sprint", "active sprint" | `jtk sprints current --board ID` | Board ID |
| "list sprints" | `jtk sprints list --board ID` | Board ID |
| "sprint issues", "what's in the sprint" | `jtk sprints issues SPRINT_ID` | Sprint ID |
| "what's in our current sprint" (shortcut) | `jtk issues list --project KEY --sprint current` | Project key |
| "add to sprint" | `jtk sprints add SPRINT_ID PROJ-1 PROJ-2 ...` | Sprint ID + issue keys |

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
# Get board ID from customization or ask user
jtk sprints current --board BOARD_ID
```

### List Sprint Issues

Two paths — pick based on what's known:

**Path A: Board-based (when you have a board ID)**
```bash
# Get current sprint first
jtk sprints current --board BOARD_ID

# Then list issues in that sprint
jtk sprints issues SPRINT_ID
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
jtk sprints add SPRINT_ID PROJ-123

# Multiple issues
jtk sprints add SPRINT_ID PROJ-123 PROJ-456 PROJ-789
```

## Output Format

- **Boards:** table with ID, Name, Type (Scrum/Kanban)
- **Sprint info:** Name, State (active/future/closed), Start/End dates
- **Sprint issues:** table with Key, Summary, Status, Assignee — grouped by status if possible

## Post-Action

After any action:
1. For sprint listings: state total issue count and breakdown by status
2. For add-to-sprint: confirm which issues were added and to which sprint by name
