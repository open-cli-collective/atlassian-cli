package page

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

// mockCopyServer creates a test server that handles page get and copy operations
func mockCopyServer(t *testing.T, getHandler, copyHandler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/api/v2/pages/") {
			if getHandler != nil {
				getHandler(w, r)
				return
			}
			// Default: return a valid page
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": "12345", "title": "Original Page", "spaceId": "SRCSPACE", "version": {"number": 1}}`))
			return
		}
		if r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/rest/api/content/") && strings.HasSuffix(r.URL.Path, "/copy") {
			if copyHandler != nil {
				copyHandler(w, r)
				return
			}
			// Default: return a successful copy
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "99999",
				"title": "Copied Page",
				"space": {"key": "TEST"},
				"version": {"number": 1},
				"_links": {"webui": "/pages/99999"}
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
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

func TestRunCopy_Success(t *testing.T) {
	server := mockCopyServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/rest/api/content/12345/copy", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "99999",
			"title": "Copied Page",
			"space": {"key": "TEST"},
			"version": {"number": 1},
			"_links": {"webui": "/pages/99999"}
		}`))
	})
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "user@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &copyOptions{
		Options: rootOpts,
		title:   "Copied Page",
		space:   "TEST",
	}

	err := runCopy("12345", opts)
	require.NoError(t, err)
}

func TestRunCopy_InfersSourceSpace(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v2/pages/12345":
			// GetPage to infer space
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Original",
				"spaceId": "123456",
				"version": {"number": 1}
			}`))
		case r.Method == "GET" && r.URL.Path == "/api/v2/spaces/123456":
			// GetSpace to get space key from numeric ID
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "123456",
				"key": "SRCSPACE",
				"name": "Source Space",
				"type": "global"
			}`))
		case r.Method == "POST" && r.URL.Path == "/rest/api/content/12345/copy":
			// Copy request
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "99999",
				"title": "Copied Page",
				"space": {"key": "SRCSPACE"},
				"version": {"number": 1},
				"_links": {}
			}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "user@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &copyOptions{
		Options: rootOpts,
		title:   "Copied Page",
		space:   "", // Not specified - should infer from source
	}

	err := runCopy("12345", opts)
	require.NoError(t, err)
	assert.Equal(t, 3, callCount) // GetPage + GetSpace + CopyPage
}

func TestRunCopy_PageNotFound(t *testing.T) {
	server := mockCopyServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Page not found"}`))
	})
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "user@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &copyOptions{
		Options: rootOpts,
		title:   "Copied Page",
		space:   "TEST",
	}

	err := runCopy("99999", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to copy page")
}

func TestRunCopy_JSONOutput(t *testing.T) {
	server := mockCopyServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "99999",
			"title": "Copied Page",
			"space": {"key": "TEST"},
			"version": {"number": 1},
			"_links": {}
		}`))
	})
	defer server.Close()

	rootOpts := newTestRootOptions()
	rootOpts.Output = "json"
	client := api.NewClient(server.URL, "user@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &copyOptions{
		Options: rootOpts,
		title:   "Copied Page",
		space:   "TEST",
	}

	err := runCopy("12345", opts)
	require.NoError(t, err)
}

func TestRunCopy_InvalidOutputFormat(t *testing.T) {
	rootOpts := newTestRootOptions()
	rootOpts.Output = "invalid"
	client := api.NewClient("http://unused", "user@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &copyOptions{
		Options: rootOpts,
		title:   "Copied Page",
		space:   "TEST",
	}

	err := runCopy("12345", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
}

func TestRunCopy_GetSourcePageFails(t *testing.T) {
	server := mockCopyServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message": "Page not found"}`))
		},
		nil,
	)
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "user@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &copyOptions{
		Options: rootOpts,
		title:   "Copied Page",
		space:   "", // Empty - will try to get source page
	}

	err := runCopy("invalid", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get source page")
}

func TestRunCopy_WithNoAttachments(t *testing.T) {
	server := mockCopyServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "99999",
			"title": "Copied Page",
			"space": {"key": "TEST"},
			"version": {"number": 1},
			"_links": {}
		}`))
	})
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "user@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &copyOptions{
		Options:       rootOpts,
		title:         "Copied Page",
		space:         "TEST",
		noAttachments: true,
	}

	err := runCopy("12345", opts)
	require.NoError(t, err)
}

func TestRunCopy_WithNoLabels(t *testing.T) {
	server := mockCopyServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "99999",
			"title": "Copied Page",
			"space": {"key": "TEST"},
			"version": {"number": 1},
			"_links": {}
		}`))
	})
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "user@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &copyOptions{
		Options:  rootOpts,
		title:    "Copied Page",
		space:    "TEST",
		noLabels: true,
	}

	err := runCopy("12345", opts)
	require.NoError(t, err)
}

func TestRunCopy_PermissionDenied(t *testing.T) {
	server := mockCopyServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message": "You do not have permission to copy this page"}`))
	})
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "user@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &copyOptions{
		Options: rootOpts,
		title:   "Copied Page",
		space:   "TEST",
	}

	err := runCopy("12345", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to copy page")
}

func TestRunCopy_GetSpaceFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/api/v2/pages/"):
			// GetPage succeeds
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Original",
				"spaceId": "999999",
				"version": {"number": 1}
			}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/api/v2/spaces/"):
			// GetSpace fails
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message": "Space not found"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "user@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &copyOptions{
		Options: rootOpts,
		title:   "Copied Page",
		space:   "", // Empty - will try to get space
	}

	err := runCopy("12345", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get space")
}
