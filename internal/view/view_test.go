package view

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{"empty (default)", "", false},
		{"table", "table", false},
		{"json", "json", false},
		{"plain", "plain", false},
		{"invalid", "invalid", true},
		{"xml", "xml", true},
		{"TABLE uppercase", "TABLE", true}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFormat(tt.format)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid output format")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidFormats(t *testing.T) {
	formats := ValidFormats()
	assert.Contains(t, formats, "table")
	assert.Contains(t, formats, "json")
	assert.Contains(t, formats, "plain")
	assert.Len(t, formats, 3)
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncate with ellipsis", "hello world", 8, "hello..."},
		{"very short max", "hello", 3, "hel"},
		{"empty string", "", 10, ""},
		{"unicode bytes", "héllo wörld", 8, "héll..."}, // Truncate works on bytes, not runes
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRenderer_RenderTable_Table(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(FormatTable, true)
	r.SetWriter(&buf)

	headers := []string{"ID", "NAME", "STATUS"}
	rows := [][]string{
		{"1", "First", "active"},
		{"2", "Second", "inactive"},
	}

	r.RenderTable(headers, rows)

	output := buf.String()
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "First")
	assert.Contains(t, output, "Second")
}

func TestRenderer_RenderTable_JSON(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(FormatJSON, true)
	r.SetWriter(&buf)

	headers := []string{"ID", "NAME"}
	rows := [][]string{
		{"1", "First"},
		{"2", "Second"},
	}

	r.RenderTable(headers, rows)

	// Verify it's valid JSON
	var result []map[string]string
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "1", result[0]["id"])
	assert.Equal(t, "First", result[0]["name"])
}

func TestRenderer_RenderTable_Plain(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(FormatPlain, true)
	r.SetWriter(&buf)

	headers := []string{"ID", "NAME"}
	rows := [][]string{
		{"1", "First"},
		{"2", "Second"},
	}

	r.RenderTable(headers, rows)

	// Plain format should use tabs and not include headers
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 2)
	assert.Contains(t, lines[0], "1\tFirst")
	assert.Contains(t, lines[1], "2\tSecond")
}

func TestRenderer_RenderJSON(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(FormatJSON, true)
	r.SetWriter(&buf)

	data := map[string]string{
		"status": "ok",
		"count":  "5",
	}

	err := r.RenderJSON(data)
	require.NoError(t, err)

	// Verify output is valid JSON
	var result map[string]string
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "ok", result["status"])
}

func TestRenderer_RenderJSON_Array(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(FormatJSON, true)
	r.SetWriter(&buf)

	err := r.RenderJSON([]interface{}{})
	require.NoError(t, err)

	output := strings.TrimSpace(buf.String())
	assert.Equal(t, "[]", output)
}

func TestRenderer_RenderText(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(FormatTable, true)
	r.SetWriter(&buf)

	r.RenderText("Hello, World!")

	output := strings.TrimSpace(buf.String())
	assert.Equal(t, "Hello, World!", output)
}

func TestRenderer_Success(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(FormatTable, true)
	r.SetWriter(&buf)

	r.Success("Operation completed")

	output := buf.String()
	assert.Contains(t, output, "✓")
	assert.Contains(t, output, "Operation completed")
}

func TestRenderer_Error(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(FormatTable, true)
	r.SetWriter(&buf)

	r.Error("Something went wrong")

	output := buf.String()
	assert.Contains(t, output, "✗")
	assert.Contains(t, output, "Something went wrong")
}

func TestRenderer_RenderKeyValue_Table(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(FormatTable, true)
	r.SetWriter(&buf)

	r.RenderKeyValue("Status", "Active")

	output := buf.String()
	assert.Contains(t, output, "Status")
	assert.Contains(t, output, "Active")
}

func TestRenderer_RenderKeyValue_JSON(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(FormatJSON, true)
	r.SetWriter(&buf)

	r.RenderKeyValue("status", "active")

	output := strings.TrimSpace(buf.String())
	assert.Equal(t, `{"status": "active"}`, output)
}

func TestRenderer_EmptyTable(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(FormatTable, true)
	r.SetWriter(&buf)

	r.RenderTable([]string{"ID", "NAME"}, [][]string{})

	// Should still print headers
	output := buf.String()
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "NAME")
}

func TestRenderer_EmptyTable_JSON(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(FormatJSON, true)
	r.SetWriter(&buf)

	r.RenderTable([]string{"ID", "NAME"}, [][]string{})

	// Should print null (empty array gives nil slice)
	var result []map[string]string
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestRenderer_RowWithFewerColumns(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(FormatJSON, true)
	r.SetWriter(&buf)

	headers := []string{"ID", "NAME", "STATUS"}
	rows := [][]string{
		{"1", "First"}, // Missing STATUS
	}

	r.RenderTable(headers, rows)

	var result []map[string]string
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "1", result[0]["id"])
	assert.Equal(t, "First", result[0]["name"])
	// status should not be set
	_, exists := result[0]["status"]
	assert.False(t, exists)
}
