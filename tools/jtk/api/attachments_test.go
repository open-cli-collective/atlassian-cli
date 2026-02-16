package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestFlexibleID_UnmarshalJSON(t *testing.T) {
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

	attachments, err := client.GetIssueAttachments("PROJ-123")
	testutil.RequireNoError(t, err)
	testutil.Len(t, attachments, 1)

	att := attachments[0]
	testutil.Equal(t, att.ID.String(), "10001")
	testutil.Equal(t, att.Filename, "test.txt")
	testutil.Equal(t, att.Size, int64(1024))
	testutil.Equal(t, att.Author.DisplayName, "Test User")
}

func TestGetIssueAttachments_EmptyIssueKey(t *testing.T) {
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	_, err := client.GetIssueAttachments("")
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "issue key is required")
}

func TestGetAttachment(t *testing.T) {
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

	att, err := client.GetAttachment("10001")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, att.ID.String(), "10001")
	testutil.Equal(t, att.Filename, "document.pdf")
	testutil.Equal(t, att.Size, int64(2048))
}

func TestGetAttachment_EmptyID(t *testing.T) {
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	_, err := client.GetAttachment("")
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "attachment ID is required")
}

func TestDeleteAttachment(t *testing.T) {
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

	err = client.DeleteAttachment("10001")
	testutil.NoError(t, err)
}

func TestDeleteAttachment_EmptyID(t *testing.T) {
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	err := client.DeleteAttachment("")
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "attachment ID is required")
}

func TestDownloadAttachment(t *testing.T) {
	content := []byte("Test file content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	err = client.DownloadAttachment(att, outPath)
	testutil.RequireNoError(t, err)

	downloaded, err := os.ReadFile(outPath)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, downloaded, content)
}

func TestDownloadAttachment_ToDirectory(t *testing.T) {
	content := []byte("Test file content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	err = client.DownloadAttachment(att, tmpDir)
	testutil.RequireNoError(t, err)

	// Should use original filename
	downloaded, err := os.ReadFile(filepath.Join(tmpDir, "original.txt"))
	testutil.RequireNoError(t, err)
	testutil.Equal(t, downloaded, content)
}

func TestDownloadAttachment_NilAttachment(t *testing.T) {
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	err := client.DownloadAttachment(nil, "/tmp/test.txt")
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "attachment is required")
}

func TestDownloadAttachment_NoContentURL(t *testing.T) {
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	att := &Attachment{Filename: "test.txt"}
	err := client.DownloadAttachment(att, "/tmp/test.txt")
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "no content URL")
}

func TestFormatFileSize(t *testing.T) {
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
			result := FormatFileSize(tt.bytes)
			testutil.Equal(t, result, tt.expected)
		})
	}
}
