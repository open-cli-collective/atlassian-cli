package page

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

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
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Contains(t, r.URL.Path, "/pages/12345")
		testutil.Equal(t, "GET", r.Method)
		testutil.Equal(t, "storage", r.URL.Query().Get("body-format"))

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

	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)

	stdout := rootOpts.Stdout.(*bytes.Buffer)
	testutil.Contains(t, stdout.String(), "Hello")
	testutil.Contains(t, stdout.String(), "World")
}

func TestRunView_RawFormat(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)

	stdout := rootOpts.Stdout.(*bytes.Buffer)
	testutil.Contains(t, stdout.String(), "<p>Raw HTML Content</p>")
}

func TestRunView_JSONOutput(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
}

func TestRunView_PageNotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	err := runView(context.Background(), "99999", opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "getting page")
}

func TestRunView_EmptyContent(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
}

func TestRunView_InvalidOutputFormat(t *testing.T) {
	t.Parallel()
	rootOpts := newViewTestRootOptions()
	rootOpts.Output = "invalid"

	opts := &viewOptions{
		Options: rootOpts,
	}

	err := runView(context.Background(), "12345", opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "invalid output format")
}

func TestRunView_ShowMacros(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
}

func TestRunView_ContentOnly(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
	// Output should only contain markdown content, no Title:/ID:/Version: headers
}

func TestRunView_ContentOnly_Raw(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
	// Output should only contain raw XHTML, no Title:/ID:/Version: headers
}

func TestRunView_ContentOnly_ShowMacros(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
	// Output should contain markdown with [TOC] macro placeholder
}

func TestRunView_ContentOnly_JSON_Error(t *testing.T) {
	t.Parallel()
	rootOpts := newViewTestRootOptions()
	rootOpts.Output = "json"

	opts := &viewOptions{
		Options:     rootOpts,
		contentOnly: true,
	}

	err := runView(context.Background(), "12345", opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "--content-only is incompatible with --output json")
}

func TestRunView_ContentOnly_Web_Error(t *testing.T) {
	t.Parallel()
	rootOpts := newViewTestRootOptions()

	opts := &viewOptions{
		Options:     rootOpts,
		contentOnly: true,
		web:         true,
	}

	err := runView(context.Background(), "12345", opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "--content-only is incompatible with --web")
}

func TestRunView_ContentOnly_EmptyBody(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
	// Output should be "(No content)" without metadata headers
}

func TestRunView_WithSpaceKey(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call: GetPage
			testutil.Contains(t, r.URL.Path, "/pages/12345")
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
			testutil.Contains(t, r.URL.Path, "/spaces/98765")
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

	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 2, callCount)
}

func TestRunView_SpaceLookupFails_Graceful(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
}

func TestEnrichPageWithSpaceKey(t *testing.T) {
	t.Parallel()
	page := &api.Page{
		ID:      "12345",
		Title:   "Test Page",
		SpaceID: "98765",
	}

	enriched := enrichPageWithSpaceKey(page, "DEV")

	testutil.Equal(t, "12345", enriched.ID)
	testutil.Equal(t, "Test Page", enriched.Title)
	testutil.Equal(t, "DEV", enriched.SpaceKey)
}

func TestTruncateContent(t *testing.T) {
	t.Parallel()
	t.Run("short content is not truncated", func(t *testing.T) {
		t.Parallel()
		opts := &viewOptions{}
		result := truncateContent("short", opts)
		testutil.Equal(t, "short", result)
	})

	t.Run("long content is truncated by default", func(t *testing.T) {
		t.Parallel()
		opts := &viewOptions{}
		long := strings.Repeat("x", maxViewChars+100)
		result := truncateContent(long, opts)
		testutil.Len(t, strings.SplitN(result, "\n\n... [truncated", 2)[0], maxViewChars)
		testutil.Contains(t, result, fmt.Sprintf("... [truncated at %d chars, use --full for complete text]", maxViewChars))
	})

	t.Run("--full bypasses truncation", func(t *testing.T) {
		t.Parallel()
		opts := &viewOptions{full: true}
		long := strings.Repeat("x", maxViewChars+100)
		result := truncateContent(long, opts)
		testutil.Equal(t, long, result)
	})

	t.Run("--content-only implies full", func(t *testing.T) {
		t.Parallel()
		opts := &viewOptions{contentOnly: true}
		long := strings.Repeat("x", maxViewChars+100)
		result := truncateContent(long, opts)
		testutil.Equal(t, long, result)
	})

	t.Run("content at exact limit is not truncated", func(t *testing.T) {
		t.Parallel()
		opts := &viewOptions{}
		exact := strings.Repeat("x", maxViewChars)
		result := truncateContent(exact, opts)
		testutil.Equal(t, exact, result)
	})
}

func TestRunView_ADFPage_FallbackToAtlasDocFormat(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if strings.Contains(r.URL.Path, "/pages/12345") && r.Method == "GET" {
			switch r.URL.Query().Get("body-format") {
			case "storage":
				// Storage returns empty body for this ADF page
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "12345",
					"title": "ADF Page",
					"spaceId": "98765",
					"version": {"number": 3},
					"body": {"storage": {"representation": "storage", "value": ""}},
					"_links": {"webui": "/pages/12345"}
				}`))
			case "atlas_doc_format":
				// ADF format returns content
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "12345",
					"title": "ADF Page",
					"spaceId": "98765",
					"version": {"number": 3},
					"body": {"atlas_doc_format": {"representation": "atlas_doc_format", "value": "{\"type\":\"doc\",\"version\":1,\"content\":[{\"type\":\"paragraph\",\"content\":[{\"type\":\"text\",\"text\":\"Hello ADF\"}]}]}"}},
					"_links": {"webui": "/pages/12345"}
				}`))
			}
			return
		}
		if strings.Contains(r.URL.Path, "/spaces/") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": "98765", "key": "TEST"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	opts := &viewOptions{Options: rootOpts}

	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)

	// Should make 3 calls: storage (empty), atlas_doc_format (has content), GetSpace
	testutil.Equal(t, 3, callCount)

	stdout := rootOpts.Stdout.(*bytes.Buffer)
	testutil.Contains(t, stdout.String(), "Hello ADF")
}

func TestRunView_ADFPage_RawFormat(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pages/12345") {
			switch r.URL.Query().Get("body-format") {
			case "storage":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "12345",
					"title": "ADF Page",
					"spaceId": "98765",
					"version": {"number": 1},
					"body": {"storage": {"representation": "storage", "value": ""}},
					"_links": {"webui": "/pages/12345"}
				}`))
			case "atlas_doc_format":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "12345",
					"title": "ADF Page",
					"spaceId": "98765",
					"version": {"number": 1},
					"body": {"atlas_doc_format": {"representation": "atlas_doc_format", "value": "{\"type\":\"doc\",\"version\":1,\"content\":[{\"type\":\"paragraph\",\"content\":[{\"type\":\"text\",\"text\":\"Raw ADF\"}]}]}"}},
					"_links": {"webui": "/pages/12345"}
				}`))
			}
			return
		}
		if strings.Contains(r.URL.Path, "/spaces/") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": "98765", "key": "TEST"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	opts := &viewOptions{Options: rootOpts, raw: true}

	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)

	stdout := rootOpts.Stdout.(*bytes.Buffer)
	testutil.Contains(t, stdout.String(), "Raw ADF")
	testutil.Contains(t, stdout.String(), `"type"`)
}

func TestRunView_StoragePage_NoFallback(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if strings.Contains(r.URL.Path, "/pages/12345") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Storage Page",
				"spaceId": "98765",
				"version": {"number": 1},
				"body": {"storage": {"representation": "storage", "value": "<p>Has content</p>"}},
				"_links": {"webui": "/pages/12345"}
			}`))
			return
		}
		if strings.Contains(r.URL.Path, "/spaces/") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": "98765", "key": "DEV"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	opts := &viewOptions{Options: rootOpts}

	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)

	// Should only make 2 calls: GetPage (storage has content) + GetSpace, no fallback
	testutil.Equal(t, 2, callCount)

	stdout := rootOpts.Stdout.(*bytes.Buffer)
	testutil.Contains(t, stdout.String(), "Has content")
}

func TestRunView_ADFPage_NullBody(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pages/12345") {
			switch r.URL.Query().Get("body-format") {
			case "storage":
				// Body field is completely empty (no storage, no ADF)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "12345",
					"title": "Empty Page",
					"version": {"number": 1},
					"body": {},
					"_links": {"webui": "/pages/12345"}
				}`))
			case "atlas_doc_format":
				// ADF also returns empty
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "12345",
					"title": "Empty Page",
					"version": {"number": 1},
					"body": {},
					"_links": {"webui": "/pages/12345"}
				}`))
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	opts := &viewOptions{Options: rootOpts}

	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)

	stdout := rootOpts.Stdout.(*bytes.Buffer)
	testutil.Contains(t, stdout.String(), "(No content)")
}
