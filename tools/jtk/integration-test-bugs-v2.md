# Integration Test Bugs v2

Full pass through `integration-tests.md` against a live Jira instance (Basic Auth).
Each entry has a minimal repro and delta between expected and actual.

---

## 1. Config & Me

### `me` default output missing Active field and email in JSON

Default table output omits the Active field. JSON output omits `emailAddress` and `active`.

**Repro:**
```bash
jtk me
# Actual:   60e09bae... | Rian Stockbower | rian@monitapp.io
# Expected: Account ID | Display Name | Email | Active

jtk me -o json
# Actual:   {"accountId":"...","displayName":"..."}
# Expected: includes "emailAddress" and "active" keys
```

---

## 2. Issues (Read-Only)

### `issues list -o plain` outputs table format, not tab-separated

`-o plain` has no effect on `issues list` — output is identical to the default table format.

**Repro:**
```bash
jtk issues list -p MON --max 1 -o plain
# Actual:   KEY | STATUS | TYPE | PTS | ASSIGNEE | SUMMARY  (pipe-delimited table)
# Expected: tab-separated values, no header row
```

### `issues list -o json` returns `results` key, not top-level array

Test doc uses `jq '.issues | length'` but the actual key is `results`.

**Repro:**
```bash
jtk issues list -p MON --max 2 -o json | jq '.issues | length'
# Actual:   null (key is "results", not "issues")
# Expected: 2

# Correct path:
jtk issues list -p MON --max 2 -o json | jq '.results | length'
# Output: 2
```

### `issues get -o json` uses flat structure, not `fields.*` nested structure

Test doc checks `.fields.summary` and `.fields.status.name` but the JSON output is flat.

**Repro:**
```bash
jtk issues get MON-5094 -o json | jq '.fields.summary'
# Actual:   null  (no nested "fields" key)
# Expected: "Centrix provider: ..."

# Correct path:
jtk issues get MON-5094 -o json | jq '.summary'
# Output: "Centrix provider: ..."
```

### `issues get` default output missing URL field

The test doc expects the URL to be shown in the default view, but it is absent.

**Repro:**
```bash
jtk issues get MON-5094
# Actual output has no URL line
# Expected: URL: https://monitproduct.atlassian.net/browse/MON-5094
```

### `issues search` JSON uses `results` key and `_meta` instead of `issues` / `pagination`

Test doc pagination checks use `.issues | length` and `.pagination` but actual keys are `results` and `_meta`.

**Repro:**
```bash
jtk issues search --jql "project = MON" --max 200 -o json | jq '.issues | length'
# Actual:   null
# Expected: ≥ 101

jtk issues search --jql "project = MON" --max 200 -o json | jq '.pagination'
# Actual:   null  (key is "_meta")
```

### `--fields` flag blocked when combined with `-o json`

The test suite has several cases that combine `--fields` with `-o json`, but the command rejects this combination.

**Repro:**
```bash
jtk issues search --jql "project = MON" --max 1 --fields summary,status -o json
# Actual:   exit 1: --fields is not supported with --output json
# Expected: JSON array with only "summary" and "status" in .fields keys
```

### `issues fields` / `fields list` missing CUSTOM column; wrong column order

Both commands emit `ID | TYPE | NAME` but the test doc expects `ID | NAME | TYPE | CUSTOM`.

**Repro:**
```bash
jtk issues fields
# Actual:   ID | TYPE | NAME
# Expected: ID | NAME | TYPE | CUSTOM

jtk fields list
# Same issue
```

### `issues field-options` uses deprecated `--issue` flag in test doc

The test doc calls `jtk issues field-options priority --issue $EXISTING_ISSUE` but `--issue` is deprecated. The new interface is positional: `jtk issues field-options <issue-key> <field>`.

**Repro:**
```bash
jtk issues field-options priority --issue MON-5094
# Prints: Flag --issue has been deprecated, use positional arg: jtk issues field-options <issue-key> <field>
# Doc should use: jtk issues field-options MON-5094 priority
```

### `issues create` / `issues update` / `issues assign` / `transitions do` show post-state view instead of confirmation

All mutation commands show a full issue detail block (like `issues get`) instead of the expected one-line confirmation.

**Repro:**
```bash
jtk issues create -p MON -t SDLC -s "[Test] Example"
# Actual:   MON-XXXX  [Test] Example
#           Status: Backlog   Type: SDLC   Priority: Medium   Points: -
#           Assignee: Unassigned   Updated: 2026-04-29
# Expected: ✓ Created issue MON-XXXX
#           URL: https://...

jtk issues update MON-XXXX -d "desc"
# Actual:   full issue detail block
# Expected: ✓ Updated issue MON-XXXX

jtk issues assign MON-XXXX $ACCOUNT_ID
# Actual:   full issue detail block
# Expected: ✓ Assigned issue MON-XXXX to Rian Stockbower

jtk transitions do MON-XXXX "Ready"
# Actual:   full issue detail block
# Expected: ✓ Transitioned MON-XXXX
```

### `issues assign --unassign` / `issues update --assignee none` show post-state view

Same pattern as above — both unassign variants show the full issue detail instead of a confirmation.

**Repro:**
```bash
jtk issues assign MON-XXXX --unassign
# Actual:   full issue detail block
# Expected: ✓ Unassigned issue MON-XXXX

jtk issues update MON-XXXX --assignee none
# Actual:   full issue detail block
# Expected: ✓ Updated issue MON-XXXX
```

### `issues get -o json` unassignment verification path differs from test doc

Test doc verifies unassignment with `jq '.fields.assignee'` but the JSON has no `fields` wrapper.

**Repro:**
```bash
jtk issues get MON-XXXX -o json | jq '.fields.assignee'
# Actual:   null  (because there is no .fields key)
# Expected: null  (but for the right reason — unassigned)

# Correct path:
jtk issues get MON-XXXX -o json | jq '.assignee'
# Output: null
```

### `issues delete` output missing ✓ prefix and "issue" word

**Repro:**
```bash
jtk issues delete MON-XXXX --force
# Actual:   Deleted MON-XXXX
# Expected: ✓ Deleted issue MON-XXXX
```

### `issues create` / `issues create -s` error message missing "Error: " prefix

**Repro:**
```bash
jtk issues create -p MON
# Actual:   required flag(s) "summary" not set
# Expected: Error: required flag(s) "summary" not set
```

---

## 3. Projects (Read-Only)

### `projects list` wrong column order — NAME is last instead of second

**Repro:**
```bash
jtk projects list --max 3
# Actual:   KEY | TYPE | LEAD | NAME
# Expected: KEY | NAME | TYPE | LEAD
```

### `projects get` missing ID field; extra fields not in test doc

**Repro:**
```bash
jtk projects get MON
# Actual:   MON  Platform Development
#           Type: software   Lead: Rian Stockbower   Style: classic
#           Issue Types: Epic, Kanban, SDLC
#           Components: 25   Versions: 0
# Expected: shows Key, Name, ID, Type, Lead, Issue Types (ID is absent, extra Style/Components/Versions present)
```

### `projects types` column name "NAME" instead of "FORMATTED"; no combined format

Test doc expects `KEY | FORMATTED` with values like `software/Software`, but actual is `KEY | NAME` with the values separate.

**Repro:**
```bash
jtk projects types
# Actual:   KEY | NAME  (e.g. "software | Software")
# Expected: KEY | FORMATTED  (e.g. "software/Software")
```

### `projects create` / `projects update` / `projects restore` show post-state view instead of confirmation

**Repro:**
```bash
jtk projects create --key ZTEST --name "..." --type software --lead $ACCOUNT_ID
# Actual:   full project detail block
# Expected: ✓ Created project ZTEST (Integration Test Project)

jtk projects update ZTEST --name "Updated"
# Actual:   full project detail block
# Expected: ✓ Updated project ZTEST

jtk projects restore ZTEST
# Actual:   full project detail block
# Expected: ✓ Restored project ZTEST (Updated Test Project)
```

### `projects delete` output has extra verbiage vs doc expectation (no ✓ prefix)

**Repro:**
```bash
jtk projects delete ZTEST --force
# Actual:   Deleted project ZTEST (moved to trash — recoverable for 60 days via projects restore)
# Expected: ✓ Deleted project ZTEST (moved to trash)
```

---

## 4. Boards & Sprints (Read-Only)

### `boards list` wrong column order — NAME is last instead of second

**Repro:**
```bash
jtk boards list --max 5
# Actual:   ID | TYPE | PROJECT | NAME
# Expected: ID | NAME | TYPE | PROJECT
```

### `sprints list` wrong column order — NAME is last instead of second

**Repro:**
```bash
jtk sprints list -b 23 -s active
# Actual:   ID | STATE | START | END | NAME
# Expected: ID | NAME | STATE | START | END
```

### `sprints list` without `--board` wrong error message format

**Repro:**
```bash
jtk sprints list
# Actual:   --board is required
# Expected: Error: required flag(s) "board" not set
```

### `sprints add` shows table view instead of confirmation

**Repro:**
```bash
jtk sprints add 125 MON-XXXX
# Actual:   KEY | STATUS | TYPE | PTS | ASSIGNEE | SUMMARY  (table row for the added issue)
# Expected: ✓ Moved 1 issue(s) to sprint 125
```

---

## 5. Links (Read-Only)

### `links list` column named `LINK_ID` instead of `ID`

**Repro:**
```bash
jtk links list MON-XXXX
# Actual:   LINK_ID | TYPE | DIRECTION | ISSUE | SUMMARY
# Expected: ID | TYPE | DIRECTION | ISSUE | SUMMARY
```

### `links create` shows table view instead of confirmation

**Repro:**
```bash
jtk links create MON-XXXX MON-YYYY --type Blocker
# Actual:   LINK_ID | TYPE | DIRECTION | ISSUE | SUMMARY  (table row)
# Expected: Created Blocker link: MON-XXXX → MON-YYYY
```

### `links create` invalid type error message uses cache language

**Repro:**
```bash
jtk links create MON-XXXX MON-XXXX --type "NonexistentType"
# Actual:   Unknown link type "NonexistentType" — not found in cache. Try `jtk refresh linktypes` if this link type was recently added.
# Expected: link type "NonexistentType" not found (available: ...)
```

---

## 6. Dashboards (Read-Only)

### `dashboards list --search` "no results" message omits search term

**Repro:**
```bash
jtk dashboards list --search "xyznonexistent999"
# Actual:   No dashboards found
# Expected: No dashboards found matching "xyznonexistent999"
```

### `dashboards gadgets list` columns differ from test doc; MODULE column empty

Test doc expects `ID | TITLE | MODULE | POSITION` but actual `dashboards gadgets list` shows `ID | POSITION | TITLE | TYPE` with TYPE values populated. The inline gadget table inside `dashboards get` shows `ID | TITLE | MODULE` with MODULE empty.

**Repro:**
```bash
jtk dashboards gadgets list 10000
# Actual:   ID | POSITION | TITLE | TYPE
# Expected: ID | TITLE | MODULE | POSITION

jtk dashboards get 10000  # inline gadgets table
# Actual:   ID | TITLE | MODULE  (MODULE always empty)
# Expected: ID | TITLE | MODULE | POSITION
```

### `dashboards get` missing Description field

**Repro:**
```bash
jtk dashboards get 10001
# Actual:   ID, Name, Owner, URL  (no Description)
# Expected: ID, Name, Description, Owner, URL
```

### `dashboards create` outputs table row, not confirmation message

**Repro:**
```bash
jtk dashboards create --name "[Test] Integration Dashboard"
# Actual:   ID | GADGETS | OWNER | FAVOURITE | NAME  (table row)
# Expected: Created dashboard [Test] Integration Dashboard (10141)
```

### `dashboards gadgets add --type com.atlassian.jira.gadgets:filter-results-gadget` fails

The module key used in the test doc is not present in this Jira instance's gadget directory. A valid module key must be discovered from `jtk dashboards gadgets list <id>`.

**Repro:**
```bash
jtk dashboards gadgets add $DASHBOARD_ID --type com.atlassian.jira.gadgets:filter-results-gadget
# Actual:   bad request: The module key of this gadget is not present in the directory
```

---

## 7. Users (Read-Only)

### `users get` default output missing Active field

`users search` includes an ACTIVE column but `users get` omits it.

**Repro:**
```bash
jtk users get 60e09bae7fcd820073089249
# Actual:   60e09bae... | Rian Stockbower | rian@monitapp.io
# Expected: includes Active column/field

jtk users search "Rian"
# Actual (correct): ACCOUNT_ID | NAME | EMAIL | ACTIVE  ← has Active
```

---

## 8. Automation (Read-Only)

### `auto list` missing LABELS column; wrong column order

**Repro:**
```bash
jtk auto list
# Actual:   ID | STATE | NAME
# Expected: UUID | NAME | STATE | LABELS
```

### `auto get --show-components` outputs indented tree, not flat table

**Repro:**
```bash
jtk auto get $AUTO_UUID --show-components
# Actual:   indented tree (TRIGGER / CONDITION / ACTION with nesting)
# Expected: # | COMPONENT | TYPE  (flat table)
```

### `auto create` / `auto disable` / `auto enable` / `auto update` show post-state view

All automation mutation commands show full rule detail instead of a confirmation line.

**Repro:**
```bash
jtk auto create --file /tmp/auto-clean.json
# Actual:   full rule detail block (Name, UUID, State, Description, Components)
# Expected: ✓ Created automation rule (UUID: ...)

jtk auto disable $UUID
# Actual:   full rule detail block showing State: DISABLED
# Expected: ✓ Rule "...": ENABLED → DISABLED

jtk auto enable $UUID
# Actual:   full rule detail block showing State: ENABLED
# Expected: ✓ Rule "...": DISABLED → ENABLED

jtk auto update $UUID --file /tmp/auto-rt.json
# Actual:   full rule detail block
# Expected: ✓ Updated automation rule $UUID
```

### `auto delete` prompts for confirmation; output differs from doc

Test doc step 11 runs `jtk auto delete $UUID` without `--force` and expects silent deletion.

**Repro:**
```bash
jtk auto delete $UUID
# Actual:   prompts "Are you sure? [y/N]:"
# Doc says: Expected: Rule deleted (auto-disables if ENABLED)
# Fix: add --force to step 11 in integration-tests.md

# Also: after confirmation, the output message is different:
echo "y" | jtk auto delete $UUID
# Actual:   Deleted automation $UUID
# Expected: Rule deleted (auto-disables if ENABLED)
```

---

## 9. Fields (Read-Only)

### `fields list` missing CUSTOM column; wrong column order (same as §2 Issues)

See §2 above — `fields list` and `issues fields` share the same bug.

---

## 10. Fields (Mutations)

### `fields create` outputs table row, not confirmation message

**Repro:**
```bash
jtk fields create --name "[Test] Integration Select" --type com.atlassian.jira.plugin.system.customfieldtypes:select
# Actual:   ID | TYPE | NAME  (table row)
# Expected: ✓ Created field customfield_XXXXX ([Test] Integration Select)
```

### `fields options add` outputs table row, not confirmation message

**Repro:**
```bash
jtk fields options add $TEST_FIELD --value "Option A"
# Actual:   ID | VALUE | DISABLED  (table row)
# Expected: ✓ Added option 10181 (Option A)
```

### `fields options update` outputs table row, not confirmation message

**Repro:**
```bash
jtk fields options update $TEST_FIELD --option $OPT_ID --value "Option A (updated)"
# Actual:   ID | VALUE | DISABLED  (table row)
# Expected: ✓ Updated option $OPT_ID
```

### `fields contexts create` without `--project` fails on global fields

A global context already exists on newly-created fields. The test doc (step 9) omits `--project`, which always fails.

**Repro:**
```bash
jtk fields contexts create $TEST_FIELD --name "[Test] Context"
# Actual:   bad request: Only one global context is allowed per field.
# Fix: add --project <numeric-project-id> to the command in the test doc
#      e.g.: jtk fields contexts create $TEST_FIELD --name "[Test] Context" --project 10022
```

### `fields contexts create` outputs table row, not confirmation message

**Repro:**
```bash
jtk fields contexts create $TEST_FIELD --name "[Test] Context" --project 10022
# Actual:   ID | NAME | GLOBAL | ANY_ISSUE_TYPE  (table row)
# Expected: ✓ Created context XXXXX ([Test] Context)
```

### `fields contexts delete` output missing ✓ prefix

**Repro:**
```bash
jtk fields contexts delete $TEST_FIELD $CTX_ID --force
# Actual:   Deleted context 10513 from field customfield_10294
# Expected: ✓ Deleted context 10513 from field customfield_10294
```

### `fields options delete` says "from context", not "from field"

**Repro:**
```bash
jtk fields options delete $TEST_FIELD --option $OPT_ID --force
# Actual:   Deleted option 10181 from context 10512
# Expected: ✓ Deleted option 10181 from field $TEST_FIELD
```

### `fields delete` says "Deleted field … moved to trash", not "Trashed field"

**Repro:**
```bash
jtk fields delete $TEST_FIELD --force
# Actual:   Deleted field customfield_10294 (moved to trash — use fields restore to recover)
# Expected: ✓ Trashed field customfield_10294
```

### `fields restore` returns "post-state unavailable" fallback

**Repro:**
```bash
jtk fields restore customfield_10294
# Actual:   post-state unavailable; showing confirmation only
#           Restored field customfield_10294
# Expected: ✓ Restored field customfield_10294  (with full field detail)
```

---

## 11. Comments

### `comments add` outputs inline comment view instead of confirmation

**Repro:**
```bash
jtk comments add MON-XXXX -b "Line one\nLine two"
# Actual:   MON-XXXX #21840 — Rian Stockbower, 2026-04-29
#           Line oneLine two
# Expected: ✓ Added comment 21840 to MON-XXXX
```

### Escape sequences in comment body not interpreted

`\n` and `\t` in `-b` values are stored as literal backslash-n / backslash-t.

**Repro:**
```bash
jtk comments add MON-XXXX -b "Line one\nLine two\n\tIndented line"
# Body stored as: Line oneLine twoIndented line  (no actual newlines or tab)
# Expected: three distinct lines with real newline and tab characters
```

### `comments delete --force` flag does not exist

Test doc step 12 calls `jtk comments delete $ISSUE $COMMENT_ID --force`, but `--force` is not supported.

**Repro:**
```bash
jtk comments delete MON-XXXX 21840 --force
# Actual:   Error: unknown flag: --force
# Fix: remove --force from the test doc, or add --force support to the command
```

### `comments delete` output missing ✓ prefix

**Repro:**
```bash
jtk comments delete MON-XXXX 21840
# Actual:   Deleted comment 21840 from MON-XXXX
# Expected: ✓ Deleted comment 21840 from MON-XXXX
```

---

## 17. Global Flags & Aliases

### `jtk f list --max 1` / `jtk field list --max 1` — `fields list` has no `--max` flag

Test doc alias rows 11 and 12 use `--max` with `fields list`, but the flag does not exist.

**Repro:**
```bash
jtk f list --max 1
# Actual:   unknown flag: --max

jtk field list --max 1
# Actual:   unknown flag: --max
```

---

## Integration Test Doc Bugs (not code bugs)

These are errors in `integration-tests.md` that need correction:

| Section | Command in doc | Problem |
|---------|----------------|---------|
| §10 step 12 | `jtk comments delete $ISSUE $COMMENT_ID --force` | `--force` not supported |
| §14 step 11 | `jtk auto delete $TEST_AUTO_UUID` | needs `--force` to skip prompt |
| §16 step 9 | `jtk fields contexts create $FIELD --name "[Test] Context"` | fails on global fields; needs `--project <ID>` |
| §17 alias #11 | `jtk f list --max 1` | `fields list` has no `--max` flag |
| §17 alias #12 | `jtk field list --max 1` | same |
| §6/§13 gadgets add | `--type com.atlassian.jira.gadgets:filter-results-gadget` | module key absent in this Jira instance |
| §2 field-options | `jtk issues field-options priority --issue $EXISTING_ISSUE` | `--issue` is deprecated; positional form is `jtk issues field-options <issue-key> <field>` |
| §10 step 11c | `jq '.fields.assignee'` | JSON is flat; correct path is `.assignee` |
| §2 pagination | `jq '.issues \| length'` / `jq '.pagination'` | keys are `results` and `_meta` |
| §2 `--fields` | `--fields … -o json` | blocked by the CLI; `--fields` is incompatible with `-o json` |
