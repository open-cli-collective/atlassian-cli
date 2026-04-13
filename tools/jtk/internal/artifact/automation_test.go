package artifact

import (
	"encoding/json"
	"testing"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestProjectAutomationRule_AgentMode(t *testing.T) {
	t.Parallel()

	rule := &api.AutomationRule{
		UUID:        "abc-123-def",
		Name:        "Close stale issues",
		State:       "ENABLED",
		Description: "Closes issues after 30 days of inactivity",
		Labels:      []string{"cleanup", "maintenance"},
		Tags:        []string{"automated"},
		Components: []api.RuleComponent{
			{Component: "TRIGGER", Type: "scheduled"},
			{Component: "CONDITION", Type: "jql"},
			{Component: "ACTION", Type: "transition"},
		},
	}

	art := ProjectAutomationRule(rule, artifact.Agent)

	// Agent fields populated
	testutil.Equal(t, art.ID, "abc-123-def")
	testutil.Equal(t, art.Name, "Close stale issues")
	testutil.Equal(t, art.State, "ENABLED")
	testutil.Equal(t, art.ComponentSummary, "3 total — 1 trigger(s), 1 condition(s), 1 action(s)")

	// Full-only fields empty
	testutil.Equal(t, art.Description, "")
	testutil.Nil(t, art.Labels)
	testutil.Nil(t, art.Tags)
}

func TestProjectAutomationRule_FullMode(t *testing.T) {
	t.Parallel()

	rule := &api.AutomationRule{
		UUID:        "abc-123-def",
		Name:        "Close stale issues",
		State:       "ENABLED",
		Description: "Closes issues after 30 days of inactivity",
		Labels:      []string{"cleanup", "maintenance"},
		Tags:        []string{"automated"},
		Components: []api.RuleComponent{
			{Component: "TRIGGER", Type: "scheduled"},
			{Component: "ACTION", Type: "transition"},
		},
	}

	art := ProjectAutomationRule(rule, artifact.Full)

	// Agent fields populated
	testutil.Equal(t, art.ID, "abc-123-def")
	testutil.Equal(t, art.Name, "Close stale issues")
	testutil.Equal(t, art.State, "ENABLED")
	testutil.Equal(t, art.ComponentSummary, "2 total — 1 trigger(s), 1 action(s)")

	// Full-only fields populated
	testutil.Equal(t, art.Description, "Closes issues after 30 days of inactivity")
	testutil.Equal(t, len(art.Labels), 2)
	testutil.Equal(t, art.Labels[0], "cleanup")
	testutil.Equal(t, len(art.Tags), 1)
	testutil.Equal(t, art.Tags[0], "automated")
}

func TestProjectAutomationRule_EmptyOptionalFields(t *testing.T) {
	t.Parallel()

	rule := &api.AutomationRule{
		UUID:  "xyz-789",
		Name:  "Simple rule",
		State: "DISABLED",
		// No description, labels, tags, or components
	}

	art := ProjectAutomationRule(rule, artifact.Full)

	testutil.Equal(t, art.ID, "xyz-789")
	testutil.Equal(t, art.Name, "Simple rule")
	testutil.Equal(t, art.State, "DISABLED")
	testutil.Equal(t, art.ComponentSummary, "none")
	testutil.Equal(t, art.Description, "")
	testutil.Nil(t, art.Labels)
	testutil.Nil(t, art.Tags)
}

func TestProjectAutomationRule_IdentifierFallback(t *testing.T) {
	t.Parallel()

	t.Run("uses UUID when available", func(t *testing.T) {
		t.Parallel()
		rule := &api.AutomationRule{
			UUID:    "uuid-value",
			RuleKey: "rulekey-value",
			ID:      "123",
			Name:    "Test",
			State:   "ENABLED",
		}
		art := ProjectAutomationRule(rule, artifact.Agent)
		testutil.Equal(t, art.ID, "uuid-value")
	})

	t.Run("falls back to RuleKey", func(t *testing.T) {
		t.Parallel()
		rule := &api.AutomationRule{
			RuleKey: "rulekey-value",
			ID:      "123",
			Name:    "Test",
			State:   "ENABLED",
		}
		art := ProjectAutomationRule(rule, artifact.Agent)
		testutil.Equal(t, art.ID, "rulekey-value")
	})

	t.Run("falls back to numeric ID", func(t *testing.T) {
		t.Parallel()
		rule := &api.AutomationRule{
			ID:    "456",
			Name:  "Test",
			State: "ENABLED",
		}
		art := ProjectAutomationRule(rule, artifact.Agent)
		testutil.Equal(t, art.ID, "456")
	})
}

func TestProjectAutomationRule_JSONSerialization(t *testing.T) {
	t.Parallel()

	t.Run("agent mode omits full-only fields", func(t *testing.T) {
		t.Parallel()
		rule := &api.AutomationRule{
			UUID:        "abc",
			Name:        "Test",
			State:       "ENABLED",
			Description: "A description",
			Labels:      []string{"label1"},
			Tags:        []string{"tag1"},
		}
		art := ProjectAutomationRule(rule, artifact.Agent)

		data, err := json.Marshal(art)
		testutil.RequireNoError(t, err)

		var parsed map[string]any
		err = json.Unmarshal(data, &parsed)
		testutil.RequireNoError(t, err)

		_, exists := parsed["description"]
		testutil.False(t, exists)
		_, exists = parsed["labels"]
		testutil.False(t, exists)
		_, exists = parsed["tags"]
		testutil.False(t, exists)
	})

	t.Run("full mode includes all fields", func(t *testing.T) {
		t.Parallel()
		rule := &api.AutomationRule{
			UUID:        "abc",
			Name:        "Test",
			State:       "ENABLED",
			Description: "A description",
			Labels:      []string{"label1"},
			Tags:        []string{"tag1"},
		}
		art := ProjectAutomationRule(rule, artifact.Full)

		data, err := json.Marshal(art)
		testutil.RequireNoError(t, err)

		var parsed map[string]any
		err = json.Unmarshal(data, &parsed)
		testutil.RequireNoError(t, err)

		testutil.Equal(t, parsed["description"], "A description")
		labels, ok := parsed["labels"].([]any)
		testutil.True(t, ok)
		testutil.Equal(t, len(labels), 1)
	})
}

func TestSummarizeComponents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		components []api.RuleComponent
		expected   string
	}{
		{
			name:       "empty",
			components: nil,
			expected:   "none",
		},
		{
			name: "triggers only",
			components: []api.RuleComponent{
				{Component: "TRIGGER", Type: "scheduled"},
				{Component: "TRIGGER", Type: "manual"},
			},
			expected: "2 total — 2 trigger(s)",
		},
		{
			name: "actions only",
			components: []api.RuleComponent{
				{Component: "ACTION", Type: "transition"},
			},
			expected: "1 total — 1 action(s)",
		},
		{
			name: "mixed",
			components: []api.RuleComponent{
				{Component: "TRIGGER", Type: "scheduled"},
				{Component: "CONDITION", Type: "jql"},
				{Component: "CONDITION", Type: "field"},
				{Component: "ACTION", Type: "transition"},
				{Component: "ACTION", Type: "comment"},
			},
			expected: "5 total — 1 trigger(s), 2 condition(s), 2 action(s)",
		},
		{
			name: "with unknown component types",
			components: []api.RuleComponent{
				{Component: "TRIGGER", Type: "scheduled"},
				{Component: "BRANCH", Type: "parallel"},
				{Component: "ACTION", Type: "transition"},
			},
			expected: "3 total — 1 trigger(s), 1 action(s), 1 other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := summarizeComponents(tt.components)
			testutil.Equal(t, result, tt.expected)
		})
	}
}
