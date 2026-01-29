package page

import (
	"bytes"
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
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

func newEditTestRootOptions() *root.Options {
	return &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
}

func TestRunEdit_Success(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# Updated Content\n\nNew text here."), 0644)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/pages/12345"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 5},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/pages/12345"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 6},
				"_links": {"webui": "/pages/12345"}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = nil
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		file:    mdFile,
	}

	err = runEdit(opts)
	require.NoError(t, err)
}

func TestRunEdit_TitleOnly(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/pages/12345"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Old Title",
				"version": {"number": 3},
				"body": {"storage": {"representation": "storage", "value": "<p>Keep this</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/pages/12345"):
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "New Title",
				"version": {"number": 4},
				"_links": {"webui": "/pages/12345"}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = nil
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		title:   "New Title",
	}

	// Note: Without file input and with a title, the current implementation
	// will still try to open an editor. For this test to work properly,
	// we need to provide a file to avoid the editor path.
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("<p>Keep this</p>"), 0644)
	require.NoError(t, err)

	useMd := false
	opts.file = mdFile
	opts.markdown = &useMd

	err = runEdit(opts)
	require.NoError(t, err)

	// Verify title was changed
	assert.Equal(t, "New Title", receivedBody["title"])
}

func TestRunEdit_PageNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Page not found"}`))
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = nil
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "99999",
		title:   "New Title",
	}

	err := runEdit(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get page")
}

func TestRunEdit_UpdateFailed(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# New Content"), 0644)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"message": "Permission denied"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = nil
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		file:    mdFile,
	}

	err = runEdit(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update page")
}

func TestRunEdit_VersionIncrement(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# Updated"), 0644)
	require.NoError(t, err)

	var receivedVersion int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 7},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)
			if v, ok := req["version"].(map[string]interface{}); ok {
				receivedVersion = int(v["number"].(float64))
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 8},
				"_links": {"webui": "/pages/12345"}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = nil
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		file:    mdFile,
	}

	err = runEdit(opts)
	require.NoError(t, err)

	// Verify version was incremented from 7 to 8
	assert.Equal(t, 8, receivedVersion)
}

func TestRunEdit_HTMLFile(t *testing.T) {
	tmpDir := t.TempDir()
	htmlFile := filepath.Join(tmpDir, "content.html")
	err := os.WriteFile(htmlFile, []byte("<p>Direct HTML</p>"), 0644)
	require.NoError(t, err)

	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = nil
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		file:    htmlFile,
		legacy:  true, // Use legacy mode for HTML files

	}

	err = runEdit(opts)
	require.NoError(t, err)

	// Verify HTML was not converted (storage format in legacy mode)
	bodyMap := receivedBody["body"].(map[string]interface{})
	storageMap := bodyMap["storage"].(map[string]interface{})
	content := storageMap["value"].(string)
	assert.Equal(t, "<p>Direct HTML</p>", content)
}

func TestRunEdit_NoMarkdownFlag(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("<p>Raw XHTML in .md file</p>"), 0644)
	require.NoError(t, err)

	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	useMd := false
	rootOpts.Stdin = nil
	opts := &editOptions{
		Options:  rootOpts,
		pageID:   "12345",
		file:     mdFile,
		markdown: &useMd,
		legacy:   true, // Use legacy mode for storage format
	}

	err = runEdit(opts)
	require.NoError(t, err)

	// Verify content was not converted (storage format in legacy mode)
	bodyMap := receivedBody["body"].(map[string]interface{})
	storageMap := bodyMap["storage"].(map[string]interface{})
	content := storageMap["value"].(string)
	assert.Equal(t, "<p>Raw XHTML in .md file</p>", content)
}

func TestRunEdit_MarkdownToADF(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# Updated\n\nNew **bold** text."), 0644)
	require.NoError(t, err)

	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = nil
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		file:    mdFile,

		// Default: not legacy, uses ADF
	}

	err = runEdit(opts)
	require.NoError(t, err)

	// Verify ADF format was used (default)
	bodyMap := receivedBody["body"].(map[string]interface{})
	adfMap := bodyMap["atlas_doc_format"].(map[string]interface{})
	content := adfMap["value"].(string)

	// Should be valid ADF JSON
	assert.Contains(t, content, `"type":"doc"`)
	assert.Contains(t, content, `"type":"heading"`)
	assert.Contains(t, content, `"type":"strong"`)
}

func TestRunEdit_JSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# Updated"), 0644)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = nil
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		file:    mdFile,
	}

	err = runEdit(opts)
	require.NoError(t, err)
}

func TestRunEdit_FileReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "12345",
			"title": "Test",
			"version": {"number": 1},
			"body": {"storage": {"value": "<p>Old</p>"}},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = nil
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		file:    "/nonexistent/file.md",
	}

	err := runEdit(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestRunEdit_Stdin_ADF(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = strings.NewReader("# Heading\n\nSome **bold** text.")
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
	}

	err := runEdit(opts)
	require.NoError(t, err)

	// Verify ADF format was used
	bodyMap := receivedBody["body"].(map[string]interface{})
	adfMap := bodyMap["atlas_doc_format"].(map[string]interface{})
	content := adfMap["value"].(string)

	assert.Contains(t, content, `"type":"doc"`)
	assert.Contains(t, content, `"type":"heading"`)
	assert.Contains(t, content, `"type":"strong"`)
}

func TestRunEdit_Stdin_Legacy(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = strings.NewReader("# Heading\n\nSome **bold** text.")
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		legacy:  true,
	}

	err := runEdit(opts)
	require.NoError(t, err)

	// Verify storage format was used
	bodyMap := receivedBody["body"].(map[string]interface{})
	storageMap := bodyMap["storage"].(map[string]interface{})
	content := storageMap["value"].(string)

	assert.Contains(t, content, "<h1")
	assert.Contains(t, content, "<strong>bold</strong>")
}

func TestRunEdit_TitleAndContent(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Old Title",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "New Title",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# New Content\n\nUpdated text here."), 0644)
	require.NoError(t, err)

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		title:   "New Title",
		file:    mdFile,
	}

	err = runEdit(opts)
	require.NoError(t, err)

	// Verify both title and content were updated
	assert.Equal(t, "New Title", receivedBody["title"])
	bodyMap := receivedBody["body"].(map[string]interface{})
	adfMap := bodyMap["atlas_doc_format"].(map[string]interface{})
	assert.NotNil(t, adfMap["value"])
}

func TestRunEdit_ComplexMarkdown_ADF(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
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

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "complex.md")
	err := os.WriteFile(mdFile, []byte(complexMarkdown), 0644)
	require.NoError(t, err)

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		file:    mdFile,
	}

	err = runEdit(opts)
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

func TestRunEdit_MoveToParent(t *testing.T) {
	moveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Content</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/rest/api/content/12345/move/append/67890"):
			moveCalled = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = nil
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		title:   "Test Page", // Keep same title to avoid editor
		parent:  "67890",
	}

	err := runEdit(opts)
	require.NoError(t, err)
	assert.True(t, moveCalled, "MovePage should have been called")
}

func TestRunEdit_MoveAndRename(t *testing.T) {
	var receivedTitle string
	moveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Old Title",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Content</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)
			receivedTitle = req["title"].(string)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "New Title",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/rest/api/content/12345/move/append/67890"):
			moveCalled = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = nil
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		title:   "New Title",
		parent:  "67890",
	}

	err := runEdit(opts)
	require.NoError(t, err)
	assert.True(t, moveCalled, "MovePage should have been called")
	assert.Equal(t, "New Title", receivedTitle)
}

func TestRunEdit_MoveFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Content</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/rest/api/content/12345/move"):
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "Target page not found"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = nil
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		title:   "Test Page",
		parent:  "99999", // Invalid parent

	}

	err := runEdit(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to move page to new parent")
}

func TestRunEdit_MoveWithContent(t *testing.T) {
	moveCalled := false
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/rest/api/content/12345/move/append/67890"):
			moveCalled = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = strings.NewReader("# New Content\n\nUpdated during move.")
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		parent:  "67890",
	}

	err := runEdit(opts)
	require.NoError(t, err)
	assert.True(t, moveCalled, "MovePage should have been called")

	// Verify content was also updated
	bodyMap := receivedBody["body"].(map[string]interface{})
	adfMap := bodyMap["atlas_doc_format"].(map[string]interface{})
	assert.NotNil(t, adfMap["value"])
}

func TestRunEdit_EmptyContentFromStdin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old content</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = strings.NewReader("")
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
	}

	err := runEdit(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "page content cannot be empty")
}

func TestRunEdit_WhitespaceOnlyFromStdin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old content</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = strings.NewReader("   \n\t\n  ")
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
	}

	err := runEdit(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "page content cannot be empty")
}

func TestRunEdit_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.md")
	err := os.WriteFile(emptyFile, []byte(""), 0644)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old content</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = nil
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		file:    emptyFile,
	}

	err = runEdit(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "page content cannot be empty")
}

func TestRunEdit_WhitespaceOnlyFile(t *testing.T) {
	tmpDir := t.TempDir()
	whitespaceFile := filepath.Join(tmpDir, "whitespace.md")
	err := os.WriteFile(whitespaceFile, []byte("   \n\t\n   "), 0644)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old content</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = nil
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		file:    whitespaceFile,
	}

	err = runEdit(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "page content cannot be empty")
}

func TestRunEdit_TitleOnlyUpdate_NoContentValidation(t *testing.T) {
	// When updating title only (with file providing content), validation should pass
	updateCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Old Title",
				"version": {"number": 1},
				"body": {"storage": {"representation": "storage", "value": "<p>Existing content</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			updateCalled = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "New Title",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	// Provide a file with valid content to avoid editor
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# Valid Content"), 0644)
	require.NoError(t, err)

	rootOpts.Stdin = nil
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		title:   "New Title",
		file:    mdFile,
	}

	err = runEdit(opts)
	require.NoError(t, err)
	assert.True(t, updateCalled, "Update should have been called")
}

func TestRunEdit_MoveOnly_NoEditorOpened(t *testing.T) {
	// Test: cfl page edit 12345 --parent 67890
	// Verifies: page is moved without content change, no editor opened
	moveCalled := false
	updateCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 1},
				"body": {"storage": {"representation": "storage", "value": "<p>Original content</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			updateCalled = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/rest/api/content/12345/move/append/67890"):
			moveCalled = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = nil
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		parent:  "67890",
	}

	err := runEdit(opts)
	require.NoError(t, err)
	assert.True(t, updateCalled, "UpdatePage should have been called")
	assert.True(t, moveCalled, "MovePage should have been called")
}

func TestRunEdit_MoveWithTitleOnly_NoEditorOpened(t *testing.T) {
	// Test: cfl page edit 12345 --parent 67890 --title "New Title"
	// Verifies: page is moved and title updated, body preserved, no editor opened
	moveCalled := false
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Old Title",
				"version": {"number": 1},
				"body": {"storage": {"representation": "storage", "value": "<p>Original content</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "New Title",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/rest/api/content/12345/move/append/67890"):
			moveCalled = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = nil
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		title:   "New Title",
		parent:  "67890",
	}

	err := runEdit(opts)
	require.NoError(t, err)
	assert.True(t, moveCalled, "MovePage should have been called")
	assert.Equal(t, "New Title", receivedBody["title"])
}

func TestRunEdit_MoveOnly_BodyPreserved(t *testing.T) {
	// Test: move-only operation preserves original body exactly
	// Verifies: received body in PUT request matches original page body
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 1},
				"body": {"storage": {"representation": "storage", "value": "<p>Original content that must be preserved</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/rest/api/content/12345/move/append/67890"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.Stdin = nil
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		parent:  "67890",
	}

	err := runEdit(opts)
	require.NoError(t, err)

	// Verify body was preserved from original page
	bodyMap := receivedBody["body"].(map[string]interface{})
	storageMap := bodyMap["storage"].(map[string]interface{})
	assert.Equal(t, "<p>Original content that must be preserved</p>", storageMap["value"])
}
