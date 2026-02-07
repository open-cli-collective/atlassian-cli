# Integration Tests

This document catalogs the manual integration test suite for `jtk`. These tests verify real-world behavior against a live Jira instance and catch edge cases that are difficult to cover with unit tests.

## Test Environment Setup

### Prerequisites
- A configured `jtk` instance (`jtk init` completed)
- Access to a test project (e.g., `TEST`) with permission to create, edit, and delete issues
- At least one agile board with an active sprint
- At least one ENABLED and one DISABLED automation rule
- At least one automation rule with multiple components (trigger + conditions + actions)

### Test Data Conventions
- Test issues use `[Test]` prefix: `[Test] My Issue`
- Test automation copies use `[Test]` prefix in the rule name
- Always clean up test data after tests complete
- Run read-only tests first, then mutation tests, then cleanup
- If a test reveals a bug, **record the bug and continue testing** rather than stopping to fix it

---

## Init

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Fresh init | `jtk init` (interactive) | Creates ~/.config/jtk/config.json with URL, email, token |
| Init with flags | `jtk init --url https://x.atlassian.net --email a@b.com --token tok` | Config created non-interactively |
| Init with --no-verify | `jtk init --url https://x.atlassian.net --email a@b.com --token tok --no-verify` | Config saved without testing connection |
| Init with existing config | `jtk init` when config exists | Prompts to overwrite or skip |
| Verify connection | After init, run `jtk me` | Connection works, user info shown |
| Invalid credentials | Init with bad API token, then `jtk me` | Error: 401 |
| Invalid URL | Init with malformed URL | Error during verification |

---

## Config Operations

### config show

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Show configuration | `jtk config show` | Table with KEY, VALUE, SOURCE columns |
| API token masked | `jtk config show` | Token shown as `****...` |
| JSON output | `jtk config show -o json` | Valid JSON object |
| Shows env var source | Set `JIRA_URL` env var, then `jtk config show` | URL source shows "env" |

### config test

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Test connection | `jtk config test` | Shows authentication status, user name, account ID |
| Bad credentials | Temporarily set `JIRA_API_TOKEN=bad`, then `jtk config test` | Error: 401 |

### config clear

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Clear with confirmation | `jtk config clear` (type "y") | Config file removed |
| Clear cancelled | `jtk config clear` (type "n") | Config file preserved |
| Clear with --force | `jtk config clear --force` | Config removed without prompt |
| Shows active env vars | `jtk config clear --force` with `ATLASSIAN_*` set | Lists active env variables |

---

## Me

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Show current user | `jtk me` | Shows Account ID, Display Name, Email, Active |
| JSON output | `jtk me -o json` | Full user object as JSON |
| Plain output | `jtk me -o plain` | Account ID only |

---

## Issue Operations

### issues list

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| List issues in project | `jtk issues list --project TEST` | Table with KEY, SUMMARY, STATUS, ASSIGNEE, TYPE |
| Limit results | `jtk issues list --project TEST --max 5` | At most 5 issues shown |
| Filter by sprint | `jtk issues list --project TEST --sprint current` | Only issues in active sprint |
| JSON output | `jtk issues list --project TEST -o json` | Valid JSON array |
| Plain output | `jtk issues list --project TEST -o plain` | Tab-separated values |
| No results | `jtk issues list --project NONEXISTENT` | Error or empty results |
| Default max (50) | `jtk issues list --project TEST` (>50 issues) | Shows 50 issues |

### issues get

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Get issue details | `jtk issues get TEST-1` | Shows Key, Summary, Status, Type, Priority, Assignee, Description, URL |
| Get with --full | `jtk issues get TEST-1 --full` | Full description without truncation |
| Truncated description | `jtk issues get <long-desc-issue>` (without --full) | Description truncated with indicator |
| JSON output | `jtk issues get TEST-1 -o json` | Full issue object as JSON |
| Non-existent issue | `jtk issues get TEST-99999` | Error: 404 |

### issues create

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Create basic issue | `jtk issues create -p TEST -s "[Test] Basic Task"` | Issue created, shows key and URL |
| Create with type | `jtk issues create -p TEST -t Bug -s "[Test] Bug Report"` | Bug type issue created |
| Create with description | `jtk issues create -p TEST -s "[Test] Described" -d "Some details"` | Issue created with description |
| Create with custom field | `jtk issues create -p TEST -s "[Test] Custom" -f priority=High` | Issue created with custom field value |
| Default type is Task | `jtk issues create -p TEST -s "[Test] Default Type"` | Issue type is "Task" |
| Missing project | `jtk issues create -s "No Project"` | Error: project required |
| Missing summary | `jtk issues create -p TEST` | Error: summary required |
| JSON output | `jtk issues create -p TEST -s "[Test] JSON" -o json` | Created issue as JSON |

### issues update

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Update summary | `jtk issues update TEST-X -s "[Test] Updated Summary"` | Success message |
| Update description | `jtk issues update TEST-X -d "New description"` | Success message |
| Update custom field | `jtk issues update TEST-X -f priority=Low` | Field updated |
| Verify update | `jtk issues get TEST-X` | Shows updated values |
| Non-existent issue | `jtk issues update TEST-99999 -s "Nope"` | Error: 404 |

### issues delete

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Delete with confirmation | `jtk issues delete TEST-X` (type "y") | Issue deleted |
| Delete cancelled | `jtk issues delete TEST-X` (type "n") | "Deletion cancelled" |
| Delete with --force | `jtk issues delete TEST-X --force` | Deleted without confirmation |
| Non-existent issue | `jtk issues delete TEST-99999 --force` | Error: 404 |

### issues assign

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Assign to user | `jtk issues assign TEST-X <account-id>` | Success: assigned to user name |
| Unassign | `jtk issues assign TEST-X --unassign` | Success: unassigned |
| Verify assignment | `jtk issues get TEST-X` | Assignee shows expected user |
| Non-existent issue | `jtk issues assign TEST-99999 abc123` | Error: 404 |

### issues search

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Search with JQL | `jtk issues search --jql "project = TEST AND status = 'To Do'"` | Matching issues in table |
| Limit results | `jtk issues search --jql "project = TEST" --max 3` | At most 3 results |
| JSON output | `jtk issues search --jql "project = TEST" -o json` | Valid JSON array |
| No results | `jtk issues search --jql "project = TEST AND summary ~ 'xyznonexistent'"` | Empty result / "No issues found" |
| Invalid JQL | `jtk issues search --jql "invalid jql ((("` | Error from API |

### issues fields

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| List all fields | `jtk issues fields` | Table with ID, NAME, TYPE, CUSTOM columns |
| List for specific issue | `jtk issues fields TEST-1` | Editable fields for that issue |
| Custom fields only | `jtk issues fields --custom` | Only custom fields shown |
| JSON output | `jtk issues fields -o json` | Valid JSON array |

### issues field-options

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| List field options | `jtk issues field-options priority` | Table with VALUE, ID |
| With issue context | `jtk issues field-options priority --issue TEST-1` | Context-specific options |
| JSON output | `jtk issues field-options priority -o json` | Valid JSON array |
| Invalid field | `jtk issues field-options nonexistent-field-xyz` | Error |

### issues types

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| List issue types | `jtk issues types -p TEST` | Table with ID, NAME, SUBTASK, DESCRIPTION |
| JSON output | `jtk issues types -p TEST -o json` | Valid JSON array |
| Missing project | `jtk issues types` | Error: project required |
| Invalid project | `jtk issues types -p NONEXISTENT` | Error: 404 or empty |

### issues move

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Move single issue | `jtk issues move TEST-X --to-project DEST` | "Moved 1 issue(s) to DEST" |
| Move multiple issues | `jtk issues move TEST-X TEST-Y --to-project DEST` | "Moved 2 issue(s) to DEST" |
| Move with type change | `jtk issues move TEST-X --to-project DEST --to-type Bug` | Issue moved and retyped |
| Move without notifications | `jtk issues move TEST-X --to-project DEST --notify=false` | Moved without sending notifications |
| Async move | `jtk issues move TEST-X --to-project DEST --wait=false` | Returns task ID |
| Non-existent issue | `jtk issues move TEST-99999 --to-project DEST` | Error |
| Non-existent target project | `jtk issues move TEST-X --to-project NONEXISTENT` | Error |

### issues move-status

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Check completed move | `jtk issues move-status <task-id>` | Shows Status, Progress, Successful/Failed keys |
| JSON output | `jtk issues move-status <task-id> -o json` | Full MoveTaskStatus as JSON |
| Invalid task ID | `jtk issues move-status 99999` | Error |

---

## Transition Operations

### transitions list

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| List transitions | `jtk transitions list TEST-X` | Table with ID, NAME, TO STATUS |
| List with fields | `jtk transitions list TEST-X --fields` | Adds REQUIRED FIELDS column |
| JSON output | `jtk transitions list TEST-X -o json` | Valid JSON array with transition objects |
| Non-existent issue | `jtk transitions list TEST-99999` | Error: 404 |

### transitions do

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Transition by name | `jtk transitions do TEST-X "In Progress"` | Success message |
| Verify transition | `jtk issues get TEST-X` | Status shows new value |
| Transition by ID | `jtk transitions do TEST-X 31` | Success message |
| Transition with field | `jtk transitions do TEST-X "Done" -f resolution=Done` | Transition with required field |
| Invalid transition | `jtk transitions do TEST-X "Nonexistent"` | Error: transition not found |
| Non-existent issue | `jtk transitions do TEST-99999 "Done"` | Error: 404 |

---

## Comment Operations

### comments list

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| List comments | `jtk comments list TEST-X` | Table with ID, AUTHOR, CREATED, BODY |
| List with --full | `jtk comments list TEST-X --full` | Full comment bodies without truncation |
| Truncated bodies | `jtk comments list TEST-X` (long comments) | Bodies truncated |
| Limit results | `jtk comments list TEST-X --max 2` | At most 2 comments |
| JSON output | `jtk comments list TEST-X -o json` | Valid JSON array |
| No comments | `jtk comments list <issue-with-no-comments>` | "No comments found" or empty table |
| Non-existent issue | `jtk comments list TEST-99999` | Error: 404 |

### comments add

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Add comment | `jtk comments add TEST-X -b "Test comment"` | Success with comment ID |
| Verify comment | `jtk comments list TEST-X` | New comment appears |
| Missing body | `jtk comments add TEST-X` | Error: body required |
| Non-existent issue | `jtk comments add TEST-99999 -b "Nope"` | Error: 404 |

### comments delete

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Delete comment | `jtk comments delete TEST-X <comment-id>` | Success message |
| Verify deletion | `jtk comments list TEST-X` | Comment no longer appears |
| Non-existent comment | `jtk comments delete TEST-X 99999999` | Error |

---

## Attachment Operations

### attachments list

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| List attachments | `jtk attachments list TEST-X` | Table with ID, FILENAME, SIZE, CREATED, AUTHOR |
| No attachments | `jtk attachments list <issue-with-no-attachments>` | "No attachments found" or empty table |
| JSON output | `jtk attachments list TEST-X -o json` | Valid JSON array |
| Alias: ls | `jtk attachments ls TEST-X` | Same as `list` |
| Non-existent issue | `jtk attachments list TEST-99999` | Error: 404 |

### attachments add

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Upload single file | `jtk attachments add TEST-X -f test.txt` | Success with filename and size |
| Upload multiple files | `jtk attachments add TEST-X -f a.txt -f b.txt` | Success for each file |
| Verify upload | `jtk attachments list TEST-X` | New attachment(s) appear |
| Non-existent file | `jtk attachments add TEST-X -f /nonexistent.txt` | Error: file not found |
| Missing --file | `jtk attachments add TEST-X` | Error: file required |
| Non-existent issue | `jtk attachments add TEST-99999 -f test.txt` | Error: 404 |

### attachments get

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Download to current dir | `jtk attachments get <att-id>` | File saved with original filename |
| Download to specific path | `jtk attachments get <att-id> -o /tmp/` | File saved to specified path |
| Verify content integrity | Upload then download, compare | Files match exactly |
| Alias: download | `jtk attachments download <att-id>` | Same as `get` |
| Non-existent attachment | `jtk attachments get 99999` | Error |

### attachments delete

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Delete attachment | `jtk attachments delete <att-id>` | Success message |
| Verify deletion | `jtk attachments list TEST-X` | Attachment no longer appears |
| Alias: rm | `jtk attachments rm <att-id>` | Same as `delete` |
| Non-existent attachment | `jtk attachments delete 99999` | Error |

---

## Board Operations

### boards list

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| List all boards | `jtk boards list` | Table with ID, NAME, TYPE, PROJECT |
| Filter by project | `jtk boards list -p TEST` | Only boards for TEST project |
| Limit results | `jtk boards list --max 3` | At most 3 boards |
| JSON output | `jtk boards list -o json` | Valid JSON array |

### boards get

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Get board details | `jtk boards get <board-id>` | Shows ID, Name, Type, Project |
| JSON output | `jtk boards get <board-id> -o json` | Full board object as JSON |
| Non-existent board | `jtk boards get 99999` | Error: 404 |

---

## Sprint Operations

### sprints list

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| List sprints | `jtk sprints list -b <board-id>` | Table with ID, NAME, STATE, START, END |
| Filter active | `jtk sprints list -b <board-id> -s active` | Only active sprints |
| Filter closed | `jtk sprints list -b <board-id> -s closed` | Only closed sprints |
| Filter future | `jtk sprints list -b <board-id> -s future` | Only future sprints |
| Limit results | `jtk sprints list -b <board-id> --max 3` | At most 3 sprints |
| JSON output | `jtk sprints list -b <board-id> -o json` | Valid JSON array |
| Missing board | `jtk sprints list` | Error: board required |

### sprints current

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Show current sprint | `jtk sprints current -b <board-id>` | Shows ID, Name, State, Start, End, Goal |
| JSON output | `jtk sprints current -b <board-id> -o json` | Sprint object as JSON |
| No active sprint | `jtk sprints current -b <board-with-no-active-sprint>` | Error or "no active sprint" |
| Missing board | `jtk sprints current` | Error: board required |

### sprints issues

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| List sprint issues | `jtk sprints issues <sprint-id>` | Table with KEY, SUMMARY, STATUS, ASSIGNEE, TYPE |
| Limit results | `jtk sprints issues <sprint-id> --max 5` | At most 5 issues |
| JSON output | `jtk sprints issues <sprint-id> -o json` | Valid JSON array |
| Empty sprint | `jtk sprints issues <empty-sprint-id>` | Empty table or "No issues" |
| Non-existent sprint | `jtk sprints issues 99999` | Error |

### sprints add

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Add issue to sprint | `jtk sprints add <sprint-id> TEST-X` | Success with count |
| Add multiple issues | `jtk sprints add <sprint-id> TEST-X TEST-Y` | "Moved 2 issue(s)" |
| Verify addition | `jtk sprints issues <sprint-id>` | Moved issues appear |
| Non-existent sprint | `jtk sprints add 99999 TEST-X` | Error |
| Non-existent issue | `jtk sprints add <sprint-id> TEST-99999` | Error |

---

## User Operations

### users search

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Search by name | `jtk users search "john"` | Table with ACCOUNT_ID, NAME, EMAIL, ACTIVE |
| Limit results | `jtk users search "a" --max 3` | At most 3 users |
| JSON output | `jtk users search "john" -o json` | Valid JSON array |
| No results | `jtk users search "xyznonexistent999"` | Empty table or "No users found" |

---

## Automation Operations

### Important: No API Delete

The Jira Automation API does not support deleting rules. Cleanup is done by disabling and renaming test rules to `[DELETEME] ...`, then manually purging them through the Jira UI. See [Cleanup](#cleanup) for details.

### automation list

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| List all rules | `jtk auto list` | Table with UUID, NAME, STATE, LABELS columns |
| Filter enabled | `jtk auto list --state ENABLED` | Only ENABLED rules shown |
| Filter disabled | `jtk auto list --state DISABLED` | Only DISABLED rules shown |
| Case-insensitive filter | `jtk auto list --state enabled` | Works (uppercased internally) |
| JSON output | `jtk auto list -o json` | Valid JSON array, parseable by `jq` |
| No rules match filter | `jtk auto list --state DISABLED` (if none disabled) | "No automation rules found" |

### automation get

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Get rule details | `jtk auto get <uuid>` | Shows Name, UUID, State, Components summary |
| Get with --full | `jtk auto get <uuid> --full` | Shows component details: [1] TRIGGER: type, [2] ACTION: type, etc. |
| JSON output | `jtk auto get <uuid> -o json` | Full rule object as valid JSON |
| Non-existent rule | `jtk auto get 99999999` | Error: 404 or similar |

### automation export

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Export pretty-printed | `jtk auto export <uuid>` | Indented JSON to stdout |
| Export is valid JSON | `jtk auto export <uuid> \| jq .` | Parses without errors |
| Export compact | `jtk auto export <uuid> --compact` | Single-line JSON |
| Export to file | `jtk auto export <uuid> > /tmp/rule.json` | File written, readable by `cat` |
| Ignores -o flag | `jtk auto export <uuid> -o plain` | Still outputs JSON (export always outputs JSON) |
| Non-existent rule | `jtk auto export 99999999` | Error |

### automation create

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Create from export | `jtk auto export <uuid> > /tmp/rule.json && jtk auto create --file /tmp/rule.json` | New rule created, shows new UUID |
| Verify created | `jtk auto get <new-uuid>` | Rule exists with same components as source |
| Create with modified name | Edit JSON to change name, then create | Rule created with new name |
| Missing --file | `jtk auto create` | Error: required flag "file" not set |
| Invalid JSON file | `echo "not json" > /tmp/bad.json && jtk auto create --file /tmp/bad.json` | Error: does not contain valid JSON |
| Non-existent file | `jtk auto create --file /tmp/nope.json` | Error: failed to read file |

### automation enable / disable

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Disable enabled rule | `jtk auto disable <enabled-uuid>` | Success: "Rule ... ENABLED -> DISABLED" |
| Verify disabled | `jtk auto get <uuid>` | State: DISABLED |
| Re-enable rule | `jtk auto enable <uuid>` | Success: "Rule ... DISABLED -> ENABLED" |
| Verify re-enabled | `jtk auto get <uuid>` | State: ENABLED |
| Enable already-enabled | `jtk auto enable <enabled-uuid>` | "already ENABLED" (idempotent, no API call) |
| Disable already-disabled | `jtk auto disable <disabled-uuid>` | "already DISABLED" (idempotent, no API call) |
| Non-existent rule | `jtk auto enable 99999999` | Error |

### automation update (on test copy only)

All update mutation tests operate on a **copy** of a real rule. Never modify production rules.

**Setup:** `jtk auto export <source-uuid> > /tmp/rule.json` -> edit name to "[Test] ..." -> `jtk auto create --file /tmp/rule.json` -> note `<test-uuid>`

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| No-op round-trip | `jtk auto export <test-uuid> > /tmp/rt.json && jtk auto update <test-uuid> --file /tmp/rt.json` | Success; rule unchanged |
| Verify round-trip | `jtk auto get <test-uuid>` | Name, state, component count identical |
| Metadata edit (name) | Export, change name in JSON, update | `jtk auto get` shows new name |
| Metadata edit (description) | Export, change description, update | Description updated |
| Insert component | Export, add an ACTION component to components array, update | Component count increases by 1; `--full` shows new action |
| Verify inserted component | `jtk auto get <test-uuid> --full` | New component appears in list |
| Remove inserted component | Export (with new component), remove it from array, update | Component count back to original |
| Verify removal | `jtk auto get <test-uuid> --full` | Component list matches original |
| Missing --file flag | `jtk auto update <test-uuid>` | Error: required flag "file" not set |
| Invalid JSON file | `echo "garbage" > /tmp/bad.json && jtk auto update <test-uuid> --file /tmp/bad.json` | Error: does not contain valid JSON |
| Non-existent file | `jtk auto update <test-uuid> --file /tmp/nope.json` | Error: failed to read file |
| Non-existent rule | `jtk auto update 99999999 --file /tmp/rt.json` | Error |

**Teardown:** See [Cleanup](#cleanup)

---

## Shell Completion

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Bash completion | `jtk completion bash` | Valid bash completion script |
| Zsh completion | `jtk completion zsh` | Valid zsh completion script |
| Fish completion | `jtk completion fish` | Valid fish completion script |
| PowerShell completion | `jtk completion powershell` | Valid PowerShell completion script |

---

## End-to-End Workflows

### Issue Lifecycle

| Test Case | Steps | Expected Result |
|-----------|-------|-----------------|
| Full issue lifecycle | 1. `jtk issues create -p TEST -s "[Test] Lifecycle"` -> note key<br>2. `jtk issues get <key>`<br>3. `jtk issues update <key> -d "Added description"`<br>4. `jtk issues assign <key> <account-id>`<br>5. `jtk comments add <key> -b "Working on it"`<br>6. `jtk attachments add <key> -f test.txt`<br>7. `jtk transitions do <key> "In Progress"`<br>8. `jtk transitions do <key> "Done"`<br>9. `jtk issues delete <key> --force` | All steps succeed; issue created, updated, commented, attached, transitioned, deleted |

### Sprint Workflow

| Test Case | Steps | Expected Result |
|-----------|-------|-----------------|
| Sprint discovery and management | 1. `jtk boards list -p TEST` -> note board ID<br>2. `jtk sprints list -b <board-id> -s active` -> note sprint ID<br>3. `jtk sprints current -b <board-id>`<br>4. `jtk sprints issues <sprint-id>`<br>5. Create test issue<br>6. `jtk sprints add <sprint-id> <key>`<br>7. `jtk sprints issues <sprint-id>` -> verify issue appears<br>8. Clean up test issue | Board found, sprint listed, issues managed |

### Automation Clone Workflow

| Test Case | Steps | Expected Result |
|-----------|-------|-----------------|
| Safe copy->mutate->cleanup | 1. Export complex rule<br>2. Rename to "[Test] Copy" in JSON<br>3. `jtk auto create --file`<br>4. Run mutation tests on copy<br>5. Disable + rename to "[DELETEME] ..." | Copy created, mutated, marked for cleanup |
| Full read cycle | 1. `jtk auto list` -> pick UUID<br>2. `jtk auto get <uuid>`<br>3. `jtk auto export <uuid> \| jq .` | All succeed, data is consistent |
| Toggle cycle (on copy) | 1. Create test copy<br>2. Disable<br>3. Verify DISABLED<br>4. Enable<br>5. Verify ENABLED<br>6. Cleanup | State toggles correctly |
| Component round-trip | 1. Create test copy<br>2. Export<br>3. Add action<br>4. Update<br>5. Export again<br>6. Remove action<br>7. Update<br>8. Verify matches original<br>9. Cleanup | Components survive insert/remove cycle |

---

## Edge Cases & Error Handling

### Global Flags

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| No color output | `jtk issues list -p TEST --no-color` | No ANSI color codes in output |
| Verbose mode | `jtk issues list -p TEST --verbose` | Additional debug output |
| JSON output format | `jtk issues list -p TEST -o json \| jq .` | Valid JSON, parseable |
| Plain output format | `jtk issues list -p TEST -o plain` | Tab-separated, scriptable |

### Command Aliases

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| `issue` alias | `jtk issue list -p TEST` | Same as `jtk issues list` |
| `i` alias | `jtk i list -p TEST` | Same as `jtk issues list` |
| `comment` alias | `jtk comment list TEST-1` | Same as `jtk comments list` |
| `c` alias | `jtk c list TEST-1` | Same as `jtk comments list` |
| `transition` alias | `jtk transition list TEST-1` | Same as `jtk transitions list` |
| `tr` alias | `jtk tr list TEST-1` | Same as `jtk transitions list` |
| `attachment` alias | `jtk attachment list TEST-1` | Same as `jtk attachments list` |
| `att` alias | `jtk att list TEST-1` | Same as `jtk attachments list` |
| `auto` alias | `jtk auto list` | Same as `jtk automation list` |
| `board` alias | `jtk board list` | Same as `jtk boards list` |
| `b` alias | `jtk b list` | Same as `jtk boards list` |
| `sprint` alias | `jtk sprint list -b 1` | Same as `jtk sprints list` |
| `sp` alias | `jtk sp list -b 1` | Same as `jtk sprints list` |
| `user` alias | `jtk user search "a"` | Same as `jtk users search` |
| `u` alias | `jtk u search "a"` | Same as `jtk users search` |
| `attachments ls` | `jtk att ls TEST-1` | Same as `attachments list` |
| `attachments download` | `jtk att download <id>` | Same as `attachments get` |
| `attachments rm` | `jtk att rm <id>` | Same as `attachments delete` |

### Error Messages

| Scenario | Expected Error |
|----------|----------------|
| Invalid credentials | API error (status 401) |
| Permission denied | API error (status 403) |
| Resource not found | API error (status 404) |
| No configuration | URL/email/token required |
| Network unreachable | Request failed |
| Cloud ID fetch fails | Failed to fetch cloud ID |

### Output Formats

| Format | Flag | Verified With |
|--------|------|---------------|
| Table (default) | (none) | Visual inspection |
| JSON | `--output json` | `jq .` parsing |
| Plain | `--output plain` | Tab-separated, scriptable |

---

## Test Execution Checklist

### Setup
- [ ] Build latest: `make build-jtk`
- [ ] Verify config: `jtk me` works
- [ ] Note test project key and board ID
- [ ] Verify automation rules exist: `jtk auto list` shows rules

### Config & Init
- [ ] `jtk config show` displays config
- [ ] `jtk config test` passes
- [ ] `jtk me` shows current user

### Issue CRUD
- [ ] Create issue (basic)
- [ ] Create issue (with type, description, custom field)
- [ ] Get issue (table)
- [ ] Get issue (--full)
- [ ] Get issue (JSON)
- [ ] List issues (with project filter)
- [ ] List issues (with sprint filter)
- [ ] Search issues (JQL)
- [ ] Update issue (summary, description)
- [ ] Assign issue
- [ ] Unassign issue
- [ ] List fields
- [ ] List field options
- [ ] List issue types
- [ ] Transition issue (by name)
- [ ] Transition issue (with field)
- [ ] Add comment
- [ ] List comments (--full)
- [ ] Delete comment
- [ ] Upload attachment
- [ ] List attachments
- [ ] Download attachment
- [ ] Verify download integrity
- [ ] Delete attachment
- [ ] Delete issue (--force)

### Sprint & Board
- [ ] List boards
- [ ] Get board details
- [ ] List sprints (with state filter)
- [ ] Show current sprint
- [ ] List sprint issues
- [ ] Add issue to sprint

### Users
- [ ] Search users

### Automation (read-only first)
- [ ] List all rules
- [ ] List filtered by state
- [ ] Get a rule (table and --full)
- [ ] Export a rule (pretty and compact)

### Automation (mutations on test copy)
- [ ] Create test copy from export
- [ ] No-op round-trip
- [ ] Metadata edit (name)
- [ ] Component insert/remove cycle
- [ ] Enable/disable toggle
- [ ] Idempotent enable/disable

### End-to-End
- [ ] Full issue lifecycle (create -> transition -> delete)
- [ ] Sprint discovery workflow

### Error Cases
- [ ] Non-existent resource (404)
- [ ] Missing required flags
- [ ] Invalid input (bad JQL, bad JSON)

### Global Flags
- [ ] `--no-color` suppresses color
- [ ] `--verbose` shows debug output
- [ ] `-o json` produces valid JSON
- [ ] `-o plain` produces tab-separated output

### Cleanup
- [ ] Delete all [Test] prefixed issues: `jtk issues delete <key> --force`
- [ ] Disable + rename automation test copies to [DELETEME]
- [ ] Manually delete [DELETEME] rules via Jira UI
- [ ] Verify no test data remains

---

## Cleanup

### Issues, Comments, and Attachments

Test issues can be deleted directly via the API:

```bash
# Delete all test issues
jtk issues search --jql "summary ~ '[Test]'" -o json | jq -r '.[].key' | while read key; do
  jtk issues delete "$key" --force
done
```

### Automation Rules

The Jira Automation API does not expose a delete endpoint. Test rules must be cleaned up manually.

**After running integration tests:**

1. Disable and rename all test rules:
   ```bash
   # For each test rule:
   jtk auto disable <test-uuid>
   jtk auto export <test-uuid> > /tmp/cleanup.json
   # Edit /tmp/cleanup.json: change "name" to "[DELETEME] <name>"
   jtk auto update <test-uuid> --file /tmp/cleanup.json
   ```

2. Manually purge `[DELETEME]` rules in the Jira UI:
   - **Global rules:** Settings -> System -> Automation rules
   - **Project rules:** Project Settings -> Automation

3. Verify no test rules remain:
   ```bash
   jtk auto list -o json | jq '.[] | select(.name | startswith("[Test]") or startswith("[DELETEME]"))'
   ```

---

## Adding New Tests

When adding new features or fixing bugs:

1. Add test cases to the appropriate section above
2. Include both happy path and error cases
3. Document any known limitations or edge cases
4. Update the "Test Execution Checklist" if needed
5. Record bugs discovered during testing and continue — don't stop to fix
