package automation

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
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
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
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "uuid-123", false)
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "uuid-123")
	testutil.NotContains(t, out, "My Rule")
	testutil.NotContains(t, out, "State")
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
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "uuid-123", true)
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "TRIGGER  issue.created")
	testutil.Contains(t, out, "  ACTION  assign.issue")
}
