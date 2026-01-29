package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Search_Success(t *testing.T) {
	testData := loadTestData(t, "search.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/search", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Query().Get("cql"), `text ~ "test"`)
		assert.Equal(t, "highlight", r.URL.Query().Get("excerpt"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	result, err := client.Search(context.Background(), &SearchOptions{
		Text: "test",
	})

	require.NoError(t, err)
	assert.Len(t, result.Results, 2)
	assert.Equal(t, 50, result.TotalSize)
	assert.True(t, result.HasMore())

	// Check first result
	first := result.Results[0]
	assert.Equal(t, "12345", first.Content.ID)
	assert.Equal(t, "page", first.Content.Type)
	assert.Equal(t, "Getting Started Guide", first.Content.Title)
	assert.Equal(t, "Development", first.ResultGlobalContainer.Title)
}

func TestClient_Search_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [],
			"start": 0,
			"limit": 25,
			"size": 0,
			"totalSize": 0,
			"searchDuration": 5
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	result, err := client.Search(context.Background(), &SearchOptions{
		Text: "nonexistent",
	})

	require.NoError(t, err)
	assert.Len(t, result.Results, 0)
	assert.False(t, result.HasMore())
}

func TestClient_Search_WithAllOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cql := r.URL.Query().Get("cql")
		assert.Contains(t, cql, `text ~ "search term"`)
		assert.Contains(t, cql, `space = "DEV"`)
		assert.Contains(t, cql, `type = "page"`)
		assert.Contains(t, cql, `title ~ "guide"`)
		assert.Contains(t, cql, `label = "documentation"`)
		assert.Equal(t, "50", r.URL.Query().Get("limit"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": [], "totalSize": 0}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	_, err := client.Search(context.Background(), &SearchOptions{
		Text:  "search term",
		Space: "DEV",
		Type:  "page",
		Title: "guide",
		Label: "documentation",
		Limit: 50,
	})
	require.NoError(t, err)
}

func TestClient_Search_RawCQL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Raw CQL should be used as-is
		cql := r.URL.Query().Get("cql")
		assert.Equal(t, `type=page AND lastModified > now("-7d")`, cql)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": [], "totalSize": 0}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	_, err := client.Search(context.Background(), &SearchOptions{
		CQL:  `type=page AND lastModified > now("-7d")`,
		Text: "ignored", // Should be ignored when CQL is set
	})
	require.NoError(t, err)
}

func TestClient_Search_NoQuery(t *testing.T) {
	client := NewClient("http://unused", "user@example.com", "token")

	_, err := client.Search(context.Background(), &SearchOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search requires a query or filters")
}

func TestClient_Search_NilOptions(t *testing.T) {
	client := NewClient("http://unused", "user@example.com", "token")

	_, err := client.Search(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search requires a query or filters")
}

func TestClient_Search_APIError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		errContain string
	}{
		{
			name:       "400 bad request",
			statusCode: 400,
			response:   `{"message": "Invalid CQL query"}`,
			errContain: "Invalid CQL query",
		},
		{
			name:       "401 unauthorized",
			statusCode: 401,
			response:   `{"message": "Authentication required"}`,
			errContain: "Authentication required",
		},
		{
			name:       "403 forbidden",
			statusCode: 403,
			response:   `{"message": "Access denied"}`,
			errContain: "Access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient(server.URL, "user@example.com", "token")
			_, err := client.Search(context.Background(), &SearchOptions{Text: "test"})

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContain)
		})
	}
}

func TestClient_Search_MalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	_, err := client.Search(context.Background(), &SearchOptions{Text: "test"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse search response")
}

func TestClient_Search_Pagination(t *testing.T) {
	tests := []struct {
		name     string
		start    int
		size     int
		total    int
		expected bool
	}{
		{"has more results", 0, 25, 100, true},
		{"last page", 75, 25, 100, false},
		{"exact fit", 0, 100, 100, false},
		{"no results", 0, 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &SearchResponse{
				Start:     tt.start,
				Size:      tt.size,
				TotalSize: tt.total,
			}
			assert.Equal(t, tt.expected, resp.HasMore())
		})
	}
}

func TestBuildCQL_TextOnly(t *testing.T) {
	opts := &SearchOptions{Text: "hello world"}
	cql := buildCQL(opts)
	assert.Equal(t, `text ~ "hello world"`, cql)
}

func TestBuildCQL_SpaceFilter(t *testing.T) {
	opts := &SearchOptions{Space: "DEV"}
	cql := buildCQL(opts)
	assert.Equal(t, `space = "DEV"`, cql)
}

func TestBuildCQL_TypeFilter(t *testing.T) {
	opts := &SearchOptions{Type: "page"}
	cql := buildCQL(opts)
	assert.Equal(t, `type = "page"`, cql)
}

func TestBuildCQL_TitleFilter(t *testing.T) {
	opts := &SearchOptions{Title: "Getting Started"}
	cql := buildCQL(opts)
	assert.Equal(t, `title ~ "Getting Started"`, cql)
}

func TestBuildCQL_LabelFilter(t *testing.T) {
	opts := &SearchOptions{Label: "documentation"}
	cql := buildCQL(opts)
	assert.Equal(t, `label = "documentation"`, cql)
}

func TestBuildCQL_Combined(t *testing.T) {
	opts := &SearchOptions{
		Text:  "api",
		Space: "DEV",
		Type:  "page",
	}
	cql := buildCQL(opts)
	assert.Contains(t, cql, `text ~ "api"`)
	assert.Contains(t, cql, `space = "DEV"`)
	assert.Contains(t, cql, `type = "page"`)
	assert.Contains(t, cql, " AND ")
}

func TestBuildCQL_Empty(t *testing.T) {
	opts := &SearchOptions{}
	cql := buildCQL(opts)
	assert.Empty(t, cql)
}

func TestBuildCQL_QuotesInValue(t *testing.T) {
	opts := &SearchOptions{Text: `search "quoted" term`}
	cql := buildCQL(opts)
	// Go's %q escapes quotes properly
	assert.Contains(t, cql, `text ~ "search \"quoted\" term"`)
}
