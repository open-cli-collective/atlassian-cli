package page

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/pageview"
)

func TestGetPageWithBodyFormat_UnavailableRepresentation(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "atlas_doc_format", r.URL.Query().Get("body-format"))
		_, _ = w.Write([]byte(`{"id":"12345","body":{}}`))
	}))
	defer server.Close()

	_, err := getPageWithBodyFormat(context.Background(), api.NewClient(server.URL, "test@example.com", "token"), "12345", bodyFormatADF)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "does not provide the requested adf")
}

func TestGetPageVersionWithBodyFormat_ADF(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v2/pages/12345":
			_, _ = w.Write([]byte(`{"id":"12345","title":"Page","version":{"number":2}}`))
		case strings.Contains(r.URL.Path, "/versions") && r.URL.Query().Get("body-format") == "":
			_, _ = w.Write([]byte(`{"results":[{"number":2}]}`))
		case strings.Contains(r.URL.Path, "/versions"):
			testutil.Equal(t, "atlas_doc_format", r.URL.Query().Get("body-format"))
			_, _ = w.Write([]byte(`{"results":[{"number":2,"page":{"id":"12345","body":{"atlas_doc_format":{"representation":"atlas_doc_format","value":"{\"type\":\"doc\",\"version\":1,\"content\":[]}"}}}}]}`))
		default:
			t.Fatalf("unexpected request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
	}))
	defer server.Close()

	page, err := getPageVersionWithBodyFormat(context.Background(), api.NewClient(server.URL, "test@example.com", "token"), "12345", 2, bodyFormatADF)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, `{"type":"doc","version":1,"content":[]}`, page.Body.AtlasDocFormat.Value)
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
			testutil.Equal(t, tt.expected, pageview.HasStorageContent(tt.page))
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
			testutil.Equal(t, tt.expected, pageview.HasADFContent(tt.page))
		})
	}
}

// Ensure the strings import is used (needed for existing test helpers).
var _ = strings.Contains
