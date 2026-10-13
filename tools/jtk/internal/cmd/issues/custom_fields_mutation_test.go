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
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// customFieldsMutationServer serves create, update and assign writes for
// TEST-1, the post-write GET with one populated custom field, and /field so
// the custom field's name can be resolved.
func customFieldsMutationServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/rest/api/3/field" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode([]api.Field{
				{ID: "customfield_10005", Name: "Change type", Custom: true, Schema: api.FieldSchema{Type: "option"}},
			})
		case r.URL.Path == "/rest/api/3/issue" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(api.Issue{Key: "TEST-1", ID: "10001"})
		case r.URL.Path == "/rest/api/3/issue/TEST-1" && r.Method == http.MethodPut,
			r.URL.Path == "/rest/api/3/issue/TEST-1/assignee" && r.Method == http.MethodPut:
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/rest/api/3/issue/TEST-1" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"key": "TEST-1",
				"fields": map[string]any{
					"summary":           "New issue",
					"status":            map[string]any{"name": "Backlog"},
					"issuetype":         map[string]any{"name": "Task"},
					"priority":          map[string]any{"name": "Medium"},
					"updated":           "2026-04-16T00:00:00.000+0000",
					"customfield_10005": map[string]any{"value": "Feature"},
				},
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestMutations_CustomFieldsFlag(t *testing.T) {
	commands := []struct {
		name string
		run  func(ctx context.Context, opts *root.Options, customFields bool) error
	}{
		{"create", func(ctx context.Context, opts *root.Options, cf bool) error {
			return runCreate(ctx, opts, "PROJ", "Task", "New issue", "", "", "", nil, cf)
		}},
		{"update", func(ctx context.Context, opts *root.Options, cf bool) error {
			return runUpdate(ctx, opts, "TEST-1", "New issue", "", "", "", "", "", nil, cf)
		}},
		{"assign", func(ctx context.Context, opts *root.Options, cf bool) error {
			return runAssign(ctx, opts, "TEST-1", "", true, cf)
		}},
	}
	for _, c := range commands {
		for _, customFields := range []bool{true, false} {
			name := c.name + map[bool]string{true: " with --custom-fields", false: " without --custom-fields"}[customFields]
			t.Run(name, func(t *testing.T) {
				seedCacheForIssues(t)
				server := customFieldsMutationServer(t)
				defer server.Close()

				client, err := api.New(api.ClientConfig{URL: server.URL, Email: "e@x", APIToken: "t"})
				testutil.RequireNoError(t, err)
				var stdout bytes.Buffer
				opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
				opts.SetAPIClient(client)

				testutil.RequireNoError(t, c.run(context.Background(), opts, customFields))

				out := stdout.String()
				testutil.Contains(t, out, "TEST-1")
				if customFields {
					testutil.Contains(t, out, "Custom Fields:")
					testutil.Contains(t, out, "Change type: Feature")
				} else if strings.Contains(out, "Custom Fields:") {
					t.Fatalf("custom fields shown without the flag:\n%s", out)
				}
			})
		}
	}
}
