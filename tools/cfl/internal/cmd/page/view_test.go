package page

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/confluence-cli/api"
)

func TestRunView_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/pages/12345")
		assert.Equal(t, "GET", r.Method)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "12345",
			"title": "Test Page",
			"version": {"number": 3},
			"body": {"storage": {"value": "<p>Hello <strong>World</strong></p>"}},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &viewOptions{
		noColor: true,
	}

	err := runView("12345", opts, client)
	require.NoError(t, err)
}

func TestRunView_RawFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "12345",
			"title": "Test Page",
			"version": {"number": 1},
			"body": {"storage": {"value": "<p>Raw HTML Content</p>"}},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &viewOptions{
		raw:     true,
		noColor: true,
	}

	err := runView("12345", opts, client)
	require.NoError(t, err)
}

func TestRunView_JSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "12345",
			"title": "Test Page",
			"version": {"number": 1},
			"body": {"storage": {"value": "<p>Content</p>"}},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &viewOptions{
		output:  "json",
		noColor: true,
	}

	err := runView("12345", opts, client)
	require.NoError(t, err)
}

func TestRunView_PageNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Page not found"}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &viewOptions{
		noColor: true,
	}

	err := runView("99999", opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get page")
}

func TestRunView_EmptyContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "12345",
			"title": "Empty Page",
			"version": {"number": 1},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &viewOptions{
		noColor: true,
	}

	err := runView("12345", opts, client)
	require.NoError(t, err)
}

func TestRunView_InvalidOutputFormat(t *testing.T) {
	client := api.NewClient("http://unused", "test@example.com", "token")
	opts := &viewOptions{
		output:  "invalid",
		noColor: true,
	}

	err := runView("12345", opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
}

func TestRunView_ShowMacros(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "12345",
			"title": "Page with Macros",
			"version": {"number": 1},
			"body": {"storage": {"value": "<ac:structured-macro ac:name=\"toc\"><ac:parameter ac:name=\"maxLevel\">2</ac:parameter></ac:structured-macro><p>Content</p>"}},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &viewOptions{
		showMacros: true,
		noColor:    true,
	}

	err := runView("12345", opts, client)
	require.NoError(t, err)
}

func TestRunView_ContentOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "12345",
			"title": "Test Page",
			"version": {"number": 3},
			"body": {"storage": {"value": "<p>Hello <strong>World</strong></p>"}},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &viewOptions{
		contentOnly: true,
		noColor:     true,
	}

	err := runView("12345", opts, client)
	require.NoError(t, err)
	// Output should only contain markdown content, no Title:/ID:/Version: headers
}

func TestRunView_ContentOnly_Raw(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "12345",
			"title": "Test Page",
			"version": {"number": 1},
			"body": {"storage": {"value": "<p>Raw HTML Content</p>"}},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &viewOptions{
		contentOnly: true,
		raw:         true,
		noColor:     true,
	}

	err := runView("12345", opts, client)
	require.NoError(t, err)
	// Output should only contain raw XHTML, no Title:/ID:/Version: headers
}

func TestRunView_ContentOnly_ShowMacros(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "12345",
			"title": "Page with Macros",
			"version": {"number": 1},
			"body": {"storage": {"value": "<ac:structured-macro ac:name=\"toc\"><ac:parameter ac:name=\"maxLevel\">2</ac:parameter></ac:structured-macro><p>Content</p>"}},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &viewOptions{
		contentOnly: true,
		showMacros:  true,
		noColor:     true,
	}

	err := runView("12345", opts, client)
	require.NoError(t, err)
	// Output should contain markdown with [TOC] macro placeholder
}

func TestRunView_ContentOnly_JSON_Error(t *testing.T) {
	client := api.NewClient("http://unused", "test@example.com", "token")
	opts := &viewOptions{
		contentOnly: true,
		output:      "json",
		noColor:     true,
	}

	err := runView("12345", opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--content-only is incompatible with --output json")
}

func TestRunView_ContentOnly_Web_Error(t *testing.T) {
	client := api.NewClient("http://unused", "test@example.com", "token")
	opts := &viewOptions{
		contentOnly: true,
		web:         true,
		noColor:     true,
	}

	err := runView("12345", opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--content-only is incompatible with --web")
}

func TestRunView_ContentOnly_EmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "12345",
			"title": "Empty Page",
			"version": {"number": 1},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &viewOptions{
		contentOnly: true,
		noColor:     true,
	}

	err := runView("12345", opts, client)
	require.NoError(t, err)
	// Output should be "(No content)" without metadata headers
}

// Ensure strings is used
var _ = strings.NewReader("")
