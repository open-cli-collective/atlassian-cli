package automation

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newListTestServer(t *testing.T, rules []api.AutomationRuleSummary) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.AutomationRuleSummaryResponse{Data: rules})
	}))
}

func TestRunList_Default_ColumnOrder(t *testing.T) {
	rules := []api.AutomationRuleSummary{
		{UUID: "uuid-1", Name: "Rule A", State: "ENABLED"},
		{UUID: "uuid-2", Name: "Rule B", State: "DISABLED"},
	}

	server := newListTestServer(t, rules)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "ID")
	testutil.Contains(t, out, "STATE")
	testutil.Contains(t, out, "NAME")
	testutil.Contains(t, out, "uuid-1")
	testutil.Contains(t, out, "ENABLED")
	testutil.NotContains(t, out, "UUID")
}

func TestRunList_IDOnly(t *testing.T) {
	rules := []api.AutomationRuleSummary{
		{UUID: "uuid-1", Name: "Rule A", State: "ENABLED"},
		{UUID: "uuid-2", Name: "Rule B", State: "DISABLED"},
	}

	server := newListTestServer(t, rules)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "uuid-1")
	testutil.Contains(t, out, "uuid-2")
	testutil.NotContains(t, out, "Rule A")
	testutil.NotContains(t, out, "STATE")
}

func TestRunList_Extended(t *testing.T) {
	rules := []api.AutomationRuleSummary{
		{
			UUID:            "uuid-1",
			Name:            "Rule A",
			State:           "ENABLED",
			Labels:          []string{"onboarding"},
			Tags:            []string{"auto-create"},
			AuthorAccountID: "acct-1",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
			return
		}
		if r.URL.Path == "/rest/api/3/user" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"accountId":"acct-1","displayName":"Rian Stockbower"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.AutomationRuleSummaryResponse{Data: rules})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, Extended: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "LABELS")
	testutil.Contains(t, out, "TAGS")
	testutil.Contains(t, out, "AUTHOR")
	testutil.Contains(t, out, "Rian Stockbower")
	testutil.Contains(t, out, "onboarding")
	testutil.Contains(t, out, "auto-create")
}

func TestRunList_Extended_AuthorFallback(t *testing.T) {
	rules := []api.AutomationRuleSummary{
		{
			UUID:            "uuid-1",
			Name:            "Rule A",
			State:           "ENABLED",
			AuthorAccountID: "acct-unknown",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
			return
		}
		if r.URL.Path == "/rest/api/3/user" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.AutomationRuleSummaryResponse{Data: rules})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, Extended: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "acct-unknown")
}

func TestRunList_IDOnly_Empty(t *testing.T) {
	server := newListTestServer(t, nil)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "")
	testutil.RequireNoError(t, err)
	if stdout.String() != "" {
		t.Errorf("--id with empty results should emit nothing, got %q", stdout.String())
	}
}

func TestRunList_NumericTimestamps(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"uuid":"uuid-ts","name":"Timestamp Rule","state":"ENABLED","created":1701482354.625000000,"updated":1701568754.000000000}]}`))
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "uuid-ts")
	testutil.Contains(t, out, "ENABLED")
	testutil.Contains(t, out, "Timestamp Rule")
}

func TestRunList_Empty(t *testing.T) {
	server := newListTestServer(t, nil)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "No automation rules found")
}
