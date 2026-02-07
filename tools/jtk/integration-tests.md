# Integration Tests

This document catalogs the manual integration test suite for `jtk` automation commands. These tests verify real-world behavior against a live Jira instance and catch edge cases that are difficult to cover with unit tests.

## Test Environment Setup

### Prerequisites
- A configured `jtk` instance (`jtk init` completed, `jtk me` works)
- At least one ENABLED and one DISABLED automation rule
- At least one rule with multiple components (trigger + conditions + actions)
- Rule IDs are numeric — discover via `jtk auto list`

### Test Data Conventions
- Test copies use `[Test]` prefix in the rule name
- Always clean up test data after tests complete (see [Cleanup](#cleanup))

### Important: No API Delete

The Jira Automation API does not support deleting rules. Cleanup is done by disabling and renaming test rules to `[DELETEME] ...`, then manually purging them through the Jira UI. See [Cleanup](#cleanup) for details.

---

## automation list

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| List all rules | `jtk auto list` | Table with ID, NAME, STATE, LABELS columns |
| Filter enabled | `jtk auto list --state ENABLED` | Only ENABLED rules shown |
| Filter disabled | `jtk auto list --state DISABLED` | Only DISABLED rules shown |
| Case-insensitive filter | `jtk auto list --state enabled` | Works (uppercased internally) |
| JSON output | `jtk auto list -o json` | Valid JSON array, parseable by `jq` |
| No rules match filter | `jtk auto list --state DISABLED` (if none disabled) | "No automation rules found" |

---

## automation get

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Get rule details | `jtk auto get <id>` | Shows Name, ID, State, Components summary |
| Get with --full | `jtk auto get <id> --full` | Shows component details: [1] TRIGGER: type, [2] ACTION: type, etc. |
| JSON output | `jtk auto get <id> -o json` | Full rule object as valid JSON with components array |
| Non-existent rule | `jtk auto get 99999999` | Error: 404 or similar |

---

## automation export

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Export pretty-printed | `jtk auto export <id>` | Indented JSON to stdout |
| Export is valid JSON | `jtk auto export <id> \| jq .` | Parses without errors |
| Export compact | `jtk auto export <id> --compact` | Single-line JSON |
| Export to file | `jtk auto export <id> > /tmp/rule.json` | File written, readable by `cat` |
| Ignores -o flag | `jtk auto export <id> -o plain` | Still outputs JSON (export always outputs JSON) |
| Non-existent rule | `jtk auto export 99999999` | Error |

---

## automation create

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Create from export | `jtk auto export <id> > /tmp/rule.json && jtk auto create --file /tmp/rule.json` | New rule created, shows new ID |
| Verify created | `jtk auto get <new-id>` | Rule exists with same components as source |
| Create with modified name | Edit JSON to change name, then create | Rule created with new name |
| Missing --file | `jtk auto create` | Error: required flag "file" not set |
| Invalid JSON file | `echo "not json" > /tmp/bad.json && jtk auto create --file /tmp/bad.json` | Error: does not contain valid JSON |
| Non-existent file | `jtk auto create --file /tmp/nope.json` | Error: failed to read file |

---

## automation enable / disable

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Disable enabled rule | `jtk auto disable <enabled-id>` | Success: "Rule ... ENABLED → DISABLED" |
| Verify disabled | `jtk auto get <id>` | State: DISABLED |
| Re-enable rule | `jtk auto enable <id>` | Success: "Rule ... DISABLED → ENABLED" |
| Verify re-enabled | `jtk auto get <id>` | State: ENABLED |
| Enable already-enabled | `jtk auto enable <enabled-id>` | "already ENABLED" (idempotent, no API call) |
| Disable already-disabled | `jtk auto disable <disabled-id>` | "already DISABLED" (idempotent, no API call) |
| Non-existent rule | `jtk auto enable 99999999` | Error |

---

## automation update (on test copy only)

All update mutation tests operate on a **copy** of a real rule. Never modify production rules.

**Setup:** `jtk auto export <source-id> > /tmp/rule.json` → edit name to "[Test] ..." → `jtk auto create --file /tmp/rule.json` → note `<test-id>`

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| No-op round-trip | `jtk auto export <test-id> > /tmp/rt.json && jtk auto update <test-id> --file /tmp/rt.json` | Success; rule unchanged |
| Verify round-trip | `jtk auto get <test-id>` | Name, state, component count identical |
| Metadata edit (name) | Export, change name in JSON, update | `jtk auto get` shows new name |
| Metadata edit (description) | Export, change description, update | Description updated |
| Insert component | Export, add an ACTION component to components array, update | Component count increases by 1; `--full` shows new action |
| Verify inserted component | `jtk auto get <test-id> --full` | New component appears in list |
| Remove inserted component | Export (with new component), remove it from array, update | Component count back to original |
| Verify removal | `jtk auto get <test-id> --full` | Component list matches original |
| Missing --file flag | `jtk auto update <test-id>` | Error: required flag "file" not set |
| Invalid JSON file | `echo "garbage" > /tmp/bad.json && jtk auto update <test-id> --file /tmp/bad.json` | Error: does not contain valid JSON |
| Non-existent file | `jtk auto update <test-id> --file /tmp/nope.json` | Error: failed to read file |
| Non-existent rule | `jtk auto update 99999999 --file /tmp/rt.json` | Error |

**Teardown:** See [Cleanup](#cleanup)

---

## End-to-End Workflows

| Test Case | Steps | Expected Result |
|-----------|-------|-----------------|
| Full read cycle | 1. `jtk auto list` → pick ID 2. `jtk auto get <id>` 3. `jtk auto export <id> \| jq .` | All succeed, data is consistent |
| Safe copy→mutate→cleanup | 1. Export complex rule 2. Rename to "[Test] Copy" in JSON 3. `jtk auto create --file` 4. Run mutation tests on copy 5. Disable + rename to "[DELETEME] ..." | Copy created, mutated, marked for cleanup |
| Toggle cycle (on copy) | 1. Create test copy 2. Disable 3. Verify DISABLED 4. Enable 5. Verify ENABLED 6. Cleanup | State toggles correctly |
| Component round-trip | 1. Create test copy 2. Export 3. Add action 4. Update 5. Export again 6. Remove action 7. Update 8. Verify matches original 9. Cleanup | Components survive insert/remove cycle |

---

## Error Handling

| Scenario | Expected Error |
|----------|----------------|
| Invalid credentials | API error (status 401) |
| No configuration | URL/email/token required |
| Network unreachable | Request failed |
| Cloud ID fetch fails | Failed to fetch cloud ID |

---

## Test Execution Checklist

### Setup
- [ ] Build latest: `make build-jtk`
- [ ] Verify config: `jtk me` works
- [ ] Verify rules exist: `jtk auto list` shows rules

### Read-Only Tests (safe, run first)
- [ ] List all rules
- [ ] List filtered by ENABLED
- [ ] List filtered by DISABLED
- [ ] List with JSON output
- [ ] Get a rule (table)
- [ ] Get a rule (--full)
- [ ] Get a rule (JSON)
- [ ] Get non-existent rule (expect error)
- [ ] Export a rule (pretty)
- [ ] Export a rule (compact)
- [ ] Export to file and verify with jq

### Create Test Copy
- [ ] Export a complex rule with multiple components
- [ ] Edit JSON: rename to "[Test] Integration Test Copy"
- [ ] Create copy via `jtk auto create --file`
- [ ] Verify copy exists via `jtk auto get`
- [ ] Verify component count matches source

### Mutation Tests (on test copy ONLY)
- [ ] No-op round-trip (export → update, no changes)
- [ ] Metadata edit: change name, verify, change back
- [ ] Component insert: add an action, verify count increased
- [ ] Component remove: remove the added action, verify count restored
- [ ] Enable/disable toggle cycle
- [ ] Idempotent enable (already enabled)
- [ ] Idempotent disable (already disabled)

### Error Cases
- [ ] Update with missing --file flag
- [ ] Update with invalid JSON
- [ ] Update with non-existent file
- [ ] Create with invalid JSON

### Cleanup
- [ ] Disable test copies: `jtk auto disable <test-id>`
- [ ] Rename test copies: export, change name to `[DELETEME] <original-name>`, update
- [ ] Verify disabled: `jtk auto get <test-id>` shows DISABLED
- [ ] Manually delete `[DELETEME]` rules via Jira UI (see below)

---

## Cleanup

The Jira Automation API does not expose a delete endpoint. Test rules must be cleaned up manually.

**After running integration tests:**

1. Disable and rename all test rules:
   ```bash
   # For each test rule:
   jtk auto disable <test-id>
   jtk auto export <test-id> > /tmp/cleanup.json
   # Edit /tmp/cleanup.json: change "name" to "[DELETEME] <name>"
   jtk auto update <test-id> --file /tmp/cleanup.json
   ```

2. Manually purge `[DELETEME]` rules in the Jira UI:
   - **Global rules:** https://monitproduct.atlassian.net/jira/settings/automation
   - **Project rules:** https://monitproduct.atlassian.net/jira/software/c/projects/ON/settings/automate

3. Verify no test rules remain:
   ```bash
   jtk auto list -o json | jq '.[] | select(.name | startswith("[Test]") or startswith("[DELETEME]"))'
   ```

---

## Adding New Tests

When adding new automation features or fixing bugs:

1. Add test cases to the appropriate section above
2. Include both happy path and error cases
3. Document any known limitations or edge cases
4. Update the "Test Execution Checklist" if needed
