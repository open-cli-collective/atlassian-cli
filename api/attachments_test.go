package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ListAttachments(t *testing.T) {
	testData := loadTestData(t, "attachments.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/pages/98765/attachments", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	result, err := client.ListAttachments(context.Background(), "98765", nil)

	require.NoError(t, err)
	assert.Len(t, result.Results, 2)

	// Check first attachment
	att := result.Results[0]
	assert.Equal(t, "att111", att.ID)
	assert.Equal(t, "screenshot.png", att.Title)
	assert.Equal(t, "image/png", att.MediaType)
	assert.Equal(t, int64(245678), att.FileSize)
}

func TestClient_ListAttachments_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "50", r.URL.Query().Get("limit"))
		assert.Equal(t, "image/png", r.URL.Query().Get("mediaType"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	opts := &ListAttachmentsOptions{
		Limit:     50,
		MediaType: "image/png",
	}
	_, err := client.ListAttachments(context.Background(), "98765", opts)
	require.NoError(t, err)
}

func TestClient_GetAttachment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/attachments/att111", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "att111",
			"title": "screenshot.png",
			"mediaType": "image/png",
			"fileSize": 245678,
			"downloadLink": "/download/attachments/98765/screenshot.png"
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	att, err := client.GetAttachment(context.Background(), "att111")

	require.NoError(t, err)
	assert.Equal(t, "att111", att.ID)
	assert.Equal(t, "screenshot.png", att.Title)
	assert.Equal(t, int64(245678), att.FileSize)
}

func TestClient_DownloadAttachment(t *testing.T) {
	fileContent := []byte("fake image content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/attachments/att111/download" {
			// Return redirect
			w.Header().Set("Location", "/download/attachments/98765/screenshot.png")
			w.WriteHeader(http.StatusFound)
			return
		}

		if r.URL.Path == "/download/attachments/98765/screenshot.png" {
			w.Header().Set("Content-Type", "image/png")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fileContent)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	reader, err := client.DownloadAttachment(context.Background(), "att111")
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	// Read and verify content
	buf := make([]byte, 100)
	n, _ := reader.Read(buf)
	assert.Equal(t, fileContent, buf[:n])
}
