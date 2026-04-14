package view

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/artifact"
)

func TestValidFormats(t *testing.T) {
	t.Parallel()
	formats := ValidFormats()

	expected := []string{"table", "json", "plain"}
	if len(formats) != len(expected) {
		t.Errorf("ValidFormats() returned %d formats, want %d", len(formats), len(expected))
	}

	for _, exp := range expected {
		found := false
		for _, f := range formats {
			if f == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ValidFormats() missing %q", exp)
		}
	}
}

func TestValidateFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		format  string
		wantErr bool
	}{
		{"", false},
		{"table", false},
		{"json", false},
		{"plain", false},
		{"xml", true},
		{"csv", true},
		{"INVALID", true},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			t.Parallel()
			err := ValidateFormat(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFormat(%q) error = %v, wantErr = %v", tt.format, err, tt.wantErr)
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()
	t.Run("default options", func(t *testing.T) {
		t.Parallel()
		v := New(FormatTable, false)

		if v.Format != FormatTable {
			t.Errorf("Format = %v, want table", v.Format)
		}

		if v.NoColor {
			t.Error("NoColor should be false")
		}

		if v.Out == nil {
			t.Error("Out should not be nil")
		}

		if v.Err == nil {
			t.Error("Err should not be nil")
		}
	})

	t.Run("with noColor", func(t *testing.T) {
		t.Parallel()
		v := New(FormatJSON, true)

		if !v.NoColor {
			t.Error("NoColor should be true")
		}
	})
}

func TestNewWithFormat(t *testing.T) {
	t.Parallel()
	v := NewWithFormat("json", false)

	if v.Format != FormatJSON {
		t.Errorf("Format = %v, want json", v.Format)
	}
}

func TestView_Table(t *testing.T) {
	t.Parallel()
	headers := []string{"ID", "NAME", "STATUS"}
	rows := [][]string{
		{"1", "Item One", "Active"},
		{"2", "Item Two", "Inactive"},
	}

	t.Run("table format", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatTable, true) // noColor for predictable output
		v.SetOutput(buf)

		err := v.Table(headers, rows)
		if err != nil {
			t.Fatalf("Table() error = %v", err)
		}

		output := buf.String()

		// Check headers are present
		if !strings.Contains(output, "ID") {
			t.Error("Output should contain header 'ID'")
		}
		if !strings.Contains(output, "NAME") {
			t.Error("Output should contain header 'NAME'")
		}

		// Check rows are present
		if !strings.Contains(output, "Item One") {
			t.Error("Output should contain 'Item One'")
		}
		if !strings.Contains(output, "Item Two") {
			t.Error("Output should contain 'Item Two'")
		}
	})

	t.Run("json format via Table", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.SetOutput(buf)

		err := v.Table(headers, rows)
		if err != nil {
			t.Fatalf("Table() error = %v", err)
		}

		// Verify it's valid JSON
		var result []map[string]string
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Output is not valid JSON: %v", err)
		}

		// Headers should be lowercase
		if result[0]["id"] != "1" {
			t.Errorf("Expected id=1, got %v", result[0]["id"])
		}
		if result[0]["name"] != "Item One" {
			t.Errorf("Expected name='Item One', got %v", result[0]["name"])
		}
	})

	t.Run("plain format via Table", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatPlain, false)
		v.SetOutput(buf)

		err := v.Table(headers, rows)
		if err != nil {
			t.Fatalf("Table() error = %v", err)
		}

		output := buf.String()

		// Should not contain headers
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) != 2 {
			t.Errorf("Expected 2 lines, got %d", len(lines))
		}

		// First line should be first data row
		if !strings.Contains(lines[0], "Item One") {
			t.Errorf("First line should contain 'Item One': %s", lines[0])
		}
	})
}

func TestView_JSON(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	v := New(FormatJSON, false)
	v.SetOutput(buf)

	data := map[string]interface{}{
		"id":   123,
		"name": "Test",
	}

	err := v.JSON(data)
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if result["name"] != "Test" {
		t.Errorf("Expected name='Test', got %v", result["name"])
	}
}

func TestView_Plain(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	v := New(FormatPlain, false)
	v.SetOutput(buf)

	rows := [][]string{
		{"a", "b", "c"},
		{"d", "e", "f"},
	}

	err := v.Plain(rows)
	if err != nil {
		t.Fatalf("Plain() error = %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}

	if !strings.Contains(lines[0], "a\tb\tc") {
		t.Errorf("First line should be tab-separated: %s", lines[0])
	}
}

func TestView_Render(t *testing.T) {
	t.Parallel()
	headers := []string{"KEY", "VALUE"}
	rows := [][]string{{"k1", "v1"}}
	jsonData := map[string]string{"key": "value"}

	t.Run("table format", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatTable, true)
		v.SetOutput(buf)

		err := v.Render(headers, rows, jsonData)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}

		if !strings.Contains(buf.String(), "KEY") {
			t.Error("Should render as table")
		}
	})

	t.Run("json format", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.SetOutput(buf)

		err := v.Render(headers, rows, jsonData)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}

		if !strings.Contains(buf.String(), `"key"`) {
			t.Error("Should render as JSON")
		}
	})

	t.Run("plain format", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatPlain, false)
		v.SetOutput(buf)

		err := v.Render(headers, rows, jsonData)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}

		output := buf.String()
		if strings.Contains(output, "KEY") {
			t.Error("Plain should not include headers")
		}
		if !strings.Contains(output, "k1") {
			t.Error("Should contain row data")
		}
	})
}

func TestView_Messages(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatTable, true)
		v.SetOutput(buf)

		v.Success("Operation %s", "completed")

		if !strings.Contains(buf.String(), "✓") {
			t.Error("Success should contain checkmark")
		}
		if !strings.Contains(buf.String(), "Operation completed") {
			t.Error("Success should contain formatted message")
		}
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatTable, true)
		v.SetError(buf)

		v.Error("Failed: %s", "reason")

		if !strings.Contains(buf.String(), "✗") {
			t.Error("Error should contain X mark")
		}
		if !strings.Contains(buf.String(), "Failed: reason") {
			t.Error("Error should contain formatted message")
		}
	})

	t.Run("Warning", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatTable, true)
		v.SetError(buf)

		v.Warning("Caution: %s", "be careful")

		if !strings.Contains(buf.String(), "⚠") {
			t.Error("Warning should contain warning symbol")
		}
		if !strings.Contains(buf.String(), "Caution: be careful") {
			t.Error("Warning should contain formatted message")
		}
	})

	t.Run("Info", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatTable, false)
		v.SetOutput(buf)

		v.Info("Status: %s", "ready")

		if !strings.Contains(buf.String(), "Status: ready") {
			t.Error("Info should contain formatted message")
		}
	})

	t.Run("Print", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatTable, false)
		v.SetOutput(buf)

		v.Print("no newline: %d", 42)

		output := buf.String()
		if output != "no newline: 42" {
			t.Errorf("Print output = %q, want 'no newline: 42'", output)
		}
	})

	t.Run("Println", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatTable, false)
		v.SetOutput(buf)

		v.Println("with newline: %d", 42)

		output := buf.String()
		if output != "with newline: 42\n" {
			t.Errorf("Println output = %q, want 'with newline: 42\\n'", output)
		}
	})
}

func TestTruncate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is too long", 10, "this is..."},
		{"ab", 3, "ab"},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
		{"a", 1, "a"},
		{"abc", 2, "ab"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := Truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestView_SetOutput(t *testing.T) {
	t.Parallel()
	v := New(FormatTable, false)

	buf := &bytes.Buffer{}
	v.SetOutput(buf)

	v.Println("test")

	if !strings.Contains(buf.String(), "test") {
		t.Error("Output should go to custom writer")
	}
}

func TestView_SetError(t *testing.T) {
	t.Parallel()
	v := New(FormatTable, true)

	buf := &bytes.Buffer{}
	v.SetError(buf)

	v.Error("test error")

	if !strings.Contains(buf.String(), "test error") {
		t.Error("Errors should go to custom writer")
	}
}

func TestView_RenderList(t *testing.T) {
	t.Parallel()
	headers := []string{"ID", "NAME"}
	rows := [][]string{
		{"1", "First"},
		{"2", "Second"},
	}

	t.Run("table format", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatTable, true)
		v.SetOutput(buf)

		err := v.RenderList(headers, rows, false)
		if err != nil {
			t.Fatalf("RenderList() error = %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "ID") {
			t.Error("Should contain header 'ID'")
		}
		if !strings.Contains(output, "First") {
			t.Error("Should contain row data")
		}
	})

	t.Run("json format with hasMore=false", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.SetOutput(buf)

		err := v.RenderList(headers, rows, false)
		if err != nil {
			t.Fatalf("RenderList() error = %v", err)
		}

		var result ListResponse
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Output is not valid JSON: %v", err)
		}

		if result.Meta.Count != 2 {
			t.Errorf("Expected count=2, got %d", result.Meta.Count)
		}
		if result.Meta.HasMore {
			t.Error("Expected hasMore=false")
		}
		if len(result.Results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(result.Results))
		}
		if result.Results[0]["id"] != "1" {
			t.Errorf("Expected id=1, got %v", result.Results[0]["id"])
		}
	})

	t.Run("json format with hasMore=true", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.SetOutput(buf)

		err := v.RenderList(headers, rows, true)
		if err != nil {
			t.Fatalf("RenderList() error = %v", err)
		}

		var result ListResponse
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Output is not valid JSON: %v", err)
		}

		if !result.Meta.HasMore {
			t.Error("Expected hasMore=true")
		}
	})
}

func TestView_RenderKeyValue(t *testing.T) {
	t.Parallel()
	t.Run("table format", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatTable, true)
		v.SetOutput(buf)

		v.RenderKeyValue("Name", "TestValue")

		output := buf.String()
		if !strings.Contains(output, "Name:") {
			t.Error("Should contain key with colon")
		}
		if !strings.Contains(output, "TestValue") {
			t.Error("Should contain value")
		}
	})

	t.Run("json format", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.SetOutput(buf)

		v.RenderKeyValue("Name", "TestValue")

		output := buf.String()
		if !strings.Contains(output, `"Name"`) {
			t.Error("Should contain JSON key")
		}
		if !strings.Contains(output, `"TestValue"`) {
			t.Error("Should contain JSON value")
		}
	})
}

func TestView_RenderText(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	v := New(FormatTable, false)
	v.SetOutput(buf)

	v.RenderText("Hello World")

	output := buf.String()
	if output != "Hello World\n" {
		t.Errorf("RenderText output = %q, want 'Hello World\\n'", output)
	}
}

func TestView_RenderArtifact(t *testing.T) {
	t.Parallel()

	t.Run("renders struct as JSON", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.SetOutput(buf)

		type TestArtifact struct {
			Key     string `json:"key"`
			Summary string `json:"summary"`
		}
		artifact := TestArtifact{Key: "PROJ-1", Summary: "Test issue"}

		err := v.RenderArtifact(artifact)
		if err != nil {
			t.Fatalf("RenderArtifact() error = %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if result["key"] != "PROJ-1" {
			t.Errorf("key = %v, want PROJ-1", result["key"])
		}
		if result["summary"] != "Test issue" {
			t.Errorf("summary = %v, want 'Test issue'", result["summary"])
		}
	})

	t.Run("omits empty fields with omitempty", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.SetOutput(buf)

		type ArtifactWithOptional struct {
			Key     string `json:"key"`
			Created string `json:"created,omitempty"`
		}
		artifact := ArtifactWithOptional{Key: "PROJ-1"} // Created is empty

		err := v.RenderArtifact(artifact)
		if err != nil {
			t.Fatalf("RenderArtifact() error = %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if _, ok := result["created"]; ok {
			t.Error("empty field with omitempty should not appear in output")
		}
	})
}

func TestView_RenderArtifactList(t *testing.T) {
	t.Parallel()

	t.Run("renders list with metadata", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.SetOutput(buf)

		type ItemArtifact struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		items := []*ItemArtifact{
			{ID: "1", Name: "First"},
			{ID: "2", Name: "Second"},
		}
		result := artifact.NewListResult(items, true)

		err := v.RenderArtifactList(result)
		if err != nil {
			t.Fatalf("RenderArtifactList() error = %v", err)
		}

		var parsed struct {
			Results []map[string]any `json:"results"`
			Meta    struct {
				Count   int  `json:"count"`
				HasMore bool `json:"hasMore"`
			} `json:"_meta"`
		}
		if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if parsed.Meta.Count != 2 {
			t.Errorf("Meta.Count = %d, want 2", parsed.Meta.Count)
		}
		if !parsed.Meta.HasMore {
			t.Error("Meta.HasMore = false, want true")
		}
		if len(parsed.Results) != 2 {
			t.Errorf("len(Results) = %d, want 2", len(parsed.Results))
		}
	})

	t.Run("handles empty list", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.SetOutput(buf)

		items := []string{}
		result := artifact.NewListResult(items, false)

		err := v.RenderArtifactList(result)
		if err != nil {
			t.Fatalf("RenderArtifactList() error = %v", err)
		}

		output := buf.String()
		// Verify _meta is present in raw output (not just parsed as zero values)
		if !strings.Contains(output, `"_meta"`) {
			t.Error("output should contain _meta key")
		}
		if !strings.Contains(output, `"hasMore"`) {
			t.Error("output should contain hasMore key even when false")
		}

		var parsed struct {
			Results []any `json:"results"`
			Meta    struct {
				Count   int  `json:"count"`
				HasMore bool `json:"hasMore"`
			} `json:"_meta"`
		}
		if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if parsed.Meta.Count != 0 {
			t.Errorf("Meta.Count = %d, want 0", parsed.Meta.Count)
		}
		if len(parsed.Results) != 0 {
			t.Errorf("len(Results) = %d, want 0", len(parsed.Results))
		}
	})
}
