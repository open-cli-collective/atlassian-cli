package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestResolveFieldOptions_EditMetaHit(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/editmeta") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"fields": map[string]any{
					"priority": map[string]any{
						"allowedValues": []map[string]any{
							{"id": "1", "name": "High"},
							{"id": "2", "name": "Low"},
						},
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, _ := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	opts, err := ResolveFieldOptions(context.Background(), client, "TEST-1", "priority")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(opts), 2)
	testutil.Equal(t, opts[0].Name, "High")
}

func TestResolveFieldOptions_FallsBackToContext(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/editmeta"):
			_ = json.NewEncoder(w).Encode(map[string]any{"fields": map[string]any{}})
		case strings.HasSuffix(r.URL.Path, "/context") && !strings.Contains(r.URL.Path, "/option"):
			_ = json.NewEncoder(w).Encode(FieldContextsResponse{
				Values: []FieldContext{{ID: "100", Name: "Default", IsGlobalContext: true}},
				IsLast: true,
			})
		case strings.Contains(r.URL.Path, "/context/100/option"):
			_ = json.NewEncoder(w).Encode(FieldContextOptionsResponse{
				Values: []FieldContextOption{
					{ID: "10", Value: "Platform"},
					{ID: "20", Value: "Integration"},
				},
				IsLast: true,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, _ := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	opts, err := ResolveFieldOptions(context.Background(), client, "TEST-1", "customfield_10050")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(opts), 2)
	testutil.Equal(t, opts[0].Value, "Platform")
}

func TestResolveFieldOptions_NonAbsentEditMetaError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client, _ := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	_, err := ResolveFieldOptions(context.Background(), client, "TEST-1", "priority")
	testutil.NotNil(t, err)
	testutil.Contains(t, err.Error(), "resolving field options")
}
