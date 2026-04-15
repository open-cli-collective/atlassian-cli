package api

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestParseEditMeta_BasicFields(t *testing.T) {
	t.Parallel()

	fieldsData := map[string]any{
		"summary": map[string]any{
			"name":     "Summary",
			"required": true,
			"schema": map[string]any{
				"type": "string",
			},
		},
		"description": map[string]any{
			"name":     "Description",
			"required": false,
			"schema": map[string]any{
				"type": "string",
			},
		},
	}

	result := ParseEditMeta(fieldsData)

	testutil.Equal(t, len(result), 2)

	// Find fields by ID (map iteration order is random)
	fieldMap := make(map[string]EditFieldMeta)
	for _, f := range result {
		fieldMap[f.ID] = f
	}

	summary := fieldMap["summary"]
	testutil.Equal(t, summary.Name, "Summary")
	testutil.Equal(t, summary.Type, "string")
	testutil.True(t, summary.Required)

	desc := fieldMap["description"]
	testutil.Equal(t, desc.Name, "Description")
	testutil.Equal(t, desc.Type, "string")
	testutil.False(t, desc.Required)
}

func TestParseEditMeta_EmptyInput(t *testing.T) {
	t.Parallel()

	result := ParseEditMeta(nil)
	testutil.Equal(t, len(result), 0)

	result = ParseEditMeta(map[string]any{})
	testutil.Equal(t, len(result), 0)
}

func TestParseEditMeta_SkipsInvalidFieldData(t *testing.T) {
	t.Parallel()

	fieldsData := map[string]any{
		"valid": map[string]any{
			"name": "Valid Field",
		},
		"invalid_string": "not a map",
		"invalid_nil":    nil,
		"invalid_number": 42,
	}

	result := ParseEditMeta(fieldsData)

	// Should only parse the valid field
	testutil.Equal(t, len(result), 1)
	testutil.Equal(t, result[0].ID, "valid")
	testutil.Equal(t, result[0].Name, "Valid Field")
}

func TestParseEditMeta_MissingSchema(t *testing.T) {
	t.Parallel()

	fieldsData := map[string]any{
		"no_schema": map[string]any{
			"name":     "No Schema",
			"required": true,
		},
	}

	result := ParseEditMeta(fieldsData)

	testutil.Equal(t, len(result), 1)
	testutil.Equal(t, result[0].ID, "no_schema")
	testutil.Equal(t, result[0].Name, "No Schema")
	testutil.Equal(t, result[0].Type, "") // Empty when no schema
	testutil.True(t, result[0].Required)
}

func TestParseEditMeta_InvalidSchema(t *testing.T) {
	t.Parallel()

	fieldsData := map[string]any{
		"bad_schema": map[string]any{
			"name":   "Bad Schema",
			"schema": "not a map",
		},
	}

	result := ParseEditMeta(fieldsData)

	testutil.Equal(t, len(result), 1)
	testutil.Equal(t, result[0].Type, "") // Empty when schema isn't a map
}

func TestParseEditMeta_RequiredFalseExplicitly(t *testing.T) {
	t.Parallel()

	fieldsData := map[string]any{
		"optional": map[string]any{
			"name":     "Optional",
			"required": false,
		},
	}

	result := ParseEditMeta(fieldsData)

	testutil.Equal(t, len(result), 1)
	testutil.False(t, result[0].Required)
}

func TestParseEditMeta_RequiredMissing(t *testing.T) {
	t.Parallel()

	fieldsData := map[string]any{
		"no_required": map[string]any{
			"name": "No Required Field",
		},
	}

	result := ParseEditMeta(fieldsData)

	testutil.Equal(t, len(result), 1)
	testutil.False(t, result[0].Required) // Defaults to false
}

func TestParseEditMeta_RequiredInvalidType(t *testing.T) {
	t.Parallel()

	fieldsData := map[string]any{
		"bad_required": map[string]any{
			"name":     "Bad Required",
			"required": "yes", // String instead of bool
		},
	}

	result := ParseEditMeta(fieldsData)

	testutil.Equal(t, len(result), 1)
	testutil.False(t, result[0].Required) // Falls back to false when not bool
}

func TestSafeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"empty_string", "", ""},
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
		{"bool", true, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testutil.Equal(t, safeString(tt.input), tt.want)
		})
	}
}
