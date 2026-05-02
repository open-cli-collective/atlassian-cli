# Integration Test Bugs

Discovered during a full pass through `integration-tests.md` against a live Jira instance.
Each entry includes a minimal repro and the delta between expected and actual output.

---

## Issues

### `issues check --id` exits 0 when fields are missing

`--id` should exit 1 when any required fields are missing (the test doc says so) but always exits 0.

**Repro:**
```bash
# Use an issue that has at least one MISSING field
jtk issues check MON-3714 --id
# Output: labels
# Exit:   0   ← should be 1
```

---

### Escape sequences in comment body not interpreted

`\n` and `\t` in `-b` values are stored/displayed as literal backslash-n and backslash-t rather than newline and tab.

**Repro:**
```bash
jtk comments add MON-XXXX -b "Line one\nLine two\n\tIndented line"
# Output: Line oneLine twoIndented line
# Expected: three distinct lines with actual newline and tab
```

---

### `issues delete` output missing "issue" word

**Repro:**
```bash
jtk issues delete MON-XXXX --force
# Actual:   Deleted MON-XXXX
# Expected: Deleted issue MON-XXXX
```

The integration test doc writes `✓ Deleted issue $KEY` — the `✓` and the word "issue" are both absent.

---

## Fields

### `fields list` / `issues fields` missing CUSTOM column; wrong column order

Both commands emit `ID | TYPE | NAME` but the integration test expects `ID | NAME | TYPE | CUSTOM`.

**Repro:**
```bash
jtk fields list
# Actual:   ID | TYPE | NAME
# Expected: ID | NAME | TYPE | CUSTOM

jtk issues fields
# Same issue
```

### `fields contexts list --id` returns full table instead of IDs

`--id` flag has no effect on `fields contexts list`.

**Repro:**
```bash
jtk fields contexts list customfield_10035 --id
# Actual:   ID | NAME | GLOBAL | ANY_ISSUE_TYPE  (full table)
# Expected: 10135
```

### `fields options add` returns table row, not confirmation message

**Repro:**
```bash
jtk fields options add customfield_10260 --value "Option A"
# Actual:   ID | VALUE | DISABLED / 10147 | Option A | no   (table row)
# Expected: Added option 10147 (Option A)
```

### `fields options update` returns table row, not confirmation message

**Repro:**
```bash
jtk fields options update customfield_10260 --option 10147 --value "Option A (updated)"
# Actual:   ID | VALUE | DISABLED / 10147 | Option A (updated) | no
# Expected: Updated option 10147
```

### `fields options delete` says "from context", not "from field"

**Repro:**
```bash
jtk fields options delete customfield_10260 --option 10147 --force
# Actual:   Deleted option 10147 from context 10476
# Expected: Deleted option 10147 from field customfield_10260
```

### `fields contexts create` returns table row, not confirmation message

**Repro:**
```bash
jtk fields contexts create customfield_10260 --name "[Test] Context" --project 10022
# Actual:   ID | NAME | GLOBAL | ANY_ISSUE_TYPE / 10478 | [Test] Context | no | no
# Expected: Created context 10478 ([Test] Context)
```

Also: creating a context without `--project` on a global field fails with `bad request: Only one global context is allowed per field.` — the integration test doc doesn't use `--project`, which will always fail for newly-created (global) fields.

### `fields contexts delete` output missing `✓` prefix

**Repro:**
```bash
jtk fields contexts delete customfield_10260 10478 --force
# Actual:   Deleted context 10478 from field customfield_10260
# Expected: ✓ Deleted context 10478 from field customfield_10260
```

### `fields delete` says "Deleted field … moved to trash", not "Trashed field"

**Repro:**
```bash
jtk fields delete customfield_10260 --force
# Actual:   Deleted field customfield_10260 (moved to trash — use fields restore to recover)
# Expected: Trashed field customfield_10260
```

### `fields restore` returns "post-state unavailable" fallback instead of full field detail

**Repro:**
```bash
jtk fields restore customfield_10260
# Actual:   post-state unavailable; showing confirmation only
#           Restored field customfield_10260
# Expected: Full field detail block (ID, Name, Type, Contexts, etc.)
```

---

## Comments

### `comments delete` has no `--force` flag

The integration test doc calls `jtk comments delete $ISSUE $COMMENT_ID --force` but the flag does not exist.

**Repro:**
```bash
jtk comments delete MON-XXXX 21769 --force
# Error: unknown flag: --force
```

### `comments delete` output missing `✓` prefix

**Repro:**
```bash
jtk comments delete MON-XXXX 21769
# Actual:   Deleted comment 21769 from MON-XXXX
# Expected: ✓ Deleted comment 21769 from MON-XXXX
```

---

## Projects

### `projects delete` output has extra text vs doc expectation

**Repro:**
```bash
jtk projects delete ZTST2 --force
# Actual:   Deleted project ZTST2 (moved to trash — recoverable for 60 days via projects restore)
# Expected: ✓ Deleted project ZTST2 (moved to trash)
```

---

## Automation

### `auto list` missing LABELS column; wrong column order

Integration test expects `UUID, NAME, STATE, LABELS` but actual is `ID | STATE | NAME`.

**Repro:**
```bash
jtk auto list
# Actual:   ID | STATE | NAME
# Expected: UUID | NAME | STATE | LABELS  (or similar)
```

### `auto get --show-components` outputs indented tree, not flat table

Integration test expects `# | COMPONENT | TYPE` flat table but the code outputs an indented tree.

**Repro:**
```bash
jtk auto get $AUTO_UUID --show-components
# Actual:   TRIGGER  jira.issue.event.trigger:created
#             CONDITION  jira.jql.condition
#             ACTION  jira.create.variable
#             ...
# Expected: # | COMPONENT | TYPE (flat table)
```

### `auto delete` without `--force` prompts for confirmation

The integration test doc step 11 runs `jtk auto delete $TEST_AUTO_UUID` without `--force` and expects silent deletion, but the command always prompts.

**Repro:**
```bash
jtk auto delete 019dd3c3-97eb-7948-a948-cda7ed4a6579
# Prompts: Are you sure? [y/N]:
# Doc says: Expected: Rule deleted (auto-disables if ENABLED)
```

**Fix for doc:** add `--force` to step 11.

### `auto disable --id` hits idempotent path in test ordering

The test sequence disables the rule, then immediately calls `disable --id` a second time. The second call hits the idempotent path and emits `Rule "…" is already DISABLED` instead of the UUID.

**Repro:**
```bash
jtk auto disable $UUID       # succeeds, rule is now DISABLED
jtk auto disable $UUID --id  # Actual: Rule "…" is already DISABLED
                             # Expected: $UUID
```

**Fix for doc:** test `--id` before (or instead of) the first disable, or re-enable between calls.

---

## Dashboards

### `dashboards create` outputs table row, not confirmation message

**Repro:**
```bash
jtk dashboards create --name "[Test] Integration Dashboard"
# Actual:   ID | GADGETS | OWNER | FAVOURITE | NAME / 10107 | 0 | Rian Stockbower | yes | [Test] Integration Dashboard
# Expected: Created dashboard [Test] Integration Dashboard (10107)
```

### `dashboards list --search` "no results" message doesn't include the search term

**Repro:**
```bash
jtk dashboards list --search "xyznonexistent999"
# Actual:   No dashboards found
# Expected: No dashboards found matching "xyznonexistent999"
```

### `dashboards gadgets list --id` emits nothing (no IDs, no "No gadgets" message) when empty

**Repro:**
```bash
jtk dashboards gadgets list 10001 --id   # dashboard with no gadgets
# Actual:   (empty output)
# Expected: either empty output or "No gadgets on dashboard 10001"
```

---

## Users / Me

### `me` default output missing Active field

Integration test expects "Detail block: Account ID, Display Name, Email, Active" but Active is absent.

**Repro:**
```bash
jtk me
# Actual:   60e09bae... | Rian Stockbower | rian@monitapp.io
# Expected: includes Active: yes/no
```

### `users get` default output missing Active field

Same issue as `me`: `users search` includes ACTIVE column but `users get` does not.

**Repro:**
```bash
jtk users get 60e09bae7fcd820073089249
# Actual:   60e09bae... | Rian Stockbower | rian@monitapp.io
# Expected: includes Active column
```

---

## Links

### `links types --extended` output identical to default (no extra columns)

**Repro:**
```bash
jtk links types
jtk links types --extended
# Both emit: ID | NAME | INWARD | OUTWARD  (identical output)
```

---

## Sprints

### `sprints list` without `--board` error message differs from expected

**Repro:**
```bash
jtk sprints list
# Actual:   --board is required
# Expected: Error: required flag(s) "board" not set
```

---

## Global Flags / Output

### `-o plain` outputs pipe-delimited format, not tab-separated

The legacy `plain` format is supposed to produce tab-separated rows with no headers. For commands that have migrated to the new `present.Emit` path (e.g. `issues list`), `-o plain` has no effect — output is identical to default table format.

**Repro:**
```bash
jtk issues list -p MON --max 1 -o plain
# Actual:   KEY | STATUS | TYPE | PTS | ASSIGNEE | SUMMARY  (pipe-delimited, same as default)
# Expected: tab-separated values, no header row
```

Root cause: `opts.RenderMode()` always returns `RenderModeAgent`; the `-o` flag is never consulted in the `present.Emit` code path.

---

## Integration Test Doc Bugs (not code bugs)

These are errors in `integration-tests.md` that don't reflect actual command behavior and need to be corrected:

| Section | Command in doc | Problem |
|---------|----------------|---------|
| §10 step 6 | `jtk comments delete $ISSUE $COMMENT_ID_2 --force` | `--force` not supported on `comments delete` |
| §14 step 11 | `jtk auto delete $TEST_AUTO_UUID` | needs `--force` to suppress prompt |
| §14 step 5 "Also test --id" | `jtk auto disable $UUID --id` | rule is already disabled at this point; hits idempotent path |
| §16 step 10 | `jtk fields contexts create $FIELD --name "[Test] Context"` | fails on global fields; needs `--project <ID>` |
| §17 alias row 11 | `jtk f list --max 1` | `fields list` has no `--max` flag |
| §6/§13 | `dashboards gadgets add … --type com.atlassian.jira.gadgets:filter-results-gadget` | Module key not present in this Jira instance; will need a valid module key from `gadgets list` |
