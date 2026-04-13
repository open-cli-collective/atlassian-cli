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

func TestView_JSON_Compact(t *testing.T) {
	t.Parallel()

	t.Run("strips null fields", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.Compact = true
		v.SetOutput(buf)

		data := map[string]any{
			"key":         "MON-123",
			"summary":     "Fix bug",
			"description": nil,
			"labels":      nil,
		}
		if err := v.JSON(data); err != nil {
			t.Fatalf("JSON() error = %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if _, ok := result["description"]; ok {
			t.Error("compact should strip null 'description'")
		}
		if _, ok := result["labels"]; ok {
			t.Error("compact should strip null 'labels'")
		}
		if result["key"] != "MON-123" {
			t.Error("compact should preserve non-null fields")
		}
	})

	t.Run("strips avatarUrls", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.Compact = true
		v.SetOutput(buf)

		data := map[string]any{
			"displayName": "Alice",
			"avatarUrls": map[string]any{
				"48x48": "https://avatar.example.com/48",
				"32x32": "https://avatar.example.com/32",
			},
		}
		if err := v.JSON(data); err != nil {
			t.Fatalf("JSON() error = %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if _, ok := result["avatarUrls"]; ok {
			t.Error("compact should strip avatarUrls")
		}
		if result["displayName"] != "Alice" {
			t.Error("compact should preserve displayName")
		}
	})

	t.Run("strips self links", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.Compact = true
		v.SetOutput(buf)

		data := map[string]any{
			"id":   "123",
			"self": "https://example.atlassian.net/rest/api/3/issue/123",
		}
		if err := v.JSON(data); err != nil {
			t.Fatalf("JSON() error = %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if _, ok := result["self"]; ok {
			t.Error("compact should strip self API links")
		}
	})

	t.Run("strips self for any HTTP URL", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.Compact = true
		v.SetOutput(buf)

		// Confluence v2 uses /wiki/api/v2/ paths, not /rest/
		data := map[string]any{
			"id":   "123",
			"self": "https://example.atlassian.net/wiki/api/v2/pages/123",
		}
		if err := v.JSON(data); err != nil {
			t.Fatalf("JSON() error = %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if _, ok := result["self"]; ok {
			t.Error("compact should strip self for Confluence v2 URLs")
		}
	})

	t.Run("preserves self when not a URL", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.Compact = true
		v.SetOutput(buf)

		data := map[string]any{
			"self": "this is not a URL",
		}
		if err := v.JSON(data); err != nil {
			t.Fatalf("JSON() error = %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if _, ok := result["self"]; !ok {
			t.Error("compact should preserve non-URL 'self' values")
		}
	})

	t.Run("preserves originally-empty arrays and maps", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.Compact = true
		v.SetOutput(buf)

		data := map[string]any{
			"key":        "MON-1",
			"labels":     []any{},
			"components": []any{},
			"subtasks":   []any{},
		}
		if err := v.JSON(data); err != nil {
			t.Fatalf("JSON() error = %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if _, ok := result["labels"]; !ok {
			t.Error("compact should preserve originally-empty 'labels' array")
		}
		if _, ok := result["components"]; !ok {
			t.Error("compact should preserve originally-empty 'components' array")
		}
	})

	t.Run("strips _links and _expandable", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.Compact = true
		v.SetOutput(buf)

		data := map[string]any{
			"title":       "My Page",
			"_links":      map[string]any{"webui": "/spaces/ENG/pages/123"},
			"_expandable": map[string]any{"body": ""},
		}
		if err := v.JSON(data); err != nil {
			t.Fatalf("JSON() error = %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if _, ok := result["_links"]; ok {
			t.Error("compact should strip _links")
		}
		if _, ok := result["_expandable"]; ok {
			t.Error("compact should strip _expandable")
		}
		if result["title"] != "My Page" {
			t.Error("compact should preserve title")
		}
	})

	t.Run("recurses into nested objects and arrays", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.Compact = true
		v.SetOutput(buf)

		data := map[string]any{
			"fields": map[string]any{
				"assignee": map[string]any{
					"displayName": "Bob",
					"avatarUrls":  map[string]any{"48x48": "https://avatar.example.com"},
					"self":        "https://example.atlassian.net/rest/api/3/user/bob",
				},
				"customfield_10001": nil,
			},
			"items": []any{
				map[string]any{"id": "1", "self": "https://example.atlassian.net/rest/api/3/item/1"},
				map[string]any{"id": "2", "extra": nil},
			},
		}
		if err := v.JSON(data); err != nil {
			t.Fatalf("JSON() error = %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		fields := result["fields"].(map[string]any)
		assignee := fields["assignee"].(map[string]any)
		if assignee["displayName"] != "Bob" {
			t.Error("should preserve nested displayName")
		}
		if _, ok := assignee["avatarUrls"]; ok {
			t.Error("should strip nested avatarUrls")
		}
		if _, ok := assignee["self"]; ok {
			t.Error("should strip nested self links")
		}
		if _, ok := fields["customfield_10001"]; ok {
			t.Error("should strip nested null fields")
		}

		items := result["items"].([]any)
		item0 := items[0].(map[string]any)
		if _, ok := item0["self"]; ok {
			t.Error("should strip self in array items")
		}
		item1 := items[1].(map[string]any)
		if _, ok := item1["extra"]; ok {
			t.Error("should strip null in array items")
		}
	})

	t.Run("strips empty maps after pruning", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.Compact = true
		v.SetOutput(buf)

		data := map[string]any{
			"key": "MON-1",
			"assignee": map[string]any{
				"avatarUrls": map[string]any{"48x48": "https://example.com"},
				"self":       "https://example.atlassian.net/rest/api/3/user/1",
			},
		}
		if err := v.JSON(data); err != nil {
			t.Fatalf("JSON() error = %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if _, ok := result["assignee"]; ok {
			t.Error("compact should strip maps left empty after pruning")
		}
	})

	t.Run("works with top-level array", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.Compact = true
		v.SetOutput(buf)

		data := []any{
			map[string]any{
				"id":   "1",
				"self": "https://example.atlassian.net/rest/api/3/issue/1",
				"name": nil,
			},
			map[string]any{
				"id": "2",
			},
		}
		if err := v.JSON(data); err != nil {
			t.Fatalf("JSON() error = %v", err)
		}

		var result []map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 items, got %d", len(result))
		}
		if _, ok := result[0]["self"]; ok {
			t.Error("should strip self in array items")
		}
		if _, ok := result[0]["name"]; ok {
			t.Error("should strip null in array items")
		}
		if result[0]["id"] != "1" {
			t.Error("should preserve non-null fields in array items")
		}
	})

	t.Run("drops empty array items after pruning", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.Compact = true
		v.SetOutput(buf)

		data := []any{
			map[string]any{
				"id":         "1",
				"avatarUrls": map[string]any{"48x48": "https://example.com"},
			},
			map[string]any{
				// This item has only stripped fields — should be dropped entirely
				"avatarUrls": map[string]any{"48x48": "https://example.com"},
				"self":       "https://example.atlassian.net/rest/api/3/user/1",
			},
			map[string]any{
				"id": "3",
			},
		}
		if err := v.JSON(data); err != nil {
			t.Fatalf("JSON() error = %v", err)
		}

		var result []map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 items after pruning empty, got %d", len(result))
		}
		if result[0]["id"] != "1" || result[1]["id"] != "3" {
			t.Error("surviving items should be id=1 and id=3")
		}
	})

	t.Run("compact is no-op for table format", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatTable, true)
		v.Compact = true
		v.SetOutput(buf)

		headers := []string{"KEY", "STATUS"}
		rows := [][]string{{"MON-1", "Open"}}
		if err := v.Table(headers, rows); err != nil {
			t.Fatalf("Table() error = %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "MON-1") {
			t.Error("table output should still contain row data when compact is set")
		}
	})

	t.Run("compact is no-op for plain format", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatPlain, false)
		v.Compact = true
		v.SetOutput(buf)

		rows := [][]string{{"a", "b"}}
		if err := v.Plain(rows); err != nil {
			t.Fatalf("Plain() error = %v", err)
		}
		if !strings.Contains(buf.String(), "a\tb") {
			t.Error("plain output should be unchanged when compact is set")
		}
	})

	t.Run("no-op when compact is false", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		// Compact is false by default
		v.SetOutput(buf)

		data := map[string]any{
			"self":        "https://example.atlassian.net/rest/api/3/issue/1",
			"description": nil,
		}
		if err := v.JSON(data); err != nil {
			t.Fatalf("JSON() error = %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if _, ok := result["self"]; !ok {
			t.Error("non-compact should preserve self links")
		}
		// Note: encoding/json does include null values from map[string]any
		// but nil in the original map becomes json null
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

	t.Run("does not apply Compact post-processing", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.Compact = true // Setting this should have no effect on RenderArtifact
		v.SetOutput(buf)

		data := map[string]any{
			"key":  "PROJ-1",
			"self": "https://example.atlassian.net/rest/api/3/issue/1",
		}

		err := v.RenderArtifact(data)
		if err != nil {
			t.Fatalf("RenderArtifact() error = %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		// With Compact=true, JSON() would strip "self", but RenderArtifact should preserve it
		if _, ok := result["self"]; !ok {
			t.Error("RenderArtifact should not apply Compact — 'self' should be preserved")
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

	t.Run("does not apply Compact post-processing", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		v := New(FormatJSON, false)
		v.Compact = true
		v.SetOutput(buf)

		items := []map[string]any{
			{
				"id":   "1",
				"self": "https://example.atlassian.net/rest/api/3/item/1",
			},
		}
		result := artifact.NewListResult(items, false)

		err := v.RenderArtifactList(result)
		if err != nil {
			t.Fatalf("RenderArtifactList() error = %v", err)
		}

		var parsed struct {
			Results []map[string]any `json:"results"`
		}
		if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if _, ok := parsed.Results[0]["self"]; !ok {
			t.Error("RenderArtifactList should not apply Compact — 'self' should be preserved")
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
