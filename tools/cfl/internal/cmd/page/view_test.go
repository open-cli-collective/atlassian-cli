package page

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
		assert.Equal(t, "storage", r.URL.Query().Get("body-format"), "must request body-format=storage to get page content")

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

func TestRunView_WithSpaceKey(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call: GetPage
			assert.Contains(t, r.URL.Path, "/pages/12345")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"spaceId": "98765",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Content</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		} else {
			// Second call: GetSpace
			assert.Contains(t, r.URL.Path, "/spaces/98765")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "98765",
				"key": "DEV",
				"name": "Development",
				"type": "global"
			}`))
		}
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
	assert.Equal(t, 2, callCount, "should call both GetPage and GetSpace")
}

func TestRunView_SpaceLookupFails_Graceful(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call: GetPage
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"spaceId": "98765",
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Content</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		} else {
			// Second call: GetSpace - fails
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message": "Space not found"}`))
		}
	}))
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &viewOptions{
		Options: rootOpts,
	}

	// Should succeed even if space lookup fails
	err := runView("12345", opts)
	require.NoError(t, err)
}

func TestEnrichPageWithSpaceKey(t *testing.T) {
	page := &api.Page{
		ID:      "12345",
		Title:   "Test Page",
		SpaceID: "98765",
	}

	enriched := enrichPageWithSpaceKey(page, "DEV")

	assert.Equal(t, "12345", enriched.ID)
	assert.Equal(t, "Test Page", enriched.Title)
	assert.Equal(t, "DEV", enriched.SpaceKey)
}

func TestTruncateContent(t *testing.T) {
	t.Run("short content is not truncated", func(t *testing.T) {
		opts := &viewOptions{}
		result := truncateContent("short", opts)
		assert.Equal(t, "short", result)
	})

	t.Run("long content is truncated by default", func(t *testing.T) {
		opts := &viewOptions{}
		long := strings.Repeat("x", maxViewChars+100)
		result := truncateContent(long, opts)
		assert.Len(t, strings.SplitN(result, "\n\n... [truncated", 2)[0], maxViewChars)
		assert.Contains(t, result, fmt.Sprintf("... [truncated at %d chars, use --full for complete text]", maxViewChars))
	})

	t.Run("--full bypasses truncation", func(t *testing.T) {
		opts := &viewOptions{full: true}
		long := strings.Repeat("x", maxViewChars+100)
		result := truncateContent(long, opts)
		assert.Equal(t, long, result)
	})

	t.Run("--content-only implies full", func(t *testing.T) {
		opts := &viewOptions{contentOnly: true}
		long := strings.Repeat("x", maxViewChars+100)
		result := truncateContent(long, opts)
		assert.Equal(t, long, result)
	})

	t.Run("content at exact limit is not truncated", func(t *testing.T) {
		opts := &viewOptions{}
		exact := strings.Repeat("x", maxViewChars)
		result := truncateContent(exact, opts)
		assert.Equal(t, exact, result)
	})
}
