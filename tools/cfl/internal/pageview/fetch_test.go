package pageview

import (
	"context"
	"net/http"
	"net/http/httptest"
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
	page, err := GetPageWithBodyFallback(context.Background(), client, "12345")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 1, callCount)
	testutil.True(t, HasStorageContent(page))
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
	page, err := GetPageWithBodyFallback(context.Background(), client, "12345")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 2, callCount)
	testutil.True(t, HasADFContent(page))
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
	page, err := GetPageWithBodyFallback(context.Background(), client, "12345")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 2, callCount)
	testutil.True(t, HasADFContent(page))
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
	page, err := GetPageWithBodyFallback(context.Background(), client, "12345")
	testutil.RequireNoError(t, err)
	testutil.False(t, HasStorageContent(page))
	testutil.False(t, HasADFContent(page))
}

func TestGetPageWithBodyFallback_GetPageError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Page not found"}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	_, err := GetPageWithBodyFallback(context.Background(), client, "99999")
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
	page, err := GetPageWithBodyFallback(context.Background(), client, "12345")
	testutil.RequireNoError(t, err)
	testutil.False(t, HasStorageContent(page))
	testutil.False(t, HasADFContent(page))
}
