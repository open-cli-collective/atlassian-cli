package space

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

func newTestRootOptions() *root.Options {
	return &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
}

func TestRunList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/spaces")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [
				{
					"id": "123456",
					"key": "DEV",
					"name": "Development",
					"type": "global",
					"description": {"plain": {"value": "Development team space"}}
				},
				{
					"id": "789012",
					"key": "DOCS",
					"name": "Documentation",
					"type": "global",
					"description": {"plain": {"value": "Product documentation"}}
				}
			]
		}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		limit:   25,
	}

	err := runList(opts)
	require.NoError(t, err)
}

func TestRunList_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		limit:   25,
	}

	err := runList(opts)
	require.NoError(t, err)
}

func TestRunList_JSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [
				{"id": "123456", "key": "DEV", "name": "Development", "type": "global"}
			]
		}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	rootOpts.Output = "json"
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		limit:   25,
	}

	err := runList(opts)
	require.NoError(t, err)
}

func TestRunList_InvalidOutputFormat(t *testing.T) {
	rootOpts := newTestRootOptions()
	rootOpts.Output = "invalid"

	opts := &listOptions{
		Options: rootOpts,
		limit:   25,
	}

	err := runList(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
}

func TestRunList_NegativeLimit(t *testing.T) {
	rootOpts := newTestRootOptions()

	opts := &listOptions{
		Options: rootOpts,
		limit:   -1,
	}

	err := runList(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid limit")
}

func TestRunList_ZeroLimit(t *testing.T) {
	rootOpts := newTestRootOptions()

	opts := &listOptions{
		Options: rootOpts,
		limit:   0,
	}

	// Zero limit should return empty without making API call
	err := runList(opts)
	require.NoError(t, err)
}

func TestRunList_ZeroLimitJSON(t *testing.T) {
	rootOpts := newTestRootOptions()
	rootOpts.Output = "json"

	opts := &listOptions{
		Options: rootOpts,
		limit:   0,
	}

	// Zero limit should return empty JSON array without making API call
	err := runList(opts)
	require.NoError(t, err)
}

func TestRunList_WithTypeFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "global", r.URL.Query().Get("type"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [
				{"id": "123456", "key": "DEV", "name": "Development", "type": "global"}
			]
		}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options:   rootOpts,
		limit:     25,
		spaceType: "global",
	}

	err := runList(opts)
	require.NoError(t, err)
}

func TestRunList_WithLimitParameter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "50", r.URL.Query().Get("limit"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		limit:   50,
	}

	err := runList(opts)
	require.NoError(t, err)
}

func TestRunList_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message": "Authentication required"}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		limit:   25,
	}

	err := runList(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list spaces")
}

func TestRunList_HasMore(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [
				{"id": "123456", "key": "DEV", "name": "Development", "type": "global"}
			],
			"_links": {"next": "/wiki/api/v2/spaces?cursor=abc123"}
		}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		limit:   25,
	}

	err := runList(opts)
	require.NoError(t, err)
}

func TestRunList_NullDescription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [
				{"id": "123456", "key": "DEV", "name": "Development", "type": "global", "description": null}
			]
		}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		limit:   25,
	}

	err := runList(opts)
	require.NoError(t, err)
}
