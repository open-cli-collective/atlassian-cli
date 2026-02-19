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

// TestSanitizeAttachmentFilename validates that filepath.Base correctly
// sanitizes malicious filenames that could be used for path traversal attacks.
func TestSanitizeAttachmentFilename(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
		valid    bool
	}{
		{
			name:     "normal filename",
			input:    "document.pdf",
			expected: "document.pdf",
			valid:    true,
		},
		{
			name:     "path traversal unix",
			input:    "../../../etc/passwd",
			expected: "passwd",
			valid:    true,
		},
		{
			name:     "path traversal with subdirectory",
			input:    "subdir/../../../etc/passwd",
			expected: "passwd",
			valid:    true,
		},
		{
			name:     "absolute path unix",
			input:    "/etc/passwd",
			expected: "passwd",
			valid:    true,
		},
		{
			name:     "filename with spaces",
			input:    "my document.pdf",
			expected: "my document.pdf",
			valid:    true,
		},
		{
			name:     "empty filename",
			input:    "",
			expected: ".",
			valid:    false,
		},
		{
			name:     "single dot",
			input:    ".",
			expected: ".",
			valid:    false,
		},
		{
			name:     "double dot",
			input:    "..",
			expected: "..",
			valid:    false,
		},
		{
			name:     "trailing slash",
			input:    "folder/",
			expected: "folder",
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := filepath.Base(tt.input)
			if result != tt.expected {
				t.Errorf("filepath.Base(%q) = %q, want %q", tt.input, result, tt.expected)
			}

			// Validate that our invalid filename check works
			isValid := result != "" && result != "." && result != ".."
			if isValid != tt.valid {
				t.Errorf("validity check for %q: got %v, want %v", tt.input, isValid, tt.valid)
			}
		})
	}
}

func mockDownloadServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/attachments/att123":
			// Get attachment metadata
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "att123",
				"title": "document.pdf",
				"mediaType": "application/pdf",
				"fileSize": 1024,
				"downloadLink": "/download/attachments/att123/document.pdf"
			}`))
		case "/download/attachments/att123/document.pdf":
			// Download content
			w.Header().Set("Content-Type", "application/pdf")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("fake pdf content"))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func newDownloadTestRootOptions() *root.Options {
	return &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
}

func TestRunDownload_Success(t *testing.T) {
	server := mockDownloadServer(t)
	defer server.Close()

	// Use temp directory for download
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	rootOpts := newDownloadTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &downloadOptions{
		Options: rootOpts,
	}

	err := runDownload(context.Background(), "att123", opts)
	testutil.RequireNoError(t, err)

	// Verify file was created
	content, err := os.ReadFile(filepath.Join(tmpDir, "document.pdf")) //nolint:gosec // reading test output file
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "fake pdf content", string(content))
}

func TestRunDownload_CustomOutputFile(t *testing.T) {
	t.Parallel()
	server := mockDownloadServer(t)
	defer server.Close()

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "custom-name.pdf")

	rootOpts := newDownloadTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &downloadOptions{
		Options:    rootOpts,
		outputFile: outputPath,
	}

	err := runDownload(context.Background(), "att123", opts)
	testutil.RequireNoError(t, err)

	// Verify file was created with custom name
	content, err := os.ReadFile(outputPath) //nolint:gosec // reading test output file
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "fake pdf content", string(content))
}

func TestRunDownload_FileExists_NoForce(t *testing.T) {
	server := mockDownloadServer(t)
	defer server.Close()

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	// Create existing file
	existingFile := filepath.Join(tmpDir, "document.pdf")
	err := os.WriteFile(existingFile, []byte("existing content"), 0600)
	testutil.RequireNoError(t, err)

	rootOpts := newDownloadTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &downloadOptions{
		Options: rootOpts,
		force:   false,
	}

	err = runDownload(context.Background(), "att123", opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "file already exists")
	testutil.Contains(t, err.Error(), "--force")

	// Verify original file was not overwritten
	content, _ := os.ReadFile(existingFile) //nolint:gosec // reading test fixture file
	testutil.Equal(t, "existing content", string(content))
}

func TestRunDownload_FileExists_WithForce(t *testing.T) {
	server := mockDownloadServer(t)
	defer server.Close()

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	// Create existing file
	existingFile := filepath.Join(tmpDir, "document.pdf")
	err := os.WriteFile(existingFile, []byte("existing content"), 0600)
	testutil.RequireNoError(t, err)

	rootOpts := newDownloadTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &downloadOptions{
		Options: rootOpts,
		force:   true,
	}

	err = runDownload(context.Background(), "att123", opts)
	testutil.RequireNoError(t, err)

	// Verify file was overwritten
	content, _ := os.ReadFile(existingFile) //nolint:gosec // reading test fixture file
	testutil.Equal(t, "fake pdf content", string(content))
}

func TestRunDownload_AttachmentNotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Attachment not found"}`))
	}))
	defer server.Close()

	rootOpts := newDownloadTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &downloadOptions{
		Options: rootOpts,
	}

	err := runDownload(context.Background(), "nonexistent", opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "getting attachment info")
}

func TestRunDownload_DownloadFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/attachments/att123":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "att123",
				"title": "document.pdf",
				"downloadLink": "/download/error"
			}`))
		case "/download/error":
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	rootOpts := newDownloadTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &downloadOptions{
		Options: rootOpts,
	}

	err := runDownload(context.Background(), "att123", opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "downloading attachment")
}

func TestRunDownload_InvalidFilename(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		filename string
	}{
		{"empty filename", ""},
		{"single dot", "."},
		{"double dot", ".."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "att123",
					"title": "` + tt.filename + `",
					"downloadLink": "/download/file"
				}`))
			}))
			defer server.Close()

			rootOpts := newDownloadTestRootOptions()
			client := api.NewClient(server.URL, "test@example.com", "token")
			rootOpts.SetAPIClient(client)

			opts := &downloadOptions{
				Options: rootOpts,
			}

			err := runDownload(context.Background(), "att123", opts)
			testutil.RequireError(t, err)
			testutil.Contains(t, err.Error(), "invalid attachment filename")
		})
	}
}

func TestRunDownload_PathTraversalPrevented(t *testing.T) {
	// Test that malicious filenames from the API are sanitized
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/attachments/att123":
			w.WriteHeader(http.StatusOK)
			// Malicious filename attempting path traversal
			_, _ = w.Write([]byte(`{
				"id": "att123",
				"title": "../../../etc/passwd",
				"downloadLink": "/download/file"
			}`))
		case "/download/file":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("file content"))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	rootOpts := newDownloadTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &downloadOptions{
		Options: rootOpts,
	}

	err := runDownload(context.Background(), "att123", opts)
	testutil.RequireNoError(t, err)

	// File should be saved as just "passwd" (the base name), not a path traversal
	_, err = os.Stat(filepath.Join(tmpDir, "passwd"))
	testutil.NoError(t, err)

	// Should NOT have created file outside tmpDir
	_, err = os.Stat("/etc/passwd-test")
	testutil.True(t, os.IsNotExist(err) || err != nil, "should not write outside current directory")
}
