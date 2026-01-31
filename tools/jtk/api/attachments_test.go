package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, id)
				assert.Equal(t, string(tt.expected), id.String())
			}
		})
	}
}

func TestGetIssueAttachments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue/PROJ-123", r.URL.Path)
		assert.Equal(t, "attachment", r.URL.Query().Get("fields"))
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
	require.NoError(t, err)

	attachments, err := client.GetIssueAttachments("PROJ-123")
	require.NoError(t, err)
	require.Len(t, attachments, 1)

	att := attachments[0]
	assert.Equal(t, "10001", att.ID.String())
	assert.Equal(t, "test.txt", att.Filename)
	assert.Equal(t, int64(1024), att.Size)
	assert.Equal(t, "Test User", att.Author.DisplayName)
}

func TestGetIssueAttachments_EmptyIssueKey(t *testing.T) {
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	_, err := client.GetIssueAttachments("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "issue key is required")
}

func TestGetAttachment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/attachment/10001", r.URL.Path)
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
	require.NoError(t, err)

	att, err := client.GetAttachment("10001")
	require.NoError(t, err)
	assert.Equal(t, "10001", att.ID.String())
	assert.Equal(t, "document.pdf", att.Filename)
	assert.Equal(t, int64(2048), att.Size)
}

func TestGetAttachment_EmptyID(t *testing.T) {
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	_, err := client.GetAttachment("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "attachment ID is required")
}

func TestDeleteAttachment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/rest/api/3/attachment/10001", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	require.NoError(t, err)

	err = client.DeleteAttachment("10001")
	assert.NoError(t, err)
}

func TestDeleteAttachment_EmptyID(t *testing.T) {
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	err := client.DeleteAttachment("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "attachment ID is required")
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
	require.NoError(t, err)

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "downloaded.txt")

	att := &Attachment{
		Filename: "test.txt",
		Content:  server.URL + "/attachment/content",
	}

	err = client.DownloadAttachment(att, outPath)
	require.NoError(t, err)

	downloaded, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.Equal(t, content, downloaded)
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
	require.NoError(t, err)

	tmpDir := t.TempDir()

	att := &Attachment{
		Filename: "original.txt",
		Content:  server.URL + "/attachment/content",
	}

	err = client.DownloadAttachment(att, tmpDir)
	require.NoError(t, err)

	// Should use original filename
	downloaded, err := os.ReadFile(filepath.Join(tmpDir, "original.txt"))
	require.NoError(t, err)
	assert.Equal(t, content, downloaded)
}

func TestDownloadAttachment_NilAttachment(t *testing.T) {
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	err := client.DownloadAttachment(nil, "/tmp/test.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "attachment is required")
}

func TestDownloadAttachment_NoContentURL(t *testing.T) {
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	att := &Attachment{Filename: "test.txt"}
	err := client.DownloadAttachment(att, "/tmp/test.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no content URL")
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
			assert.Equal(t, tt.expected, result)
		})
	}
}
