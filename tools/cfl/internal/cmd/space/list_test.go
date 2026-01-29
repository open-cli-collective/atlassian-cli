package space

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/confluence-cli/api"
)

func TestRunList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/spaces")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
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

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		limit:   25,
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}

func TestRunList_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		limit:   25,
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}

func TestRunList_JSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"results": [
				{"id": "123456", "key": "DEV", "name": "Development", "type": "global"}
			]
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		limit:   25,
		output:  "json",
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}

func TestRunList_InvalidOutputFormat(t *testing.T) {
	opts := &listOptions{
		limit:   25,
		output:  "invalid",
		noColor: true,
	}

	err := runList(opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
}

func TestRunList_NegativeLimit(t *testing.T) {
	opts := &listOptions{
		limit:   -1,
		noColor: true,
	}

	err := runList(opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid limit")
}

func TestRunList_ZeroLimit(t *testing.T) {
	opts := &listOptions{
		limit:   0,
		noColor: true,
	}

	// Zero limit should return empty without making API call
	err := runList(opts, nil)
	require.NoError(t, err)
}

func TestRunList_ZeroLimitJSON(t *testing.T) {
	opts := &listOptions{
		limit:   0,
		output:  "json",
		noColor: true,
	}

	// Zero limit should return empty JSON array without making API call
	err := runList(opts, nil)
	require.NoError(t, err)
}

func TestRunList_WithTypeFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "global", r.URL.Query().Get("type"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"results": [
				{"id": "123456", "key": "DEV", "name": "Development", "type": "global"}
			]
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		limit:     25,
		spaceType: "global",
		noColor:   true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}

func TestRunList_WithLimitParameter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "50", r.URL.Query().Get("limit"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		limit:   50,
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}

func TestRunList_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message": "Authentication required"}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		limit:   25,
		noColor: true,
	}

	err := runList(opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list spaces")
}

func TestRunList_HasMore(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"results": [
				{"id": "123456", "key": "DEV", "name": "Development", "type": "global"}
			],
			"_links": {"next": "/wiki/api/v2/spaces?cursor=abc123"}
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		limit:   25,
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}

func TestRunList_NullDescription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"results": [
				{"id": "123456", "key": "DEV", "name": "Development", "type": "global", "description": null}
			]
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &listOptions{
		limit:   25,
		noColor: true,
	}

	err := runList(opts, client)
	require.NoError(t, err)
}
