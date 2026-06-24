package page

import (
	"bytes"
	"context"
	"encoding/json"
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
	testutil.NotContains(t, err.Error(), "getting page: getting page:")
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

func TestRunView_VersionContentOnly(t *testing.T) {
	t.Parallel()
	server := mockVersionedViewServer(t, "<p>Historical <strong>Content</strong></p>")
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	rootOpts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))

	opts := &viewOptions{
		Options:     rootOpts,
		version:     2,
		contentOnly: true,
	}

	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)

	stdout := rootOpts.Stdout.(*bytes.Buffer).String()
	testutil.Contains(t, stdout, "Historical")
	testutil.Contains(t, stdout, "Content")
	testutil.False(t, strings.Contains(stdout, "Title:"), "content-only output should omit metadata")
	testutil.False(t, strings.Contains(stdout, "ID:"), "content-only output should omit metadata")
	testutil.False(t, strings.Contains(stdout, "Version:"), "content-only output should omit metadata")
}

func TestRunView_VersionRaw(t *testing.T) {
	t.Parallel()
	server := mockVersionedViewServer(t, "<p>Historical Raw</p>")
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	rootOpts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))

	opts := &viewOptions{
		Options: rootOpts,
		version: 2,
		raw:     true,
	}

	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, rootOpts.Stdout.(*bytes.Buffer).String(), "<p>Historical Raw</p>")
}

func TestRunView_VersionNewerThanCurrent_PreservesVersionContext(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v2/pages/12345":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Versioned Page",
				"version": {"number": 3},
				"_links": {"webui": "/pages/12345"}
			}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	rootOpts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))

	opts := &viewOptions{
		Options: rootOpts,
		version: 10,
	}

	err := runView(context.Background(), "12345", opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "page version 10 is newer than current version 3")
	testutil.NotContains(t, err.Error(), "getting page: page version")
}

func TestRunView_VersionTruncatesByDefault(t *testing.T) {
	t.Parallel()
	server := mockVersionedViewServer(t, "<p>"+strings.Repeat("a", maxViewChars+10)+"</p>")
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	rootOpts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))

	opts := &viewOptions{
		Options: rootOpts,
		version: 2,
		raw:     true,
	}

	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, rootOpts.Stdout.(*bytes.Buffer).String(), "truncated at")
}

func TestRunView_VersionNoTruncate(t *testing.T) {
	t.Parallel()
	server := mockVersionedViewServer(t, "<p>"+strings.Repeat("a", maxViewChars+10)+"</p>")
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	rootOpts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))

	opts := &viewOptions{
		Options:    rootOpts,
		version:    2,
		raw:        true,
		noTruncate: true,
	}

	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
	stdout := rootOpts.Stdout.(*bytes.Buffer).String()
	testutil.False(t, strings.Contains(stdout, "truncated at"), "no-truncate should show full historical content")
	testutil.Contains(t, stdout, strings.Repeat("a", maxViewChars+10))
}

func TestRunView_VersionShowMacros(t *testing.T) {
	t.Parallel()
	storage := `<ac:structured-macro ac:name="toc"><ac:parameter ac:name="maxLevel">2</ac:parameter></ac:structured-macro><p>Content</p>`
	server := mockVersionedViewServer(t, storage)
	defer server.Close()

	rootOpts := newViewTestRootOptions()
	rootOpts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))

	opts := &viewOptions{
		Options:     rootOpts,
		version:     2,
		contentOnly: true,
		showMacros:  true,
	}

	err := runView(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, rootOpts.Stdout.(*bytes.Buffer).String(), "[TOC")
}

func TestRunView_VersionWebError(t *testing.T) {
	t.Parallel()
	rootOpts := newViewTestRootOptions()
	opts := &viewOptions{
		Options: rootOpts,
		version: 2,
		web:     true,
	}

	err := runView(context.Background(), "12345", opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "--version is incompatible with --web")
}

func TestRunView_NegativeVersionError(t *testing.T) {
	t.Parallel()
	rootOpts := newViewTestRootOptions()
	opts := &viewOptions{
		Options: rootOpts,
		version: -1,
	}

	err := runView(context.Background(), "12345", opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "invalid version")
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
		testutil.Contains(t, result, fmt.Sprintf("... [truncated at %d chars, use --no-truncate for complete text]", maxViewChars))
	})

	t.Run("--full bypasses truncation", func(t *testing.T) {
		t.Parallel()
		opts := &viewOptions{noTruncate: true}
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

func mockVersionedViewServer(t *testing.T, storage string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v2/pages/12345":
			testutil.Empty(t, r.URL.Query().Get("body-format"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Versioned Page",
				"version": {"number": 3}
			}`))
		case r.URL.Path == "/api/v2/pages/12345/versions" && r.URL.Query().Get("body-format") == "" && r.URL.Query().Get("cursor") == "":
			testutil.Equal(t, "1", r.URL.Query().Get("limit"))
			testutil.Equal(t, "-modified-date", r.URL.Query().Get("sort"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"results": [{"number": 3}],
				"_links": {"next": "/api/v2/pages/12345/versions?cursor=cursor-v2"}
			}`))
		case r.URL.Path == "/api/v2/pages/12345/versions" && r.URL.Query().Get("body-format") == "" && r.URL.Query().Get("cursor") == "cursor-v2":
			testutil.Equal(t, "1", r.URL.Query().Get("limit"))
			testutil.Equal(t, "-modified-date", r.URL.Query().Get("sort"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"results": [{"number": 2}]}`))
		case r.URL.Path == "/api/v2/pages/12345/versions" && r.URL.Query().Get("body-format") == "storage":
			testutil.Equal(t, "1", r.URL.Query().Get("limit"))
			testutil.Equal(t, "cursor-v2", r.URL.Query().Get("cursor"))
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]any{{
					"number": 2,
					"page": map[string]any{
						"id":    "12345",
						"title": "Versioned Page",
						"body": map[string]any{
							"storage": map[string]string{
								"representation": "storage",
								"value":          storage,
							},
						},
					},
				}},
			})
		default:
			t.Fatalf("unexpected request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
	}))
}
