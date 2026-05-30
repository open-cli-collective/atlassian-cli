package space

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

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
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "GET", r.Method)
		testutil.Contains(t, r.URL.Path, "/spaces")

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

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunList_EmptyResults(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunList_InvalidOutputFormat(t *testing.T) {
	t.Parallel()
	rootOpts := newTestRootOptions()
	rootOpts.Output = "invalid"

	opts := &listOptions{
		Options: rootOpts,
		limit:   25,
	}

	err := runList(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "invalid output format")
}

func TestRunList_NegativeLimit(t *testing.T) {
	t.Parallel()
	rootOpts := newTestRootOptions()

	opts := &listOptions{
		Options: rootOpts,
		limit:   -1,
	}

	err := runList(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "invalid limit")
}

func TestRunList_ZeroLimit(t *testing.T) {
	t.Parallel()
	rootOpts := newTestRootOptions()

	opts := &listOptions{
		Options: rootOpts,
		limit:   0,
	}

	// Zero limit should return empty without making API call
	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunList_WithTypeFilter(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "global", r.URL.Query().Get("type"))

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

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunList_PreservesRawSpaceTypes(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if key := r.URL.Query().Get("keys"); key != "" {
			spaces := map[string]api.Space{
				"GLOBAL":     {ID: "1", Key: "GLOBAL", Name: "Space With Words", Type: "global", Status: "current"},
				"~123":       {ID: "2", Key: "~123", Name: "Space With Words", Type: "personal", Status: "current"},
				"CONFLUENCE": {ID: "3", Key: "CONFLUENCE", Name: "Space With Words", Type: "collaboration", Status: "current"},
				"Education":  {ID: "4", Key: "Education", Name: "Space With Words", Type: "knowledge_base", Status: "current"},
			}
			_ = json.NewEncoder(w).Encode(api.PaginatedResponse[api.Space]{
				Results: []api.Space{spaces[key]},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(api.PaginatedResponse[api.Space]{
			Results: []api.Space{
				{ID: "1", Key: "GLOBAL", Name: "Global Space", Type: "global", Status: "current"},
				{ID: "2", Key: "~123", Name: "Personal Space", Type: "personal", Status: "current"},
				{ID: "3", Key: "CONFLUENCE", Name: "Confluence CLI", Type: "collaboration", Status: "current"},
				{ID: "4", Key: "Education", Name: "Education Space", Type: "knowledge_base", Status: "current"},
			},
		})
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	rootOpts := newTestRootOptions()
	rootOpts.Stdout = stdout
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		limit:   25,
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
	gotTypes := spaceTypesByKey(stdout.String())
	testutil.Equal(t, "global", gotTypes["GLOBAL"])
	testutil.Equal(t, "personal", gotTypes["~123"])
	testutil.Equal(t, "collaboration", gotTypes["CONFLUENCE"])
	testutil.Equal(t, "knowledge_base", gotTypes["Education"])

	for key, listType := range gotTypes {
		viewStdout := &bytes.Buffer{}
		viewRootOpts := newTestRootOptions()
		viewRootOpts.Stdout = viewStdout
		viewRootOpts.SetAPIClient(client)

		viewOpts := &viewOptions{Options: viewRootOpts}
		err = runView(context.Background(), key, viewOpts)
		testutil.RequireNoError(t, err)
		testutil.Contains(t, viewStdout.String(), "Type: "+listType)
	}
}

func spaceTypesByKey(output string) map[string]string {
	types := make(map[string]string)
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return types
	}
	header := lines[0]
	typeStart := strings.Index(header, "TYPE")
	descStart := strings.Index(header, "DESCRIPTION")
	if typeStart < 0 || descStart < 0 || descStart <= typeStart {
		return types
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" || len(line) <= typeStart {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		end := descStart
		if len(line) < end {
			end = len(line)
		}
		types[fields[0]] = strings.TrimSpace(line[typeStart:end])
	}
	return types
}

func TestRunList_WithLimitParameter(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "50", r.URL.Query().Get("limit"))

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

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunList_APIError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	err := runList(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "listing spaces")
}

func TestRunList_HasMore(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunList_NullDescription(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunList_WithCursor(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "abc123", r.URL.Query().Get("cursor"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [
				{"id": "789012", "key": "DOCS", "name": "Documentation", "type": "global"}
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
		cursor:  "abc123",
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunList_DisplaysNextCursor(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [
				{"id": "123456", "key": "DEV", "name": "Development", "type": "global"}
			],
			"_links": {"next": "/wiki/api/v2/spaces?cursor=nextPageCursor123"}
		}`))
	}))
	defer server.Close()

	stderr := &bytes.Buffer{}
	rootOpts := newTestRootOptions()
	rootOpts.Stderr = stderr
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		limit:   25,
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stderr.String(), "nextPageCursor123")
	testutil.Contains(t, stderr.String(), "--cursor")
}

func TestExtractCursor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		nextLink string
		want     string
	}{
		{
			name:     "valid cursor",
			nextLink: "/wiki/api/v2/spaces?cursor=abc123&limit=25",
			want:     "abc123",
		},
		{
			name:     "empty link",
			nextLink: "",
			want:     "",
		},
		{
			name:     "no cursor param",
			nextLink: "/wiki/api/v2/spaces?limit=25",
			want:     "",
		},
		{
			name:     "full URL",
			nextLink: "https://example.atlassian.net/wiki/api/v2/spaces?cursor=xyz789",
			want:     "xyz789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractCursor(tt.nextLink)
			testutil.Equal(t, tt.want, got)
		})
	}
}
