package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestFlexibleID_UnmarshalJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected FlexibleID
		wantErr  bool
	}{
		{
			name:     "string ID",
			input:    `"12345"`,
			expected: FlexibleID("12345"),
		},
		{
			name:     "number ID",
			input:    `12345`,
			expected: FlexibleID("12345"),
		},
		{
			name:     "large number ID",
			input:    `9876543210`,
			expected: FlexibleID("9876543210"),
		},
		{
			name:    "invalid type",
			input:   `true`,
			wantErr: true,
		},
		{
			name:    "null",
			input:   `null`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var id FlexibleID
			err := json.Unmarshal([]byte(tt.input), &id)
			if tt.wantErr {
				testutil.Error(t, err)
			} else {
				testutil.RequireNoError(t, err)
				testutil.Equal(t, id, tt.expected)
				testutil.Equal(t, id.String(), string(tt.expected))
			}
		})
	}
}

func TestGetIssueAttachments(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/issue/PROJ-123")
		testutil.Equal(t, r.URL.Query().Get("fields"), "attachment")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"fields": {
				"attachment": [
					{
						"id": "10001",
						"filename": "test.txt",
						"size": 1024,
						"created": "2024-01-15T10:30:00.000+0000",
						"author": {"displayName": "Test User"},
						"mimeType": "text/plain",
						"content": "https://example.com/attachments/10001"
					}
				]
			}
		}`))
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	attachments, err := client.GetIssueAttachments(context.Background(), "PROJ-123")
	testutil.RequireNoError(t, err)
	testutil.Len(t, attachments, 1)

	att := attachments[0]
	testutil.Equal(t, att.ID.String(), "10001")
	testutil.Equal(t, att.Filename, "test.txt")
	testutil.Equal(t, att.Size, int64(1024))
	testutil.Equal(t, att.Author.DisplayName, "Test User")
}

func TestGetIssueAttachments_EmptyIssueKey(t *testing.T) {
	t.Parallel()
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	_, err := client.GetIssueAttachments(context.Background(), "")
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "issue key is required")
}

func TestGetAttachment(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/attachment/10001")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "10001",
			"filename": "document.pdf",
			"size": 2048,
			"mimeType": "application/pdf",
			"content": "https://example.com/attachments/10001"
		}`))
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	att, err := client.GetAttachment(context.Background(), "10001")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, att.ID.String(), "10001")
	testutil.Equal(t, att.Filename, "document.pdf")
	testutil.Equal(t, att.Size, int64(2048))
}

func TestGetAttachment_EmptyID(t *testing.T) {
	t.Parallel()
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	_, err := client.GetAttachment(context.Background(), "")
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "attachment ID is required")
}

func TestDeleteAttachment(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodDelete)
		testutil.Equal(t, r.URL.Path, "/rest/api/3/attachment/10001")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	err = client.DeleteAttachment(context.Background(), "10001")
	testutil.NoError(t, err)
}

func TestDeleteAttachment_EmptyID(t *testing.T) {
	t.Parallel()
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	err := client.DeleteAttachment(context.Background(), "")
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "attachment ID is required")
}

func TestDownloadAttachment(t *testing.T) {
	t.Parallel()
	content := []byte("Test file content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "downloaded.txt")

	att := &Attachment{
		Filename: "test.txt",
		Content:  server.URL + "/attachment/content",
	}

	err = client.DownloadAttachment(context.Background(), att, outPath)
	testutil.RequireNoError(t, err)

	downloaded, err := os.ReadFile(outPath) //nolint:gosec // test reading known temp file
	testutil.RequireNoError(t, err)
	testutil.Equal(t, downloaded, content)
}

func TestDownloadAttachment_ToDirectory(t *testing.T) {
	t.Parallel()
	content := []byte("Test file content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	tmpDir := t.TempDir()

	att := &Attachment{
		Filename: "original.txt",
		Content:  server.URL + "/attachment/content",
	}

	err = client.DownloadAttachment(context.Background(), att, tmpDir)
	testutil.RequireNoError(t, err)

	// Should use original filename
	downloaded, err := os.ReadFile(filepath.Join(tmpDir, "original.txt")) //nolint:gosec // test reading known temp file
	testutil.RequireNoError(t, err)
	testutil.Equal(t, downloaded, content)
}

func TestDownloadAttachment_NilAttachment(t *testing.T) {
	t.Parallel()
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	err := client.DownloadAttachment(context.Background(), nil, "/tmp/test.txt")
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "attachment is required")
}

func TestDownloadAttachment_NoContentURL(t *testing.T) {
	t.Parallel()
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	att := &Attachment{Filename: "test.txt"}
	err := client.DownloadAttachment(context.Background(), att, "/tmp/test.txt")
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "no content URL")
}

func TestFormatFileSize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			result := FormatFileSize(tt.bytes)
			testutil.Equal(t, result, tt.expected)
		})
	}
}
