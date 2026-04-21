# ManageIssue Workflow

Create, update, assign, and transition **a single** Jira issue.

> **For multi-issue operations** — creating more than one issue, parent+sub-task hierarchies, bulk updates across multiple issues, mixing creates with updates, or decomposing a spec into tickets — use `Workflows/ManageIssueSet.md` instead. That workflow handles discovery, plan confirmation, ordering, and partial-failure recovery for coordinated multi-issue work.

## Intent-to-Flag Mapping

### Action Selection

| User Says | Action | Command |
|-----------|--------|---------|
| "create issue", "new ticket", "file a bug" | Create | `jtk issues create` |
| "update issue", "change field", "set priority" | Update | `jtk issues update PROJ-123 --field FIELD=VALUE` |
| "assign to X" (any form: `me`, email, display name, account ID) | Assign | `jtk issues update PROJ-123 --assignee VALUE` |
| "unassign" | Unassign | `jtk issues update PROJ-123 --assignee none` |
| "move to done", "close", "transition", "change status" | Transition | `jtk transitions do PROJ-123 "Target Status"` |

### Choosing an Issue Type (for create)

Issue types are **project-configurable** — each project has its own set, and names vary. A project might have `Support Request` instead of `Bug`, or `Feature` instead of `Story`, or entirely custom types like `Capability`, `Initiative`, `PRD`, `Tech Design`, `UAT`, etc. Some types are *sub-task types* (Jira's `subtask: true` flag), which **require `--parent KEY`** pointing at a non-subtask parent issue.

Common Jira type names you may encounter: `Bug`, `Story`, `Task`, `Epic`, and one or more sub-task types (typically but not always named `Sub-task`). Many projects add custom types.

**Before creating, run `jtk issues types --project KEY`** to list the project's actual types. Output includes a `SUBTASK` column — rows with `SUBTASK: yes` are sub-task types. Match the user's phrasing (e.g., "bug," "user story," "design task") to the discovered list.

If `jtk issues create --type X` fails with an invalid-type error, the project's scheme doesn't include `X`. Re-run `jtk issues types --project KEY` and present the available options to the user.

### Confirming ambiguous or multi-ticket type choices

After discovering the project's types, confirm with the user — don't silently pick — when any of these apply:

- **Multiple types could match the intent.** If the user says "file a design task" and the project has `Tech Design`, `UX Design`, and `Development`, ask which. If the user says "create a release ticket" and the project has `UAT`, `Pilot Release`, and `Full Release`, ask.
- **Nothing matches cleanly.** If no discovered type fits the user's phrasing, list the available types and ask which one they want.
- **Hierarchy / sub-task creation.** When the user wants a parent + sub-task, confirm (a) the parent's type, and (b) the sub-task type specifically — some projects have more than one sub-task type (e.g., `Technical Sub-task`, `Bug Sub-task`). Ask which one.
- **Creating multiple tickets in one request.** If the user asks for more than one issue in a single request, you've likely landed in this workflow by mistake — route to **ManageIssueSet** instead.

Use the user's confirmed choice — don't fall back to guessing a "close enough" type once you've surfaced an ambiguity.

### Choosing a Priority (for create/update)

Priority values are **project-scoped**, not universal. A fresh Jira Cloud project ships with something like `Highest / High / Medium / Low / Lowest`, but many instances replace or extend that with schemes like `Blocker / Critical / Major / Minor / Trivial` or ten-level custom priorities. The skill does not assume any priority value is valid on any project.

**Before setting priority, run `jtk issues field-options priority --issue PROJ-123`** to list the valid values for the issue's project. `--issue PROJ-123` is strongly recommended — priority schemes are usually project-scoped, and without a project-scoped issue reference the CLI falls back to whatever global scheme data is available, which may be incomplete or empty. For a create where the issue doesn't exist yet, run the call against an existing issue in the target project to learn the scheme. If an `--issue`-less call returns empty or missing values, retry with `--issue PROJ-123`.

Match the user's phrasing (e.g., "urgent," "critical," "low priority") to the discovered values. If the user's wording is ambiguous (e.g., "high" in a scheme that has both `High` and `Highest`), ask.

If `jtk issues update --field priority=X` or `jtk issues create ... --field priority=X` fails with a "not found" or "invalid option" error, re-run the `field-options` command and present the available values to the user.

## Execute

### Create Issue

```bash
jtk issues create \
  --project KEY \
  --type TYPE \
  --summary "Summary text" \
  --description "Description text"
```

Optional flags:
```bash
  --assignee me                 # or an email, display name, or raw account ID
  --parent PROJ-100              # attach to epic/parent
  --field priority=High \
  --field labels=label1,label2
```

### Update Issue (fields)

```bash
jtk issues update PROJ-123 \
  --field FIELD=VALUE
```

Common direct flags instead of `--field`: `--summary`, `--description`, `--assignee`, `--type`, `--parent`.

**Multi-value fields** (multi-select customfields, labels, etc.): repeat `--field` with the same key to accumulate values, e.g. `--field customfield_10050=Option1 --field customfield_10050=Option2`.

**Text escapes:** `--summary`, `--description`, and `--body`-style flags support `\n` (newline), `\t` (tab), and `\\` (literal backslash) escape sequences. Use these for multi-line content passed on the command line.

**Discovering editable fields and valid values:** `jtk issues fields PROJ-123` lists the fields you can edit on a specific issue. `jtk issues field-options FIELD --issue PROJ-123` lists the allowed values for enum-type fields (priority, status categories, select customfields). Many fields — including `priority` on most instances — have project-scoped or issue-scoped schemes, so `--issue PROJ-123` is effectively required. You can omit `--issue` for truly global fields, but if the call fails with "Field key '...' is not valid" or similar, add `--issue` and retry.

**Changing issue type (`--type`):** uses the Jira Cloud bulk-move API rather than a direct edit, so the operation is asynchronous and behaves slightly differently from a plain field update. Safe, but expect a brief delay before the change is visible.

### Assign Issue

Use `jtk issues update KEY --assignee VALUE`. The resolver accepts four forms: `me`, email address, display name, or raw account ID. All four flow through the same User resolver and produce the same result — pick whichever the user provided verbatim.

```bash
jtk issues update PROJ-123 --assignee me
jtk issues update PROJ-123 --assignee user@example.com
jtk issues update PROJ-123 --assignee "Alice Cooper"             # display name
jtk issues update PROJ-123 --assignee 5b10ac8d82e05b22cc7d4ef5   # raw accountId
jtk issues update PROJ-123 --assignee none                       # unassign
```

Note: `jtk issues assign KEY VALUE` (positional user arg) and `jtk issues assign KEY --unassign` are equivalent alternatives — same resolver, same accepted forms. This workflow standardizes on `--assignee` for uniformity across the assign/unassign/update surface. `jtk me --id` returns your own account ID if you need it for scripting.

#### Fallback when the resolver fails

If `--assignee VALUE` returns an error ("user not found", "ambiguous match", "multiple users matched", or similar), the resolver couldn't map the input to a single account. Fall back to explicit search:

```bash
jtk users search "QUERY" --max 10
```

Then apply match-count logic — mirror the pattern used in SpecToTickets stage 2 step 4:

| Match count | Action |
|-------------|--------|
| **0** | Surface to the user: "No matches for `QUERY` — did you mean someone else, or should I leave unassigned?" Wait for user direction. Do not silently substitute. |
| **1, with clear fit** (exact display name, exact email, or the single result is the obvious intended user) | Use that account ID — retry with `--assignee <ACCOUNT_ID>`. |
| **1, with possible doubt** (common first name, loose partial match, single result but weak fit) | Surface the single match to the user for confirmation before using. |
| **2–9** | Surface the full candidate list (display name + email for each). Ask the user which account is correct. Do NOT auto-pick. |
| **10 (the cap)** | The intended user may be past the cap. Surface as ambiguous, ask the user to refine the search (fuller name, specific email domain, exact email address). |

Never silently commit to an account ID without evidence of correctness. When any doubt remains, error back to the user for disambiguation rather than guessing.

### Transition Issue (workflow status)

Status changes happen through `jtk transitions do`, **not** through `jtk issues update`.

```bash
# Step 1: List available transitions (names are instance-dependent)
jtk transitions list PROJ-123

# Step 1b (optional but useful): show which transitions require fields
jtk transitions list PROJ-123 --fields

# Step 2: Apply the transition by name
jtk transitions do PROJ-123 "In Progress"

# Or by numeric ID (stable even if names change)
jtk transitions do PROJ-123 21

# If `transitions list --fields` showed a required field, pass it. Only
# pass --field values that the list-fields output explicitly flagged as
# required — many workflows set resolution via post-function or hide it
# from the transition screen, and a speculative --field resolution=Done
# will fail with "Field 'resolution' cannot be set. It is not on the
# appropriate screen, or unknown." If that happens, retry without --field.
jtk transitions do PROJ-123 "Done" --field NAME=VALUE
```

Common transition names (instance-dependent — always `list` first):

| User Says | Typical Target |
|-----------|----------------|
| "start working", "in progress" | `In Progress` |
| "ready for review", "in review" | `In Review` |
| "done", "close", "complete", "resolve" | `Done` |
| "reopen" | `To Do` |
| "block", "blocked" | `Blocked` (if configured) |

## Post-Action

After any action:
1. Confirm success with the issue key
2. For creates: show the new issue key. **Note: the `--id` global flag is currently ignored by `jtk issues create` — the command emits its full `Created issue PROJ-123 (https://...)` message regardless. For scripting, parse the key out of the decorated output (e.g., `grep -oE '[A-Z]+-[0-9]+' | head -1`) rather than relying on `--id`.**
3. For transitions: show old status → new status
4. For assignments: confirm assignee display name
