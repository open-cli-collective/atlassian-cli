# jtk CLI Reference

Reference for the `jtk` command line tool from [open-cli-collective/atlassian-cli](https://github.com/open-cli-collective/atlassian-cli).

## Authentication

**Config file** (recommended): typical paths — `~/.config/jira-ticket-cli/config.json` (Linux), `~/Library/Application Support/jira-ticket-cli/config.json` (macOS). Run `jtk config show` for the resolved path on your system.

Set up interactively:
```bash
jtk init
```

Verify:
```bash
jtk config test    # reports connection + identity
jtk config show    # shows config path + current values (credentials masked)
```

Prompts for: Atlassian instance URL, email, API token (from https://id.atlassian.com/manage-profile/security/api-tokens).

**Env-var auth** (alternative / overrides): precedence `JIRA_*` → `ATLASSIAN_*` → config. Primary vars: `*_URL`, `*_EMAIL`, `*_API_TOKEN`, `*_AUTH_METHOD` (`basic` or `bearer`), `*_CLOUD_ID` (bearer only). Use `ATLASSIAN_*` for credentials shared with `cfl` (Confluence CLI). `JIRA_DEFAULT_PROJECT` sets a default project key.

**Bearer auth** (scoped service account tokens): Does NOT support Agile operations (boards, sprints, automation, dashboards) due to Atlassian platform limitations. Requires a Cloud ID — find it at `https://your-site.atlassian.net/_edge/tenant_info`. Use classic API token auth for full functionality.

## Global Flags

| Flag | Description |
|------|-------------|
| `--extended` | Include admin/schema/audit fields in output |
| `--fulltext` | Disable truncation of descriptions and comments (`--no-truncate` is a deprecated alias kept during migration) |
| `--id` | Emit only the primary identifier (useful for scripting). Takes precedence over `--extended` and `--fulltext` |
| `--no-color` | Disable colored output |
| `-v, --verbose` | Enable verbose output |

> `-o table|json|plain` (`--output` / `-o`) is retained for backward compatibility but hidden from `--help`. Prefer `--id` over format-parsing. JSON output is still first-class for scripting needs that require structured data.

## Command Structure

```
jtk [resource] [action] [KEY/ID] [flags]
```

## Current User

| Command | Description |
|---------|-------------|
| `jtk me` | Show current authenticated user |
| `jtk me --id` | Print just the account ID (script-friendly) |

## Issues

| Command | Description |
|---------|-------------|
| `jtk issues list --project KEY` | List issues in project |
| `jtk issues list --project KEY --sprint current` | List issues in current sprint |
| `jtk issues get PROJ-123` | Get issue details |
| `jtk issues create --project KEY --type TYPE --summary "TEXT"` | Create issue |
| `jtk issues update PROJ-123 --field FIELD=VALUE` | Update issue fields |
| `jtk issues search --jql "JQL"` | Search with JQL query (flag is **required**) |
| `jtk issues update PROJ-123 --assignee VALUE` | Assign issue. Resolver accepts `me`, email, display name, or raw account ID |
| `jtk issues update PROJ-123 --assignee none` | Unassign via update |
| `jtk issues assign PROJ-123 VALUE` | Dedicated assign command; same resolver inputs as `--assignee` (user is a positional arg, not a flag) |
| `jtk issues assign PROJ-123 --unassign` | Unassign via dedicated command (equivalent to `--assignee none` on update) |
| `jtk issues move PROJ-1 [PROJ-2 ...] --to-project NEWPROJ` | Move one or more issues to another project (Jira Cloud only). By default synchronous — waits for completion. Max 1000 issues per request. See move flags below |
| `jtk issues move-status TASK_ID` | Check status of an async move operation (used with `--no-wait`) |
| `jtk issues delete PROJ-123` | Permanently delete an issue. Interactive `y/N` prompt by default (prompt goes to stderr, reads from stdin); pass `--force` to skip. Destructive and irreversible |
| `jtk issues types --project KEY` | List valid issue types for a project (output includes the `SUBTASK` column; use values from this list as `--type` on create) |
| `jtk issues fields [PROJ-123]` | List available fields (all fields, or editable fields for a specific issue) |
| `jtk issues field-options FIELD_NAME_OR_ID [--issue PROJ-123]` | List allowed values for a field (e.g. priority, custom selects) |

### Create Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--project KEY` / `-p` | Yes | Project key |
| `--type TYPE` / `-t` | No | Issue type (default Task). Task, Bug, Story, Epic, Sub-task |
| `--summary "TEXT"` / `-s` | Yes | Issue title |
| `--description "TEXT"` / `-d` | No | Issue description |
| `--assignee VALUE` / `-a` | No | Assignee (resolver accepts `me`, email, display name, or raw account ID) |
| `--parent KEY` | No | Parent issue key (epic or parent) |
| `--field NAME=VALUE` / `-f` | No | Set custom field (repeatable) |

### Update Flags

| Flag | Description |
|------|-------------|
| `--summary "TEXT"` / `-s` | New summary |
| `--description "TEXT"` / `-d` | New description |
| `--assignee VALUE` / `-a` | Reassign (resolver accepts `me`, email, display name, or raw account ID; use `none` to unassign). Note: `jtk issues update --help` flag text underclaims the resolver (omits display name) — the resolver implementation accepts all four, same as `issues create` and `issues assign` |
| `--type TYPE` / `-t` | Change issue type (uses bulk move API) |
| `--parent KEY` | Change parent/epic |
| `--field NAME=VALUE` / `-f` | Update custom field (repeatable; repeating the same key accumulates values for multi-select fields) |

**Notes:**
- `jtk issues update` does **not** change workflow status. Use `jtk transitions do` for status changes (see Transitions below).
- `--type` on update uses the Jira Cloud bulk-move API (different path than a plain edit) — safe, but behaves asynchronously.
- `--description` and other text flags support `\n`, `\t`, `\\` escape sequences.

### Move Flags (`jtk issues move`)

| Flag | Required | Description |
|------|----------|-------------|
| `--to-project KEY_OR_NAME` | Yes | Target project (accepts key or name) |
| `--to-type TYPE` | No | Target issue type name. If omitted, uses the same type as the source issue (resolved via cache; may need `jtk refresh issuetypes` on a cold cache) |
| `--wait` / `--no-wait` | No | `--wait` (default) polls the move task to completion; `--no-wait` returns the task ID immediately. Use `jtk issues move-status TASK_ID` to check an async move later |
| `--notify` / `--no-notify` | No | `--notify` (default) sends Jira notifications for the move; `--no-notify` suppresses them |

Positional: one or more `<issue-key>` (up to 1000 per request). Jira Cloud only — not available on Server or Data Center.

### Search & List Flags (shared by `jtk issues search` and `jtk issues list`)

| Flag | Description |
|------|-------------|
| `--jql "QUERY"` | JQL query string (required for `search`) |
| `--project KEY` / `-p` | Filter by project key (for `list`) |
| `--sprint current` / `-s current` | Filter by current sprint (for `list`) |
| `--max N` / `-m N` | Maximum results (default 25; auto-paginates) |
| `--next-page-token TOKEN` | Resume from previous page token |
| `--all-fields` | Include all fields (e.g. description) |
| `--fields summary,status,customfield_10005` | Comma-separated list of specific fields |

### Common Issue Types

Task, Bug, Story, Epic, Sub-task (instance-dependent)

## Transitions (Workflow Status Changes)

Status changes happen via `jtk transitions do`, **not** `jtk issues update`.

| Command | Description |
|---------|-------------|
| `jtk transitions list PROJ-123` | List available transitions for issue |
| `jtk transitions list PROJ-123 --fields` | Show required fields for each transition |
| `jtk transitions do PROJ-123 "Transition Name"` | Apply transition by name |
| `jtk transitions do PROJ-123 21` | Apply transition by numeric ID |
| `jtk transitions do PROJ-123 "Done" --field NAME=VALUE` | Apply with required fields (only when `transitions list --fields` shows a required field) |

Common transition names: "To Do", "In Progress", "In Review", "Done" (instance-dependent — always run `transitions list` first).

> **Do not speculatively pass `--field resolution=Done` (or any other field) unless `jtk transitions list --fields PROJ-123` explicitly shows it is required for the transition you're applying.** Many Jira workflows set resolution via post-function or hide it from the transition screen — speculatively providing `--field resolution=Done` will fail with "Field 'resolution' cannot be set. It is not on the appropriate screen, or unknown." In that case, re-run the transition without the `--field` flag.

## Projects

| Command | Description |
|---------|-------------|
| `jtk projects list` | List all projects |
| `jtk projects get KEY` | Get project details |

## Sprints

| Command | Description |
|---------|-------------|
| `jtk sprints list --board ID` | List sprints for board |
| `jtk sprints current --board ID` | Get active sprint |
| `jtk sprints issues SPRINT_ID` | List issues in sprint |
| `jtk sprints add SPRINT_ID PROJ-1 PROJ-2 ...` | Add issues to sprint (issues are positional) |

## Boards

| Command | Description |
|---------|-------------|
| `jtk boards list` | List all boards |
| `jtk boards get ID` | Get board details |

## Comments

| Command | Description |
|---------|-------------|
| `jtk comments list PROJ-123` | List comments on issue |
| `jtk comments add PROJ-123 --body "TEXT"` | Add comment (`--body` / `-b` is **required**; supports `\n`, `\t`, `\\` escapes) |
| `jtk comments delete PROJ-123 COMMENT_ID` | Delete a comment |

## Attachments

| Command | Description |
|---------|-------------|
| `jtk attachments list PROJ-123` | List attachments on issue |
| `jtk attachments add PROJ-123 --file PATH` | Upload attachment (`--file` / `-f` repeatable for multiple) |
| `jtk attachments get ATTACHMENT_ID` | Download attachment (alias: `download`) |
| `jtk attachments get ATTACHMENT_ID --output ./dir/` | Download to specific directory |
| `jtk attachments get ATTACHMENT_ID --output ./renamed.pdf` | Download with custom filename |
| `jtk attachments delete ATTACHMENT_ID` | Delete attachment |

## Users

| Command | Description |
|---------|-------------|
| `jtk users search "QUERY"` | Search for users (matches display name, email, etc.) |
| `jtk users search "QUERY" --max 1 --id` | Resolve a query to a single account ID (script-friendly) |
| `jtk users get ACCOUNT_ID` | Get user details by account ID |
| `jtk users get ACCOUNT_ID --id` | Echo just the account ID (useful in pipelines) |
| `jtk me` | Show current authenticated user (see Current User section above) |

## Common JQL Patterns

| Intent | JQL |
|--------|-----|
| My open issues | `assignee = currentUser() AND status != Done` |
| My in-progress | `assignee = currentUser() AND status = "In Progress"` |
| Project bugs | `project = KEY AND type = Bug` |
| High priority | `project = KEY AND priority = High` |
| Updated this week | `project = KEY AND updated >= -7d` |
| Created today | `project = KEY AND created >= startOfDay()` |
| Unassigned | `project = KEY AND assignee is EMPTY` |
| Sprint issues | `sprint in openSprints() AND project = KEY` |
| Overdue | `project = KEY AND duedate < now() AND status != Done` |

## Output

- Data goes to stdout (pipeable)
- Diagnostics/logs go to stderr
- **Pagination continuation notices (`More results available ...`) go to STDOUT, not stderr** — this is intentional per `jtk`'s output contract, and applies even with `--id`. When using `--id` in command substitution or piping to a tool that reads line-by-line, size `--max` to match your expectation, or post-filter with `grep -oE '[A-Z]+-[0-9]+' | head -1` (or equivalent) to isolate just the identifier from any trailing notice.
- Use `--id` global flag for just the primary identifier (useful when piping to another command; note the pagination caveat above)
- Use `--fulltext` global flag to disable truncation of descriptions/comments
- Use standard shell tools for filtering: `jtk issues list --project KEY | grep "Bug"`

## Scope of This Reference

This reference covers `jtk`'s daily-use operator surface — issues, transitions, sprints, boards, comments, attachments, projects, users. It intentionally does **not** cover administrative surfaces, which are out of scope for the workflows in this skill:

- `jtk fields` — custom field management (create, delete, restore, contexts, options)
- `jtk dashboards` — dashboard and gadget management
- `jtk automation` — automation rule management (list, export, create, update, enable/disable)

These subcommands exist and work — run `jtk <subcommand> --help` for discovery, or see the [upstream README](https://github.com/open-cli-collective/atlassian-cli/blob/main/tools/jtk/README.md) for the full command reference.
