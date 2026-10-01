package attachment

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

func newListTestRootOptions() *root.Options {
	return &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
}

func TestRunList_Success(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [
				{"id": "att1", "title": "doc.pdf", "mediaType": "application/pdf", "fileSize": 1024},
				{"id": "att2", "title": "image.png", "mediaType": "image/png", "fileSize": 2048}
			]
		}`))
	}))
	defer server.Close()

	rootOpts := newListTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		pageID:  "12345",
		limit:   25,
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

func TestRunList_PlainFullExact(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [
				{"id": "att1", "title": "doc.pdf", "mediaType": "application/pdf", "fileSize": 1024, "status": "current", "comment": "latest"}
			],
			"_links": {"next": "/api/v2/pages/12345/attachments?cursor=abc"}
		}`))
	}))
	defer server.Close()

	rootOpts := newListTestRootOptions()
	rootOpts.Output = "plain"
	rootOpts.Full = true
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		pageID:  "12345",
		limit:   25,
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "ID\tTITLE\tMEDIA TYPE\tFILE SIZE\tSTATUS\tCOMMENT\natt1\tdoc.pdf\tapplication/pdf\t1.0 KB\tcurrent\tlatest\n", rootOpts.Stdout.(*bytes.Buffer).String())
	testutil.Equal(t, "(showing first 1 results, use --limit to see more)\n", rootOpts.Stderr.(*bytes.Buffer).String())
}

func TestRunList_TableOutputExact(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [
				{"id": "att1", "title": "doc.pdf", "mediaType": "application/pdf", "fileSize": 1024}
			]
		}`))
	}))
	defer server.Close()

	rootOpts := newListTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		pageID:  "12345",
		limit:   25,
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "ID    TITLE    MEDIA TYPE       FILE SIZE\natt1  doc.pdf  application/pdf  1.0 KB\n", rootOpts.Stdout.(*bytes.Buffer).String())
	testutil.Equal(t, "", rootOpts.Stderr.(*bytes.Buffer).String())
}

func TestRunList_Empty(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	rootOpts := newListTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		pageID:  "12345",
		limit:   25,
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
	// The empty-results banner always prints to stderr.
	testutil.Contains(t, rootOpts.Stderr.(*bytes.Buffer).String(), "No attachments found.")
}

func TestRunList_APIError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Page not found"}`))
	}))
	defer server.Close()

	rootOpts := newListTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		pageID:  "99999",
		limit:   25,
	}

	err := runList(context.Background(), opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "listing attachments")
}

func TestRunList_Limits(t *testing.T) {
	for _, tt := range []struct {
		name        string
		limit       int
		wantRequest bool
	}{
		{name: "zero", limit: 0},
		{name: "negative", limit: -1},
		{name: "default", limit: 25, wantRequest: true},
		{name: "positive", limit: 50, wantRequest: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				testutil.Equal(t, fmt.Sprint(tt.limit), r.URL.Query().Get("limit"))
				_, _ = w.Write([]byte(`{"results":[]}`))
			}))
			defer server.Close()
			rootOpts := newListTestRootOptions()
			rootOpts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))

			err := runList(context.Background(), &listOptions{Options: rootOpts, pageID: "12345", limit: tt.limit})
			if tt.wantRequest {
				testutil.RequireNoError(t, err)
				testutil.Equal(t, 1, requests)
				return
			}
			testutil.RequireError(t, err)
			testutil.Contains(t, err.Error(), "must be greater than 0")
			testutil.Equal(t, 0, requests)
		})
	}
}

func TestList_InvalidLimitFailsBeforeConfig(t *testing.T) {
	rootCmd, rootOpts := root.NewCmd()
	rootOpts.ConfigPath = filepath.Join(t.TempDir(), "missing.yml")
	Register(rootCmd, rootOpts)
	rootCmd.SetArgs([]string{"attachment", "list", "--page", "12345", "--limit", "0"})

	err := rootCmd.Execute()
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "must be greater than 0")
	testutil.NotContains(t, err.Error(), "config")
}

func TestRunList_UnusedFlag(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		switch r.URL.Path {
		case "/api/v2/pages/12345/attachments":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"results": [
					{"id": "att1", "title": "used.png", "mediaType": "image/png", "fileSize": 1024},
					{"id": "att2", "title": "unused.pdf", "mediaType": "application/pdf", "fileSize": 2048}
				]
			}`))
		case "/api/v2/pages/12345":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"body": {
					"storage": {
						"representation": "storage",
						"value": "<ac:image><ri:attachment ri:filename=\"used.png\"/></ac:image>"
					}
				}
			}`))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	rootOpts := newListTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		pageID:  "12345",
		limit:   25,
		unused:  true,
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 2, requestCount) // Both attachments and page content fetched
}

func TestRunList_UnusedFlag_NoUnused(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/pages/12345/attachments":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"results": [
					{"id": "att1", "title": "used.png", "mediaType": "image/png", "fileSize": 1024}
				]
			}`))
		case "/api/v2/pages/12345":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Test Page",
				"body": {
					"storage": {
						"representation": "storage",
						"value": "<ac:image><ri:attachment ri:filename=\"used.png\"/></ac:image>"
					}
				}
			}`))
		}
	}))
	defer server.Close()

	rootOpts := newListTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &listOptions{
		Options: rootOpts,
		pageID:  "12345",
		limit:   25,
		unused:  true,
	}

	err := runList(context.Background(), opts)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, rootOpts.Stderr.(*bytes.Buffer).String(), "No unused attachments found.")
}
