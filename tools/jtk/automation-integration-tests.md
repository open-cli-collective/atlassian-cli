# Automation Builder Integration Tests

This document is a sequential runbook for testing automation rules built with the `api` builder module against a live Jira instance. These tests validate that builder-generated JSON is accepted by the Automation REST API and that round-tripped rules preserve their structure.

> **Basic Auth only** — Automation endpoints are not available with scoped tokens. These tests cannot run with Bearer Auth.

If a test reveals a discrepancy between the builder's output and what the API expects, **record the finding and continue testing**. Feed results back into unit test assertions.

---

## Prerequisites

- A configured `jtk` instance with Basic Auth (`jtk init` completed)
- Access to a project with Automation enabled
- Build: `make build-jtk`

## Discover Test Values

Run these commands and capture the values. They are referenced as `$VARIABLES` throughout.

```bash
# $ACCOUNT_ID — your Atlassian account ID (required for authorAccountId and actor)
jtk me -o json | jq -r .accountId

# $PROJECT — pick a project you have full access to
jtk projects list --max 10

# $CLOUD_ID — your Atlassian Cloud ID
curl -s https://YOUR-SITE.atlassian.net/_edge/tenant_info | jq -r .cloudId

# $PROJECT_ARI — construct from cloud ID and project ID
# Format: ari:cloud:jira:$CLOUD_ID:project/$PROJECT_ID
jtk projects get $PROJECT -o json | jq -r '.id'
# → ari:cloud:jira:$CLOUD_ID:project/$PROJECT_ID

# $CUSTOM_SELECT_FIELD — a single-select custom field ID
jtk fields list --custom -o json | jq '.[] | select(.schema.custom | test("select$")) | {id, name}'
# Note the ID (e.g., customfield_10037) and name (e.g., "Banking Platform")

# $CUSTOM_MULTISELECT_FIELD — a multi-select/checkbox custom field ID
jtk fields list --custom -o json | jq '.[] | select(.schema.custom | test("multi|checkbox")) | {id, name}'
# Note the ID (e.g., customfield_10038) and name (e.g., "Products")

# $SELECT_OPTION_ID — an option ID for the select field
jtk fields options list $CUSTOM_SELECT_FIELD -o json | jq '.[0].id'

# $MULTISELECT_OPTION_ID — an option ID for the multiselect field
jtk fields options list $CUSTOM_MULTISELECT_FIELD -o json | jq '.[0].id'

# $EXISTING_AUTO_UUID — an existing automation rule to use as round-trip reference
jtk auto list --state ENABLED --max 5
# Note a UUID
```

---

## Test 1: JQL Condition Rule

Creates a minimal rule with a JQL condition and a comment action.

1. **Write the JSON file:**
   ```bash
   cat > /tmp/auto-jql.json << 'EOF'
   {
     "rule": {
       "name": "[Test] JQL Condition Rule",
       "state": "DISABLED",
       "description": "Integration test: JQL condition",
       "authorAccountId": "$ACCOUNT_ID",
       "actor": {"type": "ACCOUNT_ID", "actor": "$ACCOUNT_ID"},
       "writeAccessType": "UNRESTRICTED",
       "canOtherRuleTrigger": false,
       "notifyOnError": "FIRSTERROR",
       "trigger": {
         "component": "TRIGGER",
         "type": "jira.issue.event.trigger:created",
         "schemaVersion": 1,
         "value": {"eventKey": "jira:issue_created", "issueEvent": "issue_created"},
         "children": [],
         "conditions": []
       },
       "components": [
         {
           "component": "CONDITION",
           "type": "jira.jql.condition",
           "schemaVersion": 1,
           "value": "project = '$PROJECT' AND issuetype = Epic",
           "children": [],
           "conditions": []
         },
         {
           "component": "ACTION",
           "type": "jira.issue.comment",
           "schemaVersion": 1,
           "value": {"comment": "JQL condition matched"},
           "children": [],
           "conditions": []
         }
       ],
       "ruleScopeARIs": ["$PROJECT_ARI"]
     },
     "connections": []
   }
   EOF
   ```
   (Replace `$PROJECT`, `$PROJECT_ARI`, and `$ACCOUNT_ID` with your values)

2. **Create the rule:**
   ```bash
   jtk auto create --file /tmp/auto-jql.json
   ```
   Expected: `✓ Created automation rule: [Test] JQL Condition Rule (UUID: ...)`
   Capture the UUID → `$JQL_UUID`

3. **Verify creation:**
   ```bash
   jtk auto get $JQL_UUID
   ```
   Expected: Name = `[Test] JQL Condition Rule`, State = DISABLED, 2 components

4. **Export and verify round-trip:**
   ```bash
   jtk auto export $JQL_UUID > /tmp/auto-jql-export.json
   cat /tmp/auto-jql-export.json | jq '.rule.components[0]'
   ```
   Expected: Component type = `jira.jql.condition`, value is the JQL string

5. **Record findings:**
   - Did the API accept `schemaVersion: 1` for JQL conditions? ____
   - Did the API modify the `value` field? ____
   - What additional fields did the API add to the component? ____

6. **Cleanup:**
   ```bash
   jtk auto delete $JQL_UUID
   ```

---

## Test 2: Comparator + Variable Rule

Creates a rule using the pattern from real backups: extract a custom field to a variable, then compare.

1. **Write the JSON file:**
   ```bash
   cat > /tmp/auto-comparator.json << 'EOF'
   {
     "rule": {
       "name": "[Test] Comparator Variable Rule",
       "state": "DISABLED",
       "description": "Integration test: variable extraction + comparator condition",
       "authorAccountId": "$ACCOUNT_ID",
       "actor": {"type": "ACCOUNT_ID", "actor": "$ACCOUNT_ID"},
       "writeAccessType": "UNRESTRICTED",
       "canOtherRuleTrigger": false,
       "notifyOnError": "FIRSTERROR",
       "trigger": {
         "component": "TRIGGER",
         "type": "jira.issue.event.trigger:created",
         "schemaVersion": 1,
         "value": {"eventKey": "jira:issue_created", "issueEvent": "issue_created"},
         "children": [],
         "conditions": []
       },
       "components": [
         {
           "component": "ACTION",
           "type": "jira.create.variable",
           "schemaVersion": 1,
           "value": {
             "id": "_customsmartvalue_id_test_1",
             "name": {"type": "FREE", "value": "testPlatform"},
             "type": "SMART",
             "query": {"type": "SMART", "value": "{{triggerIssue.$CUSTOM_SELECT_FIELD}}"},
             "lazy": false
           },
           "children": [],
           "conditions": []
         },
         {
           "component": "CONDITION",
           "type": "jira.comparator.condition",
           "schemaVersion": 1,
           "value": {
             "first": "{{testPlatform}}",
             "second": "Q2",
             "operator": "EQUALS"
           },
           "children": [],
           "conditions": []
         },
         {
           "component": "ACTION",
           "type": "jira.issue.comment",
           "schemaVersion": 1,
           "value": {"comment": "Comparator matched"},
           "children": [],
           "conditions": []
         }
       ],
       "ruleScopeARIs": ["$PROJECT_ARI"]
     },
     "connections": []
   }
   EOF
   ```

2. **Create and verify:**
   ```bash
   jtk auto create --file /tmp/auto-comparator.json
   ```
   Capture UUID → `$COMP_UUID`
   ```bash
   jtk auto get $COMP_UUID --show-components
   jtk auto export $COMP_UUID > /tmp/auto-comparator-export.json
   ```

3. **Record findings:**
   - Did the API accept the custom `id` for the variable? ____
   - Did the comparator condition survive round-trip? ____
   - Verify: `jq '.rule.components[1].value' /tmp/auto-comparator-export.json`

4. **Cleanup:**
   ```bash
   jtk auto delete $COMP_UUID
   ```

---

## Test 3: If/Else Block Rule

Creates a rule with if/else branching — the most complex structure.

1. **Write the JSON file:**
   ```bash
   cat > /tmp/auto-ifelse.json << 'EOF'
   {
     "rule": {
       "name": "[Test] If/Else Block Rule",
       "state": "DISABLED",
       "description": "Integration test: if/else branching with comparator conditions",
       "authorAccountId": "$ACCOUNT_ID",
       "actor": {"type": "ACCOUNT_ID", "actor": "$ACCOUNT_ID"},
       "writeAccessType": "UNRESTRICTED",
       "canOtherRuleTrigger": false,
       "notifyOnError": "FIRSTERROR",
       "trigger": {
         "component": "TRIGGER",
         "type": "jira.issue.event.trigger:created",
         "schemaVersion": 1,
         "value": {"eventKey": "jira:issue_created", "issueEvent": "issue_created"},
         "children": [],
         "conditions": []
       },
       "components": [
         {
           "component": "ACTION",
           "type": "jira.create.variable",
           "schemaVersion": 1,
           "value": {
             "id": "_customsmartvalue_id_test_ifelse",
             "name": {"type": "FREE", "value": "platform"},
             "type": "SMART",
             "query": {"type": "SMART", "value": "{{triggerIssue.$CUSTOM_SELECT_FIELD}}"},
             "lazy": false
           },
           "children": [],
           "conditions": []
         },
         {
           "component": "CONDITION",
           "type": "jira.condition.container.block",
           "schemaVersion": 1,
           "value": {},
           "children": [
             {
               "component": "CONDITION_BLOCK",
               "type": "jira.condition.if.block",
               "schemaVersion": 1,
               "value": {"conditionMatchType": "ALL"},
               "conditions": [
                 {
                   "component": "CONDITION",
                   "type": "jira.comparator.condition",
                   "schemaVersion": 1,
                   "value": {"first": "{{platform}}", "second": "Q2", "operator": "EQUALS"},
                   "children": [],
                   "conditions": []
                 }
               ],
               "children": [
                 {
                   "component": "ACTION",
                   "type": "jira.issue.comment",
                   "schemaVersion": 1,
                   "value": {"comment": "Platform is Q2"},
                   "children": [],
                   "conditions": []
                 }
               ]
             },
             {
               "component": "CONDITION_BLOCK",
               "type": "jira.condition.if.block",
               "schemaVersion": 1,
               "value": {"conditionMatchType": "ALL"},
               "conditions": [],
               "children": [
                 {
                   "component": "ACTION",
                   "type": "jira.issue.comment",
                   "schemaVersion": 1,
                   "value": {"comment": "Platform is not Q2"},
                   "children": [],
                   "conditions": []
                 }
               ]
             }
           ],
           "conditions": []
         }
       ],
       "ruleScopeARIs": ["$PROJECT_ARI"]
     },
     "connections": []
   }
   EOF
   ```

2. **Create and verify:**
   ```bash
   jtk auto create --file /tmp/auto-ifelse.json
   ```
   Capture UUID → `$IFELSE_UUID`
   ```bash
   jtk auto get $IFELSE_UUID --show-components
   jtk auto export $IFELSE_UUID > /tmp/auto-ifelse-export.json
   ```

3. **Verify nested structure survived:**
   ```bash
   # Container block
   jq '.rule.components[1].type' /tmp/auto-ifelse-export.json
   # Expected: "jira.condition.container.block"

   # If block children
   jq '.rule.components[1].children | length' /tmp/auto-ifelse-export.json
   # Expected: 2

   # First block conditions
   jq '.rule.components[1].children[0].conditions[0].value' /tmp/auto-ifelse-export.json
   # Expected: {"first": "{{platform}}", "second": "Q2", "operator": "EQUALS"}

   # Else block (empty conditions)
   jq '.rule.components[1].children[1].conditions | length' /tmp/auto-ifelse-export.json
   # Expected: 0
   ```

4. **Record findings:**
   - Did the nested if/else structure survive? ____
   - Did `conditionMatchType: "ALL"` survive? ____
   - Did the else block (empty conditions) survive? ____

5. **Cleanup:**
   ```bash
   jtk auto delete $IFELSE_UUID
   ```

---

## Test 4: Multi-Condition Rule (AND)

Creates a rule with multiple conditions: platform = Q2 AND product includes CheckSync.

1. **Write the JSON file:**
   ```bash
   cat > /tmp/auto-multi.json << 'EOF'
   {
     "rule": {
       "name": "[Test] Multi-Condition AND Rule",
       "state": "DISABLED",
       "description": "Integration test: platform = Q2 AND products includes CheckSync",
       "authorAccountId": "$ACCOUNT_ID",
       "actor": {"type": "ACCOUNT_ID", "actor": "$ACCOUNT_ID"},
       "writeAccessType": "UNRESTRICTED",
       "canOtherRuleTrigger": false,
       "notifyOnError": "FIRSTERROR",
       "trigger": {
         "component": "TRIGGER",
         "type": "jira.issue.event.trigger:created",
         "schemaVersion": 1,
         "value": {"eventKey": "jira:issue_created", "issueEvent": "issue_created"},
         "children": [],
         "conditions": []
       },
       "components": [
         {
           "component": "CONDITION",
           "type": "jira.jql.condition",
           "schemaVersion": 1,
           "value": "\"$CUSTOM_SELECT_FIELD_NAME\" = \"Q2\" AND \"$CUSTOM_MULTISELECT_FIELD_NAME\" in (\"CheckSync\")",
           "children": [],
           "conditions": []
         },
         {
           "component": "ACTION",
           "type": "jira.issue.comment",
           "schemaVersion": 1,
           "value": {"comment": "Multi-condition matched"},
           "children": [],
           "conditions": []
         }
       ],
       "ruleScopeARIs": ["$PROJECT_ARI"]
     },
     "connections": []
   }
   EOF
   ```

2. **Create and verify:**
   ```bash
   jtk auto create --file /tmp/auto-multi.json
   ```
   Capture UUID → `$MULTI_UUID`
   ```bash
   jtk auto get $MULTI_UUID --show-components
   jtk auto export $MULTI_UUID > /tmp/auto-multi-export.json
   ```

3. **Record findings:**
   - Did the JQL condition with custom field names work? ____
   - Did the multi-select `in (...)` syntax work? ____

4. **Cleanup:**
   ```bash
   jtk auto delete $MULTI_UUID
   ```

---

## Test 5: Round-Trip Fidelity

Tests that exporting a rule and re-creating it produces structurally identical output.

1. **Export a known working rule:**
   ```bash
   jtk auto export $EXISTING_AUTO_UUID > /tmp/auto-rt-source.json
   ```

2. **Create a copy:**
   ```bash
   jq 'del(.rule.uuid) | del(.rule.id) | del(.rule.ruleKey) | .rule.name = "[Test] Round-Trip Copy"' \
     /tmp/auto-rt-source.json > /tmp/auto-rt-clean.json
   jtk auto create --file /tmp/auto-rt-clean.json
   ```
   Capture UUID → `$RT_UUID`

3. **Export the copy:**
   ```bash
   jtk auto export $RT_UUID > /tmp/auto-rt-copy.json
   ```

4. **Compare structures** (ignoring server-assigned IDs):
   ```bash
   # Compare component types
   diff \
     <(jq '[.rule.components[].type]' /tmp/auto-rt-source.json) \
     <(jq '[.rule.components[].type]' /tmp/auto-rt-copy.json)

   # Compare trigger type
   diff \
     <(jq '.rule.trigger.type' /tmp/auto-rt-source.json) \
     <(jq '.rule.trigger.type' /tmp/auto-rt-copy.json)
   ```
   Expected: No differences in component types and trigger type

5. **Record findings:**
   - Did component types survive? ____
   - Did component values survive? ____
   - What fields did the API add/modify? ____

6. **Cleanup:**
   ```bash
   jtk auto delete $RT_UUID
   ```

---

## Test 6: Edit Field Action

Creates a rule with a `jira.issue.edit` action that sets a custom field.

1. **Write the JSON file:**
   ```bash
   cat > /tmp/auto-edit.json << 'EOF'
   {
     "rule": {
       "name": "[Test] Edit Field Action Rule",
       "state": "DISABLED",
       "description": "Integration test: edit issue field action",
       "authorAccountId": "$ACCOUNT_ID",
       "actor": {"type": "ACCOUNT_ID", "actor": "$ACCOUNT_ID"},
       "writeAccessType": "UNRESTRICTED",
       "canOtherRuleTrigger": false,
       "notifyOnError": "FIRSTERROR",
       "trigger": {
         "component": "TRIGGER",
         "type": "jira.manual.trigger.issue",
         "schemaVersion": 1,
         "value": {},
         "children": [],
         "conditions": []
       },
       "components": [
         {
           "component": "ACTION",
           "type": "jira.issue.edit",
           "schemaVersion": 10,
           "value": {
             "operations": [
               {
                 "field": {"type": "NAME", "value": "$CUSTOM_MULTISELECT_FIELD_NAME"},
                 "fieldType": "com.atlassian.jira.plugin.system.customfieldtypes:multicheckboxes",
                 "type": "SET",
                 "value": []
               }
             ],
             "advancedFields": null,
             "sendNotifications": true
           },
           "children": [],
           "conditions": []
         }
       ],
       "ruleScopeARIs": ["$PROJECT_ARI"]
     },
     "connections": []
   }
   EOF
   ```

2. **Create and verify:**
   ```bash
   jtk auto create --file /tmp/auto-edit.json
   ```
   Capture UUID → `$EDIT_UUID`
   ```bash
   jtk auto get $EDIT_UUID --show-components
   jtk auto export $EDIT_UUID > /tmp/auto-edit-export.json
   jq '.rule.components[0].value.operations' /tmp/auto-edit-export.json
   ```

3. **Record findings:**
   - Did `schemaVersion: 10` for `jira.issue.edit` work? ____
   - Did the operations array structure survive? ____
   - Did field reference by NAME work? ____

4. **Cleanup:**
   ```bash
   jtk auto delete $EDIT_UUID
   ```

---

## Error Cases

| # | Scenario | Command | Expected |
|---|----------|---------|----------|
| 1 | Malformed JSON | `echo "not json" > /tmp/bad.json && jtk auto create --file /tmp/bad.json` | Error: does not contain valid JSON |
| 2 | Missing trigger | Create rule with no trigger field | Record API response |
| 3 | Invalid field ID in condition | Use `customfield_99999` in a field condition | Record API response |
| 4 | Missing file | `jtk auto create --file /tmp/nope.json` | Error: failed to read file |

---

## Final Cleanup

List any leftover test rules and delete them:

```bash
# Find any remaining test rules
jtk auto list | grep -E "\[Test\]|\[DELETEME\]"

# Delete each one (auto-disables ENABLED rules before deleting)
jtk auto delete $UUID
```

Automation rules can also be managed in the Jira UI at `$JIRA_URL/jira/settings/automation` (system-level settings).

---

## Findings Log

Record findings from each test run below. These feed back into unit test assertions.

### Run 1: 2026-03-09

**Summary of required fields discovered:**
- `authorAccountId` (string) — 400 without it
- `actor` (object `{"type":"ACCOUNT_ID","actor":"<id>"}`) — 400 without it
- `writeAccessType` (string, e.g. `"UNRESTRICTED"`) — 400 without it
- `ruleScopeARIs` (flat string array) — replaces nested `ruleScope.resources`
- Rules with a trigger must have at least one ACTION component

**Schema version upgrades by API:**
- `jira.issue.comment`: schemaVersion 1 → 2 (silent upgrade)
- `jira.issue.edit`: schemaVersion 10 → 12 (silent upgrade)

**Test 1 (JQL Condition):**
- API rejected initially: missing `authorAccountId`, `actor`, `writeAccessType`
- After adding required fields + comment action: accepted
- API rejected `ruleScope.resources`, accepted `ruleScopeARIs`

**Test 2 (Field Condition — `jira.issue.condition`):**
- **REMOVED** — API rejects `jira.issue.condition` regardless of schemaVersion (1 or 3) or value type (ID or VALUE). This component type is not supported via REST API. Use JQL conditions or comparator+variable patterns instead.

**Test 3 (Comparator + Variable):**
- API accepted: yes (after adding required fields + comment action)
- Variable ID preserved: yes
- Comparator survived: yes

**Test 4 (If/Else Block):**
- Nested structure survived: yes
- conditionMatchType preserved: yes
- Else block preserved: yes

**Test 5 (Multi-Condition AND):**
- JQL with custom field names: yes
- Multi-select `in (...)`: yes

**Test 6 (Round-Trip):**
- Component types match: yes (all 25 component types preserved)
- API-added fields: `id`, `ruleKey`, `uuid`, `created`, `updated`, `billingType`, etc.
- Schema versions: API silently upgrades some (comment 1→2, edit 10→12)

**Test 7 (Edit Field Action):**
- Schema version 10 accepted: yes (upgraded to 12 on export)
- Operations array: preserved
- Field by NAME: preserved

**Fixes applied (commits 1-4):**
- Removed `FieldCondition` / `jira.issue.condition` (unsupported)
- Changed `ruleScope.resources` → `ruleScopeARIs`
- Added `WithAuthor()`, `WithActor()`, `WithWriteAccessType()` to builder
- Added validation: trigger requires at least one ACTION

### Run 2: ____-__-__

**Test 1 (JQL Condition):**
- API accepted: ____
- Schema version returned: ____
- Fields added by API: ____

**Test 2 (Comparator + Variable):**
- API accepted: ____
- Variable ID preserved: ____
- Comparator survived: ____

**Test 3 (If/Else Block):**
- Nested structure survived: ____
- conditionMatchType preserved: ____
- Else block preserved: ____

**Test 4 (Multi-Condition AND):**
- JQL with custom field names: ____
- Multi-select `in (...)`: ____

**Test 5 (Round-Trip):**
- Component types match: ____
- API-added fields: ____

**Test 6 (Edit Field Action):**
- Schema version 10 accepted: ____
- Operations array: ____
- Field by NAME: ____

**Unit test updates needed:**
- [ ] ____
- [ ] ____
- [ ] ____
