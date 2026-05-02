# Integration Test Bugs v3

Full pass through `integration-tests.md` against a live Jira instance (Basic Auth, `monitproduct.atlassian.net`).
Run date: 2026-04-29. Built from `main` after merging PR #315.

Test variables: `PROJECT=MON`, `EXISTING_ISSUE=MON-5094`, `ACCOUNT_ID=60e09bae7fcd820073089249`,
`BOARD_ID=23`, `SPRINT_ID=125`, `AUTO_UUID=018c2840-57c1-7869-9393-11205cc87ce4`, `DASHBOARD_ID=10001`.

Each entry has a minimal repro and delta between expected and actual.

---

## Section 1 — Config & Init

**PASS.** `config show`, `config test`, `me`, `me --id`, `me --extended` all work correctly.

### `me` default output missing `Active` field

**Repro:**
```bash
jtk me
# Actual:   60e09bae7fcd820073089249 | Rian Stockbower | rian@monitapp.io
# Expected: Detail block: Account ID, Display Name, Email, Active
```
`Active` is present in `--extended` output but absent from the default view. Same issue in `users get`.

---

## Section 2 — Issues (Read-Only)

### `-o plain` still renders table format, not tab-separated

`--output plain` is supposed to produce tab-separated values for machine consumption.

**Repro:**
```bash
jtk issues list -p MON --max 2 -o plain
# Actual:   KEY | STATUS | TYPE | PTS | ASSIGNEE | SUMMARY
#           MON-5094 | Ready for Deployment | ...
# Expected: tab-separated rows (no | separators)
```
Same behavior on `issues search -o plain`. This matches the known v2 bug and is not yet fixed.

### `--id` output includes pagination token line

When using `--id` on list/search commands, the pagination hint leaks into the output.

**Repro:**
```bash
jtk issues list -p MON --max 3 --id
# Actual:
#   MON-5094
#   MON-5088
#   MON-5092
#   More results available (next: Ck11cGRh...)
# Expected: Issue keys only, one per line — no pagination line
```
This breaks piping: `jtk issues list -p MON --id | wc -l` counts the pagination line as a result.
Same behavior for `issues search --id`, `projects list --id`, and `sprints list active --id`.

### `issues get` default output missing URL

**Repro:**
```bash
jtk issues get MON-5094
# Actual:   MON-5094  Centrix provider: ...
#           Status: Ready for Deployment   Type: SDLC   Priority: Medium   Points: -
#           Assignee: Caleb Piekstra   Updated: 2026-04-29
#           Description: ...
# Expected: same fields plus URL (e.g., https://monitproduct.atlassian.net/browse/MON-5094)
```

### `issues get --extended` missing Sprint, Transitions, and Watchers sections

The spec says `--extended` should add "Sprint, Transitions list, raw custom fields block".

**Repro:**
```bash
jtk issues get MON-5094 --extended
# Actual adds: Reporter, Created date
# Missing: Sprint assignment, Transitions section, Watchers, Resolution, Fix Versions
```

### Error messages have operation-prefix wrapper

`get`, `search`, `update`, and similar commands wrap the API error with an operation prefix that the spec does not include.

**Repro:**
```bash
jtk issues get MON-99999
# Actual:   fetching issue: resource not found: Issue does not exist or you do not have permission to see it.
# Expected: resource not found: Issue does not exist or you do not have permission to see it.

jtk issues search --jql "invalid jql ((("
# Actual:   searching issues: bad request: Error in the JQL Query: ...
# Expected: bad request: Error in the JQL Query: ...

jtk issues update MON-99999 -s "Nope"
# Actual:   updating issue MON-99999: resource not found: ...
# Expected: resource not found: ...

jtk issues delete MON-99999 --force
# Actual:   Failed to delete MON-99999: deleting issue MON-99999: resource not found: ...
# Expected: resource not found: ... (double-wrapped for delete)
```
The double-wrap on `delete` is especially noisy.

### Required-flag errors missing `Error:` prefix

**Repro:**
```bash
jtk issues create -p MON
# Actual:   required flag(s) "summary" not set
# Expected: Error: required flag(s) "summary" not set

jtk projects create --key ZTEST
# Actual:   required flag(s) "lead", "name" not set
# Expected: Error: required flag(s) "lead", "name" not set
```
Cobra emits the `Error:` prefix for its own errors but the jtk command swallows it.

### `issues fields` (and `fields list`) missing CUSTOM column

**Repro:**
```bash
jtk issues fields
# Actual columns: ID | TYPE | NAME
# Expected cols:  ID | NAME | TYPE | CUSTOM

jtk fields list
# Actual columns: ID | TYPE | NAME
# Expected cols:  ID | NAME | TYPE | CUSTOM
```
The CUSTOM column that indicates whether a field is a custom field is not rendered. Because the column is absent, `--custom-fields` filtering works but there is no visual indicator in the output.

### `--fields` projection includes KEY even when not requested

**Repro:**
```bash
jtk issues search --jql "project = MON" --max 1 --fields summary,status
# Actual:   KEY | SUMMARY | STATUS
# Expected: SUMMARY | STATUS  (KEY absent per spec)
```

---

## Section 3 — Projects (Read-Only)

**PASS** with minor notes below.

### `projects list` column order differs from spec

**Repro:**
```bash
jtk projects list --max 1
# Actual:   KEY | TYPE | LEAD | NAME
# Expected: KEY | NAME | TYPE | LEAD
```

### `projects get` missing numeric project ID

**Repro:**
```bash
jtk projects get MON
# Actual:   MON  Platform Development
#           Type: software   Lead: Rian Stockbower   Style: classic
#           Issue Types: Epic, Kanban, SDLC
#           Components: 25   Versions: 0
# Expected: same plus internal numeric ID (e.g., ID: 10200)
```

### `projects get` error has operation prefix

```bash
jtk projects get NONEXISTENT
# Actual:   fetching project: resource not found: No project could be found with key 'NONEXISTENT'.
# Expected: resource not found: No project could be found with key 'NONEXISTENT'.
```

---

## Section 4 — Boards & Sprints (Read-Only)

**Mostly PASS.**

### `boards list` column order differs from spec

```bash
jtk boards list --max 1
# Actual:   ID | TYPE | PROJECT | NAME
# Expected: ID | NAME | TYPE | PROJECT
```

### `boards get` error has operation prefix

```bash
jtk boards get 99999
# Actual:   getting board 99999: resource not found: ...
# Expected: Error: 404 (board not found)
```

### `sprints list` column order differs from spec; required-flag error differs

```bash
jtk sprints list -b 23 -s active
# Actual:   ID | STATE | START | END | NAME
# Expected: ID | NAME | STATE | START | END

jtk sprints list
# Actual:   --board is required
# Expected: Error: required flag(s) "board" not set
```

---

## Section 5 — Links (Read-Only)

**PASS** with one note:

### `links types --extended` adds no extra columns

```bash
jtk links types --extended
# Actual:   ID | NAME | INWARD | OUTWARD  (same as default)
# Expected: Extended table with additional link type metadata
```

---

## Section 6 — Dashboards (Read-Only)

### `dashboards list` has extra GADGETS column; column order differs from spec

```bash
jtk dashboards list --max 5
# Actual:   ID | GADGETS | OWNER | FAVOURITE | NAME
# Expected: ID | NAME | OWNER | FAVOURITE
```

### `dashboards get` missing Description field

```bash
jtk dashboards get 10001
# Actual:   ID: 10001
#           Name: Epics
#           Owner: Rian Stockbower
#           URL: /jira/dashboards/10001
# Expected: same plus Description field (even if empty)
```

### `dashboards get` inline gadgets show empty MODULE column

```bash
jtk dashboards get 10000
# Actual Gadgets table:
#   ID | TITLE | MODULE
#   10001 | Spaces |
#   10002 | Assigned to Me |
#   10054 | Filter Results |
# Expected: MODULE column contains the raw gadget URI
```
The MODULE values are blank for all gadgets. `dashboards gadgets list` correctly resolves the TYPE (short key), but `dashboards get` inline gadgets show no URI.

### `dashboards list --search` does not filter results

```bash
jtk dashboards list --search "[Test] Integration"
# Actual:   Returns all 3 dashboards (Default dashboard, Epics, [Test] Integration Dashboard)
# Expected: Only dashboards matching the search term
```
The search parameter is sent to the API, but the results include unrelated dashboards. Either the API ignores partial-name search or the CLI is not filtering the response.

---

## Section 7 — Users (Read-Only)

**PASS** with the same "Active field in default output" note as Section 1.

### `users get` default output missing Active field

```bash
jtk users get 60e09bae7fcd820073089249
# Actual:   60e09bae7fcd820073089249 | Rian Stockbower | rian@monitapp.io
# Expected: Detail block: Account ID, Display Name, Email, Active
```
Active is available in `--extended` but the spec requires it in the default view.

---

## Section 8 — Automation (Read-Only)

**Mostly PASS.**

### `auto list` default table missing LABELS column

```bash
jtk auto list
# Actual:   ID | STATE | NAME
# Expected: UUID | NAME | STATE | LABELS
```

### `auto get --show-components` renders tree not flat table

The spec says `--show-components` should produce a flat table with `# | COMPONENT | TYPE` columns.

**Repro:**
```bash
jtk auto get 018c2840-57c1-7869-9393-11205cc87ce4 --show-components
# Actual: Indented tree with TRIGGER / CONDITION / ACTION labels:
#   TRIGGER  jira.issue.event.trigger:created
#     CONDITION  jira.jql.condition
#     ACTION  jira.create.variable
#     ...
# Expected:
#   # | COMPONENT | TYPE
#   1 | TRIGGER   | jira.issue.event.trigger:created
#   2 | CONDITION | jira.jql.condition
#   ...
```

---

## Section 9 — Fields (Read-Only)

### `fields list` missing CUSTOM column (same as Section 2)

See Section 2 — `issues fields` entry. Same root cause.

### `fields contexts list --id` returns full table instead of IDs

**Repro:**
```bash
jtk fields contexts list customfield_10035 --id
# Actual:   ID | NAME | GLOBAL | ANY_ISSUE_TYPE
#           10135 | Default Configuration Scheme for Story Points | yes | yes
# Expected: 10135
```
The `--id` flag is ignored for `fields contexts list`.

---

## Section 10 — Issue Mutations

### `issues create` output missing URL (and Reporter)

Same as `issues get` — mutations show post-state but the URL field is absent.

```bash
jtk issues create -p MON -t SDLC -s "[Test] Integration Test Issue"
# Actual:   MON-5103  [Test] Integration Test Issue
#           Status: Backlog   Type: SDLC   Priority: Medium   Points: -
#           Assignee: Unassigned   Updated: 2026-04-29
# Expected: same plus Reporter and URL per spec
```

### Escape sequences in comment body not rendered

`\n` and `\t` literal sequences in `-b` values are passed verbatim to the API rather than being interpreted as newline/tab.

**Repro:**
```bash
jtk comments add MON-5103 -b "Line one\nLine two\n\tIndented line"
# Actual body stored: Line oneLine twoIndented line
# Expected body:
#   Line one
#   Line two
#       Indented line
```
Verified via `jtk comments list --fulltext`: body is `Line oneLine twoIndented line` (no whitespace).

### `comments delete` does not support `--force`

The spec includes `jtk comments delete $ISSUE $COMMENT_ID --force` but the flag is unknown.

**Repro:**
```bash
jtk comments delete MON-5103 21843 --force
# Actual:   unknown flag: --force
# Expected: deletes without confirmation (or accepts --force to skip prompt)
```
Without `--force`, the command deletes immediately with no confirmation at all — so there is no safety mechanism and no way to explicitly bypass one.

---

## Section 11 — Link Mutations

### `links list` and `links create` use column header `LINK_ID` instead of `ID`

**Repro:**
```bash
jtk links list MON-5094
# Actual columns: LINK_ID | TYPE | DIRECTION | ISSUE | SUMMARY
# Expected:       ID | TYPE | DIRECTION | ISSUE | SUMMARY

jtk links create MON-5107 MON-5108 --type Blocks
# Actual columns: LINK_ID | TYPE | DIRECTION | ISSUE | SUMMARY
# Expected:       ID | TYPE | DIRECTION | ISSUE | SUMMARY
```

### `links create` invalid type error message format differs from spec

**Repro:**
```bash
jtk links create MON-5094 MON-5094 --type "NonexistentType"
# Actual:   Unknown link type "NonexistentType" — not found in cache. Try `jtk refresh linktypes` if this link type was recently added.
# Expected: link type "NonexistentType" not found (available: ...)
```
The spec expects a listing of available types in the error; the actual suggests a cache refresh.

---

## Section 12 — Project Mutations

**PASS** with notes on output format.

### `projects delete` message format differs from spec

**Repro:**
```bash
jtk projects delete ZT2 --force
# Actual:   Deleted project ZT2 (moved to trash — recoverable for 60 days via projects restore)
# Expected: ✓ Deleted project ZT2 (moved to trash)
```
The actual message is more verbose and lacks the `✓` prefix.

---

## Section 13 — Dashboard Mutations

### `dashboards create` shows table row instead of confirmation message

**Repro:**
```bash
jtk dashboards create --name "[Test] Integration Dashboard"
# Actual:
#   ID | GADGETS | OWNER | FAVOURITE | NAME
#   10142 | 0 | Rian Stockbower | yes | [Test] Integration Dashboard
# Expected: Created dashboard [Test] Integration Dashboard (10142)
```

### `dashboards gadgets add` fails with well-known Jira module type

The test gadget type from the spec cannot be added via the API.

**Repro:**
```bash
jtk dashboards gadgets add 10142 --type com.atlassian.jira.gadgets:filter-results-gadget
# Actual:   bad request: The module key of this gadget is not present in the directory
# Expected: Single table row with gadget ID, title, module, position
```
This makes the gadget add/list populated/remove sequence untestable. The exact module key that Jira Cloud requires for this gadget is not documented.

---

## Section 14 — Automation Mutations

### `auto create` and `auto update` use labeled-field format instead of standard detail block

`auto get` renders: `UUID  Name\nState: ...\n...`
`auto create` and `auto update` render: `Name: ...\nUUID: ...\nState: ...\n...`

**Repro:**
```bash
jtk auto create --file /tmp/auto-clean.json
# Actual:
#   Name: [Test] Auto Integration Copy
#   UUID: 019ddb45-2ec0-7500-be36-25c3ef66c762
#   State: ENABLED
#   ...
# Expected (same format as auto get):
#   019ddb45-2ec0-7500-be36-25c3ef66c762  [Test] Auto Integration Copy
#   State: ENABLED
#   ...
```
`auto update` has the same inconsistency.

### `auto delete` has interactive confirmation with no `--force` bypass

**Repro:**
```bash
jtk auto delete 019ddb45-2ec0-7500-be36-25c3ef66c762
# Actual:
#   This will permanently delete rule "[Test] Auto Integration Copy" (...). This action cannot be undone.
#   Are you sure? [y/N]:
# Expected (per spec): Rule deleted (auto-disables if ENABLED) — no mention of prompt
```
`echo "y" | jtk auto delete UUID` works as a workaround but there is no `--force` flag to skip the prompt programmatically.

---

## Section 15 — Sprint Mutations

**PASS.** `sprints add` shows post-state table. `--id` returns issue key only.

---

## Section 16 — Field Mutations

### `fields options add` shows table instead of confirmation message

**Repro:**
```bash
jtk fields options add customfield_10295 --value "Option A"
# Actual:
#   ID | VALUE | DISABLED
#   10183 | Option A | no
# Expected: ✓ Added option 10183 (Option A)
```
Same for `fields options update`:
```bash
jtk fields options update customfield_10295 --option 10183 --value "Option A (updated)"
# Actual:
#   ID | VALUE | DISABLED
#   10183 | Option A (updated) | no
# Expected: ✓ Updated option 10183
```

### `fields options delete` message says "from context" not "from field"

**Repro:**
```bash
jtk fields options delete customfield_10295 --option 10183 --force
# Actual:   Deleted option 10183 from context 10515
# Expected: ✓ Deleted option 10183 from field customfield_10295
```

### `fields delete` message format differs from spec (no `✓`, more verbose)

**Repro:**
```bash
jtk fields delete customfield_10295 --force
# Actual:   Deleted field customfield_10295 (moved to trash — use fields restore to recover)
# Expected: ✓ Trashed field customfield_10295
```

### `fields restore` shows "post-state unavailable" notice instead of full detail block

**Repro:**
```bash
jtk fields restore customfield_10295
# Actual:
#   post-state unavailable; showing confirmation only
#   Restored field customfield_10295
# Expected: Full field detail block
```
The Jira API does not return the restored field in the restore response; the CLI acknowledges this with the notice. May be an accepted API limitation, but it differs from the spec.

### `fields contexts create` fails when global context already exists

**Repro:**
```bash
jtk fields contexts create customfield_10295 --name "[Test] Context"
# Actual:   creating field context: bad request: Only one global context is allowed per field.
# Expected: ✓ Created context XXXXX ([Test] Context)
```
A newly created field already has a default global context. Creating a second global context is rejected. The test would need to scope the new context to specific projects (`--project` flag) to succeed, but the spec doesn't show this.

---

## Section 17 — Global Flags & Aliases

### `-o plain` still produces table format (same as Section 2)

See Section 2. Both `--output plain` and `-o plain` render tables.

### `jtk f list --max 1` and `jtk field list --max 1` fail — no `--max` on `fields list`

**Repro:**
```bash
jtk f list --max 1
# Actual:   unknown flag: --max
# Expected: One row (alias verified to work without --max)
```
`fields list` does not support `--max`. The spec uses `--max 1` in the alias verification table but the flag doesn't exist.

All other aliases (`i`, `p`, `proj`, `b`, `sp`, `u`, `auto`, `tr`, `c`, `att`, `l`, `link`, `dash`, `dashboard`) work correctly.

---

## Section 18 — Error Cases

**All error cases exit with non-zero codes.** Error message format issues are the same patterns already documented:

| Command | Actual | Issue |
|---------|--------|-------|
| `issues get MON-99999` | `fetching issue: resource not found: ...` | operation prefix |
| `issues search --jql "invalid jql ((("` | `searching issues: bad request: ...` | operation prefix |
| `issues create -p MON` | `required flag(s) "summary" not set` | missing `Error:` prefix |
| `projects get NONEXISTENT` | `fetching project: resource not found: ...` | operation prefix |
| `boards get 99999` | `getting board 99999: resource not found: ...` | operation prefix |
| `sprints list` | `--board is required` | different message than cobra default |
| `links list MON-99999` | `resource not found: ...` | **PASS** (no prefix) |
| `dashboards get 99999` | `resource not found: The dashboard with id '99999' does not exist.` | **PASS** |

---

## Summary

| Severity | Count | Notes |
|----------|-------|-------|
| Output contract violations | 10 | `-o plain`, URL missing, CUSTOM column, `--id` pagination leak, `links LINK_ID`, `auto create/update format`, `dashboards create/options add format` |
| Error message format | 6 | Operation prefix wrap, missing `Error:` prefix on required flags, `sprints list` message |
| Missing fields in output | 4 | Active in `me`/`users get`, URL in `issues get`/`issues create`, Sprint/Transitions in `--extended`, MODULE in dashboard gadgets |
| API/behavior bugs | 3 | `\n\t` not interpreted in comment body, `dashboards list --search` not filtering, `fields contexts list --id` broken |
| UX / spec mismatches | 5 | `auto delete` no `--force`, `comments delete` no `--force`, `fields contexts create` global limit, `fields restore` post-state, `fields list` no `--max` |
