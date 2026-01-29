package view

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	v := New("json", false)
	assert.Equal(t, FormatJSON, v.Format)
	assert.False(t, v.NoColor)

	v = New("table", true)
	assert.Equal(t, FormatTable, v.Format)
	assert.True(t, v.NoColor)
}

func TestView_Table(t *testing.T) {
	var buf bytes.Buffer
	v := &View{
		Format:  FormatTable,
		NoColor: true,
		Out:     &buf,
	}

	headers := []string{"KEY", "SUMMARY", "STATUS"}
	rows := [][]string{
		{"PROJ-1", "First issue", "Open"},
		{"PROJ-2", "Second issue", "Closed"},
	}

	err := v.Table(headers, rows)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "KEY")
	assert.Contains(t, output, "SUMMARY")
	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "PROJ-1")
	assert.Contains(t, output, "First issue")
	assert.Contains(t, output, "PROJ-2")
}

func TestView_Table_ErrorForJSON(t *testing.T) {
	var buf bytes.Buffer
	v := &View{
		Format: FormatJSON,
		Out:    &buf,
	}

	err := v.Table([]string{"A"}, [][]string{{"1"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JSON")
}

func TestView_JSON(t *testing.T) {
	var buf bytes.Buffer
	v := &View{
		Format: FormatJSON,
		Out:    &buf,
	}

	data := map[string]interface{}{
		"key":     "PROJ-123",
		"summary": "Test issue",
		"status":  "Open",
	}

	err := v.JSON(data)
	require.NoError(t, err)

	// Parse output and verify
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "PROJ-123", result["key"])
	assert.Equal(t, "Test issue", result["summary"])
}

func TestView_JSON_Slice(t *testing.T) {
	var buf bytes.Buffer
	v := &View{Format: FormatJSON, Out: &buf}

	data := []map[string]string{
		{"key": "PROJ-1"},
		{"key": "PROJ-2"},
	}

	err := v.JSON(data)
	require.NoError(t, err)

	var result []map[string]string
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestView_Plain(t *testing.T) {
	var buf bytes.Buffer
	v := &View{
		Format: FormatPlain,
		Out:    &buf,
	}

	rows := [][]string{
		{"PROJ-1", "First issue", "Open"},
		{"PROJ-2", "Second issue", "Closed"},
	}

	err := v.Plain(rows)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 2)
	assert.Equal(t, "PROJ-1\tFirst issue\tOpen", lines[0])
	assert.Equal(t, "PROJ-2\tSecond issue\tClosed", lines[1])
}

func TestView_Render(t *testing.T) {
	tests := []struct {
		name   string
		format Format
		check  func(t *testing.T, output string)
	}{
		{
			name:   "table format",
			format: FormatTable,
			check: func(t *testing.T, output string) {
				assert.Contains(t, output, "KEY")
				assert.Contains(t, output, "PROJ-1")
			},
		},
		{
			name:   "json format",
			format: FormatJSON,
			check: func(t *testing.T, output string) {
				var result []map[string]string
				err := json.Unmarshal([]byte(output), &result)
				require.NoError(t, err)
				assert.Equal(t, "PROJ-1", result[0]["key"])
			},
		},
		{
			name:   "plain format",
			format: FormatPlain,
			check: func(t *testing.T, output string) {
				assert.Contains(t, output, "PROJ-1\tFirst")
				assert.NotContains(t, output, "KEY")
			},
		},
	}

	headers := []string{"KEY", "SUMMARY"}
	rows := [][]string{{"PROJ-1", "First"}}
	jsonData := []map[string]string{{"key": "PROJ-1", "summary": "First"}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			v := &View{Format: tt.format, NoColor: true, Out: &buf}

			err := v.Render(headers, rows, jsonData)
			require.NoError(t, err)
			tt.check(t, buf.String())
		})
	}
}

func TestView_Success(t *testing.T) {
	var buf bytes.Buffer
	v := &View{NoColor: true, Out: &buf}

	v.Success("Created issue %s", "PROJ-123")

	output := buf.String()
	assert.Contains(t, output, "✓")
	assert.Contains(t, output, "Created issue PROJ-123")
}

func TestView_Error(t *testing.T) {
	var buf bytes.Buffer
	v := &View{NoColor: true, Err: &buf}

	v.Error("Failed to create issue: %s", "bad request")

	output := buf.String()
	assert.Contains(t, output, "✗")
	assert.Contains(t, output, "Failed to create issue: bad request")
}

func TestView_Warning(t *testing.T) {
	var buf bytes.Buffer
	v := &View{NoColor: true, Err: &buf}

	v.Warning("Config file not found")

	output := buf.String()
	assert.Contains(t, output, "⚠")
	assert.Contains(t, output, "Config file not found")
}

func TestView_Info(t *testing.T) {
	var buf bytes.Buffer
	v := &View{Out: &buf}

	v.Info("Processing %d issues", 5)

	output := buf.String()
	assert.Contains(t, output, "Processing 5 issues")
}

func TestView_Print(t *testing.T) {
	var buf bytes.Buffer
	v := &View{Out: &buf}

	v.Print("Hello %s", "world")

	assert.Equal(t, "Hello world", buf.String())
}

func TestView_Println(t *testing.T) {
	var buf bytes.Buffer
	v := &View{Out: &buf}

	v.Println("Hello %s", "world")

	assert.Equal(t, "Hello world\n", buf.String())
}

func TestFormat_Constants(t *testing.T) {
	assert.Equal(t, Format("table"), FormatTable)
	assert.Equal(t, Format("json"), FormatJSON)
	assert.Equal(t, Format("plain"), FormatPlain)
}
