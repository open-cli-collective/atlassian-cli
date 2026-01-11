package attachment

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rianjs/confluence-cli/api"
)

func TestRunList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"results": [
				{"id": "att1", "title": "doc.pdf", "mediaType": "application/pdf", "fileSize": 1024},
				{"id": "att2", "title": "image.png", "mediaType": "image/png", "fileSize": 2048}
			]
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		pageID:  "12345",
		limit:   25,
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}

func TestRunList_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		pageID:  "12345",
		limit:   25,
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}

func TestRunList_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Page not found"}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		pageID:  "99999",
		limit:   25,
		noColor: true,
	}

	err := runList(opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list attachments")
}

func TestRunList_JSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"results": [
				{"id": "att1", "title": "doc.pdf", "mediaType": "application/pdf", "fileSize": 1024}
			]
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		pageID:  "12345",
		limit:   25,
		output:  "json",
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}

func TestRunList_InvalidOutputFormat(t *testing.T) {
	// Don't need a server - should fail before API call
	client := api.NewClient("http://unused", "test@example.com", "token")
	opts := &listOptions{
		pageID:  "12345",
		output:  "invalid",
		noColor: true,
	}

	err := runList(opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
		{1610612736, "1.5 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatFileSize(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}
