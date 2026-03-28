package api //nolint:revive // package name is intentional

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func getTestFields() []Field {
	return []Field{
		{ID: "summary", Name: "Summary", Custom: false},
		{ID: "description", Name: "Description", Custom: false},
		{ID: "customfield_10001", Name: "Story Points", Custom: true},
		{ID: "customfield_10002", Name: "Sprint", Custom: true},
		{ID: "customfield_10003", Name: "Epic Link", Custom: true},
	}
}

func TestFindFieldByName(t *testing.T) {
	t.Parallel()
	fields := getTestFields()

	tests := []struct {
		name       string
		searchName string
		wantID     string
		wantNil    bool
	}{
		{
			name:       "exact match",
			searchName: "Summary",
			wantID:     "summary",
		},
		{
			name:       "case insensitive",
			searchName: "story points",
			wantID:     "customfield_10001",
		},
		{
			name:       "not found",
			searchName: "NonExistent",
			wantNil:    true,
		},
		{
			name:       "empty name",
			searchName: "",
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindFieldByName(fields, tt.searchName)
			if tt.wantNil {
				testutil.Nil(t, result)
			} else {
				testutil.NotNil(t, result)
				testutil.Equal(t, result.ID, tt.wantID)
			}
		})
	}
}

func TestFindFieldByID(t *testing.T) {
	fields := getTestFields()

	tests := []struct {
		name     string
		searchID string
		wantName string
		wantNil  bool
	}{
		{
			name:     "exact match",
			searchID: "summary",
			wantName: "Summary",
		},
		{
			name:     "custom field",
			searchID: "customfield_10001",
			wantName: "Story Points",
		},
		{
			name:     "not found",
			searchID: "nonexistent",
			wantNil:  true,
		},
		{
			name:     "empty id",
			searchID: "",
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindFieldByID(fields, tt.searchID)
			if tt.wantNil {
				testutil.Nil(t, result)
			} else {
				testutil.NotNil(t, result)
				testutil.Equal(t, result.Name, tt.wantName)
			}
		})
	}
}

func TestResolveFieldID(t *testing.T) {
	fields := getTestFields()

	tests := []struct {
		name      string
		nameOrID  string
		wantID    string
		wantError bool
	}{
		{
			name:     "by exact ID",
			nameOrID: "summary",
			wantID:   "summary",
		},
		{
			name:     "by name",
			nameOrID: "Story Points",
			wantID:   "customfield_10001",
		},
		{
			name:     "by name case insensitive",
			nameOrID: "epic link",
			wantID:   "customfield_10003",
		},
		{
			name:      "not found",
			nameOrID:  "NonExistent",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ResolveFieldID(fields, tt.nameOrID)
			if tt.wantError {
				testutil.Error(t, err)
			} else {
				testutil.RequireNoError(t, err)
				testutil.Equal(t, id, tt.wantID)
			}
		})
	}
}

func TestFindFieldByName_EmptySlice(t *testing.T) {
	result := FindFieldByName([]Field{}, "Summary")
	testutil.Nil(t, result)
}

func TestFindFieldByID_EmptySlice(t *testing.T) {
	result := FindFieldByID([]Field{}, "summary")
	testutil.Nil(t, result)
}

func TestResolveFieldID_EmptySlice(t *testing.T) {
	_, err := ResolveFieldID([]Field{}, "summary")
	testutil.Error(t, err)
}

func TestFormatFieldValue(t *testing.T) {
	tests := []struct {
		name  string
		field *Field
		value string
		want  any
	}{
		{
			name:  "nil field - returns string as-is",
			field: nil,
			value: "some value",
			want:  "some value",
		},
		{
			name: "option field - wraps in value map",
			field: &Field{
				ID:   "customfield_10001",
				Name: "Change Type",
				Schema: FieldSchema{
					Type: "option",
				},
			},
			value: "Bug Fix",
			want:  map[string]string{"value": "Bug Fix"},
		},
		{
			name: "array of options - wraps in array of value maps",
			field: &Field{
				ID:   "customfield_10002",
				Name: "Categories",
				Schema: FieldSchema{
					Type:  "array",
					Items: "option",
				},
			},
			value: "Frontend",
			want:  []map[string]string{{"value": "Frontend"}},
		},
		{
			name: "array of strings - wraps in string array",
			field: &Field{
				ID:   "labels",
				Name: "Labels",
				Schema: FieldSchema{
					Type:  "array",
					Items: "string",
				},
			},
			value: "urgent",
			want:  []string{"urgent"},
		},
		{
			name: "user field - wraps in accountId map",
			field: &Field{
				ID:   "assignee",
				Name: "Assignee",
				Schema: FieldSchema{
					Type: "user",
				},
			},
			value: "abc123",
			want:  map[string]string{"accountId": "abc123"},
		},
		{
			name: "user field with none - returns nil for unassignment",
			field: &Field{
				ID:   "assignee",
				Name: "Assignee",
				Schema: FieldSchema{
					Type: "user",
				},
			},
			value: "none",
			want:  nil,
		},
		{
			name: "user field with empty string - returns nil for unassignment",
			field: &Field{
				ID:   "assignee",
				Name: "Assignee",
				Schema: FieldSchema{
					Type: "user",
				},
			},
			value: "",
			want:  nil,
		},
		{
			name: "user field with null - returns nil for unassignment",
			field: &Field{
				ID:   "assignee",
				Name: "Assignee",
				Schema: FieldSchema{
					Type: "user",
				},
			},
			value: "null",
			want:  nil,
		},
		{
			name: "string field - returns as-is",
			field: &Field{
				ID:   "summary",
				Name: "Summary",
				Schema: FieldSchema{
					Type: "string",
				},
			},
			value: "Updated summary",
			want:  "Updated summary",
		},
		{
			name: "number field - converts to float64",
			field: &Field{
				ID:   "customfield_10003",
				Name: "Story Points",
				Schema: FieldSchema{
					Type: "number",
				},
			},
			value: "5",
			want:  float64(5),
		},
		{
			name: "number field with decimal",
			field: &Field{
				ID:   "customfield_10003",
				Name: "Story Points",
				Schema: FieldSchema{
					Type: "number",
				},
			},
			value: "3.5",
			want:  float64(3.5),
		},
		{
			name: "number field with invalid value - returns string",
			field: &Field{
				ID:   "customfield_10003",
				Name: "Story Points",
				Schema: FieldSchema{
					Type: "number",
				},
			},
			value: "not-a-number",
			want:  "not-a-number",
		},
		{
			name: "priority field by name - wraps in name map",
			field: &Field{
				ID:   "priority",
				Name: "Priority",
				Schema: FieldSchema{
					Type:   "priority",
					System: "priority",
				},
			},
			value: "High",
			want:  map[string]string{"name": "High"},
		},
		{
			name: "priority field by ID - wraps in id map",
			field: &Field{
				ID:   "priority",
				Name: "Priority",
				Schema: FieldSchema{
					Type:   "priority",
					System: "priority",
				},
			},
			value: "2",
			want:  map[string]string{"id": "2"},
		},
		{
			name: "resolution field - wraps in name map",
			field: &Field{
				ID:   "resolution",
				Name: "Resolution",
				Schema: FieldSchema{
					Type:   "resolution",
					System: "resolution",
				},
			},
			value: "Done",
			want:  map[string]string{"name": "Done"},
		},
		{
			name: "issuetype field - wraps in name map",
			field: &Field{
				ID:   "issuetype",
				Name: "Issue Type",
				Schema: FieldSchema{
					Type:   "issuetype",
					System: "issuetype",
				},
			},
			value: "Bug",
			want:  map[string]string{"name": "Bug"},
		},
		{
			name: "status field - wraps in name map",
			field: &Field{
				ID:   "status",
				Name: "Status",
				Schema: FieldSchema{
					Type:   "status",
					System: "status",
				},
			},
			value: "In Progress",
			want:  map[string]string{"name": "In Progress"},
		},
		{
			name: "securitylevel field - wraps in name map",
			field: &Field{
				ID:   "security",
				Name: "Security Level",
				Schema: FieldSchema{
					Type:   "securitylevel",
					System: "security",
				},
			},
			value: "Confidential",
			want:  map[string]string{"name": "Confidential"},
		},
		{
			name: "parent field by issue key - wraps in key map",
			field: &Field{
				ID:   "parent",
				Name: "Parent",
				Schema: FieldSchema{
					Type: "",
				},
			},
			value: "PROJ-123",
			want:  map[string]string{"key": "PROJ-123"},
		},
		{
			name: "parent field by numeric ID - wraps in id map",
			field: &Field{
				ID:   "parent",
				Name: "Parent",
				Schema: FieldSchema{
					Type: "",
				},
			},
			value: "10001",
			want:  map[string]string{"id": "10001"},
		},
		{
			name: "issuelink custom field by key - wraps in key map",
			field: &Field{
				ID:   "customfield_10050",
				Name: "Blocked By",
				Schema: FieldSchema{
					Type: "issuelink",
				},
			},
			value: "PROJ-456",
			want:  map[string]string{"key": "PROJ-456"},
		},
		{
			name: "issuelink custom field by ID - wraps in id map",
			field: &Field{
				ID:   "customfield_10050",
				Name: "Blocked By",
				Schema: FieldSchema{
					Type: "issuelink",
				},
			},
			value: "20001",
			want:  map[string]string{"id": "20001"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFieldValue(tt.field, tt.value)
			testutil.Equal(t, got, tt.want)
		})
	}
}

func TestFormatFieldValue_TextareaField(t *testing.T) {
	field := &Field{
		ID:   "customfield_10046",
		Name: "QA Notes",
		Schema: FieldSchema{
			Type:   "string",
			Custom: "com.atlassian.jira.plugin.system.customfieldtypes:textarea",
		},
	}

	got := FormatFieldValue(field, "Testing notes here")

	// Textarea fields should return ADF document
	adf, ok := got.(*ADFDocument)
	testutil.True(t, ok, fmt.Sprintf("expected *ADFDocument, got %T", got))
	testutil.NotNil(t, adf)
	testutil.Equal(t, adf.Type, "doc")
	testutil.Equal(t, adf.Version, 1)
	testutil.Len(t, adf.Content, 1)
	testutil.Equal(t, adf.Content[0].Type, "paragraph")
}

func TestClient_GetFieldOptionsFromEditMeta(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Contains(t, r.URL.Path, "/issue/PROJ-123/editmeta")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"fields": {
				"priority": {
					"name": "Priority",
					"allowedValues": [
						{"id": "1", "name": "Highest"},
						{"id": "2", "name": "High"},
						{"id": "3", "name": "Medium"},
						{"id": "4", "name": "Low"},
						{"id": "5", "name": "Lowest"}
					]
				},
				"customfield_10001": {
					"name": "Change Type",
					"allowedValues": [
						{"id": "10", "value": "Feature"},
						{"id": "11", "value": "Bug Fix"},
						{"id": "12", "value": "Refactor", "disabled": true}
					]
				}
			}
		}`))
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "user@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	t.Run("priority field with name values", func(t *testing.T) {
		options, err := client.GetFieldOptionsFromEditMeta(context.Background(), "PROJ-123", "priority")
		testutil.RequireNoError(t, err)
		testutil.Len(t, options, 5)
		testutil.Equal(t, options[0].ID, "1")
		testutil.Equal(t, options[0].Name, "Highest")
	})

	t.Run("custom field with value format", func(t *testing.T) {
		options, err := client.GetFieldOptionsFromEditMeta(context.Background(), "PROJ-123", "customfield_10001")
		testutil.RequireNoError(t, err)
		testutil.Len(t, options, 3)
		testutil.Equal(t, options[0].Value, "Feature")
		testutil.Equal(t, options[2].Value, "Refactor")
		testutil.True(t, options[2].Disabled)
	})

	t.Run("field not found", func(t *testing.T) {
		_, err := client.GetFieldOptionsFromEditMeta(context.Background(), "PROJ-123", "nonexistent")
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "not found")
	})
}

func TestMergeFieldValues(t *testing.T) {
	tests := []struct {
		name     string
		existing any
		newVal   any
		want     any
	}{
		{
			name:     "merge option arrays (multi-checkbox)",
			existing: []map[string]string{{"value": "CheckSync"}},
			newVal:   []map[string]string{{"value": "MoniCore"}},
			want:     []map[string]string{{"value": "CheckSync"}, {"value": "MoniCore"}},
		},
		{
			name:     "merge string arrays (labels)",
			existing: []string{"urgent"},
			newVal:   []string{"backend"},
			want:     []string{"urgent", "backend"},
		},
		{
			name:     "non-array field overwrites",
			existing: "old value",
			newVal:   "new value",
			want:     "new value",
		},
		{
			name:     "mismatched types - new value wins",
			existing: "string value",
			newVal:   []string{"array value"},
			want:     []string{"array value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeFieldValues(tt.existing, tt.newVal)
			testutil.Equal(t, got, tt.want)
		})
	}

	t.Run("chained merges - three option values", func(t *testing.T) {
		v1 := []map[string]string{{"value": "CheckSync"}}
		v2 := []map[string]string{{"value": "MoniCore"}}
		v3 := []map[string]string{{"value": "Monit Accounting"}}

		merged := MergeFieldValues(v1, v2)
		merged = MergeFieldValues(merged, v3)

		want := []map[string]string{
			{"value": "CheckSync"},
			{"value": "MoniCore"},
			{"value": "Monit Accounting"},
		}
		testutil.Equal(t, merged, want)
	})
}

func TestIsNullValue(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"none", true},
		{"None", true},
		{"NONE", true},
		{"null", true},
		{"Null", true},
		{"NULL", true},
		{"", true},
		{" none ", true},
		{"user@example.com", false},
		{"me", false},
		{"61292e4c4f29230069621c5f", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			testutil.Equal(t, IsNullValue(tt.input), tt.want)
		})
	}
}
