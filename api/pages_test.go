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

func TestClient_CopyPage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/content/12345/copy", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req map[string]interface{}
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)

		assert.Equal(t, "New Title", req["pageTitle"])
		assert.Equal(t, true, req["copyAttachments"])
		assert.Equal(t, true, req["copyPermissions"])
		assert.Equal(t, true, req["copyProperties"])
		assert.Equal(t, true, req["copyLabels"])
		assert.Equal(t, true, req["copyCustomContents"])

		dest := req["destination"].(map[string]interface{})
		assert.Equal(t, "space", dest["type"])
		assert.Equal(t, "TEST", dest["value"])

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "99999",
			"type": "page",
			"status": "current",
			"title": "New Title",
			"space": {"id": 123, "key": "TEST", "name": "Test Space"},
			"version": {"number": 1},
			"_links": {"webui": "/spaces/TEST/pages/99999"}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	opts := &CopyPageOptions{
		Title:              "New Title",
		DestinationSpace:   "TEST",
		CopyAttachments:    true,
		CopyPermissions:    true,
		CopyProperties:     true,
		CopyLabels:         true,
		CopyCustomContents: true,
	}

	page, err := client.CopyPage(context.Background(), "12345", opts)
	require.NoError(t, err)
	assert.Equal(t, "99999", page.ID)
	assert.Equal(t, "New Title", page.Title)
	assert.Equal(t, "TEST", page.SpaceID)
	assert.Equal(t, 1, page.Version.Number)
	assert.Equal(t, "/spaces/TEST/pages/99999", page.Links.WebUI)
}

func TestClient_CopyPage_MissingTitle(t *testing.T) {
	client := NewClient("http://unused", "user@example.com", "token")

	_, err := client.CopyPage(context.Background(), "12345", &CopyPageOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "title is required")
}

func TestClient_CopyPage_NilOptions(t *testing.T) {
	client := NewClient("http://unused", "user@example.com", "token")

	_, err := client.CopyPage(context.Background(), "12345", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "title is required")
}

func TestClient_CopyPage_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Page not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	opts := &CopyPageOptions{
		Title:            "New Title",
		DestinationSpace: "TEST",
	}

	_, err := client.CopyPage(context.Background(), "99999", opts)
	require.Error(t, err)
}

func TestClient_CopyPage_WithoutAttachments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req map[string]interface{}
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)

		assert.Equal(t, false, req["copyAttachments"])

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "99999",
			"title": "New Title",
			"space": {"key": "TEST"},
			"version": {"number": 1},
			"_links": {}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	opts := &CopyPageOptions{
		Title:            "New Title",
		DestinationSpace: "TEST",
		CopyAttachments:  false,
	}

	_, err := client.CopyPage(context.Background(), "12345", opts)
	require.NoError(t, err)
}

func TestClient_CopyPage_ToDifferentSpace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req map[string]interface{}
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)

		dest := req["destination"].(map[string]interface{})
		assert.Equal(t, "space", dest["type"])
		assert.Equal(t, "OTHERSPACE", dest["value"])

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "99999",
			"title": "New Title",
			"space": {"key": "OTHERSPACE"},
			"version": {"number": 1},
			"_links": {}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	opts := &CopyPageOptions{
		Title:            "New Title",
		DestinationSpace: "OTHERSPACE",
	}

	page, err := client.CopyPage(context.Background(), "12345", opts)
	require.NoError(t, err)
	assert.Equal(t, "OTHERSPACE", page.SpaceID)
}

func TestClient_CopyPage_WithoutLabels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req map[string]interface{}
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)

		assert.Equal(t, false, req["copyLabels"])
		assert.Equal(t, true, req["copyAttachments"]) // others should still be true

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "99999",
			"title": "New Title",
			"space": {"key": "TEST"},
			"version": {"number": 1},
			"_links": {}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	opts := &CopyPageOptions{
		Title:            "New Title",
		DestinationSpace: "TEST",
		CopyAttachments:  true, // explicitly set to true
		CopyLabels:       false,
	}

	_, err := client.CopyPage(context.Background(), "12345", opts)
	require.NoError(t, err)
}

func TestClient_UpdatePage_VersionConflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{
			"message": "Version conflict: expected version 5 but page is at version 6",
			"errors": [{"title": "Version conflict"}]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	req := &UpdatePageRequest{
		ID:      "98765",
		Status:  "current",
		Title:   "Updated Title",
		Version: &Version{Number: 5}, // Stale version
	}

	_, err := client.UpdatePage(context.Background(), "98765", req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflict")
}

func TestClient_GetPage_MissingBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "98765",
			"title": "Page Without Body",
			"spaceId": "123456",
			"version": {"number": 1}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	page, err := client.GetPage(context.Background(), "98765", nil)

	require.NoError(t, err)
	assert.Equal(t, "98765", page.ID)
	assert.Equal(t, "Page Without Body", page.Title)
	assert.Nil(t, page.Body)
}

func TestClient_GetPage_EmptyBodyStorage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "98765",
			"title": "Page With Empty Body",
			"spaceId": "123456",
			"version": {"number": 1},
			"body": {
				"storage": null
			}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	page, err := client.GetPage(context.Background(), "98765", nil)

	require.NoError(t, err)
	assert.NotNil(t, page.Body)
	assert.Nil(t, page.Body.Storage)
}

func TestClient_ListPages_WithCursor(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		if callCount == 1 {
			// First call - return results with cursor
			assert.Empty(t, r.URL.Query().Get("cursor"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"results": [{"id": "1", "title": "Page 1"}],
				"_links": {"next": "/api/v2/spaces/123/pages?cursor=abc123"}
			}`))
		} else {
			// Second call - verify cursor is passed
			assert.Equal(t, "abc123", r.URL.Query().Get("cursor"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"results": [{"id": "2", "title": "Page 2"}],
				"_links": {}
			}`))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")

	// First request
	result1, err := client.ListPages(context.Background(), "123", nil)
	require.NoError(t, err)
	assert.True(t, result1.HasMore())

	// Second request with cursor
	opts := &ListPagesOptions{Cursor: "abc123"}
	result2, err := client.ListPages(context.Background(), "123", opts)
	require.NoError(t, err)
	assert.False(t, result2.HasMore())

	assert.Equal(t, 2, callCount)
}

func TestClient_ListPages_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	result, err := client.ListPages(context.Background(), "123", nil)

	require.NoError(t, err)
	assert.Empty(t, result.Results)
	assert.False(t, result.HasMore())
}

func TestClient_ListPages_NullVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [{
				"id": "98765",
				"title": "Page With Null Version",
				"spaceId": "123456",
				"version": null
			}]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	result, err := client.ListPages(context.Background(), "123", nil)

	require.NoError(t, err)
	require.Len(t, result.Results, 1)
	assert.Nil(t, result.Results[0].Version)
}
