# Integration Tests

This document is a concrete, sequential runbook for testing `jtk` against a live Jira instance. Run read-only tests first, then mutations, then cleanup.

If a test reveals a bug, **record the bug and continue testing** rather than stopping to fix it.

## Test Environment Setup

### Prerequisites
- A configured `jtk` instance (`jtk init` completed)
- Access to a project with permission to create, edit, and delete issues
- At least one agile board with an active sprint
- At least one ENABLED and one DISABLED automation rule
- At least one automation rule with multiple components (trigger + conditions + actions)

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

# $BOARD_ID — find a board for your project
jtk boards list -p $PROJECT
# Note the ID column, e.g., 23

# $SPRINT_ID — find the active sprint
jtk sprints list -b $BOARD_ID -s active
# Note the ID column, e.g., 119

# $AUTO_UUID — pick an enabled automation rule
jtk auto list --state ENABLED
# Note a UUID from the first column

# $CUSTOM_FIELD — pick a custom field ID
jtk fields list --custom
# Note an ID, e.g., customfield_10001

# $SELECT_FIELD — pick a select/multiselect custom field with options
# (same as $CUSTOM_FIELD if it's a select type)
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

## 5. Users (Read-Only)

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk users search "YOUR_NAME"` | Table with columns: ACCOUNT_ID, NAME, EMAIL, ACTIVE |
| 2 | `jtk users search "YOUR_NAME" -o json` | Valid JSON array |
| 3 | `jtk users search "xyznonexistent999"` | `No users found matching 'xyznonexistent999'` |

---

## 6. Automation (Read-Only)

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk auto list` | Table with columns: UUID, NAME, STATE, LABELS |
| 2 | `jtk auto list --state ENABLED` | Only ENABLED rules |
| 3 | `jtk auto list --state DISABLED` | Only DISABLED rules |
| 4 | `jtk auto list -o json` | Valid JSON array |
| 5 | `jtk auto get $AUTO_UUID` | Shows Name, UUID, State, Description, Components summary |
| 6 | `jtk auto get $AUTO_UUID --full` | Adds component details: `[1] CONDITION: type`, `[2] ACTION: type`, etc. |
| 7 | `jtk auto get $AUTO_UUID -o json` | Valid JSON |
| 8 | `jtk auto export $AUTO_UUID \| jq .` | Pretty-printed valid JSON (top-level keys: `rule`, `connections`) |
| 9 | `jtk auto export $AUTO_UUID --compact` | Single-line JSON |

---

## 7. Issue Mutations

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

6. **Add comment:**
   ```bash
   jtk comments add $TEST_ISSUE -b "Test comment from integration testing"
   ```
   Expected: `✓ Added comment XXXXX to $TEST_ISSUE`
   Capture the comment ID → `$COMMENT_ID`

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

11. **Unassign:**
    ```bash
    jtk issues assign $TEST_ISSUE --unassign
    ```
    Expected: `✓ Unassigned issue $TEST_ISSUE`

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

### Error cases

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues create -p $PROJECT` | `Error: required flag(s) "summary" not set` |
| 2 | `jtk issues create -s "No Project"` | `Error: required flag(s) "project" not set` |
| 3 | `jtk issues get ${PROJECT}-99999` | `resource not found: ...` |
| 4 | `jtk issues update ${PROJECT}-99999 -s "Nope"` | `resource not found: ...` |
| 5 | `jtk issues delete ${PROJECT}-99999 --force` | `resource not found: ...` |

---

## 8. Project Mutations

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

## 9. Automation Mutations

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

11. **Disable and rename for manual deletion:**
    ```bash
    jtk auto disable $TEST_AUTO_UUID
    jq '.rule.name = "[DELETEME] Auto Integration Copy"' /tmp/auto-rt.json > /tmp/auto-deleteme.json
    jtk auto update $TEST_AUTO_UUID --file /tmp/auto-deleteme.json
    ```
    Expected: Rule disabled and renamed

### Error cases

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk auto create` | `Error: required flag(s) "file" not set` |
| 2 | `echo "not json" > /tmp/bad.json && jtk auto create --file /tmp/bad.json` | Error: does not contain valid JSON |
| 3 | `jtk auto create --file /tmp/nope.json` | Error: failed to read file |
| 4 | `jtk auto enable 99999999` | Error |

---

## 10. Sprint Mutations

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

## 11. Global Flags & Aliases

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

### Shell completion

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk completion bash \| head -3` | Starts with `# bash completion for jtk` |
| 2 | `jtk completion zsh \| head -3` | Valid zsh completion script |

---

## 12. Error Cases

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk issues get ${PROJECT}-99999` | `resource not found: Issue does not exist or you do not have permission to see it.` |
| 2 | `jtk issues search --jql "invalid jql ((("` | `bad request: Error in the JQL Query: ...` |
| 3 | `jtk issues create -p $PROJECT` | `Error: required flag(s) "summary" not set` |
| 4 | `jtk projects get NONEXISTENT` | `resource not found: No project could be found with key 'NONEXISTENT'.` |
| 5 | `jtk boards get 99999` | Error: 404 |
| 6 | `jtk sprints list` | `Error: required flag(s) "board" not set` |

---

## 13. Fields (Read-Only)

### fields list

| # | Command | Expected Output |
|---|---------|-----------------|
| 1 | `jtk fields list` | Table with columns: ID, NAME, TYPE, CUSTOM |
| 2 | `jtk fields list --custom` | Same table but only rows where CUSTOM = yes |
| 3 | `jtk fields list -o json` | Valid JSON array |

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

## 14. Field Mutations

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

## Test Execution Checklist

### Setup
- [ ] `make build-jtk`
- [ ] `jtk me` works
- [ ] Discover: `$PROJECT`, `$BOARD_ID`, `$SPRINT_ID`, `$ACCOUNT_ID`, `$AUTO_UUID`, `$EXISTING_ISSUE`, `$CUSTOM_FIELD`, `$SELECT_FIELD`
- [ ] `jtk issues types -p $PROJECT` to learn `$ISSUE_TYPE`

### Config & Init (Section 1)
- [ ] `config show` (table, JSON)
- [ ] `config test`
- [ ] `me` (table, JSON, plain)

### Issues Read-Only (Section 2)
- [ ] `issues list` (table, JSON, plain, error)
- [ ] `issues get` (table, JSON, 404)
- [ ] `issues search` (results, JSON, no results, bad JQL)
- [ ] `issues types` (table, JSON, 404)
- [ ] `issues fields` (all, custom, JSON)
- [ ] `issues field-options` (with --issue, JSON)

### Projects Read-Only (Section 3)
- [ ] `projects list` (table, JSON)
- [ ] `projects get` (table, JSON, 404)
- [ ] `projects types` (table, JSON)

### Boards & Sprints Read-Only (Section 4)
- [ ] `boards list`, `boards get` (table, JSON, 404)
- [ ] `sprints list`, `sprints current`
- [ ] `sprints issues` (table, JSON)

### Users Read-Only (Section 5)
- [ ] `users search` (results, JSON, no results)

### Automation Read-Only (Section 6)
- [ ] `auto list` (all, filtered, JSON)
- [ ] `auto get` (summary, --full, JSON)
- [ ] `auto export` (pretty, compact)

### Issue Mutations (Section 7)
- [ ] Create → get → update → assign → comment → transition → unassign → delete comment → delete issue
- [ ] Error cases (missing flags, 404)

### Project Mutations (Section 8)
- [ ] Create → get → update → delete → restore → verify → delete (cleanup)
- [ ] Error cases

### Automation Mutations (Section 9)
- [ ] Create copy (strip UUID, rename)
- [ ] Toggle cycle (disable, enable, idempotent)
- [ ] Round-trip update
- [ ] Cleanup (disable + rename to DELETEME)
- [ ] Error cases

### Sprint Mutations (Section 10)
- [ ] Create issue → add to sprint → verify → delete issue

### Global Flags & Aliases (Section 11)
- [ ] `--no-color`, `--verbose`, `-o json`, `-o plain`
- [ ] All aliases verified (including `jtk f`, `jtk field`)

### Error Cases (Section 12)
- [ ] 404, bad JQL, missing flags

### Fields Read-Only (Section 13)
- [ ] `fields list` (all, custom, JSON)
- [ ] `fields contexts list` (table, JSON, 404)
- [ ] `fields options list` (table, JSON)

### Field Mutations (Section 14)
- [ ] Create field → list contexts → add options → update option → delete option
- [ ] Create context → delete context
- [ ] Trash field → restore → trash again (cleanup)
- [ ] Error cases (missing flags, 404)

### Cleanup
- [ ] Delete test projects: `jtk projects delete ZTEST --force` (etc.)
- [ ] Delete test issues: search for `[Test]` prefix, delete with `--force`
- [ ] Trash test fields: `jtk fields delete $TEST_FIELD --force`
- [ ] Disable + rename automation test copies to `[DELETEME]`
- [ ] Manually purge `[DELETEME]` rules in Jira UI (Settings → System → Automation rules)
- [ ] Verify: `jtk auto list -o json | jq '.[] | select(.name | startswith("[Test]") or startswith("[DELETEME]"))'`

---

## Adding New Tests

When adding new features or fixing bugs:

1. Add test steps to the appropriate numbered section above
2. Include both happy path and error cases with exact expected output
3. Document gotchas inline, immediately before the step where they matter
4. Update the Test Execution Checklist
5. Record bugs discovered during testing and continue — don't stop to fix
