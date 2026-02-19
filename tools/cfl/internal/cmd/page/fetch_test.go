package page

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
)

func TestGetPageWithBodyFallback_StorageHasContent(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		testutil.Equal(t, "storage", r.URL.Query().Get("body-format"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "12345",
			"title": "Page",
			"version": {"number": 1},
			"body": {"storage": {"representation": "storage", "value": "<p>Content</p>"}},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	page, err := getPageWithBodyFallback(context.Background(), client, "12345")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 1, callCount)
	testutil.True(t, hasStorageContent(page))
}

func TestGetPageWithBodyFallback_StorageEmpty_FallsBackToADF(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch r.URL.Query().Get("body-format") {
		case "storage":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "ADF Page",
				"version": {"number": 1},
				"body": {"storage": {"representation": "storage", "value": ""}},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "atlas_doc_format":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "ADF Page",
				"version": {"number": 1},
				"body": {"atlas_doc_format": {"representation": "atlas_doc_format", "value": "{\"type\":\"doc\"}"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	page, err := getPageWithBodyFallback(context.Background(), client, "12345")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 2, callCount)
	testutil.True(t, hasADFContent(page))
}

func TestGetPageWithBodyFallback_NullBody_FallsBackToADF(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch r.URL.Query().Get("body-format") {
		case "storage":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Page",
				"version": {"number": 1},
				"body": {},
				"_links": {"webui": "/pages/12345"}
			}`))
		case "atlas_doc_format":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Page",
				"version": {"number": 1},
				"body": {"atlas_doc_format": {"representation": "atlas_doc_format", "value": "{\"type\":\"doc\"}"}},
				"_links": {"webui": "/pages/12345"}
			}`))
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	page, err := getPageWithBodyFallback(context.Background(), client, "12345")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 2, callCount)
	testutil.True(t, hasADFContent(page))
}

func TestGetPageWithBodyFallback_BothEmpty(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "12345",
			"title": "Empty Page",
			"version": {"number": 1},
			"body": {},
			"_links": {"webui": "/pages/12345"}
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	page, err := getPageWithBodyFallback(context.Background(), client, "12345")
	testutil.RequireNoError(t, err)
	testutil.False(t, hasStorageContent(page))
	testutil.False(t, hasADFContent(page))
}

func TestGetPageWithBodyFallback_GetPageError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Page not found"}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	_, err := getPageWithBodyFallback(context.Background(), client, "99999")
	testutil.RequireError(t, err)
}

func TestGetPageWithBodyFallback_ADFFallbackFails_GracefulDegradation(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch r.URL.Query().Get("body-format") {
		case "storage":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "12345",
				"title": "Page",
				"version": {"number": 1},
				"body": {},
				"_links": {"webui": "/pages/12345"}
			}`))
		default:
			// ADF request fails
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message": "Internal error"}`))
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	page, err := getPageWithBodyFallback(context.Background(), client, "12345")
	testutil.RequireNoError(t, err)
	testutil.False(t, hasStorageContent(page))
	testutil.False(t, hasADFContent(page))
}

func TestHasStorageContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		page     *api.Page
		expected bool
	}{
		{"nil body", &api.Page{}, false},
		{"nil storage", &api.Page{Body: &api.Body{}}, false},
		{"empty value", &api.Page{Body: &api.Body{Storage: &api.BodyRepresentation{Value: ""}}}, false},
		{"has content", &api.Page{Body: &api.Body{Storage: &api.BodyRepresentation{Value: "<p>Hi</p>"}}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testutil.Equal(t, tt.expected, hasStorageContent(tt.page))
		})
	}
}

func TestHasADFContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		page     *api.Page
		expected bool
	}{
		{"nil body", &api.Page{}, false},
		{"nil adf", &api.Page{Body: &api.Body{}}, false},
		{"empty value", &api.Page{Body: &api.Body{AtlasDocFormat: &api.BodyRepresentation{Value: ""}}}, false},
		{"has content", &api.Page{Body: &api.Body{AtlasDocFormat: &api.BodyRepresentation{Value: `{"type":"doc"}`}}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testutil.Equal(t, tt.expected, hasADFContent(tt.page))
		})
	}
}

// Ensure the strings import is used (needed for existing test helpers).
var _ = strings.Contains
