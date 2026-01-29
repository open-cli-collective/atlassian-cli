package attachment

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

func newUploadTestRootOptions() *root.Options {
	return &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
}

func TestRunUpload_Success(t *testing.T) {
	// Create temp file to upload
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "upload.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/child/attachment")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [{
				"id": "att123",
				"title": "upload.txt",
				"mediaType": "text/plain",
				"fileSize": 12
			}]
		}`))
	}))
	defer server.Close()

	rootOpts := newUploadTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &uploadOptions{
		Options: rootOpts,
		pageID:  "12345",
		file:    testFile,
	}

	err = runUpload(opts)
	require.NoError(t, err)
}

func TestRunUpload_WithComment(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "upload.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	var receivedComment string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(10 << 20)
		require.NoError(t, err)
		receivedComment = r.FormValue("comment")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [{
				"id": "att123",
				"title": "upload.txt",
				"mediaType": "text/plain",
				"fileSize": 12
			}]
		}`))
	}))
	defer server.Close()

	rootOpts := newUploadTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &uploadOptions{
		Options: rootOpts,
		pageID:  "12345",
		file:    testFile,
		comment: "My upload comment",
	}

	err = runUpload(opts)
	require.NoError(t, err)
	assert.Equal(t, "My upload comment", receivedComment)
}

func TestRunUpload_FileNotFound(t *testing.T) {
	rootOpts := newUploadTestRootOptions()
	client := api.NewClient("http://unused", "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &uploadOptions{
		Options: rootOpts,
		pageID:  "12345",
		file:    "/nonexistent/file.txt",
	}

	err := runUpload(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open file")
}

func TestRunUpload_APIError(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "upload.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message": "Permission denied"}`))
	}))
	defer server.Close()

	rootOpts := newUploadTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &uploadOptions{
		Options: rootOpts,
		pageID:  "12345",
		file:    testFile,
	}

	err = runUpload(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to upload attachment")
}

func TestRunUpload_JSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "upload.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [{
				"id": "att123",
				"title": "upload.txt",
				"mediaType": "text/plain",
				"fileSize": 12
			}]
		}`))
	}))
	defer server.Close()

	rootOpts := newUploadTestRootOptions()
	rootOpts.Output = "json"
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &uploadOptions{
		Options: rootOpts,
		pageID:  "12345",
		file:    testFile,
	}

	err = runUpload(opts)
	require.NoError(t, err)
}
