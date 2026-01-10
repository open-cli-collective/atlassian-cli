package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ListPages(t *testing.T) {
	testData := loadTestData(t, "pages.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/spaces/123456/pages", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "25", r.URL.Query().Get("limit"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	result, err := client.ListPages(context.Background(), "123456", nil)

	require.NoError(t, err)
	assert.Len(t, result.Results, 2)
	assert.True(t, result.HasMore())

	// Check first page
	page := result.Results[0]
	assert.Equal(t, "98765", page.ID)
	assert.Equal(t, "Getting Started Guide", page.Title)
	assert.Equal(t, "123456", page.SpaceID)
}

func TestClient_ListPages_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "50", r.URL.Query().Get("limit"))
		assert.Equal(t, "current", r.URL.Query().Get("status"))
		assert.Equal(t, "title", r.URL.Query().Get("sort"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	opts := &ListPagesOptions{
		Limit:  50,
		Status: "current",
		Sort:   "title",
	}
	_, err := client.ListPages(context.Background(), "123456", opts)
	require.NoError(t, err)
}

func TestClient_GetPage(t *testing.T) {
	testData := loadTestData(t, "page.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/pages/98765", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	page, err := client.GetPage(context.Background(), "98765", nil)

	require.NoError(t, err)
	assert.Equal(t, "98765", page.ID)
	assert.Equal(t, "Getting Started Guide", page.Title)
	assert.Equal(t, "123456", page.SpaceID)
	assert.Equal(t, 5, page.Version.Number)
	assert.NotNil(t, page.Body)
	assert.NotNil(t, page.Body.Storage)
	assert.Contains(t, page.Body.Storage.Value, "<h1>Getting Started</h1>")
}

func TestClient_GetPage_WithBodyFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "storage", r.URL.Query().Get("body-format"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "98765", "title": "Test"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	opts := &GetPageOptions{BodyFormat: "storage"}
	_, err := client.GetPage(context.Background(), "98765", opts)
	require.NoError(t, err)
}

func TestClient_CreatePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/pages", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req CreatePageRequest
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)

		assert.Equal(t, "123456", req.SpaceID)
		assert.Equal(t, "New Page", req.Title)
		assert.NotNil(t, req.Body)
		assert.NotNil(t, req.Body.Storage)
		assert.Equal(t, "<p>Content</p>", req.Body.Storage.Value)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "99999",
			"title": "New Page",
			"spaceId": "123456",
			"version": {"number": 1}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	req := &CreatePageRequest{
		SpaceID: "123456",
		Title:   "New Page",
		Body: &Body{
			Storage: &BodyRepresentation{
				Representation: "storage",
				Value:          "<p>Content</p>",
			},
		},
	}
	page, err := client.CreatePage(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "99999", page.ID)
	assert.Equal(t, "New Page", page.Title)
}

func TestClient_UpdatePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/pages/98765", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req UpdatePageRequest
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)

		assert.Equal(t, "98765", req.ID)
		assert.Equal(t, "Updated Title", req.Title)
		assert.Equal(t, 6, req.Version.Number)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "98765",
			"title": "Updated Title",
			"version": {"number": 6}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	req := &UpdatePageRequest{
		ID:     "98765",
		Status: "current",
		Title:  "Updated Title",
		Body: &Body{
			Storage: &BodyRepresentation{
				Representation: "storage",
				Value:          "<p>Updated content</p>",
			},
		},
		Version: &Version{Number: 6},
	}
	page, err := client.UpdatePage(context.Background(), "98765", req)

	require.NoError(t, err)
	assert.Equal(t, "Updated Title", page.Title)
	assert.Equal(t, 6, page.Version.Number)
}

func TestClient_DeletePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/pages/98765", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	err := client.DeletePage(context.Background(), "98765")

	require.NoError(t, err)
}
