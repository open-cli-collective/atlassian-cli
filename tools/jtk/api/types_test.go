package api //nolint:revive // package name is intentional

import (
	"encoding/json"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

// jsonEq compares two JSON strings for structural equality.
func jsonEq(t *testing.T, got, want string) {
	t.Helper()
	var gotVal, wantVal any
	if err := json.Unmarshal([]byte(got), &gotVal); err != nil {
		t.Fatalf("got is not valid JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(want), &wantVal); err != nil {
		t.Fatalf("want is not valid JSON: %v", err)
	}
	// Re-marshal both to canonical form for comparison
	gotBytes, _ := json.Marshal(gotVal)
	wantBytes, _ := json.Marshal(wantVal)
	testutil.Equal(t, string(gotBytes), string(wantBytes))
}

func TestDescription_UnmarshalJSON(t *testing.T) {
	t.Parallel()
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
			testutil.RequireNoError(t, err)
			testutil.Equal(t, desc.Text, tt.wantText)
			testutil.Equal(t, desc.ADF != nil, tt.wantADF)
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
					Content: []*ADFNode{{Type: "paragraph", Content: []*ADFNode{{Type: "text", Text: "Custom ADF"}}}},
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
			testutil.RequireNoError(t, err)
			jsonEq(t, string(data), tt.want)
		})
	}
}

func TestDescription_ToPlainText(t *testing.T) {
	testutil.Equal(t, (*Description)(nil).ToPlainText(), "")
	testutil.Equal(t, (&Description{Text: "test"}).ToPlainText(), "test")
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
				Content: []*ADFNode{
					{
						Type: "paragraph",
						Content: []*ADFNode{
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
				testutil.Nil(t, got)
			} else {
				testutil.NotNil(t, got)
				testutil.Equal(t, got.Type, tt.want.Type)
				testutil.Equal(t, got.Version, tt.want.Version)
				testutil.Equal(t, len(got.Content), len(tt.want.Content))
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
				Content: []*ADFNode{
					{
						Type: "paragraph",
						Content: []*ADFNode{
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
				Content: []*ADFNode{
					{
						Type: "paragraph",
						Content: []*ADFNode{
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
				Content: []*ADFNode{
					{
						Type: "paragraph",
						Content: []*ADFNode{
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
			testutil.Equal(t, got, tt.want)
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
	testutil.RequireNoError(t, err)

	testutil.Equal(t, issue.ID, "10001")
	testutil.Equal(t, issue.Key, "PROJ-123")
	testutil.Equal(t, issue.Fields.Summary, "Test Issue")
	testutil.Equal(t, issue.Fields.Description.Text, "Plain text description")
	testutil.Equal(t, issue.Fields.Status.Name, "Open")
	testutil.Equal(t, issue.Fields.IssueType.Name, "Task")
	testutil.Equal(t, issue.Fields.Priority.Name, "Medium")
	testutil.Equal(t, issue.Fields.Assignee.DisplayName, "John Doe")
	testutil.Equal(t, issue.Fields.Labels, []string{"bug", "urgent"})
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
	testutil.RequireNoError(t, err)

	testutil.Equal(t, result.StartAt, 0)
	testutil.Equal(t, result.MaxResults, 50)
	testutil.Equal(t, result.Total, 2)
	testutil.Len(t, result.Issues, 2)
	testutil.Equal(t, result.Issues[0].Key, "PROJ-1")
	testutil.Equal(t, result.Issues[1].Key, "PROJ-2")
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
	testutil.RequireNoError(t, err)

	testutil.Equal(t, transition.ID, "21")
	testutil.Equal(t, transition.Name, "In Progress")
	testutil.Equal(t, transition.To.Name, "In Progress")
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
	testutil.RequireNoError(t, err)

	testutil.Equal(t, comment.ID, "10001")
	testutil.Equal(t, comment.Author.DisplayName, "Jane Doe")
	testutil.NotNil(t, comment.Body)
	testutil.Equal(t, comment.Body.ToPlainText(), "This is a comment\n")
}

func TestCreateIssueRequest_MarshalJSON(t *testing.T) {
	req := CreateIssueRequest{
		Fields: map[string]any{
			"project":   map[string]string{"key": "PROJ"},
			"issuetype": map[string]string{"name": "Task"},
			"summary":   "New task",
		},
	}

	data, err := json.Marshal(req)
	testutil.RequireNoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	testutil.RequireNoError(t, err)

	fields := result["fields"].(map[string]any)
	testutil.Equal(t, fields["summary"], "New task")
	project := fields["project"].(map[string]any)
	testutil.Equal(t, project["key"], "PROJ")
}

func TestTransitionRequest_MarshalJSON(t *testing.T) {
	req := TransitionRequest{
		Transition: TransitionID{ID: "21"},
		Fields: map[string]any{
			"resolution": map[string]string{"name": "Done"},
		},
	}

	data, err := json.Marshal(req)
	testutil.RequireNoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	testutil.RequireNoError(t, err)

	transition := result["transition"].(map[string]any)
	testutil.Equal(t, transition["id"], "21")

	fields := result["fields"].(map[string]any)
	resolution := fields["resolution"].(map[string]any)
	testutil.Equal(t, resolution["name"], "Done")
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
	testutil.RequireNoError(t, err)

	// Standard fields should be parsed
	testutil.Equal(t, fields.Summary, "Test Issue")
	testutil.NotNil(t, fields.Status)
	testutil.Equal(t, fields.Status.Name, "Open")

	// Custom fields should be captured
	testutil.NotNil(t, fields.CustomFields)
	testutil.Equal(t, fields.CustomFields["customfield_10001"], float64(5))

	customField10002 := fields.CustomFields["customfield_10002"].(map[string]any)
	testutil.Equal(t, customField10002["value"], "Feature")

	customField10003 := fields.CustomFields["customfield_10003"].([]any)
	testutil.Len(t, customField10003, 2)
}

func TestIssueFields_MarshalJSON_IncludesCustomFields(t *testing.T) {
	fields := IssueFields{
		Summary: "Test Issue",
		Status:  &Status{ID: "1", Name: "Open"},
		CustomFields: map[string]any{
			"customfield_10001": 5,
			"customfield_10002": map[string]string{"value": "Feature"},
		},
	}

	data, err := json.Marshal(fields)
	testutil.RequireNoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	testutil.RequireNoError(t, err)

	// Standard fields should be present
	testutil.Equal(t, result["summary"], "Test Issue")

	// Custom fields should be included
	testutil.Equal(t, result["customfield_10001"], float64(5))
	customField10002 := result["customfield_10002"].(map[string]any)
	testutil.Equal(t, customField10002["value"], "Feature")
}

func TestExtractText_Headings(t *testing.T) {
	doc := &ADFDocument{
		Type:    "doc",
		Version: 1,
		Content: []*ADFNode{
			{
				Type:  "heading",
				Attrs: map[string]any{"level": float64(1)},
				Content: []*ADFNode{
					{Type: "text", Text: "Title"},
				},
			},
			{
				Type: "paragraph",
				Content: []*ADFNode{
					{Type: "text", Text: "Body text"},
				},
			},
		},
	}

	got := doc.ToPlainText()
	testutil.Contains(t, got, "Title")
	testutil.Contains(t, got, "Body text")
	// Heading should have newlines around it for separation
	testutil.Equal(t, got, "\nTitle\nBody text\n")
}

func TestExtractText_BulletList(t *testing.T) {
	doc := &ADFDocument{
		Type:    "doc",
		Version: 1,
		Content: []*ADFNode{
			{
				Type: "bulletList",
				Content: []*ADFNode{
					{
						Type: "listItem",
						Content: []*ADFNode{
							{Type: "paragraph", Content: []*ADFNode{{Type: "text", Text: "Item one"}}},
						},
					},
					{
						Type: "listItem",
						Content: []*ADFNode{
							{Type: "paragraph", Content: []*ADFNode{{Type: "text", Text: "Item two"}}},
						},
					},
				},
			},
		},
	}

	got := doc.ToPlainText()
	testutil.Contains(t, got, "- Item one")
	testutil.Contains(t, got, "- Item two")
}

func TestExtractText_CodeBlock(t *testing.T) {
	doc := &ADFDocument{
		Type:    "doc",
		Version: 1,
		Content: []*ADFNode{
			{
				Type: "paragraph",
				Content: []*ADFNode{
					{Type: "text", Text: "Before code"},
				},
			},
			{
				Type:  "codeBlock",
				Attrs: map[string]any{"language": "go"},
				Content: []*ADFNode{
					{Type: "text", Text: "fmt.Println(\"hello\")"},
				},
			},
			{
				Type: "paragraph",
				Content: []*ADFNode{
					{Type: "text", Text: "After code"},
				},
			},
		},
	}

	got := doc.ToPlainText()
	testutil.Contains(t, got, "Before code")
	testutil.Contains(t, got, "fmt.Println(\"hello\")")
	testutil.Contains(t, got, "After code")
}

func TestExtractText_Blockquote(t *testing.T) {
	doc := &ADFDocument{
		Type:    "doc",
		Version: 1,
		Content: []*ADFNode{
			{
				Type: "blockquote",
				Content: []*ADFNode{
					{
						Type: "paragraph",
						Content: []*ADFNode{
							{Type: "text", Text: "Quoted text"},
						},
					},
				},
			},
		},
	}

	got := doc.ToPlainText()
	testutil.Contains(t, got, "> Quoted text")
}

func TestExtractText_Rule(t *testing.T) {
	doc := &ADFDocument{
		Type:    "doc",
		Version: 1,
		Content: []*ADFNode{
			{
				Type: "paragraph",
				Content: []*ADFNode{
					{Type: "text", Text: "Above"},
				},
			},
			{Type: "rule"},
			{
				Type: "paragraph",
				Content: []*ADFNode{
					{Type: "text", Text: "Below"},
				},
			},
		},
	}

	got := doc.ToPlainText()
	testutil.Contains(t, got, "Above")
	testutil.Contains(t, got, "---")
	testutil.Contains(t, got, "Below")
}

func TestExtractText_NestedList(t *testing.T) {
	doc := &ADFDocument{
		Type:    "doc",
		Version: 1,
		Content: []*ADFNode{
			{
				Type: "bulletList",
				Content: []*ADFNode{
					{
						Type: "listItem",
						Content: []*ADFNode{
							{Type: "paragraph", Content: []*ADFNode{{Type: "text", Text: "Parent"}}},
							{
								Type: "bulletList",
								Content: []*ADFNode{
									{
										Type: "listItem",
										Content: []*ADFNode{
											{Type: "paragraph", Content: []*ADFNode{{Type: "text", Text: "Child"}}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	got := doc.ToPlainText()
	testutil.Contains(t, got, "- Parent")
	testutil.Contains(t, got, "  - Child")
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
	testutil.RequireNoError(t, err)

	// Verify custom fields were captured
	testutil.Equal(t, issue.Fields.CustomFields["customfield_10001"], float64(8))

	// Marshal back to JSON
	data, err := json.Marshal(issue)
	testutil.RequireNoError(t, err)

	// Verify custom fields are in the output
	var result map[string]any
	err = json.Unmarshal(data, &result)
	testutil.RequireNoError(t, err)

	fields := result["fields"].(map[string]any)
	testutil.Equal(t, fields["customfield_10001"], float64(8))
	testutil.Equal(t, fields["customfield_10002"].(map[string]any)["value"], "Bug Fix")
}
