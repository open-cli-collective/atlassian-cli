---
name: Jira
description: Jira issue tracking and project management via jtk CLI — search, create (single or bulk), update, assign, transition issues, manage sprints and boards, comments, attachments, cross-issue links, parent/sub-task hierarchies. USE WHEN jira, issues, tickets, sprint, standup, backlog, board, jira board, JQL, create issue, create tickets, bulk create, multiple tickets, sub-tasks, parent with children, epic with stories, decompose spec into tickets, turn PRD into tickets, link issues, my tickets, assign issue, move ticket, transition issue, move to done, add comment, attach to issue, jira attachment, jira status, what am I working on.
---

# Jira

Issue tracking and project management via the `jtk` CLI tool ([open-cli-collective/atlassian-cli](https://github.com/open-cli-collective/atlassian-cli)).

## Prerequisites

All workflows share these prerequisites. Workflows do not repeat them — check these before entering any workflow.

1. **jtk installed** — verify with `jtk --version`
2. **Auth configured** — run `jtk init` to set up interactively. Verify with `jtk config test` (reports connection + identity). Run `jtk config show` to see the resolved config file path on your system (typical locations: `~/.config/jira-ticket-cli/config.json` on Linux, `~/Library/Application Support/jira-ticket-cli/config.json` on macOS).
3. **Environment-variable auth** (alternative to config file) — precedence: `JIRA_*` env vars → `ATLASSIAN_*` env vars → config file. Use `ATLASSIAN_URL` / `ATLASSIAN_EMAIL` / `ATLASSIAN_API_TOKEN` for credentials shared with `cfl` (Confluence CLI); use `JIRA_*` to override per-tool. Bearer auth additionally requires `JIRA_CLOUD_ID` (or `ATLASSIAN_CLOUD_ID`) — find the Cloud ID at `https://your-site.atlassian.net/_edge/tenant_info`.
4. **Defaults for missing inputs** — see the **Defaults & Missing Inputs** section below.

## Defaults & Missing Inputs

When the user's request doesn't specify a required input (project key, board ID, etc.), consult `jtk`'s own defaulting mechanisms before asking. If the input is still missing, ask the user — and suggest the one-time setup that would persist the default.

### Project key

Resolution order `jtk` uses: `--project` flag → `JIRA_DEFAULT_PROJECT` env var → `default_project` in config file.

If still missing: ask the user. Then suggest they persist it:

- **Env var (per-shell):** `export JIRA_DEFAULT_PROJECT=PROJ` — add to `.bashrc` / `.zshrc` / equivalent to persist
- **Config file:** edit the `default_project` field in the file shown by `jtk config show`

Env var wins over config. Once set, `jtk issues list` (and other project-scoped commands) work without `--project`.

### Board ID

`jtk` has no default-board mechanism — there's no env var or config key for a default board. If a board ID is needed and the user hasn't provided one, ask which board. Suggest `jtk boards list` or `jtk boards list --project KEY` to discover IDs.

### Sprint, issue key, attachment ID, comment ID, etc.

No defaults exist. Ask the user. Do not guess.

### Assignee / user identity shortcut

For assignment and self-scoped queries, `--assignee me` resolves to the authenticated user — no account ID lookup needed. `jtk me --id` returns the current user's account ID when a raw ID is specifically needed (e.g., embedding in a script or passing to an API-boundary tool that can't consume an email or display name). For regular assign/unassign, just use `--assignee me` — both `jtk issues update` and `jtk issues assign` accept it directly via their resolver.

## Cache Warming (first use per session)

`jtk` caches instance metadata locally — fields, projects, users, issue types, statuses, priorities, resolutions, boards, and link types — to avoid repeated API calls on every invocation. When a session first exercises a workflow that depends on a cold cache (creating issues, creating links, moving issues between projects, resolving user identities for assignment, etc.), the command will fail with an actionable error like `cannot resolve link type "X" from cache — run 'jtk refresh linktypes'` or `cannot resolve issue type ID for "Task" in project KEY from cache — run 'jtk refresh issuetypes'`.

To avoid hitting this mid-workflow, warm the caches once near the start of any session that will do significant write work or user lookups:

```bash
jtk refresh          # refresh everything (recommended)
```

Or refresh just the resources you expect to touch:

```bash
jtk refresh linktypes issuetypes users
```

Both forms are idempotent and fast. Bare `jtk refresh` is the simplest — it fetches every cacheable resource and auto-resolves dependencies; on a warm cache it's cheap. Repeated refreshes within a session are fine. If a command fails mid-run with `run jtk refresh <resource>`, run the suggested refresh (or bare `jtk refresh`) and retry.

Use `jtk refresh --status` to inspect cache freshness without fetching anything.

## Common Errors

| Symptom | Likely Cause | Remedy |
|---------|--------------|--------|
| `unauthorized` / `401` / "invalid credentials" | Missing or expired API token | Run `jtk init` to reconfigure; tokens from https://id.atlassian.com/manage-profile/security/api-tokens |
| `permission denied` on a specific issue/project | Account lacks permission on that project | Verify project membership; ask a project admin to grant access |
| `forbidden: insufficient permissions` on a delete operation (`jtk issues delete`, `jtk links delete`, `jtk comments delete`, `jtk attachments delete`) | Account lacks the specific Delete permission in this project — most Jira Cloud projects restrict Delete Issue permission to admins; other delete permissions (links, comments, attachments) are also separately granted | Inform the user the delete failed due to insufficient permissions. Stop and wait for user direction — do not attempt alternative cleanup paths (transitions, status changes, workarounds) unless the user explicitly asks |
| `not found` on a valid-looking issue key | Wrong project key, or issue deleted, or insufficient permission (Jira returns 404 instead of 403 for unauthorized reads) | Double-check the key; try `jtk projects list` to confirm project visibility |
| Agile operations fail (boards, sprints, automation, dashboards) with auth errors | Using bearer/scoped API token — scoped tokens lack Agile/Automation/Dashboard scopes (Atlassian platform limitation) | Reconfigure with a classic API token via `jtk init` |
| Bearer auth fails to connect or gateway URL is wrong | Missing or wrong Cloud ID | Set `JIRA_CLOUD_ID` (or `ATLASSIAN_CLOUD_ID`), or re-run `jtk init --auth-method bearer` with `--cloud-id`. Find the Cloud ID at `https://your-site.atlassian.net/_edge/tenant_info` |
| `Unbounded JQL queries are not allowed here` from `jtk issues list` | `jtk issues list` ran without `--project` and without a default project set | Pass `--project KEY` or set `JIRA_DEFAULT_PROJECT` / config `default_project` (see Defaults & Missing Inputs) |
| `required flag(s) "jql" not set` | Passing a bare positional query to `jtk issues search` | Use `--jql "QUERY"` — the flag is required |
| `required flag(s) "body" not set` on comment add | Passing comment text positionally | Use `--body "TEXT"` |
| Status change silently does nothing | Trying `jtk issues update --status` (no such flag) | Use `jtk transitions do KEY "Target"` instead |

## Common Patterns

### Output Contract

`jtk` commands follow a text-first output model (per the repo's [Output Artifact Contract](https://github.com/open-cli-collective/atlassian-cli/blob/main/docs/ARTIFACT_CONTRACT.md)). Three global flags shape what gets emitted:

- **`--extended`** — includes admin/schema/audit fields on top of the default output. Use when the user asks for "more detail," "all fields," or similar.
- **`--fulltext`** — disables truncation of descriptions and comments. Use when the user needs full body content (e.g., "show the full description"). `--no-truncate` is a deprecated alias kept during the migration; prefer `--fulltext`.
- **`--id`** — emits only the primary identifier (issue key, account ID, etc.). Takes precedence over `--extended` and `--fulltext`. Use whenever a downstream step will parse the output — no decoration to strip, formatting is stable. **Caveat for scripts:** when a list command's result is truncated (multi-page), the pagination continuation notice (`More results available ...`) is still appended to STDOUT even with `--id`. For command substitution or line-by-line piping, either size `--max` so all results fit on one page, or post-filter with `grep -oE '[A-Z]+-[0-9]+' | head -1` to isolate just the identifier.

`-o table|json|plain` is retained for backward compatibility but hidden from `--help`; prefer `--id` over format-parsing. JSON output is still first-class for scripting needs that require structured data.

### Pagination & Result Sizing

Most list-type commands (`jtk issues list`, `jtk issues search`, `jtk projects list`, `jtk comments list`, `jtk attachments list`, `jtk sprints list`, `jtk boards list`, `jtk users search`, `jtk dashboards list`, etc.) accept `--max N` to cap results. Defaults vary by command (typically 25–50). Commands that paginate also accept `--next-page-token TOKEN` to resume. When a listing is truncated, `jtk` prints a "More results available" notice **on stdout** (not stderr — and this holds even with `--id`; see the `--id` bullet above for the scripting caveat) — honor it or raise `--max` if the user expects more.

### Extracting Issue Keys from URLs

Many commands take an issue key (`PROJ-123`). If the user provides a browse URL, the key is the segment after `/browse/`:

```
https://INSTANCE.atlassian.net/browse/PROJ-123
                                       ^^^^^^^^
```

Use that key directly — no API lookup needed.

## Workflow Routing

| Workflow | Trigger | File |
|----------|---------|------|
| **SearchIssues** | "search jira", "find issues", "JQL", "list issues", "filter issues" | `Workflows/SearchIssues.md` |
| **ManageIssue** | "create issue" (single), "update issue", "assign", "transition", "move to done", "change status" — **single issue only** | `Workflows/ManageIssue.md` |
| **ManageIssueSet** | "create N tickets", "create these tickets", "file three bugs", "bulk update", "update these issues", "parent with sub-tasks", "epic with stories", "link these together", "create and link", "link issue A to B", "add blocks/relates link between two issues", "remove a link" — **any multi-issue operation where the user specifies tickets directly, including link operations between two existing issues (links always involve two endpoints)** | `Workflows/ManageIssueSet.md` |
| **SpecToTickets** | "decompose this spec into tickets", "turn this PRD into tickets", "break this design doc into tasks", "file tickets for this plan" — **any case where the agent is making the ticket-breakdown judgment from a spec/PRD/plan**. Hands off to ManageIssueSet for execution. | `Workflows/SpecToTickets.md` |
| **SprintBoard** | "current sprint", "sprint issues", "boards", "add to sprint", "list sprints" | `Workflows/SprintBoard.md` |
| **QuickStatus** | "my jira", "what am I working on", "jira status", "my tickets", "my issues" | `Workflows/QuickStatus.md` |
| **ManageComments** | "add comment", "view comments", "comment on issue", "list comments" | `Workflows/ManageComments.md` |
| **ManageAttachments** | "attach file", "list attachments", "download attachment", "upload attachment" | `Workflows/ManageAttachments.md` |

## Quick Reference

| Operation | Command |
|-----------|---------|
| Search by JQL | `jtk issues search --jql "JQL_QUERY"` |
| List project issues | `jtk issues list --project KEY` |
| List current sprint issues | `jtk issues list --project KEY --sprint current` |
| Get issue details | `jtk issues get PROJ-123` |
| Create issue | `jtk issues create --project KEY --type Task --summary "..."` |
| Assign (me / email / display name / account ID) | `jtk issues update PROJ-123 --assignee VALUE` (or equivalently `jtk issues assign PROJ-123 VALUE`) |
| Unassign | `jtk issues update PROJ-123 --assignee none` (or equivalently `jtk issues assign PROJ-123 --unassign`) |
| Transition issue | `jtk transitions list PROJ-123` then `jtk transitions do PROJ-123 "Done"` |
| Current user | `jtk me` (or `jtk me --id` for just the account ID) |
| Current sprint | `jtk sprints current --board ID` |
| Add comment | `jtk comments add PROJ-123 --body "..."` |

**Full CLI reference:** load `CliReference.md`

## Examples

**Example 1: Quick status check**
```
User: "What am I working on in Jira?"
→ Invokes QuickStatus workflow
→ Searches for issues assigned to current user in active statuses
→ Returns grouped list by status
```

**Example 2: Create and assign a bug**
```
User: "Create a bug in PROJ for broken login page, assign to me"
→ Invokes ManageIssue workflow
→ Creates issue with --type Bug --summary "Broken login page" --assignee me
→ Returns issue key and link
```

**Example 3: Search with JQL**
```
User: "Find all high priority issues in PROJ updated this week"
→ Invokes SearchIssues workflow
→ Runs: jtk issues search --jql "project = PROJ AND priority = High AND updated >= -7d"
→ Returns formatted results
```
