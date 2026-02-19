package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestClient_ListPages(t *testing.T) {
	t.Parallel()
	testData := loadTestData(t, "pages.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/api/v2/spaces/123456/pages", r.URL.Path)
		testutil.Equal(t, "GET", r.Method)
		testutil.Equal(t, "25", r.URL.Query().Get("limit"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	result, err := client.ListPages(context.Background(), "123456", nil)

	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Results, 2)
	testutil.True(t, result.HasMore())

	// Check first page
	page := result.Results[0]
	testutil.Equal(t, "98765", page.ID)
	testutil.Equal(t, "Getting Started Guide", page.Title)
	testutil.Equal(t, "123456", page.SpaceID)
}

func TestClient_ListPages_WithOptions(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "50", r.URL.Query().Get("limit"))
		testutil.Equal(t, "current", r.URL.Query().Get("status"))
		testutil.Equal(t, "title", r.URL.Query().Get("sort"))

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
	testutil.RequireNoError(t, err)
}

func TestClient_GetPage(t *testing.T) {
	t.Parallel()
	testData := loadTestData(t, "page.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/api/v2/pages/98765", r.URL.Path)
		testutil.Equal(t, "GET", r.Method)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	page, err := client.GetPage(context.Background(), "98765", nil)

	testutil.RequireNoError(t, err)
	testutil.Equal(t, "98765", page.ID)
	testutil.Equal(t, "Getting Started Guide", page.Title)
	testutil.Equal(t, "123456", page.SpaceID)
	testutil.Equal(t, 5, page.Version.Number)
	testutil.NotNil(t, page.Body)
	testutil.NotNil(t, page.Body.Storage)
	testutil.Contains(t, page.Body.Storage.Value, "<h1>Getting Started</h1>")
}

func TestClient_GetPage_WithBodyFormat(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "storage", r.URL.Query().Get("body-format"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "98765", "title": "Test"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	opts := &GetPageOptions{BodyFormat: "storage"}
	_, err := client.GetPage(context.Background(), "98765", opts)
	testutil.RequireNoError(t, err)
}

func TestClient_CreatePage(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/api/v2/pages", r.URL.Path)
		testutil.Equal(t, "POST", r.Method)

		body, err := io.ReadAll(r.Body)
		testutil.RequireNoError(t, err)

		var req CreatePageRequest
		err = json.Unmarshal(body, &req)
		testutil.RequireNoError(t, err)

		testutil.Equal(t, "123456", req.SpaceID)
		testutil.Equal(t, "New Page", req.Title)
		testutil.NotNil(t, req.Body)
		testutil.NotNil(t, req.Body.Storage)
		testutil.Equal(t, "<p>Content</p>", req.Body.Storage.Value)

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

	testutil.RequireNoError(t, err)
	testutil.Equal(t, "99999", page.ID)
	testutil.Equal(t, "New Page", page.Title)
}

func TestClient_UpdatePage(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/api/v2/pages/98765", r.URL.Path)
		testutil.Equal(t, "PUT", r.Method)

		body, err := io.ReadAll(r.Body)
		testutil.RequireNoError(t, err)

		var req UpdatePageRequest
		err = json.Unmarshal(body, &req)
		testutil.RequireNoError(t, err)

		testutil.Equal(t, "98765", req.ID)
		testutil.Equal(t, "Updated Title", req.Title)
		testutil.Equal(t, 6, req.Version.Number)

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

	testutil.RequireNoError(t, err)
	testutil.Equal(t, "Updated Title", page.Title)
	testutil.Equal(t, 6, page.Version.Number)
}

func TestClient_DeletePage(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/api/v2/pages/98765", r.URL.Path)
		testutil.Equal(t, "DELETE", r.Method)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	err := client.DeletePage(context.Background(), "98765")

	testutil.RequireNoError(t, err)
}

func TestClient_MovePage_Success(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/rest/api/content/12345/move/append/67890", r.URL.Path)
		testutil.Equal(t, "PUT", r.Method)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	err := client.MovePage(context.Background(), "12345", "67890")

	testutil.RequireNoError(t, err)
}

func TestClient_MovePage_NotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Page not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	err := client.MovePage(context.Background(), "99999", "67890")

	testutil.RequireError(t, err)
}

func TestClient_MovePage_PermissionDenied(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message": "You do not have permission to move this page"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	err := client.MovePage(context.Background(), "12345", "67890")

	testutil.RequireError(t, err)
}

func TestClient_CopyPage_Success(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/rest/api/content/12345/copy", r.URL.Path)
		testutil.Equal(t, "POST", r.Method)

		body, err := io.ReadAll(r.Body)
		testutil.RequireNoError(t, err)

		var req map[string]any
		err = json.Unmarshal(body, &req)
		testutil.RequireNoError(t, err)

		testutil.Equal(t, "New Title", req["pageTitle"])
		testutil.Equal(t, true, req["copyAttachments"])
		testutil.Equal(t, true, req["copyPermissions"])
		testutil.Equal(t, true, req["copyProperties"])
		testutil.Equal(t, true, req["copyLabels"])
		testutil.Equal(t, true, req["copyCustomContents"])

		dest := req["destination"].(map[string]any)
		testutil.Equal(t, "space", dest["type"])
		testutil.Equal(t, "TEST", dest["value"])

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
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "99999", page.ID)
	testutil.Equal(t, "New Title", page.Title)
	testutil.Equal(t, "TEST", page.SpaceID)
	testutil.Equal(t, 1, page.Version.Number)
	testutil.Equal(t, "/spaces/TEST/pages/99999", page.Links.WebUI)
}

func TestClient_CopyPage_MissingTitle(t *testing.T) {
	t.Parallel()
	client := NewClient("http://unused", "user@example.com", "token")

	_, err := client.CopyPage(context.Background(), "12345", &CopyPageOptions{})
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "title is required")
}

func TestClient_CopyPage_NilOptions(t *testing.T) {
	t.Parallel()
	client := NewClient("http://unused", "user@example.com", "token")

	_, err := client.CopyPage(context.Background(), "12345", nil)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "title is required")
}

func TestClient_CopyPage_APIError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	testutil.RequireError(t, err)
}

func TestClient_CopyPage_WithoutAttachments(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		testutil.RequireNoError(t, err)

		var req map[string]any
		err = json.Unmarshal(body, &req)
		testutil.RequireNoError(t, err)

		testutil.Equal(t, false, req["copyAttachments"])

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
	testutil.RequireNoError(t, err)
}

func TestClient_CopyPage_ToDifferentSpace(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		testutil.RequireNoError(t, err)

		var req map[string]any
		err = json.Unmarshal(body, &req)
		testutil.RequireNoError(t, err)

		dest := req["destination"].(map[string]any)
		testutil.Equal(t, "space", dest["type"])
		testutil.Equal(t, "OTHERSPACE", dest["value"])

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
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "OTHERSPACE", page.SpaceID)
}

func TestClient_CopyPage_WithoutLabels(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		testutil.RequireNoError(t, err)

		var req map[string]any
		err = json.Unmarshal(body, &req)
		testutil.RequireNoError(t, err)

		testutil.Equal(t, false, req["copyLabels"])
		testutil.Equal(t, true, req["copyAttachments"]) // others should still be true

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
	testutil.RequireNoError(t, err)
}

func TestClient_UpdatePage_VersionConflict(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "conflict")
}

func TestClient_GetPage_MissingBody(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	testutil.RequireNoError(t, err)
	testutil.Equal(t, "98765", page.ID)
	testutil.Equal(t, "Page Without Body", page.Title)
	testutil.Nil(t, page.Body)
}

func TestClient_GetPage_EmptyBodyStorage(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	testutil.RequireNoError(t, err)
	testutil.NotNil(t, page.Body)
	testutil.Nil(t, page.Body.Storage)
}

func TestClient_ListPages_WithCursor(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		if callCount == 1 {
			// First call - return results with cursor
			testutil.Empty(t, r.URL.Query().Get("cursor"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"results": [{"id": "1", "title": "Page 1"}],
				"_links": {"next": "/api/v2/spaces/123/pages?cursor=abc123"}
			}`))
		} else {
			// Second call - verify cursor is passed
			testutil.Equal(t, "abc123", r.URL.Query().Get("cursor"))
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
	testutil.RequireNoError(t, err)
	testutil.True(t, result1.HasMore())

	// Second request with cursor
	opts := &ListPagesOptions{Cursor: "abc123"}
	result2, err := client.ListPages(context.Background(), "123", opts)
	testutil.RequireNoError(t, err)
	testutil.False(t, result2.HasMore())

	testutil.Equal(t, 2, callCount)
}

func TestClient_ListPages_EmptyResults(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	result, err := client.ListPages(context.Background(), "123", nil)

	testutil.RequireNoError(t, err)
	testutil.Empty(t, result.Results)
	testutil.False(t, result.HasMore())
}

func TestClient_ListPages_NullVersion(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Results, 1)
	testutil.Nil(t, result.Results[0].Version)
}
