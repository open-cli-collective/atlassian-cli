package page

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

func newViewTestRootOptions() *root.Options {
	return &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
}

func TestRunView_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/pages/12345")
		assert.Equal(t, "GET", r.Method)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "12345",
			"title": "Test Page",
			"version": {"number": 3},
			"body": {"storage": {"value": "<p>Hello <strong>World</strong></p>"}},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &viewOptions{
		Options: rootOpts,
	}

	err := runView("12345", opts)
	require.NoError(t, err)
}

func TestRunView_RawFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "12345",
			"title": "Test Page",
			"version": {"number": 1},
			"body": {"storage": {"value": "<p>Raw HTML Content</p>"}},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &viewOptions{
		Options: rootOpts,
		raw:     true,
	}

	err := runView("12345", opts)
	require.NoError(t, err)
}

func TestRunView_JSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "12345",
			"title": "Test Page",
			"version": {"number": 1},
			"body": {"storage": {"value": "<p>Content</p>"}},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	rootOpts.Output = "json"
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &viewOptions{
		Options: rootOpts,
	}

	err := runView("12345", opts)
	require.NoError(t, err)
}

func TestRunView_PageNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Page not found"}`))
	}))
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &viewOptions{
		Options: rootOpts,
	}

	err := runView("99999", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get page")
}

func TestRunView_EmptyContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "12345",
			"title": "Empty Page",
			"version": {"number": 1},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &viewOptions{
		Options: rootOpts,
	}

	err := runView("12345", opts)
	require.NoError(t, err)
}

func TestRunView_InvalidOutputFormat(t *testing.T) {
	rootOpts := newViewTestRootOptions()
	rootOpts.Output = "invalid"

	opts := &viewOptions{
		Options: rootOpts,
	}

	err := runView("12345", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
}

func TestRunView_ShowMacros(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "12345",
			"title": "Page with Macros",
			"version": {"number": 1},
			"body": {"storage": {"value": "<ac:structured-macro ac:name=\"toc\"><ac:parameter ac:name=\"maxLevel\">2</ac:parameter></ac:structured-macro><p>Content</p>"}},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &viewOptions{
		Options:    rootOpts,
		showMacros: true,
	}

	err := runView("12345", opts)
	require.NoError(t, err)
}

func TestRunView_ContentOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "12345",
			"title": "Test Page",
			"version": {"number": 3},
			"body": {"storage": {"value": "<p>Hello <strong>World</strong></p>"}},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &viewOptions{
		Options:     rootOpts,
		contentOnly: true,
	}

	err := runView("12345", opts)
	require.NoError(t, err)
	// Output should only contain markdown content, no Title:/ID:/Version: headers
}

func TestRunView_ContentOnly_Raw(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "12345",
			"title": "Test Page",
			"version": {"number": 1},
			"body": {"storage": {"value": "<p>Raw HTML Content</p>"}},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &viewOptions{
		Options:     rootOpts,
		contentOnly: true,
		raw:         true,
	}

	err := runView("12345", opts)
	require.NoError(t, err)
	// Output should only contain raw XHTML, no Title:/ID:/Version: headers
}

func TestRunView_ContentOnly_ShowMacros(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "12345",
			"title": "Page with Macros",
			"version": {"number": 1},
			"body": {"storage": {"value": "<ac:structured-macro ac:name=\"toc\"><ac:parameter ac:name=\"maxLevel\">2</ac:parameter></ac:structured-macro><p>Content</p>"}},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &viewOptions{
		Options:     rootOpts,
		contentOnly: true,
		showMacros:  true,
	}

	err := runView("12345", opts)
	require.NoError(t, err)
	// Output should contain markdown with [TOC] macro placeholder
}

func TestRunView_ContentOnly_JSON_Error(t *testing.T) {
	rootOpts := newViewTestRootOptions()
	rootOpts.Output = "json"

	opts := &viewOptions{
		Options:     rootOpts,
		contentOnly: true,
	}

	err := runView("12345", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--content-only is incompatible with --output json")
}

func TestRunView_ContentOnly_Web_Error(t *testing.T) {
	rootOpts := newViewTestRootOptions()

	opts := &viewOptions{
		Options:     rootOpts,
		contentOnly: true,
		web:         true,
	}

	err := runView("12345", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--content-only is incompatible with --web")
}

func TestRunView_ContentOnly_EmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "12345",
			"title": "Empty Page",
			"version": {"number": 1},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &viewOptions{
		Options:     rootOpts,
		contentOnly: true,
	}

	err := runView("12345", opts)
	require.NoError(t, err)
	// Output should be "(No content)" without metadata headers
}
