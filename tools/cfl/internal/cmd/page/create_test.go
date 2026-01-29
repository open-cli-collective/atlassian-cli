package page

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/confluence-cli/api"
)

// mockCreateServer creates a test server that handles GetSpaceByKey and CreatePage requests
func mockCreateServer(t *testing.T, spaceKey, spaceID string, createStatus int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/spaces") && r.URL.Query().Get("keys") != "":
			// GetSpaceByKey
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"results": [{"id": "` + spaceID + `", "key": "` + spaceKey + `", "name": "Test Space", "type": "global"}]}`))
		case r.Method == "POST" && strings.Contains(r.URL.Path, "/pages"):
			// CreatePage
			w.WriteHeader(createStatus)
			if createStatus == http.StatusOK {
				w.Write([]byte(`{
					"id": "99999",
					"title": "Test Page",
					"spaceId": "` + spaceID + `",
					"version": {"number": 1},
					"_links": {"webui": "/spaces/` + spaceKey + `/pages/99999"}
				}`))
			} else {
				w.Write([]byte(`{"message": "Create failed"}`))
			}
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestRunCreate_Success(t *testing.T) {
	// Create temp markdown file
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# Hello\n\nWorld"), 0644)
	require.NoError(t, err)

	server := mockCreateServer(t, "DEV", "123456", http.StatusOK)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &createOptions{
		space:   "DEV",
		title:   "Test Page",
		file:    mdFile,
		noColor: true,
	}

	err = runCreate(opts, client)
	require.NoError(t, err)
}

func TestRunCreate_HTMLFile_Legacy(t *testing.T) {
	// Create temp HTML file - should be treated as storage format in legacy mode
	tmpDir := t.TempDir()
	htmlFile := filepath.Join(tmpDir, "content.html")
	err := os.WriteFile(htmlFile, []byte("<p>Hello World</p>"), 0644)
	require.NoError(t, err)

	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/spaces"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"results": [{"id": "123456", "key": "DEV"}]}`))
		case r.Method == "POST" && strings.Contains(r.URL.Path, "/pages"):
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": "99999", "title": "Test", "version": {"number": 1}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &createOptions{
		space:   "DEV",
		title:   "Test Page",
		file:    htmlFile,
		legacy:  true, // Use legacy mode for HTML files
		noColor: true,
	}

	err = runCreate(opts, client)
	require.NoError(t, err)

	// Verify HTML was not converted (should be passed as-is in storage format)
	bodyMap := receivedBody["body"].(map[string]interface{})
	storageMap := bodyMap["storage"].(map[string]interface{})
	content := storageMap["value"].(string)
	assert.Equal(t, "<p>Hello World</p>", content)
}

func TestRunCreate_NoMarkdownFlag_Legacy(t *testing.T) {
	// Create temp file with markdown extension
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("<p>Raw XHTML</p>"), 0644)
	require.NoError(t, err)

	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/spaces"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"results": [{"id": "123456", "key": "DEV"}]}`))
		case r.Method == "POST" && strings.Contains(r.URL.Path, "/pages"):
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": "99999", "title": "Test", "version": {"number": 1}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	useMd := false
	opts := &createOptions{
		space:    "DEV",
		title:    "Test Page",
		file:     mdFile,
		markdown: &useMd, // Force no markdown conversion
		legacy:   true,   // Use legacy mode for storage format
		noColor:  true,
	}

	err = runCreate(opts, client)
	require.NoError(t, err)

	// Verify content was not converted even though file has .md extension
	bodyMap := receivedBody["body"].(map[string]interface{})
	storageMap := bodyMap["storage"].(map[string]interface{})
	content := storageMap["value"].(string)
	assert.Equal(t, "<p>Raw XHTML</p>", content)
}

func TestRunCreate_MissingSpace(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# Hello"), 0644)
	require.NoError(t, err)

	// Don't need server - should fail before API call
	client := api.NewClient("http://unused", "test@example.com", "token")
	opts := &createOptions{
		space:   "", // No space provided
		title:   "Test Page",
		file:    mdFile,
		noColor: true,
	}

	err = runCreate(opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "space is required")
}

func TestRunCreate_SpaceNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# Hello"), 0644)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return empty results for space lookup
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &createOptions{
		space:   "INVALID",
		title:   "Test Page",
		file:    mdFile,
		noColor: true,
	}

	err = runCreate(opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find space")
}

func TestRunCreate_CreateFailed(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# Hello"), 0644)
	require.NoError(t, err)

	server := mockCreateServer(t, "DEV", "123456", http.StatusForbidden)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &createOptions{
		space:   "DEV",
		title:   "Test Page",
		file:    mdFile,
		noColor: true,
	}

	err = runCreate(opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create page")
}

func TestRunCreate_WithParent(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# Child Page"), 0644)
	require.NoError(t, err)

	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/spaces"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"results": [{"id": "123456", "key": "DEV"}]}`))
		case r.Method == "POST" && strings.Contains(r.URL.Path, "/pages"):
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": "99999", "title": "Test", "version": {"number": 1}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &createOptions{
		space:   "DEV",
		title:   "Child Page",
		parent:  "12345",
		file:    mdFile,
		noColor: true,
	}

	err = runCreate(opts, client)
	require.NoError(t, err)

	// Verify parent ID was included in request
	assert.Equal(t, "12345", receivedBody["parentId"])
}

func TestRunCreate_JSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# Hello"), 0644)
	require.NoError(t, err)

	server := mockCreateServer(t, "DEV", "123456", http.StatusOK)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &createOptions{
		space:   "DEV",
		title:   "Test Page",
		file:    mdFile,
		output:  "json",
		noColor: true,
	}

	err = runCreate(opts, client)
	require.NoError(t, err)
}

func TestRunCreate_MarkdownConversion_Legacy(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# Hello World\n\nThis is **bold** text."), 0644)
	require.NoError(t, err)

	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/spaces"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"results": [{"id": "123456", "key": "DEV"}]}`))
		case r.Method == "POST" && strings.Contains(r.URL.Path, "/pages"):
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": "99999", "title": "Test", "version": {"number": 1}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &createOptions{
		space:   "DEV",
		title:   "Test Page",
		file:    mdFile,
		legacy:  true, // Use legacy mode to test storage format
		noColor: true,
	}

	err = runCreate(opts, client)
	require.NoError(t, err)

	// Verify markdown was converted to HTML storage format
	bodyMap := receivedBody["body"].(map[string]interface{})
	storageMap := bodyMap["storage"].(map[string]interface{})
	content := storageMap["value"].(string)

	// Should have HTML heading and strong tag from markdown conversion
	assert.Contains(t, content, "<h1")
	assert.Contains(t, content, "<strong>bold</strong>")
}

func TestRunCreate_MarkdownToADF(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# Hello World\n\nThis is **bold** text."), 0644)
	require.NoError(t, err)

	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/spaces"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"results": [{"id": "123456", "key": "DEV"}]}`))
		case r.Method == "POST" && strings.Contains(r.URL.Path, "/pages"):
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": "99999", "title": "Test", "version": {"number": 1}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &createOptions{
		space:   "DEV",
		title:   "Test Page",
		file:    mdFile,
		noColor: true,
		// Default: not legacy, uses ADF
	}

	err = runCreate(opts, client)
	require.NoError(t, err)

	// Verify ADF format was used (default)
	bodyMap := receivedBody["body"].(map[string]interface{})
	adfMap := bodyMap["atlas_doc_format"].(map[string]interface{})
	content := adfMap["value"].(string)

	// Should be valid ADF JSON with heading and strong mark
	assert.Contains(t, content, `"type":"doc"`)
	assert.Contains(t, content, `"type":"heading"`)
	assert.Contains(t, content, `"type":"strong"`)
}

func TestRunCreate_FileReadError(t *testing.T) {
	server := mockCreateServer(t, "DEV", "123456", http.StatusOK)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &createOptions{
		space:   "DEV",
		title:   "Test Page",
		file:    "/nonexistent/file.md",
		noColor: true,
	}

	err := runCreate(opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestRunCreate_Stdin_ADF(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/spaces"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"results": [{"id": "123456", "key": "DEV"}]}`))
		case r.Method == "POST" && strings.Contains(r.URL.Path, "/pages"):
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": "99999", "title": "Test", "version": {"number": 1}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &createOptions{
		space:   "DEV",
		title:   "Test Page",
		stdin:   strings.NewReader("# Hello\n\nThis is **bold** text."),
		noColor: true,
	}

	err := runCreate(opts, client)
	require.NoError(t, err)

	// Verify ADF format was used
	bodyMap := receivedBody["body"].(map[string]interface{})
	adfMap := bodyMap["atlas_doc_format"].(map[string]interface{})
	content := adfMap["value"].(string)

	assert.Contains(t, content, `"type":"doc"`)
	assert.Contains(t, content, `"type":"heading"`)
	assert.Contains(t, content, `"type":"strong"`)
}

func TestRunCreate_Stdin_Legacy(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/spaces"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"results": [{"id": "123456", "key": "DEV"}]}`))
		case r.Method == "POST" && strings.Contains(r.URL.Path, "/pages"):
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": "99999", "title": "Test", "version": {"number": 1}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &createOptions{
		space:   "DEV",
		title:   "Test Page",
		stdin:   strings.NewReader("# Hello\n\nThis is **bold** text."),
		legacy:  true,
		noColor: true,
	}

	err := runCreate(opts, client)
	require.NoError(t, err)

	// Verify storage format was used
	bodyMap := receivedBody["body"].(map[string]interface{})
	storageMap := bodyMap["storage"].(map[string]interface{})
	content := storageMap["value"].(string)

	assert.Contains(t, content, "<h1")
	assert.Contains(t, content, "<strong>bold</strong>")
}

func TestRunCreate_Stdin_NoMarkdown_Legacy(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/spaces"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"results": [{"id": "123456", "key": "DEV"}]}`))
		case r.Method == "POST" && strings.Contains(r.URL.Path, "/pages"):
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": "99999", "title": "Test", "version": {"number": 1}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	useMd := false
	opts := &createOptions{
		space:    "DEV",
		title:    "Test Page",
		stdin:    strings.NewReader("<p>Raw XHTML content</p>"),
		markdown: &useMd,
		legacy:   true,
		noColor:  true,
	}

	err := runCreate(opts, client)
	require.NoError(t, err)

	// Verify raw content passed through without conversion
	bodyMap := receivedBody["body"].(map[string]interface{})
	storageMap := bodyMap["storage"].(map[string]interface{})
	content := storageMap["value"].(string)

	assert.Equal(t, "<p>Raw XHTML content</p>", content)
}

func TestRunCreate_ComplexMarkdown_ADF(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/spaces"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"results": [{"id": "123456", "key": "DEV"}]}`))
		case r.Method == "POST" && strings.Contains(r.URL.Path, "/pages"):
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": "99999", "title": "Test", "version": {"number": 1}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	complexMarkdown := `# Title

| Header 1 | Header 2 |
|----------|----------|
| Cell 1   | Cell 2   |

- Item 1
  - Nested item
- Item 2

` + "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```"

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &createOptions{
		space:   "DEV",
		title:   "Test Page",
		stdin:   strings.NewReader(complexMarkdown),
		noColor: true,
	}

	err := runCreate(opts, client)
	require.NoError(t, err)

	// Verify ADF contains complex elements
	bodyMap := receivedBody["body"].(map[string]interface{})
	adfMap := bodyMap["atlas_doc_format"].(map[string]interface{})
	content := adfMap["value"].(string)

	assert.Contains(t, content, `"type":"table"`)
	assert.Contains(t, content, `"type":"bulletList"`)
	assert.Contains(t, content, `"type":"codeBlock"`)
	assert.Contains(t, content, `"language":"go"`)
}

func TestRunCreate_EmptyContentFromStdin(t *testing.T) {
	server := mockCreateServer(t, "DEV", "123456", http.StatusOK)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &createOptions{
		space:   "DEV",
		title:   "Test Page",
		stdin:   strings.NewReader(""),
		noColor: true,
	}

	err := runCreate(opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "page content cannot be empty")
}

func TestRunCreate_WhitespaceOnlyFromStdin(t *testing.T) {
	server := mockCreateServer(t, "DEV", "123456", http.StatusOK)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &createOptions{
		space:   "DEV",
		title:   "Test Page",
		stdin:   strings.NewReader("   \n\t\n   "),
		noColor: true,
	}

	err := runCreate(opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "page content cannot be empty")
}

func TestRunCreate_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.md")
	err := os.WriteFile(emptyFile, []byte(""), 0644)
	require.NoError(t, err)

	server := mockCreateServer(t, "DEV", "123456", http.StatusOK)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &createOptions{
		space:   "DEV",
		title:   "Test Page",
		file:    emptyFile,
		noColor: true,
	}

	err = runCreate(opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "page content cannot be empty")
}

func TestRunCreate_WhitespaceOnlyFile(t *testing.T) {
	tmpDir := t.TempDir()
	whitespaceFile := filepath.Join(tmpDir, "whitespace.md")
	err := os.WriteFile(whitespaceFile, []byte("   \n\t\n   "), 0644)
	require.NoError(t, err)

	server := mockCreateServer(t, "DEV", "123456", http.StatusOK)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &createOptions{
		space:   "DEV",
		title:   "Test Page",
		file:    whitespaceFile,
		noColor: true,
	}

	err = runCreate(opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "page content cannot be empty")
}
