package api //nolint:revive // package name is intentional

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestClient_ListAttachments(t *testing.T) {
	t.Parallel()
	testData := loadTestData(t, "attachments.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/api/v2/pages/98765/attachments", r.URL.Path)
		testutil.Equal(t, "GET", r.Method)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	result, err := client.ListAttachments(context.Background(), "98765", nil)

	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Results, 2)

	// Check first attachment
	att := result.Results[0]
	testutil.Equal(t, "att111", att.ID)
	testutil.Equal(t, "screenshot.png", att.Title)
	testutil.Equal(t, "image/png", att.MediaType)
	testutil.Equal(t, int64(245678), att.FileSize)
}

func TestClient_ListAttachments_WithOptions(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "50", r.URL.Query().Get("limit"))
		testutil.Equal(t, "image/png", r.URL.Query().Get("mediaType"))

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
	testutil.RequireNoError(t, err)
}

func TestClient_GetAttachment(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/api/v2/attachments/att111", r.URL.Path)
		testutil.Equal(t, "GET", r.Method)

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

	testutil.RequireNoError(t, err)
	testutil.Equal(t, "att111", att.ID)
	testutil.Equal(t, "screenshot.png", att.Title)
	testutil.Equal(t, int64(245678), att.FileSize)
}

func TestClient_DownloadAttachment(t *testing.T) {
	t.Parallel()
	fileContent := []byte("fake image content")
	downloadCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/attachments/att111" {
			// GetAttachment returns metadata with downloadLink
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "att111",
				"title": "screenshot.png",
				"downloadLink": "/download/attachments/98765/screenshot.png"
			}`))
			return
		}

		if r.URL.Path == "/download/attachments/98765/screenshot.png" {
			downloadCalled = true
			w.Header().Set("Content-Type", "image/png")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fileContent)
			return
		}

		t.Errorf("unexpected request path: %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	reader, err := client.DownloadAttachment(context.Background(), "att111")
	testutil.RequireNoError(t, err)
	defer func() { _ = reader.Close() }()

	testutil.True(t, downloadCalled, "should have called download link")

	// Read and verify content
	buf := make([]byte, 100)
	n, _ := reader.Read(buf)
	testutil.Equal(t, fileContent, buf[:n])
}

func TestClient_DownloadAttachment_NoDownloadLink(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "att123", "title": "test.txt"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	_, err := client.DownloadAttachment(context.Background(), "att123")
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "no download link")
}

func TestClient_DeleteAttachment(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/api/v2/attachments/att111", r.URL.Path)
		testutil.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	err := client.DeleteAttachment(context.Background(), "att111")
	testutil.RequireNoError(t, err)
}

func TestClient_DeleteAttachment_NotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Attachment not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	err := client.DeleteAttachment(context.Background(), "invalid")
	testutil.RequireError(t, err)
}
