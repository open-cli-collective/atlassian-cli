package automation

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/atime"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

func TestSummarizeComponents(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		components []api.RuleComponent
		want       string
	}{
		{
			name:       "empty",
			components: nil,
			want:       "none",
		},
		{
			name: "trigger only",
			components: []api.RuleComponent{
				{Component: "TRIGGER", Type: "jira.issue.create"},
			},
			want: "1 total — 1 trigger",
		},
		{
			name: "all types",
			components: []api.RuleComponent{
				{Component: "TRIGGER", Type: "jira.issue.create"},
				{Component: "CONDITION", Type: "jira.jql.condition"},
				{Component: "ACTION", Type: "jira.issue.assign"},
			},
			want: "3 total — 1 trigger, 1 condition, 1 action",
		},
		{
			name: "multiple actions",
			components: []api.RuleComponent{
				{Component: "TRIGGER", Type: "jira.issue.create"},
				{Component: "ACTION", Type: "jira.issue.assign"},
				{Component: "ACTION", Type: "jira.issue.transition"},
				{Component: "ACTION", Type: "jira.issue.comment"},
			},
			want: "4 total — 1 trigger, 3 actions",
		},
		{
			name: "unknown component types ignored in breakdown",
			components: []api.RuleComponent{
				{Component: "TRIGGER", Type: "jira.issue.create"},
				{Component: "BRANCH", Type: "jira.issue.branch"},
			},
			want: "2 total — 1 trigger",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := jtkpresent.SummarizeComponents(tt.components)
			testutil.Equal(t, got, tt.want)
		})
	}
}

func newGetTestServer(t *testing.T, rule api.AutomationRule) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/_edge/tenant_info":
			_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
		case strings.HasPrefix(r.URL.Path, "/gateway/api/automation/"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(rule)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestRunGet_Default(t *testing.T) {
	rule := api.AutomationRule{
		UUID:        "uuid-123",
		Name:        "My Rule",
		State:       "ENABLED",
		Description: "Does stuff",
		Components: []api.RuleComponent{
			{Component: "TRIGGER", Type: "issue.created"},
			{Component: "ACTION", Type: "assign.issue"},
		},
	}

	server := newGetTestServer(t, rule)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "uuid-123", false)
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "uuid-123  My Rule")
	testutil.Contains(t, out, "State: ENABLED")
	testutil.Contains(t, out, "Components:")
	testutil.Contains(t, out, "Description: Does stuff")
}

func TestRunGet_IDOnly(t *testing.T) {
	rule := api.AutomationRule{UUID: "uuid-123", Name: "My Rule", State: "ENABLED"}

	server := newGetTestServer(t, rule)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "uuid-123", false)
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "uuid-123")
	testutil.NotContains(t, out, "My Rule")
	testutil.NotContains(t, out, "State")
}

func TestRunGet_Extended(t *testing.T) {
	rule := api.AutomationRule{
		UUID:            "uuid-123",
		Name:            "My Rule",
		State:           "ENABLED",
		Description:     "Does stuff",
		Labels:          []string{"onboarding"},
		Tags:            []string{"auto-create"},
		AuthorAccountID: "acct-1",
		Created:         &atime.AtlassianTime{Time: time.Date(2023, 12, 4, 10, 0, 0, 0, time.UTC)},
		Updated:         &atime.AtlassianTime{Time: time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC)},
		Projects:        []api.RuleProject{{ProjectKey: "MON"}, {ProjectKey: "ON"}},
		Components: []api.RuleComponent{
			{Component: "TRIGGER", Type: "issue.created"},
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
		_ = json.NewEncoder(w).Encode(rule)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, Extended: true}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "uuid-123", false)
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "uuid-123  My Rule")
	testutil.Contains(t, out, "Labels: onboarding")
	testutil.Contains(t, out, "Tags: auto-create")
	testutil.Contains(t, out, "Author: Rian Stockbower")
	testutil.Contains(t, out, "Scope: project (MON, ON)")
	testutil.Contains(t, out, "Created: 2023-12-04")
	testutil.Contains(t, out, "Updated: 2026-03-15")
}

func TestRunGet_Extended_AuthorLookupFails(t *testing.T) {
	rule := api.AutomationRule{
		UUID:            "uuid-123",
		Name:            "My Rule",
		State:           "ENABLED",
		AuthorAccountID: "acct-unknown",
		Components: []api.RuleComponent{
			{Component: "TRIGGER", Type: "issue.created"},
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
		_ = json.NewEncoder(w).Encode(rule)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, Extended: true}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "uuid-123", false)
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "Author: acct-unknown")
}

func TestRunGet_Extended_NoAuthor(t *testing.T) {
	rule := api.AutomationRule{
		UUID:  "uuid-123",
		Name:  "My Rule",
		State: "ENABLED",
		Components: []api.RuleComponent{
			{Component: "TRIGGER", Type: "issue.created"},
		},
	}

	server := newGetTestServer(t, rule)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, Extended: true}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "uuid-123", false)
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "Author: -")
}

func TestRunGet_EnvelopeWithNumericTimestamps(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
			return
		}
		if r.URL.Path == "/rest/api/3/user" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"accountId":"acct-1","displayName":"Test Author"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"rule":{"uuid":"uuid-env","name":"Envelope Rule","state":"ENABLED","authorAccountId":"acct-1","created":1701482354.625000000,"updated":1701568754.000000000,"components":[{"component":"TRIGGER","type":"issue.created"},{"component":"ACTION","type":"assign.issue"}]}}`))
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, Extended: true}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "uuid-env", false)
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "uuid-env  Envelope Rule")
	testutil.Contains(t, out, "State: ENABLED")
	testutil.Contains(t, out, "2 total")
	testutil.Contains(t, out, "Created: 2023-12-02")
	testutil.Contains(t, out, "Updated: 2023-12-03")
	testutil.Contains(t, out, "Author: Test Author")
}

func TestRunGet_ShowComponents(t *testing.T) {
	rule := api.AutomationRule{
		UUID:  "uuid-123",
		Name:  "My Rule",
		State: "ENABLED",
		Components: []api.RuleComponent{
			{Component: "TRIGGER", Type: "issue.created"},
			{Component: "ACTION", Type: "assign.issue"},
		},
	}

	server := newGetTestServer(t, rule)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "uuid-123", true)
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "COMPONENT")
	testutil.Contains(t, out, "TRIGGER")
	testutil.Contains(t, out, "ACTION")
}
