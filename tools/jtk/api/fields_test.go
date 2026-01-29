package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.wantID, result.ID)
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
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.wantName, result.Name)
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
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantID, id)
			}
		})
	}
}

func TestFindFieldByName_EmptySlice(t *testing.T) {
	result := FindFieldByName([]Field{}, "Summary")
	assert.Nil(t, result)
}

func TestFindFieldByID_EmptySlice(t *testing.T) {
	result := FindFieldByID([]Field{}, "summary")
	assert.Nil(t, result)
}

func TestResolveFieldID_EmptySlice(t *testing.T) {
	_, err := ResolveFieldID([]Field{}, "summary")
	assert.Error(t, err)
}

func TestFormatFieldValue(t *testing.T) {
	tests := []struct {
		name  string
		field *Field
		value string
		want  interface{}
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFieldValue(tt.field, tt.value)
			assert.Equal(t, tt.want, got)
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
	require.True(t, ok, "expected *ADFDocument, got %T", got)
	require.NotNil(t, adf)
	assert.Equal(t, "doc", adf.Type)
	assert.Equal(t, 1, adf.Version)
	require.Len(t, adf.Content, 1)
	assert.Equal(t, "paragraph", adf.Content[0].Type)
}

func TestClient_GetFieldOptionsFromEditMeta(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/issue/PROJ-123/editmeta")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
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

	client := &Client{
		BaseURL:    server.URL,
		Email:      "user@example.com",
		APIToken:   "token",
		HTTPClient: server.Client(),
	}

	t.Run("priority field with name values", func(t *testing.T) {
		options, err := client.GetFieldOptionsFromEditMeta("PROJ-123", "priority")
		require.NoError(t, err)
		assert.Len(t, options, 5)
		assert.Equal(t, "1", options[0].ID)
		assert.Equal(t, "Highest", options[0].Name)
	})

	t.Run("custom field with value format", func(t *testing.T) {
		options, err := client.GetFieldOptionsFromEditMeta("PROJ-123", "customfield_10001")
		require.NoError(t, err)
		assert.Len(t, options, 3)
		assert.Equal(t, "Feature", options[0].Value)
		assert.Equal(t, "Refactor", options[2].Value)
		assert.True(t, options[2].Disabled)
	})

	t.Run("field not found", func(t *testing.T) {
		_, err := client.GetFieldOptionsFromEditMeta("PROJ-123", "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}
