package page

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rianjs/confluence-cli/api"
)

// mockListServer creates a test server for page list operations
// It handles both GetSpaceByKey and ListPages endpoints
func mockListServer(t *testing.T, spaceKey, spaceID string, pages string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/spaces") && r.URL.Query().Get("keys") != "":
			// GetSpaceByKey
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"results": [{"id": "` + spaceID + `", "key": "` + spaceKey + `", "name": "Test Space", "type": "global"}]
			}`))
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/spaces/"+spaceID+"/pages"):
			// ListPages
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(pages))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestRunList_PageList_Success(t *testing.T) {
	server := mockListServer(t, "DEV", "123456", `{
		"results": [
			{"id": "11111", "title": "Page One", "status": "current", "version": {"number": 1}},
			{"id": "22222", "title": "Page Two", "status": "current", "version": {"number": 5}}
		]
	}`)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		space:   "DEV",
		limit:   25,
		status:  "current",
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}

func TestRunList_PageList_EmptyResults(t *testing.T) {
	server := mockListServer(t, "DEV", "123456", `{"results": []}`)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		space:   "DEV",
		limit:   25,
		status:  "current",
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}

func TestRunList_PageList_JSONOutput(t *testing.T) {
	server := mockListServer(t, "DEV", "123456", `{
		"results": [
			{"id": "11111", "title": "Page One", "status": "current", "version": {"number": 1}}
		]
	}`)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		space:   "DEV",
		limit:   25,
		status:  "current",
		output:  "json",
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}

func TestRunList_PageList_InvalidOutputFormat(t *testing.T) {
	opts := &listOptions{
		space:   "DEV",
		limit:   25,
		output:  "invalid",
		noColor: true,
	}

	err := runList(opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
}

func TestRunList_PageList_NegativeLimit(t *testing.T) {
	opts := &listOptions{
		space:   "DEV",
		limit:   -1,
		status:  "current",
		noColor: true,
	}

	err := runList(opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid limit")
}

func TestRunList_PageList_ZeroLimit(t *testing.T) {
	opts := &listOptions{
		space:   "DEV",
		limit:   0,
		status:  "current",
		noColor: true,
	}

	// Zero limit should return empty without making API call
	err := runList(opts, nil)
	require.NoError(t, err)
}

func TestRunList_PageList_MissingSpace(t *testing.T) {
	// Create a mock client to avoid config loading
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		space:   "", // No space provided
		limit:   25,
		status:  "current",
		noColor: true,
	}

	err := runList(opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "space is required")
}

func TestRunList_PageList_SpaceNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return empty results for space lookup
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		space:   "INVALID",
		limit:   25,
		status:  "current",
		noColor: true,
	}

	err := runList(opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find space")
}

func TestRunList_PageList_NullVersion(t *testing.T) {
	server := mockListServer(t, "DEV", "123456", `{
		"results": [
			{"id": "11111", "title": "Page Without Version", "status": "current", "version": null}
		]
	}`)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		space:   "DEV",
		limit:   25,
		status:  "current",
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}

func TestRunList_PageList_HasMore(t *testing.T) {
	server := mockListServer(t, "DEV", "123456", `{
		"results": [
			{"id": "11111", "title": "Page One", "status": "current", "version": {"number": 1}}
		],
		"_links": {"next": "/wiki/api/v2/spaces/123456/pages?cursor=abc"}
	}`)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		space:   "DEV",
		limit:   25,
		status:  "current",
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}

func TestRunList_PageList_LongTitle(t *testing.T) {
	longTitle := strings.Repeat("A", 100)
	server := mockListServer(t, "DEV", "123456", `{
		"results": [
			{"id": "11111", "title": "`+longTitle+`", "status": "current", "version": {"number": 1}}
		]
	}`)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		space:   "DEV",
		limit:   25,
		status:  "current",
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}

func TestRunList_PageList_StatusFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pages") {
			assert.Equal(t, "archived", r.URL.Query().Get("status"))
		}
		if r.URL.Query().Get("keys") != "" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"results": [{"id": "123456", "key": "DEV", "name": "Test", "type": "global"}]}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		space:   "DEV",
		limit:   25,
		status:  "archived",
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}

func TestRunList_PageList_InvalidStatus(t *testing.T) {
	opts := &listOptions{
		space:   "DEV",
		limit:   25,
		status:  "draft",
		noColor: true,
	}

	err := runList(opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")
	assert.Contains(t, err.Error(), "draft")
}

func TestRunList_PageList_TrashedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pages") {
			assert.Equal(t, "trashed", r.URL.Query().Get("status"))
		}
		if r.URL.Query().Get("keys") != "" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"results": [{"id": "123456", "key": "DEV", "name": "Test", "type": "global"}]}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		space:   "DEV",
		limit:   25,
		status:  "trashed",
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}
