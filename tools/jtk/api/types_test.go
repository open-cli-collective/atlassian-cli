package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDescription_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantText string
		wantADF  bool
	}{
		{
			name:     "string format (Agile API)",
			input:    `"This is a plain text description"`,
			wantText: "This is a plain text description",
			wantADF:  false,
		},
		{
			name: "ADF format (REST API v3)",
			input: `{
				"type": "doc",
				"version": 1,
				"content": [
					{
						"type": "paragraph",
						"content": [
							{"type": "text", "text": "Hello world"}
						]
					}
				]
			}`,
			wantText: "Hello world\n",
			wantADF:  true,
		},
		{
			name:     "null value",
			input:    `null`,
			wantText: "",
			wantADF:  false,
		},
		{
			name:     "empty string",
			input:    `""`,
			wantText: "",
			wantADF:  false,
		},
		{
			name: "ADF with multiple paragraphs",
			input: `{
				"type": "doc",
				"version": 1,
				"content": [
					{
						"type": "paragraph",
						"content": [{"type": "text", "text": "First"}]
					},
					{
						"type": "paragraph",
						"content": [{"type": "text", "text": "Second"}]
					}
				]
			}`,
			wantText: "First\nSecond\n",
			wantADF:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var desc Description
			err := json.Unmarshal([]byte(tt.input), &desc)
			require.NoError(t, err)
			assert.Equal(t, tt.wantText, desc.Text)
			assert.Equal(t, tt.wantADF, desc.ADF != nil)
		})
	}
}

func TestDescription_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		desc Description
		want string
	}{
		{
			name: "with text only - converts to ADF",
			desc: Description{Text: "Hello"},
			want: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"Hello"}]}]}`,
		},
		{
			name: "with existing ADF",
			desc: Description{
				Text: "Hello",
				ADF: &ADFDocument{
					Type:    "doc",
					Version: 1,
					Content: []ADFNode{{Type: "paragraph", Content: []ADFNode{{Type: "text", Text: "Custom ADF"}}}},
				},
			},
			want: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"Custom ADF"}]}]}`,
		},
		{
			name: "empty description",
			desc: Description{},
			want: `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(&tt.desc)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(data))
		})
	}
}

func TestDescription_ToPlainText(t *testing.T) {
	assert.Equal(t, "", (*Description)(nil).ToPlainText())
	assert.Equal(t, "test", (&Description{Text: "test"}).ToPlainText())
}

func TestNewADFDocument(t *testing.T) {
	tests := []struct {
		name string
		text string
		want *ADFDocument
	}{
		{
			name: "with text",
			text: "Hello world",
			want: &ADFDocument{
				Type:    "doc",
				Version: 1,
				Content: []ADFNode{
					{
						Type: "paragraph",
						Content: []ADFNode{
							{Type: "text", Text: "Hello world"},
						},
					},
				},
			},
		},
		{
			name: "empty text",
			text: "",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewADFDocument(tt.text)
			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tt.want.Type, got.Type)
				assert.Equal(t, tt.want.Version, got.Version)
				assert.Equal(t, len(tt.want.Content), len(got.Content))
			}
		})
	}
}

func TestADFDocument_ToPlainText(t *testing.T) {
	tests := []struct {
		name string
		doc  *ADFDocument
		want string
	}{
		{
			name: "nil document",
			doc:  nil,
			want: "",
		},
		{
			name: "simple paragraph",
			doc: &ADFDocument{
				Type:    "doc",
				Version: 1,
				Content: []ADFNode{
					{
						Type: "paragraph",
						Content: []ADFNode{
							{Type: "text", Text: "Hello"},
						},
					},
				},
			},
			want: "Hello\n",
		},
		{
			name: "multiple text nodes",
			doc: &ADFDocument{
				Type:    "doc",
				Version: 1,
				Content: []ADFNode{
					{
						Type: "paragraph",
						Content: []ADFNode{
							{Type: "text", Text: "Hello "},
							{Type: "text", Text: "World"},
						},
					},
				},
			},
			want: "Hello World\n",
		},
		{
			name: "with hard break",
			doc: &ADFDocument{
				Type:    "doc",
				Version: 1,
				Content: []ADFNode{
					{
						Type: "paragraph",
						Content: []ADFNode{
							{Type: "text", Text: "Line 1"},
							{Type: "hardBreak"},
							{Type: "text", Text: "Line 2"},
						},
					},
				},
			},
			want: "Line 1\nLine 2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.doc.ToPlainText()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIssue_UnmarshalJSON(t *testing.T) {
	// Test full issue unmarshaling with various field types
	input := `{
		"id": "10001",
		"key": "PROJ-123",
		"self": "https://example.atlassian.net/rest/api/3/issue/10001",
		"fields": {
			"summary": "Test Issue",
			"description": "Plain text description",
			"status": {
				"id": "1",
				"name": "Open",
				"statusCategory": {
					"id": 2,
					"key": "new",
					"name": "To Do"
				}
			},
			"issuetype": {
				"id": "10001",
				"name": "Task",
				"subtask": false
			},
			"priority": {
				"id": "3",
				"name": "Medium"
			},
			"assignee": {
				"accountId": "abc123",
				"displayName": "John Doe",
				"active": true
			},
			"labels": ["bug", "urgent"],
			"created": "2024-01-15T10:00:00.000Z",
			"updated": "2024-01-15T12:00:00.000Z"
		}
	}`

	var issue Issue
	err := json.Unmarshal([]byte(input), &issue)
	require.NoError(t, err)

	assert.Equal(t, "10001", issue.ID)
	assert.Equal(t, "PROJ-123", issue.Key)
	assert.Equal(t, "Test Issue", issue.Fields.Summary)
	assert.Equal(t, "Plain text description", issue.Fields.Description.Text)
	assert.Equal(t, "Open", issue.Fields.Status.Name)
	assert.Equal(t, "Task", issue.Fields.IssueType.Name)
	assert.Equal(t, "Medium", issue.Fields.Priority.Name)
	assert.Equal(t, "John Doe", issue.Fields.Assignee.DisplayName)
	assert.Equal(t, []string{"bug", "urgent"}, issue.Fields.Labels)
}

func TestSearchResult_UnmarshalJSON(t *testing.T) {
	input := `{
		"startAt": 0,
		"maxResults": 50,
		"total": 2,
		"issues": [
			{"id": "1", "key": "PROJ-1", "fields": {"summary": "Issue 1"}},
			{"id": "2", "key": "PROJ-2", "fields": {"summary": "Issue 2"}}
		]
	}`

	var result SearchResult
	err := json.Unmarshal([]byte(input), &result)
	require.NoError(t, err)

	assert.Equal(t, 0, result.StartAt)
	assert.Equal(t, 50, result.MaxResults)
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.Issues, 2)
	assert.Equal(t, "PROJ-1", result.Issues[0].Key)
	assert.Equal(t, "PROJ-2", result.Issues[1].Key)
}

func TestTransition_UnmarshalJSON(t *testing.T) {
	input := `{
		"id": "21",
		"name": "In Progress",
		"to": {
			"id": "3",
			"name": "In Progress",
			"statusCategory": {
				"id": 4,
				"key": "indeterminate",
				"name": "In Progress"
			}
		}
	}`

	var transition Transition
	err := json.Unmarshal([]byte(input), &transition)
	require.NoError(t, err)

	assert.Equal(t, "21", transition.ID)
	assert.Equal(t, "In Progress", transition.Name)
	assert.Equal(t, "In Progress", transition.To.Name)
}

func TestComment_UnmarshalJSON(t *testing.T) {
	input := `{
		"id": "10001",
		"author": {
			"accountId": "abc123",
			"displayName": "Jane Doe",
			"active": true
		},
		"body": {
			"type": "doc",
			"version": 1,
			"content": [
				{
					"type": "paragraph",
					"content": [{"type": "text", "text": "This is a comment"}]
				}
			]
		},
		"created": "2024-01-15T10:00:00.000Z",
		"updated": "2024-01-15T10:00:00.000Z"
	}`

	var comment Comment
	err := json.Unmarshal([]byte(input), &comment)
	require.NoError(t, err)

	assert.Equal(t, "10001", comment.ID)
	assert.Equal(t, "Jane Doe", comment.Author.DisplayName)
	assert.NotNil(t, comment.Body)
	assert.Equal(t, "This is a comment\n", comment.Body.ToPlainText())
}

func TestCreateIssueRequest_MarshalJSON(t *testing.T) {
	req := CreateIssueRequest{
		Fields: map[string]interface{}{
			"project":   map[string]string{"key": "PROJ"},
			"issuetype": map[string]string{"name": "Task"},
			"summary":   "New task",
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	fields := result["fields"].(map[string]interface{})
	assert.Equal(t, "New task", fields["summary"])
	project := fields["project"].(map[string]interface{})
	assert.Equal(t, "PROJ", project["key"])
}

func TestTransitionRequest_MarshalJSON(t *testing.T) {
	req := TransitionRequest{
		Transition: TransitionID{ID: "21"},
		Fields: map[string]interface{}{
			"resolution": map[string]string{"name": "Done"},
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	transition := result["transition"].(map[string]interface{})
	assert.Equal(t, "21", transition["id"])

	fields := result["fields"].(map[string]interface{})
	resolution := fields["resolution"].(map[string]interface{})
	assert.Equal(t, "Done", resolution["name"])
}

func TestIssueFields_CustomFields(t *testing.T) {
	// Test unmarshaling with custom fields
	input := `{
		"summary": "Test Issue",
		"status": {"id": "1", "name": "Open"},
		"customfield_10001": 5,
		"customfield_10002": {"value": "Feature"},
		"customfield_10003": ["label1", "label2"]
	}`

	var fields IssueFields
	err := json.Unmarshal([]byte(input), &fields)
	require.NoError(t, err)

	// Standard fields should be parsed
	assert.Equal(t, "Test Issue", fields.Summary)
	assert.NotNil(t, fields.Status)
	assert.Equal(t, "Open", fields.Status.Name)

	// Custom fields should be captured
	assert.NotNil(t, fields.CustomFields)
	assert.Equal(t, float64(5), fields.CustomFields["customfield_10001"])

	customfield_10002 := fields.CustomFields["customfield_10002"].(map[string]interface{})
	assert.Equal(t, "Feature", customfield_10002["value"])

	customfield_10003 := fields.CustomFields["customfield_10003"].([]interface{})
	assert.Len(t, customfield_10003, 2)
}

func TestIssueFields_MarshalJSON_IncludesCustomFields(t *testing.T) {
	fields := IssueFields{
		Summary: "Test Issue",
		Status:  &Status{ID: "1", Name: "Open"},
		CustomFields: map[string]interface{}{
			"customfield_10001": 5,
			"customfield_10002": map[string]string{"value": "Feature"},
		},
	}

	data, err := json.Marshal(fields)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Standard fields should be present
	assert.Equal(t, "Test Issue", result["summary"])

	// Custom fields should be included
	assert.Equal(t, float64(5), result["customfield_10001"])
	customfield_10002 := result["customfield_10002"].(map[string]interface{})
	assert.Equal(t, "Feature", customfield_10002["value"])
}

func TestIssue_RoundTrip_WithCustomFields(t *testing.T) {
	// Test full issue round-trip with custom fields
	input := `{
		"id": "10001",
		"key": "PROJ-123",
		"self": "https://example.atlassian.net/rest/api/3/issue/10001",
		"fields": {
			"summary": "Test Issue",
			"customfield_10001": 8,
			"customfield_10002": {"value": "Bug Fix"},
			"customfield_sprint": {"id": 42, "name": "Sprint 5"}
		}
	}`

	var issue Issue
	err := json.Unmarshal([]byte(input), &issue)
	require.NoError(t, err)

	// Verify custom fields were captured
	assert.Equal(t, float64(8), issue.Fields.CustomFields["customfield_10001"])

	// Marshal back to JSON
	data, err := json.Marshal(issue)
	require.NoError(t, err)

	// Verify custom fields are in the output
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	fields := result["fields"].(map[string]interface{})
	assert.Equal(t, float64(8), fields["customfield_10001"])
	assert.Equal(t, "Bug Fix", fields["customfield_10002"].(map[string]interface{})["value"])
}
