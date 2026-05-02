# Integration Tests

This document is a concrete, sequential runbook for testing `jtk` against a live Jira instance. Run read-only tests first, then mutations, then cleanup.

If a test reveals a bug, **record the bug and continue testing** rather than stopping to fix it.

## Auth Methods

jtk supports two authentication methods. The full integration test suite should be run with both:

- **Basic Auth** (default): Classic API tokens using `email:token` against the instance URL.
- **Bearer Auth**: Scoped API tokens for service accounts using `Authorization: Bearer <token>` against the `api.atlassian.com` gateway.

> **Scope limitations:** Scoped tokens don't have scopes for Agile (boards/sprints), Automation, or Dashboards. Sections 4 (Boards & Sprints), 6 (Dashboards), 8 (Automation), 13 (Dashboard Mutations), 14 (Automation Mutations), and 15 (Sprint Mutations) must be **skipped** when testing with Bearer Auth. Section 19 (Bearer Auth Guards) should be run **only** with Bearer Auth.

---

## Test Environment Setup

### Prerequisites
- A configured `jtk` instance (`jtk init` completed)
- Access to a project with permission to create, edit, and delete issues
- At least one agile board with an active sprint (Basic Auth only)
- At least one ENABLED and one DISABLED automation rule (Basic Auth only)
- At least one automation rule with multiple components (trigger + conditions + actions) (Basic Auth only)
- At least one dashboard (Basic Auth only)

### Bearer Auth Prerequisites
- An Atlassian service account with a scoped API token
- Your Cloud ID (find at `https://your-site.atlassian.net/_edge/tenant_info`)
- `jtk init --auth-method bearer` completed

### Build

```bash
make build-jtk
```

### Discover Test Values

Run these commands and capture the values. They are referenced as `$VARIABLES` throughout this document.

```bash
# $ACCOUNT_ID — your account ID (used for assignment and project lead)
jtk me --id

# $PROJECT — pick a project you have full access to
jtk projects list --max 10
# Note the KEY column value, e.g., MON

# $ISSUE_TYPES — check available issue types (not all projects have "Task")
jtk issues types -p $PROJECT
# Note a valid type name, e.g., SDLC, Bug, Task

# $EXISTING_ISSUE — pick an existing issue key for read-only tests
jtk issues list -p $PROJECT --max 3 --id
# Note a KEY, e.g., MON-3714

# $BOARD_ID — find a board for your project (Basic Auth only)
jtk boards list -p $PROJECT --id
# Note the ID column, e.g., 23

# $SPRINT_ID — find the active sprint (Basic Auth only)
jtk sprints list -b $BOARD_ID -s active --id
# Note the ID column, e.g., 119

# $AUTO_UUID — pick an enabled automation rule (Basic Auth only)
jtk auto list --state ENABLED --id
# Note a UUID from the output

# $DASHBOARD_ID — pick a dashboard (Basic Auth only)
jtk dashboards list --max 5 --id
# Note an ID, e.g., 10001

# $LINK_TYPE — check available link types
jtk links types
# Note a NAME, e.g., Blocks

# $CUSTOM_FIELD — pick a custom field ID
jtk fields list --custom-fields --id
# Note an ID, e.g., customfield_10001

# $SELECT_FIELD — pick a select/multiselect custom field with options
# (same as $CUSTOM_FIELD if it's a select type)

# $MULTI_FIELD — pick a multi-select or multi-checkbox custom field (optional)
# Used for multi-value --field tests. Skip those tests if unavailable.
```

### Test Data Conventions
- Test issues use `[Test]` prefix: `[Test] My Issue`
- Test projects use `Z`-prefixed keys: `ZTEST`, `ZT2` (sorts away from real projects)
- Test automation copies use `[Test]` prefix in the rule name
- Always clean up test data after tests complete

---

## 1. Config & Init

### config show

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk config show` | Table with columns: KEY, VALUE, SOURCE. Token is masked as `****...` |

### config test

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk config test` | `✓ Authentication successful` followed by user name and account ID |

### Bearer Auth Init & Config

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk init --auth-method bearer` (interactive) | Prompts for URL, API token, Cloud ID. Skips email prompt. Tests connection via gateway. |
| 2 | `jtk init --auth-method bearer --url URL --token TOKEN --cloud-id ID --no-verify` | Non-interactive setup completes without prompts |
| 3 | `jtk config show` (after bearer init) | Table shows `auth_method = bearer`, `cloud_id = <value>`, email row is empty |
| 4 | `jtk config test` (after bearer init) | `✓ Authentication successful` via gateway URL |

### me

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk me` | Detail block: Account ID, Display Name, Email, Active |
| 2 | `jtk me --id` | Account ID only |
| 3 | `jtk me --extended` | Extended user detail with additional fields |

---

## 2. Issues (Read-Only)

### issues list

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues list -p $PROJECT --max 3` | Table: KEY, SUMMARY, STATUS, ASSIGNEE, TYPE. At most 3 rows. |
| 2 | `jtk issues list -p $PROJECT --max 3 --id` | Issue keys only, one per line |
| 3 | `jtk issues list -p $PROJECT --max 3 --extended` | Extended table with additional columns |
| 4 | `jtk issues list -p $PROJECT --max 2 -o plain` | Tab-separated values, 2 rows |
| 5 | `jtk issues list -p NONEXISTENT` | Error message containing "not found" or empty results |

### issues get

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues get $EXISTING_ISSUE` | Detail block: Key, Summary, Status, Type, Priority, Assignee, Description (truncated), URL |
| 2 | `jtk issues get $EXISTING_ISSUE --id` | Issue key only |
| 3 | `jtk issues get $EXISTING_ISSUE --extended` | Full detail block plus Sprint, Transitions list, raw custom fields block |
| 4 | `jtk issues get $EXISTING_ISSUE --fulltext` | Full description and long text fields without truncation |
| 5 | `jtk issues get ${PROJECT}-99999` | `resource not found: Issue does not exist or you do not have permission to see it.` |

### issues search

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues search --jql "project = $PROJECT" --max 3` | Table of matching issues, at most 3 rows |
| 2 | `jtk issues search --jql "project = $PROJECT" --max 3 --id` | Issue keys only |
| 3 | `jtk issues search --jql "project = $PROJECT" --max 3 --extended` | Extended table |
| 4 | `jtk issues search --jql "project = $PROJECT AND summary ~ 'xyznonexistent999'"` | `No issues found` |
| 5 | `jtk issues search --jql "invalid jql ((("` | `bad request: Error in the JQL Query: ...` |

### Auto-pagination (issues search / issues list)

> These tests require a project with more than 100 issues. If your project has fewer, lower the `--max` value and adjust expected counts accordingly.

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues search --jql "project = $PROJECT" --max 200` | Table with > 100 rows (proves multi-page fetch) |
| 1b | `jtk issues search --jql "project = $PROJECT" --max 200 --id \| wc -l` | Number >= 101 (machine-verifiable row count) |
| 2 | `jtk issues list -p $PROJECT --max 200` | Same multi-page behavior for list |
| 2b | `jtk issues list -p $PROJECT --max 200 --id \| wc -l` | Number >= 101 (machine-verifiable row count) |

### `--fields` flag (issues search / issues list)

> `--fields` is a column projection flag. It limits which columns appear in the table output. Without `--fields`, the default columns are KEY, STATUS, TYPE, PTS, ASSIGNEE, SUMMARY.

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues search --jql "project = $PROJECT" --max 1` | Table with default columns: KEY, STATUS, TYPE, PTS, ASSIGNEE, SUMMARY |
| 2 | `jtk issues search --jql "project = $PROJECT" --max 1 --fields summary,status` | Table shows only SUMMARY and STATUS columns (KEY and others absent) |
| 3 | `jtk issues list -p $PROJECT --max 1 --fields summary,customfield_10005` | Table shows only SUMMARY and the custom field columns |

### issues types

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues types -p $PROJECT` | Table: ID, NAME, SUBTASK, DESCRIPTION |
| 2 | `jtk issues types -p $PROJECT --id` | Type IDs only |
| 3 | `jtk issues types -p NONEXISTENT` | Error: 404 |

### issues fields

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues fields` | Table: ID, NAME, TYPE, CUSTOM |
| 2 | `jtk issues fields --custom-fields` | Only rows where CUSTOM = yes |
| 3 | `jtk issues fields --id` | Field IDs only |
| 4 | `jtk issues fields --extended` | Extended table with schema info |

### issues field-options

> Positional syntax: `jtk issues field-options <ISSUE-KEY> <field-name-or-id>`

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues field-options $EXISTING_ISSUE priority` | Table: VALUE, ID (e.g., Highest/1, High/2, Medium/3, Low/4, Lowest/5) |
| 2 | `jtk issues field-options $EXISTING_ISSUE priority --id` | Option IDs only |

### issues check

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues check $EXISTING_ISSUE` | Table of fields with PRESENT/MISSING status |
| 2 | `jtk issues check $EXISTING_ISSUE --id` | Missing field IDs only (exit 1 if any missing, exit 0 if all present) |

---

## 3. Projects (Read-Only)

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk projects list --max 5` | Table: KEY, NAME, TYPE, LEAD |
| 2 | `jtk projects list --max 5 --id` | Project keys only |
| 3 | `jtk projects list --max 5 --extended` | Extended table |
| 4 | `jtk projects get $PROJECT` | Detail: Key, Name, ID, Type, Lead, Issue Types |
| 5 | `jtk projects get $PROJECT --id` | Project key only |
| 6 | `jtk projects get $PROJECT --extended` | Extended detail |
| 7 | `jtk projects get NONEXISTENT` | `resource not found: No project could be found with key 'NONEXISTENT'.` |
| 8 | `jtk projects types` | Table: KEY, FORMATTED (e.g., software/Software) |
| 9 | `jtk projects types --id` | Type keys only |
| 10 | `jtk projects types --extended` | Extended table |

---

## 4. Boards & Sprints (Read-Only)

> **Basic Auth only** — Agile endpoints (boards/sprints) are not available with scoped tokens (no Agile scope). Skip this section when testing with Bearer Auth.

### boards

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk boards list --max 5` | Table: ID, NAME, TYPE, PROJECT |
| 2 | `jtk boards list -p $PROJECT` | Only boards for that project |
| 3 | `jtk boards list --id` | Board IDs only |
| 4 | `jtk boards list --extended` | Extended table |
| 5 | `jtk boards get $BOARD_ID` | Detail: ID, Name, Type, Project |
| 6 | `jtk boards get $BOARD_ID --extended` | Extended detail including `Filter: <name> (id: <id>)` |
| 7 | `jtk boards get $BOARD_ID --id` | Board ID only |
| 8 | `jtk boards get 99999` | Error: 404 (board not found) |

### sprints

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk sprints list -b $BOARD_ID -s active` | Table: ID, NAME, STATE, START, END. State = `active` |
| 2 | `jtk sprints list -b $BOARD_ID -s active --id` | Sprint IDs only |
| 3 | `jtk sprints list -b $BOARD_ID --extended` | Extended table with additional sprint details |
| 4 | `jtk sprints current -b $BOARD_ID` | Current sprint detail: ID, Name, State, Start, End |
| 5 | `jtk sprints current -b $BOARD_ID --id` | Current sprint ID only |
| 6 | `jtk sprints current -b $BOARD_ID --extended` | Extended detail |
| 7 | `jtk sprints list` | `Error: required flag(s) "board" not set` |

### sprints issues

> The Jira Agile API endpoint is slow (~30s). Use `--max` to limit results. The client timeout is 60s.

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk sprints issues $SPRINT_ID --max 3` | Table: KEY, SUMMARY, STATUS, ASSIGNEE, TYPE |
| 2 | `jtk sprints issues $SPRINT_ID --max 3 --id` | Issue keys only |
| 3 | `jtk sprints issues $SPRINT_ID --max 3 --extended` | Extended table |
| 4 | `jtk sprints issues 99999` | Error |

---

## 5. Links (Read-Only)

### links types

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk links types` | Table: ID, NAME, OUTWARD, INWARD |
| 2 | `jtk links types --id` | Link type IDs only |
| 3 | `jtk links types --extended` | Extended table |

### links list

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk links list $EXISTING_ISSUE` | Table: ID, TYPE, DIRECTION, ISSUE, SUMMARY (or `No links on $EXISTING_ISSUE`) |
| 2 | `jtk links list $EXISTING_ISSUE --id` | Link IDs only |
| 3 | `jtk links list $EXISTING_ISSUE --extended` | Extended table |
| 4 | `jtk links list ${PROJECT}-99999` | `resource not found: ...` |

---

## 6. Dashboards (Read-Only)

> **Basic Auth only** — Dashboard endpoints are not available with scoped tokens (no Dashboard scope). Skip this section when testing with Bearer Auth.

### dashboards list

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk dashboards list --max 5` | Table: ID, NAME, OWNER, FAVOURITE |
| 2 | `jtk dashboards list --search "SEARCH_TERM"` | Filtered results matching search term |
| 3 | `jtk dashboards list --id` | Dashboard IDs only |
| 4 | `jtk dashboards list --extended` | Extended table |
| 5 | `jtk dashboards list --search "xyznonexistent999"` | `No dashboards found matching "xyznonexistent999"` |

### dashboards get

> `dashboards get` does not support `--id` or `--extended`. Gadgets are rendered inline in the detail view.

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk dashboards get $DASHBOARD_ID` | Detail: ID, Name, Description, Owner, URL; then inline Gadgets table (ID \| TITLE \| MODULE) if any. Note: `MODULE` is the raw gadget URI — this differs intentionally from `gadgets list` which uses `TYPE` (the resolved module key) and adds a `POSITION` column. |
| 2 | `jtk dashboards get 99999` | Error: 404 |

### dashboards gadgets list

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk dashboards gadgets list $DASHBOARD_ID` | Table: ID, POSITION, TITLE, TYPE |
| 2 | `jtk dashboards gadgets list $DASHBOARD_ID --id` | Gadget IDs only |

---

## 7. Users (Read-Only)

### users search

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk users search "YOUR_NAME"` | Table: ACCOUNT_ID, NAME, EMAIL, ACTIVE |
| 2 | `jtk users search "YOUR_NAME" --id` | Account IDs only |
| 3 | `jtk users search "YOUR_NAME" --extended` | Extended user table |
| 4 | `jtk users search "xyznonexistent999"` | `No users found matching 'xyznonexistent999'` |

### users get

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk users get $ACCOUNT_ID` | Detail: Account ID, Display Name, Email, Active |
| 2 | `jtk users get $ACCOUNT_ID --id` | Account ID only |
| 3 | `jtk users get $ACCOUNT_ID --extended` | Extended user detail |
| 4 | `jtk users get 000000000000000000000000` | Error: 404 (user not found) |

---

## 8. Automation (Read-Only)

> **Basic Auth only** — Automation endpoints are not available with scoped tokens (no Automation scope). Skip this section when testing with Bearer Auth.

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk auto list` | Table: UUID, NAME, STATE, LABELS |
| 2 | `jtk auto list --state ENABLED` | Only ENABLED rules |
| 3 | `jtk auto list --state DISABLED` | Only DISABLED rules |
| 4 | `jtk auto list --id` | Rule UUIDs only |
| 5 | `jtk auto list --extended` | Extended table with additional columns |
| 6 | `jtk auto get $AUTO_UUID` | Detail: Name, UUID, State, Description, Components summary |
| 7 | `jtk auto get $AUTO_UUID --extended` | Extended detail |
| 8 | `jtk auto get $AUTO_UUID --id` | Rule UUID only |
| 9 | `jtk auto get $AUTO_UUID --show-components` | Flat table: # \| COMPONENT \| TYPE |
| 10 | `jtk auto export $AUTO_UUID \| jq .` | Pretty-printed valid JSON (top-level keys: `rule`, `connections`) |
| 11 | `jtk auto export $AUTO_UUID --compact` | Single-line JSON |

---

## 9. Fields (Read-Only)

### fields list

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk fields list` | Table: ID, NAME, TYPE, CUSTOM |
| 2 | `jtk fields list --custom-fields` | Only rows where CUSTOM = yes |
| 3 | `jtk fields list --id` | Field IDs only |
| 4 | `jtk fields list --extended` | Extended table |
| 5 | `jtk fields list --name "story"` | Table showing only fields with "story" in the name |
| 6 | `jtk fields list --name "nonexistent"` | `No fields found` |
| 7 | `jtk fields list --name "story" --custom-fields` | Only custom fields matching "story" |

### fields show

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk fields show $CUSTOM_FIELD` | Flat denormalized view: contexts, project mappings, options |
| 2 | `jtk fields show $CUSTOM_FIELD --id` | Context IDs only |
| 3 | `jtk fields show customfield_99999` | Error: 404 |

### fields contexts list

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk fields contexts list $CUSTOM_FIELD` | Table: ID, NAME, GLOBAL, ANY_ISSUE_TYPE |
| 2 | `jtk fields contexts list $CUSTOM_FIELD --id` | Context IDs only |
| 3 | `jtk fields contexts list customfield_99999` | Error: 404 |

### fields options list

> Options list auto-detects the default context when `--context` is omitted.

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk fields options list $SELECT_FIELD` | Table: ID, VALUE, DISABLED |

---

## 10. Issue Mutations

Run these steps in order. Each step depends on the previous.

### Create and manipulate a test issue

1. **Check available types** (not all projects have "Task"):
   ```bash
   jtk issues types -p $PROJECT
   ```
   Note a valid type name → `$ISSUE_TYPE` (e.g., `SDLC`, `Task`, `Bug`)

2. **Create issue:**
   ```bash
   jtk issues create -p $PROJECT -t $ISSUE_TYPE -s "[Test] Integration Test Issue"
   ```
   Expected: Full issue detail block (Key, Summary, Status, Type, Priority, Reporter, URL)
   Capture the issue key → `$TEST_ISSUE`

   Also test `--id` variant:
   ```bash
   jtk issues create -p $PROJECT -t $ISSUE_TYPE -s "[Test] ID Flag Test" --id
   ```
   Expected: Issue key only (e.g., `MON-XXXX`). Delete this issue immediately after:
   ```bash
   jtk issues delete $PROJECT-XXXX --force
   ```

3. **Verify creation:**
   ```bash
   jtk issues get $TEST_ISSUE
   ```
   Expected: Key, Summary = `[Test] Integration Test Issue`, Status, Type = `$ISSUE_TYPE`

4. **Update description:**
   ```bash
   jtk issues update $TEST_ISSUE -d "Test description for integration testing"
   ```
   Expected: Full issue detail block with updated description

5. **Assign to self:**
   ```bash
   jtk issues assign $TEST_ISSUE $ACCOUNT_ID
   ```
   Expected: Full issue detail block with ASSIGNEE = your name

   Also test `--id` variant:
   ```bash
   jtk issues assign $TEST_ISSUE $ACCOUNT_ID --id
   ```
   Expected: Issue key only

6. **Add comment with escape sequences:**
   ```bash
   jtk comments add $TEST_ISSUE -b "Line one\nLine two\n\tIndented line"
   ```
   Expected: Comment detail block (Issue Key, Comment ID, Author, body excerpt)
   Capture the comment ID → `$COMMENT_ID`

   Also test `--id` variant. Capture the ID so it can be cleaned up:
   ```bash
   COMMENT_ID_2=$(jtk comments add $TEST_ISSUE -b "ID flag test comment" --id)
   echo $COMMENT_ID_2
   ```
   Expected: Comment ID only. Delete this comment immediately:
   ```bash
   jtk comments delete $TEST_ISSUE $COMMENT_ID_2 --force
   ```

6b. **Verify escape sequences rendered:**
   ```bash
   jtk comments list $TEST_ISSUE --fulltext
   ```
   Expected: Comment body shows actual newlines and tab, not literal `\n` or `\t`

7. **List comments:**
   ```bash
   jtk comments list $TEST_ISSUE
   ```
   Expected: Table showing `$COMMENT_ID`, your name, and the comment body

### Attachment sub-block

> Run before deleting `$TEST_ISSUE`.

7b. **Create a test file and upload (default output):**
    ```bash
    echo "integration test attachment" > /tmp/jtk-test-attach.txt
    jtk attachments add $TEST_ISSUE --file /tmp/jtk-test-attach.txt
    ```
    Expected: Table row with filename, attachment ID, and file size.
    Capture the attachment ID → `$ATTACH_ID`

    Also test `--id` variant. Capture the ID so it can be cleaned up:
    ```bash
    ATTACH_ID_2=$(jtk attachments add $TEST_ISSUE --file /tmp/jtk-test-attach.txt --id)
    echo $ATTACH_ID_2
    ```
    Expected: Attachment ID only (numeric). Delete this attachment immediately:
    ```bash
    jtk attachments delete $ATTACH_ID_2
    ```

7c. **List attachments:**
    ```bash
    jtk attachments list $TEST_ISSUE
    ```
    Expected: Table showing `$ATTACH_ID` with filename `jtk-test-attach.txt`

7d. **Download attachment:**
    ```bash
    jtk attachments get $ATTACH_ID --output /tmp/
    ```
    Expected: Download success message (e.g., `Downloaded $ATTACH_ID → /tmp/jtk-test-attach.txt (28 B)`)
    Clean up the download:
    ```bash
    rm /tmp/jtk-test-attach.txt
    ```

7e. **Delete attachment:**
    ```bash
    jtk attachments delete $ATTACH_ID
    ```
    Expected: `✓ Deleted attachment $ATTACH_ID`

7f. **Verify deletion:**
    ```bash
    jtk attachments list $TEST_ISSUE
    ```
    Expected: `$ATTACH_ID` no longer listed (or `No attachments on $TEST_ISSUE`)

8. **Check transitions:**
   ```bash
   jtk transitions list $TEST_ISSUE
   ```
   Expected: Table: ID, NAME, TO_STATUS
   Note a valid transition name → `$TRANSITION_NAME`

   Also verify `--extended` and `--id` variants:
   ```bash
   jtk transitions list $TEST_ISSUE --extended
   ```
   Expected: Table adds STATUS_CATEGORY, HAS_SCREEN, CONDITIONAL, REQUIRED_FIELDS columns
   ```bash
   jtk transitions list $TEST_ISSUE --id
   ```
   Expected: Transition IDs only, one per line

9. **Transition issue:**
   ```bash
   # If no required fields:
   jtk transitions do $TEST_ISSUE "$TRANSITION_NAME"
   # If required fields (e.g., Change Type):
   jtk transitions do $TEST_ISSUE "$TRANSITION_NAME" -f customfield_10005=Feature
   ```
   Expected: Full issue detail block with updated status

   Also test `--id` variant:
   ```bash
   jtk transitions do $TEST_ISSUE "$TRANSITION_NAME" --id
   ```
   Expected: Issue key only

10. **Verify transition:**
    ```bash
    jtk issues get $TEST_ISSUE
    ```
    Expected: Status shows the new value

11. **Unassign (via assign command):**
    ```bash
    jtk issues assign $TEST_ISSUE --unassign
    ```
    Expected: Full issue detail block with empty/unassigned ASSIGNEE

11b. **Re-assign, then unassign via update --assignee none:**
    ```bash
    jtk issues assign $TEST_ISSUE $ACCOUNT_ID
    jtk issues update $TEST_ISSUE --assignee none
    ```
    Expected: First command shows full detail with assignee; second command shows full detail with empty ASSIGNEE

11c. **Verify unassignment:**
    ```bash
    jtk issues get $TEST_ISSUE
    ```
    Expected: ASSIGNEE field shows empty/unassigned

12. **Delete comment:**
    ```bash
    jtk comments delete $TEST_ISSUE $COMMENT_ID
    ```
    Expected: `✓ Deleted comment $COMMENT_ID from $TEST_ISSUE`

13. **Delete issue:**
    ```bash
    jtk issues delete $TEST_ISSUE --force
    ```
    Expected: `✓ Deleted issue $TEST_ISSUE`

### Archive sub-block

> **Residual artifact:** `issues archive` has no corresponding `issues restore` in this CLI. Archived issues remain archived until restored or removed outside this CLI/runbook. This is an accepted residual — note the two archived issue keys.

1. **Create archive test issue (default output):**
   ```bash
   jtk issues create -p $PROJECT -t $ISSUE_TYPE -s "[Test] Archive Default"
   ```
   Capture the key → `$ARCHIVE_ISSUE_1`

2. **Archive (default output):**
   ```bash
   jtk issues archive $ARCHIVE_ISSUE_1
   ```
   Expected: `Archived $ARCHIVE_ISSUE_1`

3. **Create archive test issue (--id output):**
   ```bash
   jtk issues create -p $PROJECT -t $ISSUE_TYPE -s "[Test] Archive ID"
   ```
   Capture the key → `$ARCHIVE_ISSUE_2`

4. **Archive (--id output):**
   ```bash
   jtk issues archive $ARCHIVE_ISSUE_2 --id
   ```
   Expected: `$ARCHIVE_ISSUE_2` (issue key only)

> **Note:** Both `$ARCHIVE_ISSUE_1` and `$ARCHIVE_ISSUE_2` remain archived in Jira. They cannot be cleaned up via this CLI.

### Multi-value `--field` flag

> Requires a multi-select or multi-checkbox custom field (`$MULTI_FIELD`) on the project. Skip if unavailable.

1. **Create issue with multi-value field:**
   ```bash
   jtk issues create -p $PROJECT -t $ISSUE_TYPE -s "[Test] Multi-Value Field" \
     --field $MULTI_FIELD=Option1 --field $MULTI_FIELD=Option2
   ```
   Expected: Full issue detail block
   Capture the issue key → `$MV_ISSUE`

2. **Verify both values set:**
   ```bash
   jtk issues get $MV_ISSUE --extended
   ```
   Expected: Custom fields block shows both Option1 and Option2 for `$MULTI_FIELD`

3. **Cleanup:**
   ```bash
   jtk issues delete $MV_ISSUE --force
   ```

### Error cases

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues create -p $PROJECT` | `Error: required flag(s) "summary" not set` |
| 2 | `jtk issues create -s "No Project"` | `Error: required flag(s) "project" not set` |
| 3 | `jtk issues get ${PROJECT}-99999` | `resource not found: ...` |
| 4 | `jtk issues update ${PROJECT}-99999 -s "Nope"` | `resource not found: ...` |
| 5 | `jtk issues delete ${PROJECT}-99999 --force` | `resource not found: ...` |
| 6 | `jtk attachments add $TEST_ISSUE --file /tmp/nonexistent.txt` | Error: file not found |
| 7 | `jtk attachments delete 99999` | Error: 404 |

---

## 11. Link Mutations

Run these steps in order.

1. **Check link types:**
   ```bash
   jtk links types
   ```
   Note a valid type name → `$LINK_TYPE` (e.g., `Blocks`)

2. **Create two test issues:**
   ```bash
   jtk issues create -p $PROJECT -t $ISSUE_TYPE -s "[Test] Link Source"
   jtk issues create -p $PROJECT -t $ISSUE_TYPE -s "[Test] Link Target"
   ```
   Capture the keys → `$LINK_SOURCE`, `$LINK_TARGET`

3. **Create link:**
   ```bash
   jtk links create $LINK_SOURCE $LINK_TARGET --type $LINK_TYPE
   ```
   Expected: Table row with link ID, type, direction, target issue, and summary

   Also test `--id` variant:
   ```bash
   jtk links create $LINK_SOURCE $LINK_TARGET --type $LINK_TYPE --id
   ```
   Expected: Link ID only

4. **Verify link:**
   ```bash
   jtk links list $LINK_SOURCE
   ```
   Expected: Table shows link to `$LINK_TARGET` with type `$LINK_TYPE`
   Capture the link ID → `$LINK_ID`

5. **Delete link:**
   ```bash
   jtk links delete $LINK_ID
   ```
   Expected: `Deleted link $LINK_ID`

6. **Verify deletion:**
   ```bash
   jtk links list $LINK_SOURCE
   ```
   Expected: No link to `$LINK_TARGET` (or `No links on $LINK_SOURCE`)

7. **Cleanup:**
   ```bash
   jtk issues delete $LINK_SOURCE --force
   jtk issues delete $LINK_TARGET --force
   ```

### Error cases

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk links create $EXISTING_ISSUE ${PROJECT}-99999 --type $LINK_TYPE` | `resource not found: ...` |
| 2 | `jtk links create $EXISTING_ISSUE $EXISTING_ISSUE --type "NonexistentType"` | `link type "NonexistentType" not found (available: ...)` |
| 3 | `jtk links delete 99999` | Error: 404 |

---

## 12. Project Mutations

Run these steps in order.

1. **Create project:**
   ```bash
   jtk projects create --key ZTEST --name "Integration Test Project" --type software --lead $ACCOUNT_ID
   ```
   Expected: Full project detail block (Key, Name, Type, Lead, etc.)

2. **Verify creation:**
   ```bash
   jtk projects get ZTEST
   ```
   Expected: Key = ZTEST, Name = Integration Test Project

3. **Update name:**
   ```bash
   jtk projects update ZTEST --name "Updated Test Project"
   ```
   Expected: Full project detail block with Name = Updated Test Project

4. **Verify update:**
   ```bash
   jtk projects get ZTEST
   ```
   Expected: Name = Updated Test Project

5. **Delete:**
   ```bash
   jtk projects delete ZTEST --force
   ```
   Expected: `✓ Deleted project ZTEST (moved to trash)`

6. **Restore:**
   ```bash
   jtk projects restore ZTEST
   ```
   Expected: Full project detail block

7. **Verify restore:**
   ```bash
   jtk projects get ZTEST
   ```
   Expected: Project is accessible

8. **Final cleanup:**
   ```bash
   jtk projects delete ZTEST --force
   ```
   Expected: `✓ Deleted project ZTEST (moved to trash)`

### Error cases

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk projects create --key ZTEST` | `Error: required flag(s) "lead", "name" not set` |
| 2 | `jtk projects get NONEXISTENT` | `resource not found: No project could be found with key 'NONEXISTENT'.` |
| 3 | `jtk projects delete NONEXISTENT --force` | Error: 404 |

---

## 13. Dashboard Mutations

> **Basic Auth only** — Dashboard endpoints are not available with scoped tokens. Skip this section when testing with Bearer Auth.

Run these steps in order.

1. **Create dashboard:**
   ```bash
   jtk dashboards create --name "[Test] Integration Dashboard"
   ```
   Expected: `Created dashboard [Test] Integration Dashboard (XXXXX)`
   Capture the dashboard ID → `$TEST_DASH_ID`

2. **Verify creation:**
   ```bash
   jtk dashboards get $TEST_DASH_ID
   ```
   Expected: Name = `[Test] Integration Dashboard`

3. **List and search:**
   ```bash
   jtk dashboards list --search "[Test] Integration"
   ```
   Expected: Dashboard appears in results

4. **List gadgets (empty):**
   ```bash
   jtk dashboards gadgets list $TEST_DASH_ID
   ```
   Expected: `No gadgets on dashboard $TEST_DASH_ID`

5. **Add gadget:**
   ```bash
   jtk dashboards gadgets add $TEST_DASH_ID --type com.atlassian.jira.gadgets:filter-results-gadget
   ```
   Expected: Single table row with gadget ID, title, module, position

   Also test `--id` variant:
   ```bash
   jtk dashboards gadgets add $TEST_DASH_ID --type com.atlassian.jira.gadgets:filter-results-gadget --id
   ```
   Expected: Gadget ID only

6. **List gadgets (populated):**
   ```bash
   jtk dashboards gadgets list $TEST_DASH_ID
   ```
   Expected: Table with the added gadget

7. **Delete:**
   ```bash
   jtk dashboards delete $TEST_DASH_ID
   ```
   Expected: `Deleted dashboard $TEST_DASH_ID`

8. **Verify deletion:**
   ```bash
   jtk dashboards get $TEST_DASH_ID
   ```
   Expected: Error: 404

### Error cases

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk dashboards create` | `Error: required flag(s) "name" not set` |
| 2 | `jtk dashboards get 99999` | Error: 404 |
| 3 | `jtk dashboards delete 99999` | Error: 404 |
| 4 | `jtk dashboards gadgets remove 99999 1` | Error: 404 |

---

## 14. Automation Mutations

> **Basic Auth only** — Automation endpoints are not available with scoped tokens. Skip this section when testing with Bearer Auth.

Run these steps in order. All mutations operate on a **copy** of a real rule — never modify production rules.

> **Source rule state:** `$AUTO_UUID` was selected from `jtk auto list --state ENABLED`. The copy starts ENABLED. The correct toggle order is: create → disable → enable → idempotent enable → update → delete.

### Create test copy

1. **Export a rule:**
   ```bash
   jtk auto export $AUTO_UUID > /tmp/auto-source.json
   ```

2. **Strip UUID and rename** (the API rejects duplicate UUIDs):
   ```bash
   jq 'del(.rule.uuid) | .rule.name = "[Test] Auto Integration Copy"' /tmp/auto-source.json > /tmp/auto-clean.json
   ```

3. **Create the copy:**
   ```bash
   jtk auto create --file /tmp/auto-clean.json
   ```
   Expected: Full automation rule detail block (Name, UUID, State, etc.)
   Capture the UUID → `$TEST_AUTO_UUID`

   Also test `--id` variant (requires a second throwaway copy):
   ```bash
   jq 'del(.rule.uuid) | .rule.name = "[Test] Auto Integration Copy 2"' /tmp/auto-source.json > /tmp/auto-clean-2.json
   TEST_AUTO_UUID_2=$(jtk auto create --file /tmp/auto-clean-2.json --id)
   echo $TEST_AUTO_UUID_2
   ```
   Expected: UUID only (one line).
   Clean up immediately:
   ```bash
   jtk auto delete $TEST_AUTO_UUID_2
   ```

4. **Verify creation:**
   ```bash
   jtk auto get $TEST_AUTO_UUID
   ```
   Expected: Name = `[Test] Auto Integration Copy`, same component count as source

### Toggle cycle

5. **Disable (source copy starts ENABLED):**
   ```bash
   jtk auto disable $TEST_AUTO_UUID
   ```
   Expected: Full rule detail block with State = DISABLED
   (Fallback: `Rule "[Test] Auto Integration Copy": ENABLED → DISABLED`)

   Also test `--id` variant:
   ```bash
   jtk auto disable $TEST_AUTO_UUID --id
   ```
   Expected: Rule UUID only

6. **Re-enable:**
   ```bash
   jtk auto enable $TEST_AUTO_UUID
   ```
   Expected: Full rule detail block with State = ENABLED
   (Fallback: `Rule "[Test] Auto Integration Copy": DISABLED → ENABLED`)

   Also test `--id` variant:
   ```bash
   jtk auto enable $TEST_AUTO_UUID --id
   ```
   Expected: Rule UUID only

7. **Idempotent enable:**
   ```bash
   jtk auto enable $TEST_AUTO_UUID
   ```
   Expected: `Rule "[Test] Auto Integration Copy" is already ENABLED`

### Round-trip update

8. **Export the copy:**
   ```bash
   jtk auto export $TEST_AUTO_UUID > /tmp/auto-rt.json
   ```

9. **Update with no changes (round-trip):**
   ```bash
   jtk auto update $TEST_AUTO_UUID --file /tmp/auto-rt.json
   ```
   Expected: Full rule detail block

10. **Verify unchanged:**
    ```bash
    jtk auto get $TEST_AUTO_UUID
    ```
    Expected: Name, state, and component count unchanged

### Cleanup test rule

11. **Delete the test rule:**
    ```bash
    jtk auto delete $TEST_AUTO_UUID
    ```
    Expected: Rule deleted (auto-disables if ENABLED)

12. **Clean up temporary files:**
    ```bash
    rm -f /tmp/auto-source.json /tmp/auto-clean.json /tmp/auto-clean-2.json /tmp/auto-rt.json
    ```

### Error cases

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk auto create` | `Error: required flag(s) "file" not set` |
| 2 | `echo "not json" > /tmp/bad.json && jtk auto create --file /tmp/bad.json` | Error: does not contain valid JSON |
| 3 | `jtk auto create --file /tmp/nope.json` | Error: failed to read file |
| 4 | `jtk auto enable 99999999` | Error |

---

## 15. Sprint Mutations

> **Basic Auth only** — Agile endpoints are not available with scoped tokens. Skip this section when testing with Bearer Auth.
>
> Only test if you have a sprint-capable board. Sprint issues endpoint is slow (~30s).

1. **Create a test issue:**
   ```bash
   jtk issues create -p $PROJECT -t $ISSUE_TYPE -s "[Test] Sprint Add Test"
   ```
   Capture the key → `$SPRINT_TEST_ISSUE`

2. **Add issue to sprint:**
   ```bash
   jtk sprints add $SPRINT_ID $SPRINT_TEST_ISSUE
   ```
   Expected: Issues list showing the added issue(s) in the sprint context

   Also test `--id` variant:
   ```bash
   jtk sprints add $SPRINT_ID $SPRINT_TEST_ISSUE --id
   ```
   Expected: Issue key only

3. **Verify** (may be slow):
   ```bash
   jtk sprints issues $SPRINT_ID --max 50 --id
   ```
   Expected: `$SPRINT_TEST_ISSUE` appears in the list of issue keys

4. **Cleanup:**
   ```bash
   jtk issues delete $SPRINT_TEST_ISSUE --force
   ```

---

## 16. Field Mutations

Run these steps in order. Each step depends on the previous.

> Field management requires "Administer Jira" global permission. If you get 403 errors, verify your account has this permission.

> **Residual artifact:** `fields delete` moves fields to trash (not permanent deletion — no purge API exists). The field at `$TEST_FIELD` will remain trashed at the end of this section. This is an accepted residual.

### Create and manage a test field

1. **Create a select field:**
   ```bash
   jtk fields create --name "[Test] Integration Select" --type com.atlassian.jira.plugin.system.customfieldtypes:select
   ```
   Expected: Field detail row (ID, Name, Type)
   Capture the field ID → `$TEST_FIELD`

   Also test `--id` variant. Capture the ID so it can be cleaned up:
   ```bash
   FIELD_ID_2=$(jtk fields create --name "[Test] Integration Select 2" --type com.atlassian.jira.plugin.system.customfieldtypes:select --id)
   echo $FIELD_ID_2
   ```
   Expected: Field ID only (e.g. `customfield_XXXXX`). Delete this field immediately:
   ```bash
   jtk fields delete $FIELD_ID_2 --force
   ```

2. **Verify creation:**
   ```bash
   jtk fields list --name "[Test] Integration Select"
   ```
   Expected: Table showing the newly created field

3. **Inspect field detail:**
   ```bash
   jtk fields show $TEST_FIELD
   ```
   Expected: Flat view of contexts, project mappings, and options (may be sparse for new field)

   ```bash
   jtk fields show $TEST_FIELD --id
   ```
   Expected: Context IDs only

4. **List contexts:**
   ```bash
   jtk fields contexts list $TEST_FIELD
   ```
   Expected: Table showing the default context. Capture context ID → `$TEST_CTX`

5. **Add options:**
   ```bash
   jtk fields options add $TEST_FIELD --value "Option A"
   ```
   Expected: `✓ Added option XXXXX (Option A)`
   ```bash
   jtk fields options add $TEST_FIELD --value "Option B"
   ```
   Expected: `✓ Added option XXXXX (Option B)`

6. **List options:**
   ```bash
   jtk fields options list $TEST_FIELD
   ```
   Expected: Table showing Option A and Option B
   Capture an option ID → `$OPT_ID`

7. **Update option:**
   ```bash
   jtk fields options update $TEST_FIELD --option $OPT_ID --value "Option A (updated)"
   ```
   Expected: `✓ Updated option $OPT_ID`

8. **Verify update:**
   ```bash
   jtk fields options list $TEST_FIELD
   ```
   Expected: Shows "Option A (updated)" instead of "Option A"

9. **Delete option:**
   ```bash
   jtk fields options delete $TEST_FIELD --option $OPT_ID --force
   ```
   Expected: `✓ Deleted option $OPT_ID from field $TEST_FIELD`

10. **Create context:**
    ```bash
    jtk fields contexts create $TEST_FIELD --name "[Test] Context"
    ```
    Expected: `✓ Created context XXXXX ([Test] Context)`
    Capture context ID → `$NEW_CTX`

11. **Delete context:**
    ```bash
    jtk fields contexts delete $TEST_FIELD $NEW_CTX --force
    ```
    Expected: `✓ Deleted context $NEW_CTX from field $TEST_FIELD`

12. **Trash field:**
    ```bash
    jtk fields delete $TEST_FIELD --force
    ```
    Expected: `✓ Trashed field $TEST_FIELD`

13. **Restore field:**
    ```bash
    jtk fields restore $TEST_FIELD
    ```
    Expected: Full field detail block

14. **Final cleanup — trash again:**
    ```bash
    jtk fields delete $TEST_FIELD --force
    ```
    Expected: `✓ Trashed field $TEST_FIELD`

> **Accepted residual:** `$TEST_FIELD` remains in the trash. Fields must be purged through the Jira admin UI; no API exists for permanent deletion.

### Error cases

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk fields create` | `Error: required flag(s) "name", "type" not set` |
| 2 | `jtk fields delete customfield_99999 --force` | Error: 404 |
| 3 | `jtk fields contexts list customfield_99999` | Error: 404 |
| 4 | `jtk fields options add customfield_99999 --value "Nope"` | Error |

---

## 17. Global Flags & Aliases

### Output formats

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues list -p $PROJECT --max 1 --no-color \| cat -v` | No `^[[` ANSI escape sequences |
| 2 | `jtk issues list -p $PROJECT --max 1 --verbose` | Shows `→ GET ...` and `← 200 OK` debug lines |
| 3 | `jtk issues list -p $PROJECT --max 1 -o json \| jq .` | Parses without errors (compatibility smoke test) |
| 4 | `jtk issues list -p $PROJECT --max 1 -o plain` | Tab-separated, one row |

### Command aliases

Verify each alias produces the same output as the full command:

| # | Alias | Full Command |
|---|-------|-------------|
| 1 | `jtk i list -p $PROJECT --max 1` | `jtk issues list -p $PROJECT --max 1` |
| 2 | `jtk p list --max 1` | `jtk projects list --max 1` |
| 3 | `jtk proj list --max 1` | `jtk projects list --max 1` |
| 4 | `jtk b list --max 1` | `jtk boards list --max 1` |
| 5 | `jtk sp list -b $BOARD_ID -s active` | `jtk sprints list -b $BOARD_ID -s active` |
| 6 | `jtk u search "a" --max 1` | `jtk users search "a" --max 1` |
| 7 | `jtk auto list --state ENABLED` | `jtk automation list --state ENABLED` |
| 8 | `jtk tr list $EXISTING_ISSUE` | `jtk transitions list $EXISTING_ISSUE` |
| 9 | `jtk c list $EXISTING_ISSUE --max 1` | `jtk comments list $EXISTING_ISSUE --max 1` |
| 10 | `jtk att list $EXISTING_ISSUE` | `jtk attachments list $EXISTING_ISSUE` |
| 11 | `jtk f list --max 1` | `jtk fields list --max 1` |
| 12 | `jtk field list --max 1` | `jtk fields list --max 1` |
| 13 | `jtk l list $EXISTING_ISSUE` | `jtk links list $EXISTING_ISSUE` |
| 14 | `jtk link list $EXISTING_ISSUE` | `jtk links list $EXISTING_ISSUE` |
| 15 | `jtk dash list --max 1` | `jtk dashboards list --max 1` |
| 16 | `jtk dashboard list --max 1` | `jtk dashboards list --max 1` |

### Shell completion

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk completion bash \| head -3` | Starts with `# bash completion for jtk` |
| 2 | `jtk completion zsh \| head -3` | Valid zsh completion script |

---

## 18. Error Cases

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues get ${PROJECT}-99999` | `resource not found: Issue does not exist or you do not have permission to see it.` |
| 2 | `jtk issues search --jql "invalid jql ((("` | `bad request: Error in the JQL Query: ...` |
| 3 | `jtk issues create -p $PROJECT` | `Error: required flag(s) "summary" not set` |
| 4 | `jtk projects get NONEXISTENT` | `resource not found: No project could be found with key 'NONEXISTENT'.` |
| 5 | `jtk boards get 99999` | Error: 404 |
| 6 | `jtk sprints list` | `Error: required flag(s) "board" not set` |
| 7 | `jtk links list ${PROJECT}-99999` | `resource not found: ...` |
| 8 | `jtk dashboards get 99999` | Error: 404 |

---

## 19. Bearer Auth Guards

> **Bearer Auth only** — Run this section ONLY during the Bearer Auth pass. These tests verify that scope-restricted commands produce clear, actionable error messages instead of cryptic API failures.
>
> Error messages are defined in `api/client.go` as `ErrAgileUnavailable`, `ErrAutomationUnavailable`, and `ErrDashboardUnavailable`. Guards run via `PersistentPreRunE` on the parent command, so they execute before any child command's `RunE`.

### Agile API (Boards & Sprints)

| # | Command | Expected Error |
|---|---------|----------------|
| 1 | `jtk boards list` | `this command requires the Agile API, which is not available with bearer auth (scoped tokens lack the Agile scope)` |
| 2 | `jtk boards get 1` | Same Agile error |
| 3 | `jtk sprints list -b 1` | Same Agile error |
| 4 | `jtk sprints current -b 1` | Same Agile error |
| 5 | `jtk sprints issues 1` | Same Agile error |
| 6 | `jtk sprints add 1 PROJ-1` | Same Agile error |

### Automation API

| # | Command | Expected Error |
|---|---------|----------------|
| 7 | `jtk auto list` | `this command requires the Automation API, which is not available with bearer auth (scoped tokens lack the Automation scope)` |
| 8 | `jtk auto get some-uuid` | Same Automation error |
| 9 | `jtk auto export some-uuid` | Same Automation error |
| 10 | `jtk auto create --file rule.json` | Same Automation error |
| 11 | `jtk auto enable some-uuid` | Same Automation error |
| 12 | `jtk auto disable some-uuid` | Same Automation error |

### Dashboard API

| # | Command | Expected Error |
|---|---------|----------------|
| 13 | `jtk dashboards list` | `this command requires the Dashboard API, which is not available with bearer auth (scoped tokens lack the Dashboard scope)` |
| 14 | `jtk dashboards get 1` | Same Dashboard error |
| 15 | `jtk dashboards create --name "x"` | Same Dashboard error |
| 16 | `jtk dashboards delete 1` | Same Dashboard error |
| 17 | `jtk dashboards gadgets list 1` | Same Dashboard error |
| 18 | `jtk dashboards gadgets add 1 --type foo` | Same Dashboard error |
| 19 | `jtk dashboards gadgets remove 1 1` | Same Dashboard error |

---

## Test Execution Checklist

### Pass 1: Basic Auth

#### Setup (Basic Auth)
- [ ] `make build-jtk`
- [ ] `jtk init` (Basic Auth)
- [ ] `jtk config test` — Authentication successful
- [ ] `jtk me` works
- [ ] Discover: `$PROJECT`, `$BOARD_ID`, `$SPRINT_ID`, `$ACCOUNT_ID`, `$AUTO_UUID`, `$DASHBOARD_ID`, `$EXISTING_ISSUE`, `$LINK_TYPE`, `$CUSTOM_FIELD`, `$SELECT_FIELD`
- [ ] `jtk issues types -p $PROJECT` to learn `$ISSUE_TYPE`

#### Config & Init (Section 1)
- [ ] `config show` (table)
- [ ] `config test`
- [ ] `me` (table, `--id`, `--extended`)

#### Issues Read-Only (Section 2)
- [ ] `issues list` (table, `--id`, `--extended`, plain, error)
- [ ] `issues get` (table, `--id`, `--extended`, `--fulltext`, 404)
- [ ] `issues search` (results, `--id`, `--extended`, no results, bad JQL)
- [ ] Auto-pagination (search multi-page, list multi-page)
- [ ] `--fields` flag (field pass-through for search and list)
- [ ] `issues types` (table, `--id`, 404)
- [ ] `issues fields` (all, `--custom-fields`, `--id`, `--extended`)
- [ ] `issues field-options` (positional syntax: `jtk issues field-options $EXISTING_ISSUE priority`, `--id`)
- [ ] `issues check` (table, `--id`)

#### Projects Read-Only (Section 3)
- [ ] `projects list` (table, `--id`, `--extended`)
- [ ] `projects get` (table, `--id`, `--extended`, 404)
- [ ] `projects types` (table, `--id`, `--extended`)

#### Boards & Sprints Read-Only (Section 4)
- [ ] `boards list`, `boards get` (table, `--id`, `--extended`, 404)
- [ ] `boards get --extended` shows `Filter: <name> (id: <id>)`
- [ ] `sprints list`, `sprints current` (`--id`, `--extended`)
- [ ] `sprints issues` (table, `--id`, `--extended`)

#### Links Read-Only (Section 5)
- [ ] `links types` (table, `--id`, `--extended`)
- [ ] `links list` (table, `--id`, `--extended`, 404)

#### Dashboards Read-Only (Section 6)
- [ ] `dashboards list` (table, search, `--id`, `--extended`, no results)
- [ ] `dashboards get` (detail with inline gadgets, 404) — no `--id`/`--extended`
- [ ] `dashboards gadgets list` (table, `--id`)

#### Users Read-Only (Section 7)
- [ ] `users search` (results, `--id`, `--extended`, no results)
- [ ] `users get` (table, `--id`, `--extended`, 404)

#### Automation Read-Only (Section 8)
- [ ] `auto list` (all, `--state ENABLED`, `--state DISABLED`, `--id`, `--extended`)
- [ ] `auto get` (detail, `--extended`, `--id`, `--show-components` flat table)
- [ ] `auto export` (pretty JSON, compact JSON)

#### Fields Read-Only (Section 9)
- [ ] `fields list` (all, `--custom-fields`, `--id`, `--extended`)
- [ ] `fields show` (detail, `--id`, 404)
- [ ] `fields contexts list` (table, `--id`, 404)
- [ ] `fields options list` (table)

#### Issue Mutations (Section 10)
- [ ] Create (full detail output) → get → update → assign (`--id` variant) → comment → comment `--id` variant → transitions list → transition (`--id` variant) → unassign → delete comment → delete issue
- [ ] Unassign via `--assignee none` on `issues update`
- [ ] Attachment sub-block (steps 7b–7f): upload (full output + `--id` variant) → list → download → delete → verify
- [ ] Archive sub-block: create `$ARCHIVE_ISSUE_1` → archive (default) → create `$ARCHIVE_ISSUE_2` → archive `--id`
- [ ] Multi-value `--field` flag (create issue with repeated `--field` same key)
- [ ] Error cases (missing flags, 404, attachment not found)

#### Link Mutations (Section 11)
- [ ] Types → create issues → create link → verify → delete link → verify → cleanup
- [ ] Error cases (nonexistent target, invalid type, delete 404)

#### Project Mutations (Section 12)
- [ ] Create (full detail) → get → update → delete → restore → verify → delete (cleanup)
- [ ] Error cases

#### Dashboard Mutations (Section 13)
- [ ] Create → verify → list+search → gadgets add → gadgets list → delete → verify 404
- [ ] Error cases (missing flags, 404)

#### Automation Mutations (Section 14)
- [ ] Create copy (strip UUID, rename) → verify
- [ ] Toggle cycle: disable → enable → idempotent enable
- [ ] Round-trip update
- [ ] Cleanup (`jtk auto delete` + `rm -f /tmp/auto-*.json`)
- [ ] Error cases

#### Sprint Mutations (Section 15)
- [ ] Create issue → add to sprint → verify with `--id` → delete issue

#### Field Mutations (Section 16)
- [ ] Create field → `fields show` → list contexts → add options → update option → delete option
- [ ] Create context → delete context
- [ ] Trash field → restore → trash again (cleanup)
- [ ] Error cases (missing flags, 404)

#### Global Flags & Aliases (Section 17)
- [ ] `--no-color`, `--verbose`, `-o json` (compat smoke test), `-o plain`
- [ ] All aliases verified (including `jtk l`, `jtk link`, `jtk dash`, `jtk dashboard`)

#### Error Cases (Section 18)
- [ ] All error cases (404, bad JQL, missing flags)

#### Cleanup (Basic Auth)
- [ ] Delete test projects: `jtk projects delete ZTEST --force` (etc.)
- [ ] Delete test issues: search for `[Test]` prefix, delete with `--force`
- [ ] Delete test dashboards: `jtk dashboards delete $TEST_DASH_ID`
- [ ] Trash test fields: `jtk fields delete $TEST_FIELD --force`
- [ ] Delete automation test rules: `jtk auto list | grep '\[Test\]' | awk '{print $1}' | xargs -I{} jtk auto delete {}`
- [ ] Verify: `jtk auto list | grep -E '\[Test\]|\[DELETEME\]'` — should be empty
- [ ] **Accepted residuals:** `$ARCHIVE_ISSUE_1` and `$ARCHIVE_ISSUE_2` remain archived (no CLI restore); `$TEST_FIELD` remains trashed (no purge API)

---

### Pass 2: Bearer Auth

#### Setup (Bearer Auth)
- [ ] `jtk init --auth-method bearer`
- [ ] `jtk config test` — Authentication successful via gateway
- [ ] `jtk me` works
- [ ] Discover: `$PROJECT`, `$EXISTING_ISSUE`, `$ACCOUNT_ID`, `$LINK_TYPE`, `$CUSTOM_FIELD`, `$SELECT_FIELD`
- [ ] `jtk issues types -p $PROJECT` to learn `$ISSUE_TYPE`
- [ ] Skip: `$BOARD_ID`, `$SPRINT_ID`, `$AUTO_UUID`, `$DASHBOARD_ID` (unavailable with bearer auth)

#### Config & Init (Section 1)
- [ ] Bearer auth init (interactive)
- [ ] Bearer auth init (non-interactive)
- [ ] Bearer auth `config show` (auth_method = bearer, cloud_id displayed)
- [ ] Bearer auth `config test`
- [ ] `me` (table, `--id`, `--extended`)

#### Issues Read-Only (Section 2)
- [ ] `issues list` (table, `--id`, `--extended`, plain, error)
- [ ] `issues get` (table, `--id`, `--extended`, `--fulltext`, 404)
- [ ] `issues search` (results, `--id`, `--extended`, no results, bad JQL)
- [ ] Auto-pagination (search multi-page, list multi-page)
- [ ] `--fields` flag (field pass-through for search and list)
- [ ] `issues types` (table, `--id`, 404)
- [ ] `issues fields` (all, `--custom-fields`, `--id`, `--extended`)
- [ ] `issues field-options` (positional syntax: `jtk issues field-options $EXISTING_ISSUE priority`, `--id`)
- [ ] `issues check` (table, `--id`)

#### Projects Read-Only (Section 3)
- [ ] `projects list` (table, `--id`, `--extended`)
- [ ] `projects get` (table, `--id`, `--extended`, 404)
- [ ] `projects types` (table, `--id`, `--extended`)

#### Links Read-Only (Section 5)
- [ ] `links types` (table, `--id`, `--extended`)
- [ ] `links list` (table, `--id`, `--extended`, 404)

#### Users Read-Only (Section 7)
- [ ] `users search` (results, `--id`, `--extended`, no results)
- [ ] `users get` (table, `--id`, `--extended`, 404)

#### Fields Read-Only (Section 9)
- [ ] `fields list` (all, `--custom-fields`, `--id`, `--extended`)
- [ ] `fields show` (detail, `--id`, 404)
- [ ] `fields contexts list` (table, `--id`, 404)
- [ ] `fields options list` (table)

#### Issue Mutations (Section 10)
- [ ] Create → get → update → assign → comment → transitions list → transition → unassign → delete comment → delete issue
- [ ] Unassign via `--assignee none` on `issues update`
- [ ] Attachment sub-block (steps 7b–7f): upload (full output + `--id` variant) → list → download → delete → verify
- [ ] Archive sub-block (two issues)
- [ ] Multi-value `--field` flag
- [ ] Error cases (missing flags, 404, attachment not found)

#### Link Mutations (Section 11)
- [ ] Types → create issues → create link → verify → delete link → verify → cleanup
- [ ] Error cases

#### Project Mutations (Section 12)
- [ ] Create → get → update → delete → restore → verify → delete (cleanup)
- [ ] Error cases

#### Field Mutations (Section 16)
- [ ] Create field → `fields show` → list contexts → add options → update option → delete option
- [ ] Create context → delete context
- [ ] Trash field → restore → trash again (cleanup)
- [ ] Error cases

#### Bearer Auth Guards (Section 19)
- [ ] Boards: `list`, `get 1` → Agile scope error
- [ ] Sprints: `list -b 1`, `current -b 1`, `issues 1`, `add 1 PROJ-1` → Agile scope error
- [ ] Automation: `list`, `get`, `export`, `create`, `enable`, `disable` → Automation scope error
- [ ] Dashboards: `list`, `get`, `create`, `delete`, `gadgets list`, `gadgets remove` → Dashboard scope error

#### Global Flags & Aliases (Section 17)
- [ ] `--no-color`, `--verbose`, `-o json` (compat smoke test), `-o plain`
- [ ] Applicable aliases (skip `jtk b`, `jtk sp`, `jtk auto`, `jtk dash`, `jtk dashboard`)

#### Error Cases (Section 18)
- [ ] All applicable error cases (skip rows 5 and 8: boards get and dashboards get)

#### Cleanup (Bearer Auth)
- [ ] Delete test projects: `jtk projects delete ZTEST --force`
- [ ] Delete test issues: search for `[Test]` prefix, delete with `--force`
- [ ] Trash test fields: `jtk fields delete $TEST_FIELD --force`
- [ ] **Accepted residuals:** `$ARCHIVE_ISSUE_1` and `$ARCHIVE_ISSUE_2` remain archived; `$TEST_FIELD` remains trashed

---

## Adding New Tests

When adding new features or fixing bugs:

1. Add test steps to the appropriate numbered section above
2. Include both happy path and error cases with exact expected output
3. Document gotchas inline, immediately before the step where they matter
4. Update both Pass 1 and Pass 2 in the Test Execution Checklist
5. If the feature is scope-restricted, add guard tests to Section 19
6. Record bugs discovered during testing and continue — don't stop to fix
