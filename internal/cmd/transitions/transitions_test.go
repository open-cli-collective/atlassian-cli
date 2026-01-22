package transitions

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestFormatFieldValue(t *testing.T) {
	tests := []struct {
		name  string
		field *api.Field
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
			field: &api.Field{
				ID:   "customfield_10001",
				Name: "Change Type",
				Schema: api.FieldSchema{
					Type: "option",
				},
			},
			value: "Bug Fix",
			want:  map[string]string{"value": "Bug Fix"},
		},
		{
			name: "array of options - wraps in array of value maps",
			field: &api.Field{
				ID:   "customfield_10002",
				Name: "Categories",
				Schema: api.FieldSchema{
					Type:  "array",
					Items: "option",
				},
			},
			value: "Frontend",
			want:  []map[string]string{{"value": "Frontend"}},
		},
		{
			name: "array of strings - wraps in string array",
			field: &api.Field{
				ID:   "labels",
				Name: "Labels",
				Schema: api.FieldSchema{
					Type:  "array",
					Items: "string",
				},
			},
			value: "urgent",
			want:  []string{"urgent"},
		},
		{
			name: "user field - wraps in accountId map",
			field: &api.Field{
				ID:   "assignee",
				Name: "Assignee",
				Schema: api.FieldSchema{
					Type: "user",
				},
			},
			value: "abc123",
			want:  map[string]string{"accountId": "abc123"},
		},
		{
			name: "string field - returns as-is",
			field: &api.Field{
				ID:   "summary",
				Name: "Summary",
				Schema: api.FieldSchema{
					Type: "string",
				},
			},
			value: "Updated summary",
			want:  "Updated summary",
		},
		{
			name: "number field - converts to float64",
			field: &api.Field{
				ID:   "customfield_10003",
				Name: "Story Points",
				Schema: api.FieldSchema{
					Type: "number",
				},
			},
			value: "5",
			want:  float64(5),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := api.FormatFieldValue(tt.field, tt.value)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetRequiredFields(t *testing.T) {
	tests := []struct {
		name       string
		transition api.Transition
		want       string
	}{
		{
			name: "no fields",
			transition: api.Transition{
				ID:     "21",
				Name:   "In Progress",
				Fields: nil,
			},
			want: "-",
		},
		{
			name: "empty fields map",
			transition: api.Transition{
				ID:     "21",
				Name:   "In Progress",
				Fields: map[string]api.TransitionField{},
			},
			want: "-",
		},
		{
			name: "no required fields",
			transition: api.Transition{
				ID:   "21",
				Name: "In Progress",
				Fields: map[string]api.TransitionField{
					"resolution": {
						Required: false,
						Name:     "Resolution",
					},
				},
			},
			want: "-",
		},
		{
			name: "one required field",
			transition: api.Transition{
				ID:   "31",
				Name: "Done",
				Fields: map[string]api.TransitionField{
					"resolution": {
						Required: true,
						Name:     "Resolution",
					},
				},
			},
			want: "Resolution",
		},
		{
			name: "multiple required fields",
			transition: api.Transition{
				ID:   "31",
				Name: "Done",
				Fields: map[string]api.TransitionField{
					"resolution": {
						Required: true,
						Name:     "Resolution",
					},
					"customfield_10001": {
						Required: true,
						Name:     "Root Cause",
					},
					"comment": {
						Required: false,
						Name:     "Comment",
					},
				},
			},
			want: "Resolution, Root Cause",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getRequiredFields(tt.transition)
			// For multiple fields, check both are present (order may vary due to map iteration)
			if tt.name == "multiple required fields" {
				assert.Contains(t, got, "Resolution")
				assert.Contains(t, got, "Root Cause")
				assert.NotContains(t, got, "Comment")
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
