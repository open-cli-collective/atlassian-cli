package page

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

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
	t.Parallel()
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# Updated Content\n\nNew text here."), 0600)
	testutil.RequireNoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/pages/12345"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 5},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/pages/12345"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
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

	err = runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunEdit_TitleOnly(t *testing.T) {
	t.Parallel()
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/pages/12345"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Old Title",
				"version": {"number": 3},
				"body": {"storage": {"representation": "storage", "value": "<p>Keep this</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/pages/12345"):
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
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
	err := os.WriteFile(mdFile, []byte("<p>Keep this</p>"), 0600)
	testutil.RequireNoError(t, err)

	useMd := false
	opts.file = mdFile
	opts.markdown = &useMd

	err = runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)

	// Verify title was changed
	testutil.Equal(t, "New Title", receivedBody["title"])
}

func TestRunEdit_PageNotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Page not found"}`))
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

	err := runEdit(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "getting page")
}

func TestRunEdit_UpdateFailed(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# New Content"), 0600)
	testutil.RequireNoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message": "Permission denied"}`))
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

	err = runEdit(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "updating page")
}

func TestRunEdit_VersionIncrement(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# Updated"), 0600)
	testutil.RequireNoError(t, err)

	var receivedVersion int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 7},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			body, _ := io.ReadAll(r.Body)
			var req map[string]any
			_ = json.Unmarshal(body, &req)
			if v, ok := req["version"].(map[string]any); ok {
				receivedVersion = int(v["number"].(float64))
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
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

	err = runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)

	// Verify version was incremented from 7 to 8
	testutil.Equal(t, 8, receivedVersion)
}

func TestRunEdit_HTMLFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	htmlFile := filepath.Join(tmpDir, "content.html")
	err := os.WriteFile(htmlFile, []byte("<p>Direct HTML</p>"), 0600)
	testutil.RequireNoError(t, err)

	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
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

	err = runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)

	// Verify HTML was not converted (storage format in legacy mode)
	bodyMap := receivedBody["body"].(map[string]any)
	storageMap := bodyMap["storage"].(map[string]any)
	content := storageMap["value"].(string)
	testutil.Equal(t, "<p>Direct HTML</p>", content)
}

func TestRunEdit_NoMarkdownFlag(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("<p>Raw XHTML in .md file</p>"), 0600)
	testutil.RequireNoError(t, err)

	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
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

	err = runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)

	// Verify content was not converted (storage format in legacy mode)
	bodyMap := receivedBody["body"].(map[string]any)
	storageMap := bodyMap["storage"].(map[string]any)
	content := storageMap["value"].(string)
	testutil.Equal(t, "<p>Raw XHTML in .md file</p>", content)
}

func TestRunEdit_MarkdownToADF(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# Updated\n\nNew **bold** text."), 0600)
	testutil.RequireNoError(t, err)

	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
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

	err = runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)

	// Verify ADF format was used (default)
	bodyMap := receivedBody["body"].(map[string]any)
	adfMap := bodyMap["atlas_doc_format"].(map[string]any)
	content := adfMap["value"].(string)

	// Should be valid ADF JSON
	testutil.Contains(t, content, `"type":"doc"`)
	testutil.Contains(t, content, `"type":"heading"`)
	testutil.Contains(t, content, `"type":"strong"`)
}

func TestRunEdit_JSONOutput(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# Updated"), 0600)
	testutil.RequireNoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
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

	err = runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunEdit_FileReadError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
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

	err := runEdit(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "reading file")
}

func TestRunEdit_Stdin_ADF(t *testing.T) {
	t.Parallel()
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
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

	err := runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)

	// Verify ADF format was used
	bodyMap := receivedBody["body"].(map[string]any)
	adfMap := bodyMap["atlas_doc_format"].(map[string]any)
	content := adfMap["value"].(string)

	testutil.Contains(t, content, `"type":"doc"`)
	testutil.Contains(t, content, `"type":"heading"`)
	testutil.Contains(t, content, `"type":"strong"`)
}

func TestRunEdit_Stdin_Legacy(t *testing.T) {
	t.Parallel()
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
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

	err := runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)

	// Verify storage format was used
	bodyMap := receivedBody["body"].(map[string]any)
	storageMap := bodyMap["storage"].(map[string]any)
	content := storageMap["value"].(string)

	testutil.Contains(t, content, "<h1")
	testutil.Contains(t, content, "<strong>bold</strong>")
}

func TestRunEdit_TitleAndContent(t *testing.T) {
	t.Parallel()
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Old Title",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
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
	err := os.WriteFile(mdFile, []byte("# New Content\n\nUpdated text here."), 0600)
	testutil.RequireNoError(t, err)

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		title:   "New Title",
		file:    mdFile,
	}

	err = runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)

	// Verify both title and content were updated
	testutil.Equal(t, "New Title", receivedBody["title"])
	bodyMap := receivedBody["body"].(map[string]any)
	adfMap := bodyMap["atlas_doc_format"].(map[string]any)
	testutil.NotNil(t, adfMap["value"])
}

func TestRunEdit_ComplexMarkdown_ADF(t *testing.T) {
	t.Parallel()
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
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
	err := os.WriteFile(mdFile, []byte(complexMarkdown), 0600)
	testutil.RequireNoError(t, err)

	rootOpts := newEditTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		file:    mdFile,
	}

	err = runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)

	// Verify ADF contains complex elements
	bodyMap := receivedBody["body"].(map[string]any)
	adfMap := bodyMap["atlas_doc_format"].(map[string]any)
	content := adfMap["value"].(string)

	testutil.Contains(t, content, `"type":"table"`)
	testutil.Contains(t, content, `"type":"bulletList"`)
	testutil.Contains(t, content, `"type":"codeBlock"`)
	testutil.Contains(t, content, `"language":"go"`)
}

func TestRunEdit_MoveToParent(t *testing.T) {
	t.Parallel()
	moveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Content</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/rest/api/content/12345/move/append/67890"):
			moveCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
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

	err := runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)
	testutil.True(t, moveCalled, "MovePage should have been called")
}

func TestRunEdit_MoveAndRename(t *testing.T) {
	t.Parallel()
	var receivedTitle string
	moveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Old Title",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Content</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			body, _ := io.ReadAll(r.Body)
			var req map[string]any
			_ = json.Unmarshal(body, &req)
			receivedTitle = req["title"].(string)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "New Title",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/rest/api/content/12345/move/append/67890"):
			moveCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
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

	err := runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)
	testutil.True(t, moveCalled, "MovePage should have been called")
	testutil.Equal(t, "New Title", receivedTitle)
}

func TestRunEdit_MoveFailed(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Content</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/rest/api/content/12345/move"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message": "Target page not found"}`))
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

	err := runEdit(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "moving page to new parent")
}

func TestRunEdit_MoveWithContent(t *testing.T) {
	t.Parallel()
	moveCalled := false
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/rest/api/content/12345/move/append/67890"):
			moveCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
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

	err := runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)
	testutil.True(t, moveCalled, "MovePage should have been called")

	// Verify content was also updated
	bodyMap := receivedBody["body"].(map[string]any)
	adfMap := bodyMap["atlas_doc_format"].(map[string]any)
	testutil.NotNil(t, adfMap["value"])
}

func TestRunEdit_EmptyContentFromStdin(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
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

	err := runEdit(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "page content cannot be empty")
}

func TestRunEdit_WhitespaceOnlyFromStdin(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
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

	err := runEdit(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "page content cannot be empty")
}

func TestRunEdit_EmptyFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.md")
	err := os.WriteFile(emptyFile, []byte(""), 0600)
	testutil.RequireNoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
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

	err = runEdit(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "page content cannot be empty")
}

func TestRunEdit_WhitespaceOnlyFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	whitespaceFile := filepath.Join(tmpDir, "whitespace.md")
	err := os.WriteFile(whitespaceFile, []byte("   \n\t\n   "), 0600)
	testutil.RequireNoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
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

	err = runEdit(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "page content cannot be empty")
}

func TestRunEdit_TitleOnlyUpdate_NoContentValidation(t *testing.T) {
	t.Parallel()
	// When updating title only (with file providing content), validation should pass
	updateCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Old Title",
				"version": {"number": 1},
				"body": {"storage": {"representation": "storage", "value": "<p>Existing content</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			updateCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
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
	err := os.WriteFile(mdFile, []byte("# Valid Content"), 0600)
	testutil.RequireNoError(t, err)

	rootOpts.Stdin = nil
	opts := &editOptions{
		Options: rootOpts,
		pageID:  "12345",
		title:   "New Title",
		file:    mdFile,
	}

	err = runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)
	testutil.True(t, updateCalled, "Update should have been called")
}

func TestRunEdit_MoveOnly_NoEditorOpened(t *testing.T) {
	t.Parallel()
	// Test: cfl page edit 12345 --parent 67890
	// Verifies: page is moved without content change, no editor opened
	moveCalled := false
	updateCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 1},
				"body": {"storage": {"representation": "storage", "value": "<p>Original content</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			updateCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/rest/api/content/12345/move/append/67890"):
			moveCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
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

	err := runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)
	testutil.True(t, updateCalled, "UpdatePage should have been called")
	testutil.True(t, moveCalled, "MovePage should have been called")
}

func TestRunEdit_MoveWithTitleOnly_NoEditorOpened(t *testing.T) {
	t.Parallel()
	// Test: cfl page edit 12345 --parent 67890 --title "New Title"
	// Verifies: page is moved and title updated, body preserved, no editor opened
	moveCalled := false
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Old Title",
				"version": {"number": 1},
				"body": {"storage": {"representation": "storage", "value": "<p>Original content</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "New Title",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/rest/api/content/12345/move/append/67890"):
			moveCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
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

	err := runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)
	testutil.True(t, moveCalled, "MovePage should have been called")
	testutil.Equal(t, "New Title", receivedBody["title"])
}

func TestRunEdit_StorageFlag_Stdin(t *testing.T) {
	t.Parallel()
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
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
	rootOpts.Stdin = strings.NewReader(`<ac:structured-macro ac:name="toc"/><p>Content with <ac:link><ri:user ri:account-id="abc123"/></ac:link></p>`)
	useMd := false
	opts := &editOptions{
		Options:  rootOpts,
		pageID:   "12345",
		storage:  true,
		markdown: &useMd,
	}

	err := runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)

	// Verify storage format was used (not atlas_doc_format)
	bodyMap := receivedBody["body"].(map[string]any)
	storageMap := bodyMap["storage"].(map[string]any)
	content := storageMap["value"].(string)

	// Content should be passed through as-is, preserving Confluence-specific markup
	testutil.Contains(t, content, `ac:structured-macro`)
	testutil.Contains(t, content, `ri:user`)

	// Should NOT have atlas_doc_format
	testutil.Nil(t, bodyMap["atlas_doc_format"])
}

func TestRunEdit_StorageFlag_File(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	htmlFile := filepath.Join(tmpDir, "content.html")
	err := os.WriteFile(htmlFile, []byte("<p>Direct storage XHTML</p>"), 0600)
	testutil.RequireNoError(t, err)

	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "PUT":
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
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
	useMd := false
	opts := &editOptions{
		Options:  rootOpts,
		pageID:   "12345",
		file:     htmlFile,
		storage:  true,
		markdown: &useMd,
	}

	err = runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)

	// Verify storage format was used without --legacy
	bodyMap := receivedBody["body"].(map[string]any)
	storageMap := bodyMap["storage"].(map[string]any)
	content := storageMap["value"].(string)
	testutil.Equal(t, "<p>Direct storage XHTML</p>", content)

	// Should NOT have atlas_doc_format
	testutil.Nil(t, bodyMap["atlas_doc_format"])
}

func TestRunEdit_MoveOnly_BodyPreserved(t *testing.T) {
	t.Parallel()
	// Test: move-only operation preserves original body exactly
	// Verifies: received body in PUT request matches original page body
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 1},
				"body": {"storage": {"representation": "storage", "value": "<p>Original content that must be preserved</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/api/v2/pages/12345"):
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"version": {"number": 2},
				"_links": {"webui": "/pages/12345"}
			}`))
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/rest/api/content/12345/move/append/67890"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
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

	err := runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)

	// Verify body was preserved from original page
	bodyMap := receivedBody["body"].(map[string]any)
	storageMap := bodyMap["storage"].(map[string]any)
	testutil.Equal(t, "<p>Original content that must be preserved</p>", storageMap["value"])
}

func TestRunEdit_ADFPage_TitleOnly_PreservesBody(t *testing.T) {
	t.Parallel()
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/pages/12345"):
			switch r.URL.Query().Get("body-format") {
			case "storage":
				// Storage returns empty for this ADF page
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "12345",
					"title": "ADF Page",
					"version": {"number": 3},
					"body": {"storage": {"representation": "storage", "value": ""}},
					"_links": {"webui": "/pages/12345"}
				}`))
			case "atlas_doc_format":
				// ADF has content
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "12345",
					"title": "ADF Page",
					"version": {"number": 3},
					"body": {"atlas_doc_format": {"representation": "atlas_doc_format", "value": "{\"type\":\"doc\",\"version\":1,\"content\":[{\"type\":\"paragraph\",\"content\":[{\"type\":\"text\",\"text\":\"ADF body\"}]}]}"}},
					"_links": {"webui": "/pages/12345"}
				}`))
			}
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/pages/12345"):
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
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

	err := runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)

	// Verify body was preserved as ADF (not storage)
	bodyMap := receivedBody["body"].(map[string]any)
	adfMap := bodyMap["atlas_doc_format"].(map[string]any)
	testutil.Contains(t, adfMap["value"].(string), "ADF body")
}

func TestRunEdit_ADFPage_NewContent(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "content.md")
	err := os.WriteFile(mdFile, []byte("# Updated Content"), 0600)
	testutil.RequireNoError(t, err)

	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/pages/12345"):
			switch r.URL.Query().Get("body-format") {
			case "storage":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "12345",
					"title": "ADF Page",
					"version": {"number": 2},
					"body": {"storage": {"representation": "storage", "value": ""}},
					"_links": {"webui": "/pages/12345"}
				}`))
			case "atlas_doc_format":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "12345",
					"title": "ADF Page",
					"version": {"number": 2},
					"body": {"atlas_doc_format": {"representation": "atlas_doc_format", "value": "{\"type\":\"doc\",\"version\":1,\"content\":[]}"}},
					"_links": {"webui": "/pages/12345"}
				}`))
			}
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/pages/12345"):
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "ADF Page",
				"version": {"number": 3},
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

	err = runEdit(context.Background(), opts)
	testutil.RequireNoError(t, err)

	// New content should be submitted as ADF (default path)
	bodyMap := receivedBody["body"].(map[string]any)
	adfMap := bodyMap["atlas_doc_format"].(map[string]any)
	testutil.Contains(t, adfMap["value"].(string), "Updated Content")
}
