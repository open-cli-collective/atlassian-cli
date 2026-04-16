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
jtk me -o json | jq -r .accountId

# $PROJECT — pick a project you have full access to
jtk projects list --max 10
# Note the KEY column value, e.g., MON

# $ISSUE_TYPES — check available issue types (not all projects have "Task")
jtk issues types -p $PROJECT
# Note a valid type name, e.g., SDLC, Bug, Task

# $EXISTING_ISSUE — pick an existing issue key for read-only tests
jtk issues list -p $PROJECT --max 3
# Note a KEY, e.g., MON-3714

# $BOARD_ID — find a board for your project (Basic Auth only)
jtk boards list -p $PROJECT
# Note the ID column, e.g., 23

# $SPRINT_ID — find the active sprint (Basic Auth only)
jtk sprints list -b $BOARD_ID -s active
# Note the ID column, e.g., 119

# $AUTO_UUID — pick an enabled automation rule (Basic Auth only)
jtk auto list --state ENABLED
# Note a UUID from the first column

# $DASHBOARD_ID — pick a dashboard (Basic Auth only)
jtk dashboards list --max 5
# Note an ID, e.g., 10001

# $LINK_TYPE — check available link types
jtk links types
# Note a NAME, e.g., Blocks

# $CUSTOM_FIELD — pick a custom field ID
jtk fields list --custom
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
| 2 | `jtk config show -o json` | Valid JSON object with keys `url`, `email`, `api_token`, etc. |

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
| 4 | `jtk config show -o json` (after bearer init) | JSON has `"auth_method": "bearer"`, `"cloud_id": "<value>"` |
| 5 | `jtk config test` (after bearer init) | `✓ Authentication successful` via gateway URL |

### me

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk me` | Table with Account ID, Display Name, Email, Active |
| 2 | `jtk me -o json` | Valid JSON with `accountId`, `displayName`, `emailAddress`, `active` |
| 3 | `jtk me -o plain` | Account ID only (single line) |

---

## 2. Issues (Read-Only)

### issues list

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues list -p $PROJECT --max 3` | Table with columns: KEY, SUMMARY, STATUS, ASSIGNEE, TYPE. At most 3 rows. |
| 2 | `jtk issues list -p $PROJECT --max 2 -o json` | Valid JSON array with 2 elements |
| 3 | `jtk issues list -p $PROJECT --max 2 -o plain` | Tab-separated values, 2 rows |
| 4 | `jtk issues list -p NONEXISTENT` | Error message containing "not found" or empty results |

### issues get

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues get $EXISTING_ISSUE` | Shows Key, Summary, Status, Type, Priority, Assignee, Description, URL |
| 2 | `jtk issues get $EXISTING_ISSUE -o json` | Valid JSON with `key`, `fields.summary`, `fields.status.name` |
| 3 | `jtk issues get ${PROJECT}-99999` | `resource not found: Issue does not exist or you do not have permission to see it.` |

### issues search

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues search --jql "project = $PROJECT" --max 3` | Table with matching issues, at most 3 rows |
| 2 | `jtk issues search --jql "project = $PROJECT" --max 2 -o json` | Valid JSON array |
| 3 | `jtk issues search --jql "project = $PROJECT AND summary ~ 'xyznonexistent999'"` | `No issues found` |
| 4 | `jtk issues search --jql "invalid jql ((("` | `bad request: Error in the JQL Query: ...` |

### Auto-pagination (issues search / issues list)

> These tests require a project with more than 100 issues. If your project has fewer, lower the `--max` value and adjust expected counts accordingly.

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues search --jql "project = $PROJECT" --max 200 -o json \| jq '.issues \| length'` | Number >= 101 (proves multi-page fetch) |
| 2 | `jtk issues search --jql "project = $PROJECT" --max 200 -o json \| jq '.pagination'` | `total` matches count, `pageSize` <= 100 |
| 3 | `jtk issues search --jql "project = $PROJECT" --max 200` | Table output with > 100 rows |
| 4 | `jtk issues list -p $PROJECT --max 200 -o json \| jq '.issues \| length'` | Same multi-page behavior for list |

### `--fields` flag (issues search / issues list)

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues search --jql "project = $PROJECT" --max 1 -o json \| jq '.issues[0].fields \| keys'` | Includes `customfield_*` keys (default `*all`) |
| 2 | `jtk issues search --jql "project = $PROJECT" --max 1 --fields summary,status -o json \| jq '.issues[0].fields \| keys'` | Only `summary`, `status` |
| 3 | `jtk issues list -p $PROJECT --max 1 -o json \| jq '.issues[0].fields \| keys'` | Same `*all` default |
| 4 | `jtk issues list -p $PROJECT --max 1 --fields summary,customfield_10005 -o json \| jq '.issues[0].fields \| keys'` | Only requested fields |

### issues types

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues types -p $PROJECT` | Table with columns: ID, NAME, SUBTASK, DESCRIPTION |
| 2 | `jtk issues types -p $PROJECT -o json` | Valid JSON array of issue type objects |
| 3 | `jtk issues types -p NONEXISTENT` | Error: 404 |

### issues fields

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues fields` | Table with columns: ID, NAME, TYPE, CUSTOM |
| 2 | `jtk issues fields --custom` | Same table but only rows where CUSTOM = yes |
| 3 | `jtk issues fields -o json` | Valid JSON array |

### issues field-options

> `field-options` requires `--issue` for most fields. Without it, the API returns "Field key is not valid".

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues field-options priority --issue $EXISTING_ISSUE` | Table with columns: VALUE, ID (e.g., Highest/1, High/2, Medium/3, Low/4, Lowest/5) |
| 2 | `jtk issues field-options priority --issue $EXISTING_ISSUE -o json` | Valid JSON array |

---

## 3. Projects (Read-Only)

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk projects list --max 5` | Table with columns: KEY, NAME, TYPE, LEAD |
| 2 | `jtk projects list -o json --max 3` | Valid JSON array with at most 3 elements |
| 3 | `jtk projects get $PROJECT` | Shows Key, Name, ID, Type, Lead, Issue Types |
| 4 | `jtk projects get $PROJECT -o json` | Valid JSON with `key`, `name`, `id` |
| 5 | `jtk projects get NONEXISTENT` | `resource not found: No project could be found with key 'NONEXISTENT'.` |
| 6 | `jtk projects types` | Table with columns: KEY, FORMATTED (e.g., software/Software) |
| 7 | `jtk projects types -o json` | Valid JSON array |

---

## 4. Boards & Sprints (Read-Only)

> **Basic Auth only** — Agile endpoints (boards/sprints) are not available with scoped tokens (no Agile scope). Skip this section when testing with Bearer Auth.

### boards

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk boards list --max 5` | Table with columns: ID, NAME, TYPE, PROJECT |
| 2 | `jtk boards list -p $PROJECT` | Only boards for that project |
| 3 | `jtk boards get $BOARD_ID` | Shows ID, Name, Type, Project |
| 4 | `jtk boards get $BOARD_ID -o json` | Valid JSON |
| 5 | `jtk boards get 99999` | Error: 404 (board not found) |

### sprints

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk sprints list -b $BOARD_ID -s active` | Table with columns: ID, NAME, STATE, START, END. State = `active` |
| 2 | `jtk sprints list -b $BOARD_ID -o json` | Valid JSON array |
| 3 | `jtk sprints current -b $BOARD_ID` | Shows ID, Name, State, Start, End |
| 4 | `jtk sprints list` | `Error: required flag(s) "board" not set` |

### sprints issues

> The Jira Agile API endpoint is slow (~30s). Use `--max` to limit results. The client timeout is 60s.

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk sprints issues $SPRINT_ID --max 3` | Table with columns: KEY, SUMMARY, STATUS, ASSIGNEE, TYPE |
| 2 | `jtk sprints issues $SPRINT_ID --max 2 -o json` | Valid JSON array |
| 3 | `jtk sprints issues 99999` | Error |

---

## 5. Links (Read-Only)

### links types

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk links types` | Table with columns: ID, NAME, OUTWARD, INWARD |
| 2 | `jtk links types -o json` | Valid JSON array of link type objects |

### links list

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk links list $EXISTING_ISSUE` | Table with columns: ID, TYPE, DIRECTION, ISSUE, SUMMARY (or `No links on $EXISTING_ISSUE`) |
| 2 | `jtk links list $EXISTING_ISSUE -o json` | Valid JSON array |
| 3 | `jtk links list ${PROJECT}-99999` | `resource not found: ...` |

---

## 6. Dashboards (Read-Only)

> **Basic Auth only** — Dashboard endpoints are not available with scoped tokens (no Dashboard scope). Skip this section when testing with Bearer Auth.

### dashboards list

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk dashboards list --max 5` | Table with columns: ID, NAME, OWNER, FAVOURITE |
| 2 | `jtk dashboards list --search "SEARCH_TERM"` | Filtered results matching search term |
| 3 | `jtk dashboards list -o json --max 3` | Valid JSON array with at most 3 elements |
| 4 | `jtk dashboards list --search "xyznonexistent999"` | `No dashboards found matching "xyznonexistent999"` |

### dashboards get

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk dashboards get $DASHBOARD_ID` | Shows ID, Name, Description, Owner, URL, then Gadgets table (if any) |
| 2 | `jtk dashboards get $DASHBOARD_ID -o json` | Valid JSON with `dashboard` and `gadgets` keys |
| 3 | `jtk dashboards get 99999` | Error: 404 |

### dashboards gadgets list

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk dashboards gadgets list $DASHBOARD_ID` | Table with columns: ID, TITLE, MODULE, POSITION |
| 2 | `jtk dashboards gadgets list $DASHBOARD_ID -o json` | Valid JSON array |

---

## 7. Users (Read-Only)

### users search

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk users search "YOUR_NAME"` | Table with columns: ACCOUNT_ID, NAME, EMAIL, ACTIVE |
| 2 | `jtk users search "YOUR_NAME" -o json` | Valid JSON array |
| 3 | `jtk users search "xyznonexistent999"` | `No users found matching 'xyznonexistent999'` |

### users get

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk users get $ACCOUNT_ID` | Table with Account ID, Display Name, Email, Active |
| 2 | `jtk users get $ACCOUNT_ID -o json` | Valid JSON with `accountId`, `displayName`, `emailAddress`, `active` |
| 3 | `jtk users get 000000000000000000000000` | Error: 404 (user not found) |

---

## 8. Automation (Read-Only)

> **Basic Auth only** — Automation endpoints are not available with scoped tokens (no Automation scope). Skip this section when testing with Bearer Auth.

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk auto list` | Table with columns: UUID, NAME, STATE, LABELS |
| 2 | `jtk auto list --state ENABLED` | Only ENABLED rules |
| 3 | `jtk auto list --state DISABLED` | Only DISABLED rules |
| 4 | `jtk auto list -o json` | Valid JSON array |
| 5 | `jtk auto get $AUTO_UUID` | Shows Name, UUID, State, Description, Components summary |
| 6 | `jtk auto get $AUTO_UUID --show-components` | Adds component details: `[1] CONDITION: type`, `[2] ACTION: type`, etc. |
| 7 | `jtk auto get $AUTO_UUID -o json` | Valid JSON |
| 8 | `jtk auto export $AUTO_UUID \| jq .` | Pretty-printed valid JSON (top-level keys: `rule`, `connections`) |
| 9 | `jtk auto export $AUTO_UUID --compact` | Single-line JSON |

---

## 9. Fields (Read-Only)

### fields list

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk fields list` | Table with columns: ID, NAME, TYPE, CUSTOM |
| 2 | `jtk fields list --custom` | Same table but only rows where CUSTOM = yes |
| 3 | `jtk fields list -o json` | Valid JSON array |
| 4 | `jtk fields list --name "story"` | Table showing only fields with "story" in the name |
| 5 | `jtk fields list --name "story" -o json` | Valid JSON array, filtered to matching fields |
| 6 | `jtk fields list --name "nonexistent"` | `No fields found` |
| 7 | `jtk fields list --name "story" --custom` | Only custom fields matching "story" |

### fields contexts list

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk fields contexts list $CUSTOM_FIELD` | Table with columns: ID, NAME, GLOBAL, ANY_ISSUE_TYPE |
| 2 | `jtk fields contexts list $CUSTOM_FIELD -o json` | Valid JSON array |
| 3 | `jtk fields contexts list customfield_99999` | Error: 404 |

### fields options list

> Options list auto-detects the default context when `--context` is omitted.

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk fields options list $SELECT_FIELD` | Table with columns: ID, VALUE, DISABLED |
| 2 | `jtk fields options list $SELECT_FIELD -o json` | Valid JSON array |

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
   Expected: `✓ Created issue $PROJECT-XXXX` and `URL: https://...`
   Capture the issue key → `$TEST_ISSUE`

3. **Verify creation:**
   ```bash
   jtk issues get $TEST_ISSUE
   ```
   Expected: Shows Key, Summary = `[Test] Integration Test Issue`, Status, Type = `$ISSUE_TYPE`

4. **Update description:**
   ```bash
   jtk issues update $TEST_ISSUE -d "Test description for integration testing"
   ```
   Expected: `✓ Updated issue $TEST_ISSUE`

5. **Assign to self:**
   ```bash
   jtk issues assign $TEST_ISSUE $ACCOUNT_ID
   ```
   Expected: `✓ Assigned issue $TEST_ISSUE to YOUR_NAME`

6. **Add comment with escape sequences:**
   ```bash
   jtk comments add $TEST_ISSUE -b "Line one\nLine two\n\tIndented line"
   ```
   Expected: `✓ Added comment XXXXX to $TEST_ISSUE`
   Capture the comment ID → `$COMMENT_ID`

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

8. **Check transitions** (some workflows require custom fields):
   ```bash
   jtk transitions list $TEST_ISSUE --fields
   ```
   Expected: Table with columns: ID, NAME, TO STATUS, REQUIRED FIELDS
   Note a valid transition name and any required fields

9. **Transition issue:**
   ```bash
   # If no required fields:
   jtk transitions do $TEST_ISSUE "TRANSITION_NAME"
   # If required fields (e.g., Change Type):
   jtk transitions do $TEST_ISSUE "TRANSITION_NAME" -f customfield_10005=Feature
   ```
   Expected: `✓ Transitioned $TEST_ISSUE`

10. **Verify transition:**
    ```bash
    jtk issues get $TEST_ISSUE
    ```
    Expected: Status shows the new value

11. **Unassign (via assign command):**
    ```bash
    jtk issues assign $TEST_ISSUE --unassign
    ```
    Expected: `✓ Unassigned issue $TEST_ISSUE`

11b. **Re-assign, then unassign via update --assignee none:**
    ```bash
    jtk issues assign $TEST_ISSUE $ACCOUNT_ID
    jtk issues update $TEST_ISSUE --assignee none
    ```
    Expected: First command assigns, second command shows `✓ Updated issue $TEST_ISSUE`

11c. **Verify unassignment:**
    ```bash
    jtk issues get $TEST_ISSUE -o json | jq '.fields.assignee'
    ```
    Expected: `null`

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

### Multi-value `--field` flag

> Requires a multi-select or multi-checkbox custom field (`$MULTI_FIELD`) on the project. Skip if unavailable.

1. **Create issue with multi-value field:**
   ```bash
   jtk issues create -p $PROJECT -t $ISSUE_TYPE -s "[Test] Multi-Value Field" \
     --field $MULTI_FIELD=Option1 --field $MULTI_FIELD=Option2
   ```
   Expected: `✓ Created issue $PROJECT-XXXX`
   Capture the issue key → `$MV_ISSUE`

2. **Verify both values set:**
   ```bash
   jtk issues get $MV_ISSUE -o json | jq ".fields.$MULTI_FIELD"
   ```
   Expected: JSON array containing both Option1 and Option2

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
   Expected: `Created $LINK_TYPE link: $LINK_SOURCE → $LINK_TARGET`

4. **Verify link:**
   ```bash
   jtk links list $LINK_SOURCE
   ```
   Expected: Table shows link to `$LINK_TARGET` with type `$LINK_TYPE`
   Capture the link ID → `$LINK_ID`

5. **Verify JSON output:**
   ```bash
   jtk links list $LINK_SOURCE -o json
   ```
   Expected: Valid JSON array with link object

6. **Delete link:**
   ```bash
   jtk links delete $LINK_ID
   ```
   Expected: `Deleted link $LINK_ID`

7. **Verify deletion:**
   ```bash
   jtk links list $LINK_SOURCE
   ```
   Expected: No link to `$LINK_TARGET` (or `No links on $LINK_SOURCE`)

8. **Cleanup:**
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
   Expected: `✓ Created project ZTEST (Integration Test Project)`

2. **Verify creation:**
   ```bash
   jtk projects get ZTEST
   ```
   Expected: Key = ZTEST, Name = Integration Test Project

3. **Update name:**
   ```bash
   jtk projects update ZTEST --name "Updated Test Project"
   ```
   Expected: `✓ Updated project ZTEST`

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
   Expected: `✓ Restored project ZTEST (Updated Test Project)`

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

5. **Delete:**
   ```bash
   jtk dashboards delete $TEST_DASH_ID
   ```
   Expected: `Deleted dashboard $TEST_DASH_ID`

6. **Verify deletion:**
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
   Expected: `✓ Created automation rule (UUID: XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX)`
   Capture the UUID → `$TEST_AUTO_UUID`

4. **Verify creation:**
   ```bash
   jtk auto get $TEST_AUTO_UUID
   ```
   Expected: Name = `[Test] Auto Integration Copy`, same component count as source

### Toggle cycle

5. **Disable:**
   ```bash
   jtk auto disable $TEST_AUTO_UUID
   ```
   Expected: `✓ Rule "[Test] Auto Integration Copy": ENABLED → DISABLED`

6. **Re-enable:**
   ```bash
   jtk auto enable $TEST_AUTO_UUID
   ```
   Expected: `✓ Rule "[Test] Auto Integration Copy": DISABLED → ENABLED`

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
   Expected: `✓ Updated automation rule $TEST_AUTO_UUID`

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
   Expected: `✓ Moved 1 issue(s) to sprint $SPRINT_ID`

3. **Verify** (may be slow):
   ```bash
   jtk sprints issues $SPRINT_ID --max 50 -o json | jq -r '.[].key' | grep $SPRINT_TEST_ISSUE
   ```
   Expected: Issue key appears in output

4. **Cleanup:**
   ```bash
   jtk issues delete $SPRINT_TEST_ISSUE --force
   ```

---

## 16. Field Mutations

Run these steps in order. Each step depends on the previous.

> Field management requires "Administer Jira" global permission. If you get 403 errors, verify your account has this permission.

### Create and manage a test field

1. **Create a select field:**
   ```bash
   jtk fields create --name "[Test] Integration Select" --type com.atlassian.jira.plugin.system.customfieldtypes:select
   ```
   Expected: `✓ Created field customfield_XXXXX ([Test] Integration Select)`
   Capture the field ID → `$TEST_FIELD`

2. **Verify creation:**
   ```bash
   jtk fields list --name "[Test] Integration Select"
   ```
   Expected: Table showing the newly created field

   ```bash
   jtk fields list --custom -o json | jq '.[] | select(.name == "[Test] Integration Select")'
   ```
   Expected: JSON object with matching `name` and `id`

3. **List contexts:**
   ```bash
   jtk fields contexts list $TEST_FIELD
   ```
   Expected: Table showing the default context. Capture context ID → `$TEST_CTX`

4. **Add options:**
   ```bash
   jtk fields options add $TEST_FIELD --value "Option A"
   ```
   Expected: `✓ Added option XXXXX (Option A)`
   ```bash
   jtk fields options add $TEST_FIELD --value "Option B"
   ```
   Expected: `✓ Added option XXXXX (Option B)`

5. **List options:**
   ```bash
   jtk fields options list $TEST_FIELD
   ```
   Expected: Table showing Option A and Option B
   Capture an option ID → `$OPT_ID`

6. **Update option:**
   ```bash
   jtk fields options update $TEST_FIELD --option $OPT_ID --value "Option A (updated)"
   ```
   Expected: `✓ Updated option $OPT_ID`

7. **Verify update:**
   ```bash
   jtk fields options list $TEST_FIELD
   ```
   Expected: Shows "Option A (updated)" instead of "Option A"

8. **Delete option:**
   ```bash
   jtk fields options delete $TEST_FIELD --option $OPT_ID --force
   ```
   Expected: `✓ Deleted option $OPT_ID from field $TEST_FIELD`

9. **Create context:**
   ```bash
   jtk fields contexts create $TEST_FIELD --name "[Test] Context"
   ```
   Expected: `✓ Created context XXXXX ([Test] Context)`
   Capture context ID → `$NEW_CTX`

10. **Delete context:**
    ```bash
    jtk fields contexts delete $TEST_FIELD $NEW_CTX --force
    ```
    Expected: `✓ Deleted context $NEW_CTX from field $TEST_FIELD`

11. **Trash field:**
    ```bash
    jtk fields delete $TEST_FIELD --force
    ```
    Expected: `✓ Trashed field $TEST_FIELD`

12. **Restore field:**
    ```bash
    jtk fields restore $TEST_FIELD
    ```
    Expected: `✓ Restored field $TEST_FIELD`

13. **Final cleanup — trash again:**
    ```bash
    jtk fields delete $TEST_FIELD --force
    ```
    Expected: `✓ Trashed field $TEST_FIELD`

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
| 3 | `jtk issues list -p $PROJECT --max 1 -o json \| jq .` | Parses without errors |
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
| 18 | `jtk dashboards gadgets remove 1 1` | Same Dashboard error |

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
- [ ] `config show` (table, JSON)
- [ ] `config test`
- [ ] `me` (table, JSON, plain)

#### Issues Read-Only (Section 2)
- [ ] `issues list` (table, JSON, plain, error)
- [ ] `issues get` (table, JSON, 404)
- [ ] `issues search` (results, JSON, no results, bad JQL)
- [ ] Auto-pagination (search multi-page, list multi-page)
- [ ] `--fields` flag (default `*all`, explicit fields for search and list)
- [ ] `issues types` (table, JSON, 404)
- [ ] `issues fields` (all, custom, JSON)
- [ ] `issues field-options` (with --issue, JSON)

#### Projects Read-Only (Section 3)
- [ ] `projects list` (table, JSON)
- [ ] `projects get` (table, JSON, 404)
- [ ] `projects types` (table, JSON)

#### Boards & Sprints Read-Only (Section 4)
- [ ] `boards list`, `boards get` (table, JSON, 404)
- [ ] `sprints list`, `sprints current`
- [ ] `sprints issues` (table, JSON)

#### Links Read-Only (Section 5)
- [ ] `links types` (table, JSON)
- [ ] `links list` (table, JSON, 404)

#### Dashboards Read-Only (Section 6)
- [ ] `dashboards list` (table, search, JSON, no results)
- [ ] `dashboards get` (table, JSON, 404)
- [ ] `dashboards gadgets list` (table, JSON)

#### Users Read-Only (Section 7)
- [ ] `users search` (results, JSON, no results)
- [ ] `users get` (table, JSON, 404)

#### Automation Read-Only (Section 8)
- [ ] `auto list` (all, filtered, JSON)
- [ ] `auto get` (summary, --show-components, JSON)
- [ ] `auto export` (pretty, compact)

#### Fields Read-Only (Section 9)
- [ ] `fields list` (all, custom, JSON)
- [ ] `fields contexts list` (table, JSON, 404)
- [ ] `fields options list` (table, JSON)

#### Issue Mutations (Section 10)
- [ ] Create → get → update → assign → comment (with escape sequences) → transition → unassign → delete comment → delete issue
- [ ] Unassign via `--assignee none` on `issues update`
- [ ] Multi-value `--field` flag (create issue with repeated `--field` same key)
- [ ] Error cases (missing flags, 404)

#### Link Mutations (Section 11)
- [ ] Types → create issues → create link → verify → delete link → verify → cleanup
- [ ] Error cases (nonexistent target, invalid type, delete 404)

#### Project Mutations (Section 12)
- [ ] Create → get → update → delete → restore → verify → delete (cleanup)
- [ ] Error cases

#### Dashboard Mutations (Section 13)
- [ ] Create → verify → list+search → gadgets list → delete → verify 404
- [ ] Error cases (missing flags, 404)

#### Automation Mutations (Section 14)
- [ ] Create copy (strip UUID, rename)
- [ ] Toggle cycle (disable, enable, idempotent)
- [ ] Round-trip update
- [ ] Cleanup (`jtk auto delete`)
- [ ] Error cases

#### Sprint Mutations (Section 15)
- [ ] Create issue → add to sprint → verify → delete issue

#### Field Mutations (Section 16)
- [ ] Create field → list contexts → add options → update option → delete option
- [ ] Create context → delete context
- [ ] Trash field → restore → trash again (cleanup)
- [ ] Error cases (missing flags, 404)

#### Global Flags & Aliases (Section 17)
- [ ] `--no-color`, `--verbose`, `-o json`, `-o plain`
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
- [ ] `me` (table, JSON, plain)

#### Issues Read-Only (Section 2)
- [ ] `issues list` (table, JSON, plain, error)
- [ ] `issues get` (table, JSON, 404)
- [ ] `issues search` (results, JSON, no results, bad JQL)
- [ ] Auto-pagination (search multi-page, list multi-page)
- [ ] `--fields` flag (default `*all`, explicit fields for search and list)
- [ ] `issues types` (table, JSON, 404)
- [ ] `issues fields` (all, custom, JSON)
- [ ] `issues field-options` (with --issue, JSON)

#### Projects Read-Only (Section 3)
- [ ] `projects list` (table, JSON)
- [ ] `projects get` (table, JSON, 404)
- [ ] `projects types` (table, JSON)

#### Links Read-Only (Section 5)
- [ ] `links types` (table, JSON)
- [ ] `links list` (table, JSON, 404)

#### Users Read-Only (Section 7)
- [ ] `users search` (results, JSON, no results)
- [ ] `users get` (table, JSON, 404)

#### Fields Read-Only (Section 9)
- [ ] `fields list` (all, custom, JSON)
- [ ] `fields contexts list` (table, JSON, 404)
- [ ] `fields options list` (table, JSON)

#### Issue Mutations (Section 10)
- [ ] Create → get → update → assign → comment (with escape sequences) → transition → unassign → delete comment → delete issue
- [ ] Unassign via `--assignee none` on `issues update`
- [ ] Multi-value `--field` flag (create issue with repeated `--field` same key)
- [ ] Error cases (missing flags, 404)

#### Link Mutations (Section 11)
- [ ] Types → create issues → create link → verify → delete link → verify → cleanup
- [ ] Error cases

#### Project Mutations (Section 12)
- [ ] Create → get → update → delete → restore → verify → delete (cleanup)
- [ ] Error cases

#### Field Mutations (Section 16)
- [ ] Create field → list contexts → add options → update option → delete option
- [ ] Create context → delete context
- [ ] Trash field → restore → trash again (cleanup)
- [ ] Error cases

#### Bearer Auth Guards (Section 19)
- [ ] Boards: `list`, `get 1` → Agile scope error
- [ ] Sprints: `list -b 1`, `current -b 1`, `issues 1`, `add 1 PROJ-1` → Agile scope error
- [ ] Automation: `list`, `get`, `export`, `create`, `enable`, `disable` → Automation scope error
- [ ] Dashboards: `list`, `get`, `create`, `delete`, `gadgets list`, `gadgets remove` → Dashboard scope error

#### Global Flags & Aliases (Section 17)
- [ ] `--no-color`, `--verbose`, `-o json`, `-o plain`
- [ ] Applicable aliases (skip `jtk b`, `jtk sp`, `jtk auto`, `jtk dash`, `jtk dashboard`)

#### Error Cases (Section 18)
- [ ] All applicable error cases (skip rows 5 and 8: boards get and dashboards get)

#### Cleanup (Bearer Auth)
- [ ] Delete test projects: `jtk projects delete ZTEST --force`
- [ ] Delete test issues: search for `[Test]` prefix, delete with `--force`
- [ ] Trash test fields: `jtk fields delete $TEST_FIELD --force`

---

## Adding New Tests

When adding new features or fixing bugs:

1. Add test steps to the appropriate numbered section above
2. Include both happy path and error cases with exact expected output
3. Document gotchas inline, immediately before the step where they matter
4. Update both Pass 1 and Pass 2 in the Test Execution Checklist
5. If the feature is scope-restricted, add guard tests to Section 19
6. Record bugs discovered during testing and continue — don't stop to fix
