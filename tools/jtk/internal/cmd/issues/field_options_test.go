package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// These tests are non-parallel: SetRootForTest / SetInstanceKeyForTest are
// process-globals that race with t.Parallel() tests writing them.

func TestRunFieldOptions_CacheHit_SkipsFieldsFetch(t *testing.T) {
	seedCacheForIssues(t)
	testutil.RequireNoError(t, cache.WriteResource("fields", "24h", []api.Field{
		{ID: "priority", Name: "Priority", Schema: api.FieldSchema{Type: "priority"}},
	}))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			t.Fatal("live /field must not be called when cache is fresh")
		}
		// Serve options for the priority field.
		if strings.Contains(r.URL.Path, "/field/priority/option") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []api.FieldOptionValue{
					{ID: "1", Value: "Highest"},
					{ID: "2", Value: "High"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runFieldOptions(context.Background(), opts, "Priority", "")
	testutil.RequireNoError(t, err)
}

func TestRunFieldOptions_CacheMiss_FallsBackToLive(t *testing.T) {
	seedCacheForIssues(t)
	// No fields cache → miss → live /field call expected.

	fieldsCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			fieldsCalled = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]api.Field{
				{ID: "priority", Name: "Priority", Schema: api.FieldSchema{Type: "priority"}},
			})
			return
		}
		if strings.Contains(r.URL.Path, "/field/priority/option") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []api.FieldOptionValue{
					{ID: "1", Value: "Highest"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runFieldOptions(context.Background(), opts, "Priority", "")
	testutil.RequireNoError(t, err)
	if !fieldsCalled {
		t.Fatal("expected live /field call on cache miss")
	}
}

func TestRunFieldOptions_FallbackToContext(t *testing.T) {
	seedCacheForIssues(t)
	testutil.RequireNoError(t, cache.WriteResource("fields", "24h", []api.Field{
		{ID: "customfield_10050", Name: "Team", Custom: true, Schema: api.FieldSchema{Type: "option"}},
	}))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/rest/api/3/field":
			t.Fatal("live /field must not be called when cache is fresh")
		case strings.Contains(r.URL.Path, "/editmeta"):
			_ = json.NewEncoder(w).Encode(map[string]any{"fields": map[string]any{}})
		case strings.HasSuffix(r.URL.Path, "/context") && !strings.Contains(r.URL.Path, "/option"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{"id": "10100", "name": "Default", "isGlobalContext": true, "isAnyIssueType": true},
				},
				"isLast": true,
			})
		case strings.Contains(r.URL.Path, "/context/10100/option"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{"id": "20001", "value": "Platform", "disabled": false},
					{"id": "20002", "value": "Integration", "disabled": false},
				},
				"isLast": true,
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{})
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runFieldOptions(context.Background(), opts, "Team", "TEST-1")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Platform")
	testutil.Contains(t, stdout.String(), "Integration")
	testutil.Contains(t, stdout.String(), "DISABLED")
}
