package attachment

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

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
	t.Parallel()
	// Create temp file to upload
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "upload.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0600)
	testutil.RequireNoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "POST", r.Method)
		testutil.Contains(t, r.URL.Path, "/child/attachment")

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

	err = runUpload(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunUpload_WithComment(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "upload.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0600)
	testutil.RequireNoError(t, err)

	var receivedComment string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(10 << 20) //nolint:gosec // test server
		testutil.RequireNoError(t, err)
		receivedComment = r.FormValue("comment") //nolint:gosec // test server

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

	err = runUpload(context.Background(), opts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "My upload comment", receivedComment)
}

func TestRunUpload_FileNotFound(t *testing.T) {
	t.Parallel()
	rootOpts := newUploadTestRootOptions()
	client := api.NewClient("http://unused", "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &uploadOptions{
		Options: rootOpts,
		pageID:  "12345",
		file:    "/nonexistent/file.txt",
	}

	err := runUpload(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "opening file")
}

func TestRunUpload_APIError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "upload.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0600)
	testutil.RequireNoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	err = runUpload(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "uploading attachment")
}

func TestRunUpload_JSONOutput(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "upload.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0600)
	testutil.RequireNoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	err = runUpload(context.Background(), opts)
	testutil.RequireNoError(t, err)
}
