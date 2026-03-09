package api //nolint:revive // package name is intentional

import (
	"encoding/json"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

// ---------------------------------------------------------------------------
// RuleBuilder tests
// ---------------------------------------------------------------------------

func TestRuleBuilder_Build(t *testing.T) {
	t.Parallel()

	t.Run("minimal rule", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("Test Rule").WithAuthor("test-account-id")
		data, err := b.Build()
		testutil.RequireNoError(t, err)

		var payload map[string]json.RawMessage
		testutil.RequireNoError(t, json.Unmarshal(data, &payload))

		// Verify envelope structure
		testutil.NotNil(t, payload["rule"])
		testutil.NotNil(t, payload["connections"])

		var rule map[string]any
		testutil.RequireNoError(t, json.Unmarshal(payload["rule"], &rule))
		testutil.Equal(t, rule["name"], "Test Rule")
		testutil.Equal(t, rule["state"], "DISABLED")
		testutil.Equal(t, rule["notifyOnError"], "FIRSTERROR")
		testutil.Equal(t, rule["canOtherRuleTrigger"], false)

		// Components should be empty array, not null
		components, ok := rule["components"].([]any)
		testutil.True(t, ok)
		testutil.Len(t, components, 0)
	})

	t.Run("empty name returns error", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("")
		_, err := b.Build()
		testutil.RequireError(t, err)
		testutil.Contains(t, err.Error(), "rule name is required")
	})

	t.Run("trigger with no action returns error", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("No Action Rule").WithAuthor("test-account-id").
			WithTrigger(IssueCreatedTrigger()).
			AddComponent(JQLCondition("project = TEST"))
		_, err := b.Build()
		testutil.RequireError(t, err)
		testutil.Contains(t, err.Error(), "must have at least one action")
	})

	t.Run("trigger with actions only in IfElseBlock passes validation", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("IfElse Only").WithAuthor("test-account-id").
			WithTrigger(IssueCreatedTrigger()).
			AddComponent(IfElseBlock(
				IfBlock{
					MatchType:  "ALL",
					Conditions: []RuleComponent{ComparatorCondition("{{x}}", "1", "EQUALS")},
					Actions:    []RuleComponent{CommentOnIssue("matched")},
				},
			))
		_, err := b.Build()
		testutil.RequireNoError(t, err)
	})

	t.Run("trigger with empty IfElseBlock returns error", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("Empty Branch").WithAuthor("test-account-id").
			WithTrigger(IssueCreatedTrigger()).
			AddComponent(IfElseBlock())
		_, err := b.Build()
		testutil.RequireError(t, err)
		testutil.Contains(t, err.Error(), "must have at least one action")
	})

	t.Run("trigger with IfElseBlock with no actions returns error", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("No Actions Branch").WithAuthor("test-account-id").
			WithTrigger(IssueCreatedTrigger()).
			AddComponent(IfElseBlock(
				IfBlock{
					MatchType:  "ALL",
					Conditions: []RuleComponent{ComparatorCondition("{{x}}", "1", "EQUALS")},
					Actions:    []RuleComponent{},
				},
			))
		_, err := b.Build()
		testutil.RequireError(t, err)
		testutil.Contains(t, err.Error(), "must have at least one action")
	})

	t.Run("trigger with IfElseBlock with empty MatchType returns error", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("Bad MatchType").WithAuthor("test-account-id").
			WithTrigger(IssueCreatedTrigger()).
			AddComponent(IfElseBlock(
				IfBlock{
					MatchType:  "",
					Conditions: []RuleComponent{ComparatorCondition("{{x}}", "1", "EQUALS")},
					Actions:    []RuleComponent{CommentOnIssue("matched")},
				},
			))
		_, err := b.Build()
		testutil.RequireError(t, err)
		testutil.Contains(t, err.Error(), "MatchType must be")
	})

	t.Run("trigger with IfElseBlock with invalid MatchType returns error", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("Bad MatchType").WithAuthor("test-account-id").
			WithTrigger(IssueCreatedTrigger()).
			AddComponent(IfElseBlock(
				IfBlock{
					MatchType:  "NONE",
					Conditions: []RuleComponent{ComparatorCondition("{{x}}", "1", "EQUALS")},
					Actions:    []RuleComponent{CommentOnIssue("matched")},
				},
			))
		_, err := b.Build()
		testutil.RequireError(t, err)
		testutil.Contains(t, err.Error(), `got "NONE"`)
	})

	t.Run("with description", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("Named Rule").WithAuthor("test-account-id").WithDescription("My description")
		data, err := b.Build()
		testutil.RequireNoError(t, err)

		rule := extractRule(t, data)
		testutil.Equal(t, rule["description"], "My description")
	})

	t.Run("with state ENABLED", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("Enabled Rule").WithAuthor("test-account-id").WithState("ENABLED")
		data, err := b.Build()
		testutil.RequireNoError(t, err)

		rule := extractRule(t, data)
		testutil.Equal(t, rule["state"], "ENABLED")
	})

	t.Run("with project scope", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("Scoped Rule").WithAuthor("test-account-id").
			ForProjects("ari:cloud:jira:abc:project/10022", "ari:cloud:jira:abc:project/10023")
		data, err := b.Build()
		testutil.RequireNoError(t, err)

		rule := extractRule(t, data)
		aris, ok := rule["ruleScopeARIs"].([]any)
		testutil.True(t, ok)
		testutil.Len(t, aris, 2)
		testutil.Equal(t, aris[0], "ari:cloud:jira:abc:project/10022")
	})

	t.Run("no project scope omits ruleScopeARIs", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("Global Rule").WithAuthor("test-account-id")
		data, err := b.Build()
		testutil.RequireNoError(t, err)

		rule := extractRule(t, data)
		testutil.Nil(t, rule["ruleScopeARIs"])
	})

	t.Run("with trigger and components", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("Full Rule").WithAuthor("test-account-id").
			WithTrigger(IssueCreatedTrigger()).
			AddComponent(JQLCondition("project = TEST")).
			AddComponent(CommentOnIssue("Hello"))
		data, err := b.Build()
		testutil.RequireNoError(t, err)

		rule := extractRule(t, data)
		testutil.NotNil(t, rule["trigger"])
		components, ok := rule["components"].([]any)
		testutil.True(t, ok)
		testutil.Len(t, components, 2)
	})

	t.Run("allow other rule trigger", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("Chainable Rule").WithAuthor("test-account-id").AllowOtherRuleTrigger(true)
		data, err := b.Build()
		testutil.RequireNoError(t, err)

		rule := extractRule(t, data)
		testutil.Equal(t, rule["canOtherRuleTrigger"], true)
	})

	t.Run("notify on error policy", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("Noisy Rule").WithAuthor("test-account-id").NotifyOnError("ALWAYS")
		data, err := b.Build()
		testutil.RequireNoError(t, err)

		rule := extractRule(t, data)
		testutil.Equal(t, rule["notifyOnError"], "ALWAYS")
	})

	t.Run("connections is empty array", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("Test").WithAuthor("test-account-id")
		data, err := b.Build()
		testutil.RequireNoError(t, err)

		var payload map[string]json.RawMessage
		testutil.RequireNoError(t, json.Unmarshal(data, &payload))

		var connections []any
		testutil.RequireNoError(t, json.Unmarshal(payload["connections"], &connections))
		testutil.Len(t, connections, 0)
	})
}

func TestRuleBuilder_AuthorActorWriteAccess(t *testing.T) {
	t.Parallel()

	t.Run("missing author returns error", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("No Author Rule")
		_, err := b.Build()
		testutil.RequireError(t, err)
		testutil.Contains(t, err.Error(), "authorAccountId is required")
	})

	t.Run("with author sets authorAccountId and default actor", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("Author Rule").WithAuthor("abc-123")
		data, err := b.Build()
		testutil.RequireNoError(t, err)

		rule := extractRule(t, data)
		testutil.Equal(t, rule["authorAccountId"], "abc-123")

		actor := rule["actor"].(map[string]any)
		testutil.Equal(t, actor["type"], "ACCOUNT_ID")
		testutil.Equal(t, actor["actor"], "abc-123")
	})

	t.Run("with explicit actor overrides default", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("Actor Rule").
			WithAuthor("human-123").
			WithActor("service-account-456")
		data, err := b.Build()
		testutil.RequireNoError(t, err)

		rule := extractRule(t, data)
		testutil.Equal(t, rule["authorAccountId"], "human-123")

		actor := rule["actor"].(map[string]any)
		testutil.Equal(t, actor["type"], "ACCOUNT_ID")
		testutil.Equal(t, actor["actor"], "service-account-456")
	})

	t.Run("writeAccessType defaults to UNRESTRICTED", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("Default Access").WithAuthor("abc-123")
		data, err := b.Build()
		testutil.RequireNoError(t, err)

		rule := extractRule(t, data)
		testutil.Equal(t, rule["writeAccessType"], "UNRESTRICTED")
	})

	t.Run("with custom writeAccessType", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("Restricted Rule").
			WithAuthor("abc-123").
			WithWriteAccessType("RESTRICTED")
		data, err := b.Build()
		testutil.RequireNoError(t, err)

		rule := extractRule(t, data)
		testutil.Equal(t, rule["writeAccessType"], "RESTRICTED")
	})
}

// ---------------------------------------------------------------------------
// Trigger tests
// ---------------------------------------------------------------------------

func TestIssueCreatedTrigger(t *testing.T) {
	t.Parallel()

	t.Run("no filters", func(t *testing.T) {
		t.Parallel()
		trigger := IssueCreatedTrigger()
		testutil.Equal(t, trigger.Component, "TRIGGER")
		testutil.Equal(t, trigger.Type, "jira.issue.event.trigger:created")
		testutil.Equal(t, trigger.SchemaVersion, 1)

		val := unmarshalValue(t, trigger.Value)
		testutil.Equal(t, val["eventKey"], "jira:issue_created")
		testutil.Equal(t, val["issueEvent"], "issue_created")
		testutil.Nil(t, val["eventFilters"])
	})

	t.Run("with project filters", func(t *testing.T) {
		t.Parallel()
		trigger := IssueCreatedTrigger(
			"ari:cloud:jira:abc:project/10022",
			"ari:cloud:jira:abc:project/10023",
		)

		val := unmarshalValue(t, trigger.Value)
		filters, ok := val["eventFilters"].([]any)
		testutil.True(t, ok)
		testutil.Len(t, filters, 2)
		testutil.Equal(t, filters[0], "ari:cloud:jira:abc:project/10022")
	})

	t.Run("has empty children and conditions", func(t *testing.T) {
		t.Parallel()
		trigger := IssueCreatedTrigger()
		assertEmptyJSONArray(t, trigger.Children)
		assertEmptyJSONArray(t, trigger.Conditions)
	})
}

func TestIssueTransitionedTrigger(t *testing.T) {
	t.Parallel()

	t.Run("single status", func(t *testing.T) {
		t.Parallel()
		trigger := IssueTransitionedTrigger("Done")
		testutil.Equal(t, trigger.Component, "TRIGGER")
		testutil.Equal(t, trigger.Type, "jira.issue.event.trigger:transitioned")

		val := unmarshalValue(t, trigger.Value)
		toStatus, ok := val["toStatus"].([]any)
		testutil.True(t, ok)
		testutil.Len(t, toStatus, 1)

		status := toStatus[0].(map[string]any)
		testutil.Equal(t, status["type"], "NAME")
		testutil.Equal(t, status["value"], "Done")
	})

	t.Run("multiple statuses", func(t *testing.T) {
		t.Parallel()
		trigger := IssueTransitionedTrigger("Done", "Deployed", "Canceled")

		val := unmarshalValue(t, trigger.Value)
		toStatus, ok := val["toStatus"].([]any)
		testutil.True(t, ok)
		testutil.Len(t, toStatus, 3)

		status2 := toStatus[1].(map[string]any)
		testutil.Equal(t, status2["value"], "Deployed")
	})

	t.Run("fromStatus is empty array", func(t *testing.T) {
		t.Parallel()
		trigger := IssueTransitionedTrigger("Done")

		val := unmarshalValue(t, trigger.Value)
		fromStatus, ok := val["fromStatus"].([]any)
		testutil.True(t, ok)
		testutil.Len(t, fromStatus, 0)
	})
}

func TestManualTrigger(t *testing.T) {
	t.Parallel()

	t.Run("no groups", func(t *testing.T) {
		t.Parallel()
		trigger := ManualTrigger()
		testutil.Equal(t, trigger.Type, "jira.manual.trigger.issue")

		val := unmarshalValue(t, trigger.Value)
		testutil.Nil(t, val["groups"])
	})

	t.Run("with group restrictions", func(t *testing.T) {
		t.Parallel()
		trigger := ManualTrigger("jira-admins", "developers")

		val := unmarshalValue(t, trigger.Value)
		groups, ok := val["groups"].([]any)
		testutil.True(t, ok)
		testutil.Len(t, groups, 2)
		testutil.Equal(t, groups[0], "jira-admins")
	})
}

func TestScheduledTrigger(t *testing.T) {
	t.Parallel()
	trigger := ScheduledTrigger("project = TEST AND status = Open", "0 9 * * 1-5")
	testutil.Equal(t, trigger.Type, "jira.jql.scheduled")

	val := unmarshalValue(t, trigger.Value)
	testutil.Equal(t, val["jql"], "project = TEST AND status = Open")
	testutil.Equal(t, val["cron"], "0 9 * * 1-5")
}

func TestFieldChangedTrigger(t *testing.T) {
	t.Parallel()
	trigger := FieldChangedTrigger("customfield_10037")
	testutil.Equal(t, trigger.Type, "jira.issue.field.changed")

	val := unmarshalValue(t, trigger.Value)
	testutil.Equal(t, val["changeType"], "ANY_CHANGE")
	fields, ok := val["fields"].([]any)
	testutil.True(t, ok)
	testutil.Len(t, fields, 1)
	field := fields[0].(map[string]any)
	testutil.Equal(t, field["value"], "customfield_10037")
}

// ---------------------------------------------------------------------------
// Condition tests
// ---------------------------------------------------------------------------

func TestJQLCondition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		jql  string
	}{
		{"basic", "project = TEST"},
		{"with quotes", `"Banking Platform" = "Q2" AND "Products" in ("CheckSync")`},
		{"with special chars", `status = "In Progress" AND labels in ("high-priority")`},
		{"with filter reference", `filter=10039 AND issueLinkType = "blocks"`},
		{"empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := JQLCondition(tt.jql)
			testutil.Equal(t, c.Component, "CONDITION")
			testutil.Equal(t, c.Type, "jira.jql.condition")
			testutil.Equal(t, c.SchemaVersion, 1)

			// JQL condition value is a JSON string
			var val string
			testutil.RequireNoError(t, json.Unmarshal(c.Value, &val))
			testutil.Equal(t, val, tt.jql)

			assertEmptyJSONArray(t, c.Children)
			assertEmptyJSONArray(t, c.Conditions)
		})
	}
}

func TestComparatorCondition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		first    string
		second   string
		operator string
	}{
		{"EQUALS with smart value", "{{bankPlatform}}", "Q2", "EQUALS"},
		{"NOT_EQUALS", "{{bankPlatform}}", "Banno", "NOT_EQUALS"},
		{"GREATER_THAN numeric", "{{lookupIssues.size}}", "0", "GREATER_THAN"},
		{"LESS_THAN", "{{count}}", "100", "LESS_THAN"},
		{"CONTAINS string", "{{issue.summary}}", "urgent", "CONTAINS"},
		{"EQUALS with zero", "{{lookupIssues.size}}", "0", "EQUALS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := ComparatorCondition(tt.first, tt.second, tt.operator)
			testutil.Equal(t, c.Component, "CONDITION")
			testutil.Equal(t, c.Type, "jira.comparator.condition")
			testutil.Equal(t, c.SchemaVersion, 1)

			var val ComparatorConditionValue
			testutil.RequireNoError(t, json.Unmarshal(c.Value, &val))
			testutil.Equal(t, val.First, tt.first)
			testutil.Equal(t, val.Second, tt.second)
			testutil.Equal(t, val.Operator, tt.operator)
		})
	}
}

func TestIfElseBlock(t *testing.T) {
	t.Parallel()

	t.Run("single if block", func(t *testing.T) {
		t.Parallel()
		block := IfElseBlock(
			IfBlock{
				MatchType:  "ALL",
				Conditions: []RuleComponent{ComparatorCondition("{{x}}", "1", "EQUALS")},
				Actions:    []RuleComponent{CommentOnIssue("matched")},
			},
		)

		testutil.Equal(t, block.Component, "CONDITION")
		testutil.Equal(t, block.Type, "jira.condition.container.block")
		testutil.Equal(t, block.SchemaVersion, 1)

		// Value should be empty object
		var val map[string]any
		testutil.RequireNoError(t, json.Unmarshal(block.Value, &val))
		testutil.Len(t, val, 0)

		// Children should have one CONDITION_BLOCK
		var children []map[string]any
		testutil.RequireNoError(t, json.Unmarshal(block.Children, &children))
		testutil.Len(t, children, 1)

		child := children[0]
		testutil.Equal(t, child["component"], "CONDITION_BLOCK")
		testutil.Equal(t, child["type"], "jira.condition.if.block")

		childValue := child["value"].(map[string]any)
		testutil.Equal(t, childValue["conditionMatchType"], "ALL")

		conditions, ok := child["conditions"].([]any)
		testutil.True(t, ok)
		testutil.Len(t, conditions, 1)

		actions, ok := child["children"].([]any)
		testutil.True(t, ok)
		testutil.Len(t, actions, 1)
	})

	t.Run("if/else-if with multiple blocks", func(t *testing.T) {
		t.Parallel()
		block := IfElseBlock(
			IfBlock{
				MatchType:  "ALL",
				Conditions: []RuleComponent{ComparatorCondition("{{platform}}", "Banno", "EQUALS")},
				Actions:    []RuleComponent{CommentOnIssue("Banno path")},
			},
			IfBlock{
				MatchType:  "ALL",
				Conditions: []RuleComponent{ComparatorCondition("{{platform}}", "Q2", "EQUALS")},
				Actions:    []RuleComponent{CommentOnIssue("Q2 path")},
			},
		)

		var children []map[string]any
		testutil.RequireNoError(t, json.Unmarshal(block.Children, &children))
		testutil.Len(t, children, 2)

		// Verify each block has different conditions
		cond0 := extractNestedConditions(t, children[0])
		testutil.Len(t, cond0, 1)

		cond1 := extractNestedConditions(t, children[1])
		testutil.Len(t, cond1, 1)
	})

	t.Run("ANY match type for OR logic", func(t *testing.T) {
		t.Parallel()
		block := IfElseBlock(
			IfBlock{
				MatchType: "ANY",
				Conditions: []RuleComponent{
					ComparatorCondition("{{platform}}", "Banno", "EQUALS"),
					ComparatorCondition("{{platform}}", "Q2", "EQUALS"),
				},
				Actions: []RuleComponent{CommentOnIssue("matched either")},
			},
		)

		var children []map[string]any
		testutil.RequireNoError(t, json.Unmarshal(block.Children, &children))
		childValue := children[0]["value"].(map[string]any)
		testutil.Equal(t, childValue["conditionMatchType"], "ANY")

		conditions := extractNestedConditions(t, children[0])
		testutil.Len(t, conditions, 2)
	})

	t.Run("else block with no conditions", func(t *testing.T) {
		t.Parallel()
		block := IfElseBlock(
			IfBlock{
				MatchType:  "ALL",
				Conditions: []RuleComponent{ComparatorCondition("{{x}}", "1", "EQUALS")},
				Actions:    []RuleComponent{CommentOnIssue("if")},
			},
			IfBlock{
				MatchType:  "ALL",
				Conditions: []RuleComponent{},
				Actions:    []RuleComponent{CommentOnIssue("else")},
			},
		)

		var children []map[string]any
		testutil.RequireNoError(t, json.Unmarshal(block.Children, &children))
		testutil.Len(t, children, 2)

		// Else block has empty conditions
		conditions := extractNestedConditions(t, children[1])
		testutil.Len(t, conditions, 0)
	})
}

// ---------------------------------------------------------------------------
// Action tests
// ---------------------------------------------------------------------------

func TestCreateVariable(t *testing.T) {
	t.Parallel()

	t.Run("basic smart value", func(t *testing.T) {
		t.Parallel()
		action := CreateVariable("bankPlatform", "{{triggerIssue.customField_10037}}")
		testutil.Equal(t, action.Component, "ACTION")
		testutil.Equal(t, action.Type, "jira.create.variable")
		testutil.Equal(t, action.SchemaVersion, 1)

		val := unmarshalValue(t, action.Value)
		name := val["name"].(map[string]any)
		testutil.Equal(t, name["type"], "FREE")
		testutil.Equal(t, name["value"], "bankPlatform")

		query := val["query"].(map[string]any)
		testutil.Equal(t, query["type"], "SMART")
		testutil.Equal(t, query["value"], "{{triggerIssue.customField_10037}}")

		testutil.Equal(t, val["type"], "SMART")
		testutil.Equal(t, val["lazy"], false)
	})

	t.Run("has unique id", func(t *testing.T) {
		t.Parallel()
		a1 := CreateVariable("var1", "{{a}}")
		a2 := CreateVariable("var2", "{{b}}")

		v1 := unmarshalValue(t, a1.Value)
		v2 := unmarshalValue(t, a2.Value)

		id1 := v1["id"].(string)
		id2 := v2["id"].(string)
		testutil.True(t, id1 != id2)
		testutil.HasPrefix(t, id1, "_customsmartvalue_id_")
	})
}

func TestEditIssueFields(t *testing.T) {
	t.Parallel()

	t.Run("single field by ID", func(t *testing.T) {
		t.Parallel()
		action := EditIssueFields(true,
			SetField("customfield_10037", "com.atlassian.jira.plugin.system.customfieldtypes:select",
				map[string]string{"value": "Q2"}),
		)
		testutil.Equal(t, action.Component, "ACTION")
		testutil.Equal(t, action.Type, "jira.issue.edit")
		testutil.Equal(t, action.SchemaVersion, 10)

		val := unmarshalValue(t, action.Value)
		testutil.Equal(t, val["sendNotifications"], true)
		testutil.Nil(t, val["advancedFields"])

		ops, ok := val["operations"].([]any)
		testutil.True(t, ok)
		testutil.Len(t, ops, 1)

		op := ops[0].(map[string]any)
		field := op["field"].(map[string]any)
		testutil.Equal(t, field["type"], "ID")
		testutil.Equal(t, field["value"], "customfield_10037")
		testutil.Equal(t, op["type"], "SET")
	})

	t.Run("field by name", func(t *testing.T) {
		t.Parallel()
		action := EditIssueFields(false,
			SetFieldByName("Meta Status", "com.atlassian.jira.plugin.system.customfieldtypes:multicheckboxes",
				[]any{}),
		)

		val := unmarshalValue(t, action.Value)
		testutil.Equal(t, val["sendNotifications"], false)

		ops := val["operations"].([]any)
		op := ops[0].(map[string]any)
		field := op["field"].(map[string]any)
		testutil.Equal(t, field["type"], "NAME")
		testutil.Equal(t, field["value"], "Meta Status")
	})

	t.Run("multiple operations", func(t *testing.T) {
		t.Parallel()
		action := EditIssueFields(true,
			SetField("summary", "summary", "Updated summary"),
			SetFieldByName("Meta Status", "com.atlassian.jira.plugin.system.customfieldtypes:multicheckboxes", []any{}),
		)

		val := unmarshalValue(t, action.Value)
		ops := val["operations"].([]any)
		testutil.Len(t, ops, 2)
	})
}

func TestTransitionIssue(t *testing.T) {
	t.Parallel()
	action := TransitionIssue("In Progress")
	testutil.Equal(t, action.Component, "ACTION")
	testutil.Equal(t, action.Type, "jira.issue.transition")

	val := unmarshalValue(t, action.Value)
	dest := val["destinationStatus"].(map[string]any)
	testutil.Equal(t, dest["type"], "NAME")
	testutil.Equal(t, dest["value"], "In Progress")
}

func TestCommentOnIssue(t *testing.T) {
	t.Parallel()
	action := CommentOnIssue("Hello from automation")
	testutil.Equal(t, action.Component, "ACTION")
	testutil.Equal(t, action.Type, "jira.issue.comment")

	val := unmarshalValue(t, action.Value)
	testutil.Equal(t, val["comment"], "Hello from automation")
}

func TestAssignIssue(t *testing.T) {
	t.Parallel()

	t.Run("static account ID", func(t *testing.T) {
		t.Parallel()
		action := AssignIssue("5b10ac8d82e05b22cc7d4ef5")

		val := unmarshalValue(t, action.Value)
		assignee := val["assignee"].(map[string]any)
		testutil.Equal(t, assignee["type"], "SMART")
		testutil.Equal(t, assignee["value"], "5b10ac8d82e05b22cc7d4ef5")
	})

	t.Run("smart value", func(t *testing.T) {
		t.Parallel()
		action := AssignIssue("{{reporter.accountId}}")

		val := unmarshalValue(t, action.Value)
		assignee := val["assignee"].(map[string]any)
		testutil.Equal(t, assignee["value"], "{{reporter.accountId}}")
	})
}

func TestLookupIssues(t *testing.T) {
	t.Parallel()

	t.Run("basic JQL", func(t *testing.T) {
		t.Parallel()
		action := LookupIssues("blockedIssues", `issue in linkedIssues("TEST-1", "blocks")`)
		testutil.Equal(t, action.Component, "ACTION")
		testutil.Equal(t, action.Type, "jira.lookup.issues")

		val := unmarshalValue(t, action.Value)
		name := val["name"].(map[string]any)
		testutil.Equal(t, name["value"], "blockedIssues")

		query := val["query"].(map[string]any)
		testutil.Equal(t, query["type"], "SMART")
		testutil.Equal(t, query["value"], `issue in linkedIssues("TEST-1", "blocks")`)

		testutil.Equal(t, val["type"], "JQL")
		testutil.Equal(t, val["lazy"], false)
	})

	t.Run("smart value JQL", func(t *testing.T) {
		t.Parallel()
		action := LookupIssues("childIssues", "parent in ({{triggerIssue.key}})")

		val := unmarshalValue(t, action.Value)
		query := val["query"].(map[string]any)
		testutil.Equal(t, query["value"], "parent in ({{triggerIssue.key}})")
	})
}

// ---------------------------------------------------------------------------
// Round-trip tests
// ---------------------------------------------------------------------------

func TestRuleBuilder_RoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("marshal and unmarshal preserves structure", func(t *testing.T) {
		t.Parallel()
		b := NewRuleBuilder("Round Trip Rule").WithAuthor("test-account-id").
			WithDescription("Testing round-trip fidelity").
			WithTrigger(IssueCreatedTrigger()).
			AddComponent(JQLCondition("project = TEST")).
			AddComponent(CreateVariable("myVar", "{{triggerIssue.summary}}")).
			AddComponent(ComparatorCondition("{{myVar}}", "test", "EQUALS")).
			ForProjects("ari:cloud:jira:abc:project/10022")

		data, err := b.Build()
		testutil.RequireNoError(t, err)

		// Verify it's valid JSON that can be round-tripped
		var parsed map[string]any
		testutil.RequireNoError(t, json.Unmarshal(data, &parsed))

		repacked, err := json.Marshal(parsed)
		testutil.RequireNoError(t, err)

		var reparsed map[string]any
		testutil.RequireNoError(t, json.Unmarshal(repacked, &reparsed))

		// Verify key fields survived
		rule := reparsed["rule"].(map[string]any)
		testutil.Equal(t, rule["name"], "Round Trip Rule")
		testutil.Equal(t, rule["description"], "Testing round-trip fidelity")

		components := rule["components"].([]any)
		testutil.Len(t, components, 3)
	})
}

func TestRuleBuilder_MatchesBackupStructure(t *testing.T) {
	t.Parallel()

	// Build a rule that structurally matches ON-MON_Unblock_Onboarding_Tickets
	// (trigger on transition, JQL condition, lookup + variable + branch)
	b := NewRuleBuilder("[Test] Unblock Onboarding").WithAuthor("test-account-id").
		WithDescription("Unblock next ticket in a chain").
		WithState("ENABLED").
		WithTrigger(IssueTransitionedTrigger("Deployed", "Canceled", "Done")).
		AddComponent(JQLCondition(`filter=10039 AND issueLinkType = "blocks"`)).
		AddComponent(LookupIssues("lookupIssues", `issue in linkedIssues({{triggerIssue.key}}, "blocks")`))

	data, err := b.Build()
	testutil.RequireNoError(t, err)

	rule := extractRule(t, data)

	// Verify trigger structure matches backup
	trigger := rule["trigger"].(map[string]any)
	testutil.Equal(t, trigger["component"], "TRIGGER")
	testutil.Equal(t, trigger["type"], "jira.issue.event.trigger:transitioned")

	triggerVal := trigger["value"].(map[string]any)
	toStatuses := triggerVal["toStatus"].([]any)
	testutil.Len(t, toStatuses, 3)

	// Verify components
	components := rule["components"].([]any)
	testutil.Len(t, components, 2)

	comp0 := components[0].(map[string]any)
	testutil.Equal(t, comp0["component"], "CONDITION")
	testutil.Equal(t, comp0["type"], "jira.jql.condition")

	comp1 := components[1].(map[string]any)
	testutil.Equal(t, comp1["component"], "ACTION")
	testutil.Equal(t, comp1["type"], "jira.lookup.issues")
}

func TestRuleBuilder_MatchesIfElseStructure(t *testing.T) {
	t.Parallel()

	// Build a rule that structurally matches ON-MON_Create_Onboarding_Tasks if/else pattern
	b := NewRuleBuilder("[Test] Platform Branching").WithAuthor("test-account-id").
		WithTrigger(IssueCreatedTrigger()).
		AddComponent(JQLCondition("project = 'ON' AND issuetype = Epic")).
		AddComponent(CreateVariable("bankPlatform", "{{triggerIssue.customField_10037}}")).
		AddComponent(IfElseBlock(
			IfBlock{
				MatchType:  "ALL",
				Conditions: []RuleComponent{ComparatorCondition("{{bankPlatform}}", "Banno", "EQUALS")},
				Actions:    []RuleComponent{CommentOnIssue("Banno setup")},
			},
			IfBlock{
				MatchType:  "ALL",
				Conditions: []RuleComponent{ComparatorCondition("{{bankPlatform}}", "Q2", "EQUALS")},
				Actions:    []RuleComponent{CommentOnIssue("Q2 setup")},
			},
		))

	data, err := b.Build()
	testutil.RequireNoError(t, err)

	rule := extractRule(t, data)
	components := rule["components"].([]any)
	testutil.Len(t, components, 3)

	// Third component should be the if/else container
	ifElse := components[2].(map[string]any)
	testutil.Equal(t, ifElse["component"], "CONDITION")
	testutil.Equal(t, ifElse["type"], "jira.condition.container.block")

	// Container should have two CONDITION_BLOCK children
	children := ifElse["children"].([]any)
	testutil.Len(t, children, 2)

	block0 := children[0].(map[string]any)
	testutil.Equal(t, block0["component"], "CONDITION_BLOCK")
	testutil.Equal(t, block0["type"], "jira.condition.if.block")

	block0Val := block0["value"].(map[string]any)
	testutil.Equal(t, block0Val["conditionMatchType"], "ALL")

	// First block: condition checks Banno
	block0Conditions := block0["conditions"].([]any)
	testutil.Len(t, block0Conditions, 1)

	cond := block0Conditions[0].(map[string]any)
	testutil.Equal(t, cond["type"], "jira.comparator.condition")

	condVal := cond["value"].(map[string]any)
	testutil.Equal(t, condVal["first"], "{{bankPlatform}}")
	testutil.Equal(t, condVal["second"], "Banno")
	testutil.Equal(t, condVal["operator"], "EQUALS")

	// First block: action is a comment
	block0Actions := block0["children"].([]any)
	testutil.Len(t, block0Actions, 1)

	action := block0Actions[0].(map[string]any)
	testutil.Equal(t, action["type"], "jira.issue.comment")
}

// ---------------------------------------------------------------------------
// Edge case tests
// ---------------------------------------------------------------------------

func TestRuleBuilder_UnicodeInValues(t *testing.T) {
	t.Parallel()
	b := NewRuleBuilder("Règle avec des accents").WithAuthor("test-account-id").
		AddComponent(JQLCondition(`summary ~ "prüfung" AND description ~ "日本語"`))

	data, err := b.Build()
	testutil.RequireNoError(t, err)

	rule := extractRule(t, data)
	testutil.Equal(t, rule["name"], "Règle avec des accents")
}

func TestRuleBuilder_LongJQL(t *testing.T) {
	t.Parallel()
	longJQL := "project = TEST AND ("
	for i := 0; i < 50; i++ {
		if i > 0 {
			longJQL += " OR "
		}
		longJQL += `summary ~ "keyword` + string(rune('A'+i%26)) + `"`
	}
	longJQL += ")"

	c := JQLCondition(longJQL)
	var val string
	testutil.RequireNoError(t, json.Unmarshal(c.Value, &val))
	testutil.Equal(t, val, longJQL)
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func extractRule(t *testing.T, data json.RawMessage) map[string]any {
	t.Helper()
	var payload map[string]json.RawMessage
	testutil.RequireNoError(t, json.Unmarshal(data, &payload))

	var rule map[string]any
	testutil.RequireNoError(t, json.Unmarshal(payload["rule"], &rule))
	return rule
}

func unmarshalValue(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var val map[string]any
	testutil.RequireNoError(t, json.Unmarshal(raw, &val))
	return val
}

func assertEmptyJSONArray(t *testing.T, raw json.RawMessage) {
	t.Helper()
	var arr []any
	testutil.RequireNoError(t, json.Unmarshal(raw, &arr))
	testutil.Len(t, arr, 0)
}

func extractNestedConditions(t *testing.T, block map[string]any) []any {
	t.Helper()
	conditions, ok := block["conditions"].([]any)
	testutil.True(t, ok)
	return conditions
}
