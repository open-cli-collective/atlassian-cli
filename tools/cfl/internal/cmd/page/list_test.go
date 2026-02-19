package page

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

// mockListServer creates a test server for page list operations
// It handles both GetSpaceByKey and ListPages endpoints
func mockListServer(t *testing.T, spaceKey, spaceID string, pages string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/spaces") && r.URL.Query().Get("keys") != "":
			// GetSpaceByKey
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"results": [{"id": "` + spaceID + `", "key": "` + spaceKey + `", "name": "Test Space", "type": "global"}]
			}`))
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/spaces/"+spaceID+"/pages"):
			// ListPages
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pages))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func newListPageTestRootOptions() *root.Options {
	return &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
}

func TestRunList_PageList_Success(t *testing.T) {
	t.Parallel()
	server := mockListServer(t, "DEV", "123456", `{
		"results": [
			{"id": "11111", "title": "Page One", "status": "current", "version": {"number": 1}},
			{"id": "22222", "title": "Page Two", "status": "current", "version": {"number": 5}}
		]
	}`)
	defer server.Close()

	rootOpts := newListPageTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		space:   "DEV",
		limit:   25,
		status:  "current",
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunList_PageList_EmptyResults(t *testing.T) {
	t.Parallel()
	server := mockListServer(t, "DEV", "123456", `{"results": []}`)
	defer server.Close()

	rootOpts := newListPageTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		space:   "DEV",
		limit:   25,
		status:  "current",
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunList_PageList_JSONOutput(t *testing.T) {
	t.Parallel()
	server := mockListServer(t, "DEV", "123456", `{
		"results": [
			{"id": "11111", "title": "Page One", "status": "current", "version": {"number": 1}}
		]
	}`)
	defer server.Close()

	rootOpts := newListPageTestRootOptions()
	rootOpts.Output = "json"
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		space:   "DEV",
		limit:   25,
		status:  "current",
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunList_PageList_InvalidOutputFormat(t *testing.T) {
	t.Parallel()
	rootOpts := newListPageTestRootOptions()
	rootOpts.Output = "invalid"

	opts := &listOptions{
		Options: rootOpts,
		space:   "DEV",
		limit:   25,
	}

	err := runList(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "invalid output format")
}

func TestRunList_PageList_NegativeLimit(t *testing.T) {
	t.Parallel()
	rootOpts := newListPageTestRootOptions()

	opts := &listOptions{
		Options: rootOpts,
		space:   "DEV",
		limit:   -1,
		status:  "current",
	}

	err := runList(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "invalid limit")
}

func TestRunList_PageList_ZeroLimit(t *testing.T) {
	t.Parallel()
	rootOpts := newListPageTestRootOptions()

	opts := &listOptions{
		Options: rootOpts,
		space:   "DEV",
		limit:   0,
		status:  "current",
	}

	// Zero limit should return empty without making API call
	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunList_PageList_MissingSpace(t *testing.T) {
	t.Parallel()
	// Create a mock client to avoid config loading
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rootOpts := newListPageTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		space:   "", // No space provided
		limit:   25,
		status:  "current",
	}

	err := runList(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "space is required")
}

func TestRunList_PageList_SpaceNotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Return empty results for space lookup
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	rootOpts := newListPageTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		space:   "INVALID",
		limit:   25,
		status:  "current",
	}

	err := runList(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "finding space")
}

func TestRunList_PageList_NullVersion(t *testing.T) {
	t.Parallel()
	server := mockListServer(t, "DEV", "123456", `{
		"results": [
			{"id": "11111", "title": "Page Without Version", "status": "current", "version": null}
		]
	}`)
	defer server.Close()

	rootOpts := newListPageTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		space:   "DEV",
		limit:   25,
		status:  "current",
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunList_PageList_HasMore(t *testing.T) {
	t.Parallel()
	server := mockListServer(t, "DEV", "123456", `{
		"results": [
			{"id": "11111", "title": "Page One", "status": "current", "version": {"number": 1}}
		],
		"_links": {"next": "/wiki/api/v2/spaces/123456/pages?cursor=abc"}
	}`)
	defer server.Close()

	rootOpts := newListPageTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		space:   "DEV",
		limit:   25,
		status:  "current",
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunList_PageList_LongTitle(t *testing.T) {
	t.Parallel()
	longTitle := strings.Repeat("A", 100)
	server := mockListServer(t, "DEV", "123456", `{
		"results": [
			{"id": "11111", "title": "`+longTitle+`", "status": "current", "version": {"number": 1}}
		]
	}`)
	defer server.Close()

	rootOpts := newListPageTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		space:   "DEV",
		limit:   25,
		status:  "current",
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunList_PageList_StatusFilter(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pages") {
			testutil.Equal(t, "archived", r.URL.Query().Get("status"))
		}
		if r.URL.Query().Get("keys") != "" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"results": [{"id": "123456", "key": "DEV", "name": "Test", "type": "global"}]}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	rootOpts := newListPageTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		space:   "DEV",
		limit:   25,
		status:  "archived",
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunList_PageList_InvalidStatus(t *testing.T) {
	t.Parallel()
	rootOpts := newListPageTestRootOptions()

	opts := &listOptions{
		Options: rootOpts,
		space:   "DEV",
		limit:   25,
		status:  "draft",
	}

	err := runList(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "invalid status")
	testutil.Contains(t, err.Error(), "draft")
}

func TestRunList_PageList_TrashedStatus(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pages") {
			testutil.Equal(t, "trashed", r.URL.Query().Get("status"))
		}
		if r.URL.Query().Get("keys") != "" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"results": [{"id": "123456", "key": "DEV", "name": "Test", "type": "global"}]}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	rootOpts := newListPageTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		space:   "DEV",
		limit:   25,
		status:  "trashed",
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
}
