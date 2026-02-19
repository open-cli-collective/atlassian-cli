package attachments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// --- list tests ---

func TestNewListCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newListCmd(opts)

	testutil.Equal(t, cmd.Use, "list <issue-key>")
	testutil.Equal(t, cmd.Aliases, []string{"ls"})
	testutil.NotEmpty(t, cmd.Short)
}

func TestRunList_Table(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := struct {
			Fields struct {
				Attachment []api.Attachment `json:"attachment"`
			} `json:"fields"`
		}{}
		response.Fields.Attachment = []api.Attachment{
			{
				ID:       "10001",
				Filename: "screenshot.png",
				Size:     204800,
				MimeType: "image/png",
				Created:  "2024-06-15T10:30:00.000Z",
				Author:   api.User{DisplayName: "Alice"},
				Content:  "https://example.com/download/10001",
			},
			{
				ID:       "10002",
				Filename: "report.pdf",
				Size:     1048576,
				MimeType: "application/pdf",
				Created:  "2024-06-16T14:00:00.000Z",
				Author:   api.User{DisplayName: "Bob"},
				Content:  "https://example.com/download/10002",
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "TEST-1")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "screenshot.png")
	testutil.Contains(t, output, "report.pdf")
	testutil.Contains(t, output, "200.0 KB")
	testutil.Contains(t, output, "1.0 MB")
	testutil.Contains(t, output, "2024-06-15")
	testutil.Contains(t, output, "2024-06-16")
	testutil.Contains(t, output, "Alice")
	testutil.Contains(t, output, "Bob")
}

func TestRunList_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := struct {
			Fields struct {
				Attachment []api.Attachment `json:"attachment"`
			} `json:"fields"`
		}{}
		response.Fields.Attachment = []api.Attachment{
			{
				ID:       "10001",
				Filename: "screenshot.png",
				Size:     204800,
				MimeType: "image/png",
				Created:  "2024-06-15T10:30:00.000Z",
				Author:   api.User{DisplayName: "Alice"},
				Content:  "https://example.com/download/10001",
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "TEST-1")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, `"filename"`)
	testutil.Contains(t, output, "screenshot.png")
	testutil.Contains(t, output, `"mimeType"`)
	testutil.Contains(t, output, `"size"`)
}

func TestRunList_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := struct {
			Fields struct {
				Attachment []api.Attachment `json:"attachment"`
			} `json:"fields"`
		}{}
		response.Fields.Attachment = []api.Attachment{}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "TEST-1")
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), "No attachments found")
}

// --- add tests ---

func TestNewAddCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newAddCmd(opts)

	testutil.Equal(t, cmd.Use, "add <issue-key>")
	testutil.NotEmpty(t, cmd.Short)

	fileFlag := cmd.Flags().Lookup("file")
	testutil.NotNil(t, fileFlag)
	testutil.Equal(t, fileFlag.Shorthand, "f")
}

func TestRunAdd_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodPost)
		testutil.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")
		testutil.Equal(t, r.Header.Get("X-Atlassian-Token"), "no-check")

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]api.Attachment{
			{
				ID:       "10001",
				Filename: "testfile.txt",
				Size:     42,
				MimeType: "text/plain",
				Created:  "2024-06-15T10:30:00.000Z",
				Author:   api.User{DisplayName: "Alice"},
			},
		})
	}))
	defer server.Close()

	// Create a temporary file to upload
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "testfile.txt")
	err := os.WriteFile(tmpFile, []byte("hello world, this is test content"), 0600)
	testutil.RequireNoError(t, err)

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runAdd(context.Background(), opts, "TEST-1", []string{tmpFile})
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Uploaded testfile.txt")
	testutil.Contains(t, output, "10001")
}

// --- get/download tests ---

func TestNewGetCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newGetCmd(opts)

	testutil.Equal(t, cmd.Use, "get <attachment-id>")
	testutil.Equal(t, cmd.Aliases, []string{"download"})
	testutil.NotEmpty(t, cmd.Short)

	outputFlag := cmd.Flags().Lookup("output")
	testutil.NotNil(t, outputFlag)
	testutil.Equal(t, outputFlag.Shorthand, "o")
	testutil.Equal(t, outputFlag.DefValue, ".")
}

func TestRunGet_Success(t *testing.T) {
	fileContent := "This is the downloaded file content."

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/attachment/10001":
			// GetAttachment metadata request - Content URL must point back to this server
			resp := map[string]any{
				"id":       "10001",
				"filename": "downloaded.txt",
				"size":     int64(len(fileContent)),
				"mimeType": "text/plain",
				"created":  "2024-06-15T10:30:00.000Z",
				"author":   api.User{DisplayName: "Alice"},
				"content":  fmt.Sprintf("http://%s/content/10001", r.Host),
			}
			_ = json.NewEncoder(w).Encode(resp)
		case "/content/10001":
			// DownloadAttachment content request
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte(fileContent))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "downloaded.txt")

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "10001", outputPath)
	testutil.RequireNoError(t, err)

	// Verify the file was downloaded
	content, err := os.ReadFile(outputPath) //nolint:gosec // test code reads from temp dir
	testutil.RequireNoError(t, err)
	testutil.Equal(t, string(content), fileContent)

	// Verify success message
	output := stdout.String()
	testutil.Contains(t, output, "Downloaded")
	testutil.Contains(t, output, "downloaded.txt")
}

// --- delete tests ---

func TestNewDeleteCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newDeleteCmd(opts)

	testutil.Equal(t, cmd.Use, "delete <attachment-id>")
	testutil.Equal(t, cmd.Aliases, []string{"rm"})
	testutil.NotEmpty(t, cmd.Short)
}

func TestRunDelete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodDelete)
		testutil.Equal(t, r.URL.Path, "/rest/api/3/attachment/10001")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, "10001")
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), "Deleted attachment 10001")
}
