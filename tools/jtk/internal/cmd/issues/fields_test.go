package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// These tests are non-parallel: SetRootForTest / SetInstanceKeyForTest are
// process-globals that race with t.Parallel() tests writing them.

func TestRunGlobalFields_CacheHit_SkipsLiveCall(t *testing.T) {
	seedCacheForIssues(t)
	testutil.RequireNoError(t, cache.WriteResource("fields", "24h", []api.Field{
		{ID: "summary", Name: "Summary", Schema: api.FieldSchema{Type: "string"}},
		{ID: "customfield_10016", Name: "Story Points", Custom: true, Schema: api.FieldSchema{Type: "number"}},
	}))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			t.Fatal("live /field must not be called when cache is fresh")
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGlobalFields(context.Background(), opts, client, false)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Summary")
	testutil.Contains(t, stdout.String(), "Story Points")
}

func TestRunGlobalFields_CacheHit_CustomFilter_SkipsLiveCall(t *testing.T) {
	seedCacheForIssues(t)
	testutil.RequireNoError(t, cache.WriteResource("fields", "24h", []api.Field{
		{ID: "summary", Name: "Summary", Custom: false, Schema: api.FieldSchema{Type: "string"}},
		{ID: "customfield_10016", Name: "Story Points", Custom: true, Schema: api.FieldSchema{Type: "number"}},
	}))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			t.Fatal("live /field must not be called when cache is fresh")
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGlobalFields(context.Background(), opts, client, true)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Story Points")
	testutil.NotContains(t, stdout.String(), "Summary")
}

func TestRunIssueFields_ShowsFieldValues(t *testing.T) {
	seedCacheForIssues(t)
	testutil.RequireNoError(t, cache.WriteResource("fields", "24h", []api.Field{
		{ID: "summary", Name: "Summary", Schema: api.FieldSchema{Type: "string"}},
		{ID: "status", Name: "Status", Schema: api.FieldSchema{Type: "status"}},
		{ID: "customfield_10035", Name: "Story Points", Custom: true, Schema: api.FieldSchema{Type: "number"}},
	}))

	issue := api.Issue{
		Key: "TEST-1",
		Fields: api.IssueFields{
			Summary: "Test issue",
			Status:  &api.Status{Name: "Open"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			t.Fatal("live /field must not be called when cache is fresh")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(issue)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runIssueFields(context.Background(), opts, client, "TEST-1", false)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Summary")
	testutil.Contains(t, stdout.String(), "Test issue")
	testutil.Contains(t, stdout.String(), "VALUE")
}

func TestRunIssueFields_CustomOnly(t *testing.T) {
	seedCacheForIssues(t)
	testutil.RequireNoError(t, cache.WriteResource("fields", "24h", []api.Field{
		{ID: "summary", Name: "Summary", Schema: api.FieldSchema{Type: "string"}},
		{ID: "customfield_10035", Name: "Story Points", Custom: true, Schema: api.FieldSchema{Type: "number"}},
	}))

	issueJSON := `{"key":"TEST-1","fields":{"summary":"Test","customfield_10035":5}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			t.Fatal("live /field must not be called when cache is fresh")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(issueJSON))
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runIssueFields(context.Background(), opts, client, "TEST-1", true)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Story Points")
	testutil.NotContains(t, stdout.String(), "summary")
}

func TestRunGlobalFields_CacheMiss_FallsBackToLive(t *testing.T) {
	seedCacheForIssues(t)
	// No fields cache written → cache miss → live call expected.

	liveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			liveCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"summary","name":"Summary"}]`))
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

	err = runGlobalFields(context.Background(), opts, client, false)
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live /field call on cache miss")
	}
	testutil.Contains(t, stdout.String(), "Summary")
}
