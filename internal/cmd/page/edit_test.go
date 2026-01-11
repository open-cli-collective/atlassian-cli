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

	"github.com/rianjs/confluence-cli/api"
)

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

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &editOptions{
		pageID:  "12345",
		file:    mdFile,
		noColor: true,
	}

	err = runEdit(opts, client)
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

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &editOptions{
		pageID:  "12345",
		title:   "New Title",
		noColor: true,
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

	err = runEdit(opts, client)
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

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &editOptions{
		pageID:  "99999",
		title:   "New Title",
		noColor: true,
	}

	err := runEdit(opts, client)
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

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &editOptions{
		pageID:  "12345",
		file:    mdFile,
		noColor: true,
	}

	err = runEdit(opts, client)
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

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &editOptions{
		pageID:  "12345",
		file:    mdFile,
		noColor: true,
	}

	err = runEdit(opts, client)
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

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &editOptions{
		pageID:  "12345",
		file:    htmlFile,
		noColor: true,
	}

	err = runEdit(opts, client)
	require.NoError(t, err)

	// Verify HTML was not converted
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

	client := api.NewClient(server.URL, "test@example.com", "token")
	useMd := false
	opts := &editOptions{
		pageID:   "12345",
		file:     mdFile,
		markdown: &useMd,
		noColor:  true,
	}

	err = runEdit(opts, client)
	require.NoError(t, err)

	// Verify content was not converted
	bodyMap := receivedBody["body"].(map[string]interface{})
	storageMap := bodyMap["storage"].(map[string]interface{})
	content := storageMap["value"].(string)
	assert.Equal(t, "<p>Raw XHTML in .md file</p>", content)
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

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &editOptions{
		pageID:  "12345",
		file:    mdFile,
		output:  "json",
		noColor: true,
	}

	err = runEdit(opts, client)
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

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &editOptions{
		pageID:  "12345",
		file:    "/nonexistent/file.md",
		noColor: true,
	}

	err := runEdit(opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}
