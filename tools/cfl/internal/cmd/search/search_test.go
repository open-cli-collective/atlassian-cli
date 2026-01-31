package search

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

// mockSearchServer creates a test server for search operations
func mockSearchServer(t *testing.T, response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/rest/api/search") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(response))
		} else {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func newTestRootOptions() *root.Options {
	return &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
}

func TestRunSearch_Success(t *testing.T) {
	server := mockSearchServer(t, `{
		"results": [
			{
				"content": {"id": "12345", "type": "page", "status": "current", "title": "Getting Started"},
				"resultGlobalContainer": {"title": "Development"}
			},
			{
				"content": {"id": "12346", "type": "page", "status": "current", "title": "API Docs"},
				"resultGlobalContainer": {"title": "Development"}
			}
		],
		"start": 0,
		"size": 2,
		"totalSize": 2
	}`)
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &searchOptions{
		Options: rootOpts,
		query:   "test",
		limit:   25,
	}

	err := runSearch(opts)
	require.NoError(t, err)
}

func TestRunSearch_EmptyResults(t *testing.T) {
	server := mockSearchServer(t, `{
		"results": [],
		"start": 0,
		"size": 0,
		"totalSize": 0
	}`)
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &searchOptions{
		Options: rootOpts,
		query:   "nonexistent",
		limit:   25,
	}

	err := runSearch(opts)
	require.NoError(t, err)
}

func TestRunSearch_JSONOutput(t *testing.T) {
	server := mockSearchServer(t, `{
		"results": [
			{
				"content": {"id": "12345", "type": "page", "status": "current", "title": "Test Page"},
				"resultGlobalContainer": {"title": "DEV"}
			}
		],
		"start": 0,
		"size": 1,
		"totalSize": 1
	}`)
	defer server.Close()

	rootOpts := newTestRootOptions()
	rootOpts.Output = "json"
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &searchOptions{
		Options: rootOpts,
		query:   "test",
		limit:   25,
	}

	err := runSearch(opts)
	require.NoError(t, err)
}

func TestRunSearch_PlainOutput(t *testing.T) {
	server := mockSearchServer(t, `{
		"results": [
			{
				"content": {"id": "12345", "type": "page", "status": "current", "title": "Test Page"},
				"resultGlobalContainer": {"title": "DEV"}
			}
		],
		"start": 0,
		"size": 1,
		"totalSize": 1
	}`)
	defer server.Close()

	rootOpts := newTestRootOptions()
	rootOpts.Output = "plain"
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &searchOptions{
		Options: rootOpts,
		query:   "test",
		limit:   25,
	}

	err := runSearch(opts)
	require.NoError(t, err)
}

func TestRunSearch_InvalidOutputFormat(t *testing.T) {
	rootOpts := newTestRootOptions()
	rootOpts.Output = "invalid"

	opts := &searchOptions{
		Options: rootOpts,
		query:   "test",
		limit:   25,
	}

	err := runSearch(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
}

func TestRunSearch_InvalidType(t *testing.T) {
	rootOpts := newTestRootOptions()

	opts := &searchOptions{
		Options:     rootOpts,
		query:       "test",
		limit:       25,
		contentType: "invalid",
	}

	err := runSearch(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
	assert.Contains(t, err.Error(), "invalid")
}

func TestRunSearch_ValidTypes(t *testing.T) {
	validTypes := []string{"page", "blogpost", "attachment", "comment"}

	for _, contentType := range validTypes {
		t.Run(contentType, func(t *testing.T) {
			server := mockSearchServer(t, `{"results": [], "totalSize": 0}`)
			defer server.Close()

			rootOpts := newTestRootOptions()
			client := api.NewClient(server.URL, "test@example.com", "token")
			rootOpts.SetAPIClient(client)

			opts := &searchOptions{
				Options:     rootOpts,
				contentType: contentType,
				space:       "DEV", // Need at least one filter
				limit:       25,
			}

			err := runSearch(opts)
			require.NoError(t, err)
		})
	}
}

func TestRunSearch_NoQuery(t *testing.T) {
	rootOpts := newTestRootOptions()

	opts := &searchOptions{
		Options: rootOpts,
		limit:   25,
	}

	err := runSearch(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search requires a query")
}

func TestRunSearch_NegativeLimit(t *testing.T) {
	rootOpts := newTestRootOptions()

	opts := &searchOptions{
		Options: rootOpts,
		query:   "test",
		limit:   -1,
	}

	err := runSearch(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid limit")
}

func TestRunSearch_ZeroLimit(t *testing.T) {
	rootOpts := newTestRootOptions()

	opts := &searchOptions{
		Options: rootOpts,
		query:   "test",
		limit:   0,
	}

	// Zero limit should return empty without making API call
	err := runSearch(opts)
	require.NoError(t, err)
}

func TestRunSearch_WithSpaceFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cql := r.URL.Query().Get("cql")
		assert.Contains(t, cql, `space = "DEV"`)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": [], "totalSize": 0}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &searchOptions{
		Options: rootOpts,
		query:   "test",
		space:   "DEV",
		limit:   25,
	}

	err := runSearch(opts)
	require.NoError(t, err)
}

func TestRunSearch_WithTypeFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cql := r.URL.Query().Get("cql")
		assert.Contains(t, cql, `type = "page"`)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": [], "totalSize": 0}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &searchOptions{
		Options:     rootOpts,
		query:       "test",
		contentType: "page",
		limit:       25,
	}

	err := runSearch(opts)
	require.NoError(t, err)
}

func TestRunSearch_WithTitleFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cql := r.URL.Query().Get("cql")
		assert.Contains(t, cql, `title ~ "Getting Started"`)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": [], "totalSize": 0}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &searchOptions{
		Options: rootOpts,
		title:   "Getting Started",
		limit:   25,
	}

	err := runSearch(opts)
	require.NoError(t, err)
}

func TestRunSearch_WithLabelFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cql := r.URL.Query().Get("cql")
		assert.Contains(t, cql, `label = "documentation"`)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": [], "totalSize": 0}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &searchOptions{
		Options: rootOpts,
		label:   "documentation",
		limit:   25,
	}

	err := runSearch(opts)
	require.NoError(t, err)
}

func TestRunSearch_WithRawCQL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cql := r.URL.Query().Get("cql")
		// Raw CQL should be used as-is
		assert.Equal(t, `type=page AND lastModified > now("-7d")`, cql)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": [], "totalSize": 0}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &searchOptions{
		Options: rootOpts,
		cql:     `type=page AND lastModified > now("-7d")`,
		limit:   25,
	}

	err := runSearch(opts)
	require.NoError(t, err)
}

func TestRunSearch_CombinedFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cql := r.URL.Query().Get("cql")
		assert.Contains(t, cql, `text ~ "kubernetes"`)
		assert.Contains(t, cql, `space = "DEV"`)
		assert.Contains(t, cql, `type = "page"`)
		assert.Contains(t, cql, `label = "infrastructure"`)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": [], "totalSize": 0}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &searchOptions{
		Options:     rootOpts,
		query:       "kubernetes",
		space:       "DEV",
		contentType: "page",
		label:       "infrastructure",
		limit:       25,
	}

	err := runSearch(opts)
	require.NoError(t, err)
}

func TestRunSearch_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message": "Invalid CQL query"}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &searchOptions{
		Options: rootOpts,
		query:   "test",
		limit:   25,
	}

	err := runSearch(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search failed")
}

func TestRunSearch_HasMore(t *testing.T) {
	server := mockSearchServer(t, `{
		"results": [
			{
				"content": {"id": "12345", "type": "page", "status": "current", "title": "Test"},
				"resultGlobalContainer": {"title": "DEV"}
			}
		],
		"start": 0,
		"size": 1,
		"totalSize": 100
	}`)
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &searchOptions{
		Options: rootOpts,
		query:   "test",
		limit:   25,
	}

	err := runSearch(opts)
	require.NoError(t, err)
}

func TestRunSearch_LongTitle(t *testing.T) {
	longTitle := strings.Repeat("A", 100)
	server := mockSearchServer(t, `{
		"results": [
			{
				"content": {"id": "12345", "type": "page", "status": "current", "title": "`+longTitle+`"},
				"resultGlobalContainer": {"title": "Development"}
			}
		],
		"start": 0,
		"size": 1,
		"totalSize": 1
	}`)
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &searchOptions{
		Options: rootOpts,
		query:   "test",
		limit:   25,
	}

	err := runSearch(opts)
	require.NoError(t, err)
}

func TestRunSearch_SpaceOnlyFilter(t *testing.T) {
	// Space-only filter should work (no query required)
	server := mockSearchServer(t, `{"results": [], "totalSize": 0}`)
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &searchOptions{
		Options: rootOpts,
		space:   "DEV",
		limit:   25,
	}

	err := runSearch(opts)
	require.NoError(t, err)
}

func TestRunSearch_LimitParameter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := r.URL.Query().Get("limit")
		assert.Equal(t, "50", limit)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": [], "totalSize": 0}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &searchOptions{
		Options: rootOpts,
		query:   "test",
		limit:   50,
	}

	err := runSearch(opts)
	require.NoError(t, err)
}

func TestExtractSpaceKey(t *testing.T) {
	tests := []struct {
		name       string
		displayURL string
		want       string
	}{
		{
			name:       "standard space URL",
			displayURL: "/spaces/DEV/pages/12345",
			want:       "DEV",
		},
		{
			name:       "wiki prefixed URL",
			displayURL: "/wiki/spaces/DOCS/overview",
			want:       "DOCS",
		},
		{
			name:       "full URL with domain",
			displayURL: "https://example.atlassian.net/wiki/spaces/TEAM/pages/98765",
			want:       "TEAM",
		},
		{
			name:       "space key with numbers",
			displayURL: "/spaces/PROJECT123/pages/456",
			want:       "PROJECT123",
		},
		{
			name:       "empty URL",
			displayURL: "",
			want:       "",
		},
		{
			name:       "no spaces in URL",
			displayURL: "/pages/12345",
			want:       "",
		},
		{
			name:       "blogpost URL",
			displayURL: "/spaces/BLOG/blog/2024/01/post-title",
			want:       "BLOG",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSpaceKey(tt.displayURL)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRunSearch_DisplaysSpaceKey(t *testing.T) {
	server := mockSearchServer(t, `{
		"results": [
			{
				"content": {"id": "12345", "type": "page", "status": "current", "title": "Test Page"},
				"resultGlobalContainer": {"title": "Development", "displayUrl": "/spaces/DEV/pages/12345"}
			}
		],
		"start": 0,
		"size": 1,
		"totalSize": 1
	}`)
	defer server.Close()

	stdout := &bytes.Buffer{}
	rootOpts := newTestRootOptions()
	rootOpts.Stdout = stdout
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &searchOptions{
		Options: rootOpts,
		query:   "test",
		limit:   25,
	}

	err := runSearch(opts)
	require.NoError(t, err)
	// The output should contain the space key "DEV" extracted from displayUrl
}
