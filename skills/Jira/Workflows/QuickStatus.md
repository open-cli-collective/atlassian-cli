# QuickStatus Workflow

Fast "what am I working on?" overview — shows current user's active Jira issues.

## Intent-to-Flag Mapping

| User Says | JQL Query |
|-----------|-----------|
| "my jira", "my tickets", "my issues" | `assignee = currentUser() AND status != Done ORDER BY updated DESC` |
| "what am I working on" | `assignee = currentUser() AND status = "In Progress" ORDER BY updated DESC` |
| "my open issues" | `assignee = currentUser() AND status != Done ORDER BY priority DESC` |
| "anything overdue" | `assignee = currentUser() AND duedate < now() AND status != Done ORDER BY duedate ASC` |
| "my recent" | `assignee = currentUser() ORDER BY updated DESC` |
| "jira status" (with project from prefs) | `assignee = currentUser() AND project = KEY AND status != Done ORDER BY updated DESC` |

## Execute

```bash
jtk issues search --jql "SELECTED_JQL_QUERY"
```

If customization provides a default project, scope the query:
```bash
jtk issues search --jql "assignee = currentUser() AND project = KEY AND status != Done ORDER BY updated DESC"
```

Add `--max N` if the user has many issues and wants a larger result set (default is 25):
```bash
jtk issues search --jql "assignee = currentUser() AND status != Done" --max 100
```

## Output Format

Present as a concise status dashboard. The CLI returns a table by default; the skill post-processes it into a grouped view:

```
## My Jira Status

### In Progress (3)
- PROJ-123: Fix login validation — High
- PROJ-456: Update API docs — Medium
- PROJ-789: Refactor auth module — Medium

### To Do (2)
- PROJ-101: Add rate limiting — High
- PROJ-102: Update dependencies — Low

### In Review (1)
- PROJ-200: New dashboard widget — Medium
```

Group by status, show key + summary + priority. Keep it scannable.

If no results: "No open issues assigned to you." and suggest checking project scope.

## Post-Action

1. State total issue count across all statuses
2. If any issues are overdue, flag them explicitly
3. If results span multiple projects, note which projects are represented
