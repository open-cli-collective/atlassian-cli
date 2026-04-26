package api

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestExtractIssueFieldValues_TypedFields(t *testing.T) {
	t.Parallel()
	issue := &Issue{
		Key: "TEST-1",
		Fields: IssueFields{
			Summary:   "My summary",
			Status:    &Status{Name: "Open"},
			IssueType: &IssueType{Name: "Task"},
			Priority:  &Priority{Name: "High"},
			Assignee:  &User{DisplayName: "Alice"},
		},
	}
	fields := []Field{
		{ID: "summary", Name: "Summary", Schema: FieldSchema{Type: "string"}},
		{ID: "status", Name: "Status", Schema: FieldSchema{Type: "status"}},
		{ID: "issuetype", Name: "Issue Type", Schema: FieldSchema{Type: "issuetype"}},
		{ID: "priority", Name: "Priority", Schema: FieldSchema{Type: "priority"}},
		{ID: "assignee", Name: "Assignee", Schema: FieldSchema{Type: "user"}},
	}

	entries := ExtractIssueFieldValues(issue, fields)
	found := make(map[string]string)
	for _, e := range entries {
		found[e.ID] = e.Value
	}

	testutil.Equal(t, found["summary"], "My summary")
	testutil.Equal(t, found["status"], "Open")
	testutil.Equal(t, found["issuetype"], "Task")
	testutil.Equal(t, found["assignee"], "Alice")
}

func TestExtractIssueFieldValues_CustomFields(t *testing.T) {
	t.Parallel()
	issue := &Issue{
		Key: "TEST-1",
		Fields: IssueFields{
			Summary: "Test",
			CustomFields: map[string]any{
				"customfield_10035": float64(5),
				"customfield_10050": map[string]any{"value": "Platform"},
				"customfield_10099": nil,
			},
		},
	}
	fields := []Field{
		{ID: "summary", Name: "Summary", Schema: FieldSchema{Type: "string"}},
		{ID: "customfield_10035", Name: "Story Points", Schema: FieldSchema{Type: "number"}},
		{ID: "customfield_10050", Name: "Team", Schema: FieldSchema{Type: "option"}},
	}

	entries := ExtractIssueFieldValues(issue, fields)
	found := make(map[string]string)
	for _, e := range entries {
		found[e.ID] = e.Value
	}

	testutil.Equal(t, found["customfield_10035"], "5")
	testutil.Equal(t, found["customfield_10050"], "Platform")
	if _, ok := found["customfield_10099"]; ok {
		t.Error("nil custom field should be excluded")
	}
}

func TestExtractIssueFieldValues_Sorted(t *testing.T) {
	t.Parallel()
	issue := &Issue{
		Key: "TEST-1",
		Fields: IssueFields{
			Summary: "Test",
			Status:  &Status{Name: "Open"},
		},
	}
	entries := ExtractIssueFieldValues(issue, nil)
	for i := 1; i < len(entries); i++ {
		if entries[i].ID < entries[i-1].ID {
			t.Errorf("entries not sorted: %s < %s", entries[i].ID, entries[i-1].ID)
		}
	}
}

func TestFormatCustomFieldValue_Types(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"string", "hello", "hello"},
		{"float_int", float64(5), "5"},
		{"float_decimal", float64(3.14), "3.14"},
		{"option_value", map[string]any{"value": "Platform"}, "Platform"},
		{"option_name", map[string]any{"name": "High"}, "High"},
		{"user", map[string]any{"displayName": "Alice"}, "Alice"},
		{"array_strings", []any{"a", "b"}, "a, b"},
		{"array_options", []any{map[string]any{"value": "X"}, map[string]any{"value": "Y"}}, "X, Y"},
		{"bool_true", true, "yes"},
		{"bool_false", false, "no"},
		{"nil", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FormatCustomFieldValue(tt.input)
			testutil.Equal(t, got, tt.want)
		})
	}
}

func TestKnownFieldExtractors_ParityWithKnownFieldKeys(t *testing.T) {
	t.Parallel()
	for key := range knownFieldKeys {
		if _, ok := knownFieldExtractors[key]; !ok {
			t.Errorf("knownFieldKeys has %q but knownFieldExtractors does not", key)
		}
	}
	for key := range knownFieldExtractors {
		if !knownFieldKeys[key] {
			t.Errorf("knownFieldExtractors has %q but knownFieldKeys does not", key)
		}
	}
}
