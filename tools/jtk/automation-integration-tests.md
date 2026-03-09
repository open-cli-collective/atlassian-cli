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

Creates a minimal rule with a JQL condition.

1. **Write the JSON file:**
   ```bash
   cat > /tmp/auto-jql.json << 'EOF'
   {
     "rule": {
       "name": "[Test] JQL Condition Rule",
       "state": "DISABLED",
       "description": "Integration test: JQL condition",
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
         }
       ],
       "ruleScope": {"resources": ["$PROJECT_ARI"]}
     },
     "connections": []
   }
   EOF
   ```
   (Replace `$PROJECT` and `$PROJECT_ARI` with your values)

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
   Expected: Name = `[Test] JQL Condition Rule`, State = DISABLED, 1 component (CONDITION)

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
   jtk auto disable $JQL_UUID
   jq '.rule.name = "[DELETEME] JQL Condition Rule"' /tmp/auto-jql-export.json > /tmp/auto-jql-del.json
   jtk auto update $JQL_UUID --file /tmp/auto-jql-del.json
   ```

---

## Test 2: Field Condition Rule (jira.issue.condition)

Creates a rule with a structured field condition. This validates the `jira.issue.condition` schema discovered from web research (NOT from backups — this is the first live validation).

1. **Write the JSON file:**
   ```bash
   cat > /tmp/auto-field.json << 'EOF'
   {
     "rule": {
       "name": "[Test] Field Condition Rule",
       "state": "DISABLED",
       "description": "Integration test: structured field condition",
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
           "type": "jira.issue.condition",
           "schemaVersion": 3,
           "value": {
             "selectedField": {"type": "ID", "value": "$CUSTOM_SELECT_FIELD"},
             "selectedFieldType": "com.atlassian.jira.plugin.system.customfieldtypes:select",
             "comparison": "EQUALS",
             "compareValue": {"type": "ID", "value": "$SELECT_OPTION_ID", "multiValue": false}
           },
           "children": [],
           "conditions": []
         }
       ],
       "ruleScope": {"resources": ["$PROJECT_ARI"]}
     },
     "connections": []
   }
   EOF
   ```

2. **Create the rule:**
   ```bash
   jtk auto create --file /tmp/auto-field.json
   ```
   Capture UUID → `$FIELD_UUID`

3. **Verify and export:**
   ```bash
   jtk auto get $FIELD_UUID --full
   jtk auto export $FIELD_UUID > /tmp/auto-field-export.json
   cat /tmp/auto-field-export.json | jq '.rule.components[0].value'
   ```

4. **Record findings:**
   - Did the API accept `schemaVersion: 3` for `jira.issue.condition`? ____
   - Did the API accept the `selectedField`/`compareValue` structure? ____
   - Did the API modify any fields or add new ones? ____
   - What schemaVersion does the export show? ____

5. **Cleanup:**
   ```bash
   jtk auto disable $FIELD_UUID
   jq '.rule.name = "[DELETEME] Field Condition Rule"' /tmp/auto-field-export.json > /tmp/auto-field-del.json
   jtk auto update $FIELD_UUID --file /tmp/auto-field-del.json
   ```

---

## Test 3: Comparator + Variable Rule

Creates a rule using the pattern from real backups: extract a custom field to a variable, then compare.

1. **Write the JSON file:**
   ```bash
   cat > /tmp/auto-comparator.json << 'EOF'
   {
     "rule": {
       "name": "[Test] Comparator Variable Rule",
       "state": "DISABLED",
       "description": "Integration test: variable extraction + comparator condition",
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
         }
       ],
       "ruleScope": {"resources": ["$PROJECT_ARI"]}
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
   jtk auto get $COMP_UUID --full
   jtk auto export $COMP_UUID > /tmp/auto-comparator-export.json
   ```

3. **Record findings:**
   - Did the API accept the custom `id` for the variable? ____
   - Did the comparator condition survive round-trip? ____
   - Verify: `jq '.rule.components[1].value' /tmp/auto-comparator-export.json`

4. **Cleanup:**
   ```bash
   jtk auto disable $COMP_UUID
   jq '.rule.name = "[DELETEME] Comparator Variable Rule"' /tmp/auto-comparator-export.json > /tmp/auto-comp-del.json
   jtk auto update $COMP_UUID --file /tmp/auto-comp-del.json
   ```

---

## Test 4: If/Else Block Rule

Creates a rule with if/else branching — the most complex structure.

1. **Write the JSON file:**
   ```bash
   cat > /tmp/auto-ifelse.json << 'EOF'
   {
     "rule": {
       "name": "[Test] If/Else Block Rule",
       "state": "DISABLED",
       "description": "Integration test: if/else branching with comparator conditions",
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
       "ruleScope": {"resources": ["$PROJECT_ARI"]}
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
   jtk auto get $IFELSE_UUID --full
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
   jtk auto disable $IFELSE_UUID
   jq '.rule.name = "[DELETEME] If/Else Block Rule"' /tmp/auto-ifelse-export.json > /tmp/auto-ifelse-del.json
   jtk auto update $IFELSE_UUID --file /tmp/auto-ifelse-del.json
   ```

---

## Test 5: Multi-Condition Rule (AND)

Creates a rule with multiple conditions: platform = Q2 AND product includes CheckSync.

1. **Write the JSON file:**
   ```bash
   cat > /tmp/auto-multi.json << 'EOF'
   {
     "rule": {
       "name": "[Test] Multi-Condition AND Rule",
       "state": "DISABLED",
       "description": "Integration test: platform = Q2 AND products includes CheckSync",
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
         }
       ],
       "ruleScope": {"resources": ["$PROJECT_ARI"]}
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
   jtk auto get $MULTI_UUID --full
   jtk auto export $MULTI_UUID > /tmp/auto-multi-export.json
   ```

3. **Record findings:**
   - Did the JQL condition with custom field names work? ____
   - Did the multi-select `in (...)` syntax work? ____

4. **Cleanup:**
   ```bash
   jtk auto disable $MULTI_UUID
   jq '.rule.name = "[DELETEME] Multi-Condition AND Rule"' /tmp/auto-multi-export.json > /tmp/auto-multi-del.json
   jtk auto update $MULTI_UUID --file /tmp/auto-multi-del.json
   ```

---

## Test 6: Round-Trip Fidelity

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
   jtk auto disable $RT_UUID
   jq '.rule.name = "[DELETEME] Round-Trip Copy"' /tmp/auto-rt-copy.json > /tmp/auto-rt-del.json
   jtk auto update $RT_UUID --file /tmp/auto-rt-del.json
   ```

---

## Test 7: Edit Field Action

Creates a rule with a `jira.issue.edit` action that sets a custom field.

1. **Write the JSON file:**
   ```bash
   cat > /tmp/auto-edit.json << 'EOF'
   {
     "rule": {
       "name": "[Test] Edit Field Action Rule",
       "state": "DISABLED",
       "description": "Integration test: edit issue field action",
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
       "ruleScope": {"resources": ["$PROJECT_ARI"]}
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
   jtk auto get $EDIT_UUID --full
   jtk auto export $EDIT_UUID > /tmp/auto-edit-export.json
   jq '.rule.components[0].value.operations' /tmp/auto-edit-export.json
   ```

3. **Record findings:**
   - Did `schemaVersion: 10` for `jira.issue.edit` work? ____
   - Did the operations array structure survive? ____
   - Did field reference by NAME work? ____

4. **Cleanup:**
   ```bash
   jtk auto disable $EDIT_UUID
   jq '.rule.name = "[DELETEME] Edit Field Action Rule"' /tmp/auto-edit-export.json > /tmp/auto-edit-del.json
   jtk auto update $EDIT_UUID --file /tmp/auto-edit-del.json
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

List all test rules and ensure they're disabled and renamed:

```bash
jtk auto list | grep -E "\[Test\]|\[DELETEME\]"
```

All test rules should be DISABLED with `[DELETEME]` prefix. Delete them manually in the Jira UI (the Automation REST API does not support deleting rules).

---

## Findings Log

Record findings from each test run below. These feed back into unit test assertions.

### Run 1: ____-__-__

**Test 1 (JQL Condition):**
- API accepted: ____
- Schema version returned: ____
- Fields added by API: ____

**Test 2 (Field Condition):**
- API accepted: ____
- Schema version returned: ____
- Fields added by API: ____

**Test 3 (Comparator + Variable):**
- API accepted: ____
- Variable ID preserved: ____
- Comparator survived: ____

**Test 4 (If/Else Block):**
- Nested structure survived: ____
- conditionMatchType preserved: ____
- Else block preserved: ____

**Test 5 (Multi-Condition AND):**
- JQL with custom field names: ____
- Multi-select `in (...)`: ____

**Test 6 (Round-Trip):**
- Component types match: ____
- API-added fields: ____

**Test 7 (Edit Field Action):**
- Schema version 10 accepted: ____
- Operations array: ____
- Field by NAME: ____

**Unit test updates needed:**
- [ ] ____
- [ ] ____
- [ ] ____
