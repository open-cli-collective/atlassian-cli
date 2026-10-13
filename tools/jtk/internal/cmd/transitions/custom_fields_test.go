package transitions

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

// Not parallel: the field cache root is process-global.
func TestRunDo_CustomFieldsFlag(t *testing.T) {
	for _, customFields := range []bool{true, false} {
		t.Run(map[bool]string{true: "with flag", false: "without flag"}[customFields], func(t *testing.T) {
			t.Cleanup(cache.SetRootForTest(t.TempDir()))
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.URL.Path == "/rest/api/3/field" && r.Method == http.MethodGet:
					_ = json.NewEncoder(w).Encode([]api.Field{
						{ID: "customfield_10005", Name: "Change type", Custom: true, Schema: api.FieldSchema{Type: "option"}},
					})
				case strings.HasSuffix(r.URL.Path, "/transitions") && r.Method == http.MethodGet:
					_ = json.NewEncoder(w).Encode(api.TransitionsResponse{Transitions: []api.Transition{
						{ID: "31", Name: "In Development", To: api.Status{Name: "In Development"}},
					}})
				case strings.HasSuffix(r.URL.Path, "/transitions") && r.Method == http.MethodPost:
					w.WriteHeader(http.StatusNoContent)
				case r.URL.Path == "/rest/api/3/issue/TEST-1" && r.Method == http.MethodGet:
					_ = json.NewEncoder(w).Encode(map[string]any{
						"key": "TEST-1",
						"fields": map[string]any{
							"summary":           "Test issue",
							"status":            map[string]any{"name": "In Development"},
							"issuetype":         map[string]any{"name": "SDLC"},
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
			defer server.Close()

			client, err := api.New(api.ClientConfig{URL: server.URL, Email: "e@x", APIToken: "t"})
			testutil.RequireNoError(t, err)
			var stdout bytes.Buffer
			opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
			opts.SetAPIClient(client)

			testutil.RequireNoError(t, runDo(context.Background(), opts, "TEST-1", "In Development", nil, customFields))

			out := stdout.String()
			testutil.Contains(t, out, "In Development")
			if customFields {
				testutil.Contains(t, out, "Change type: Feature")
			} else if strings.Contains(out, "Custom Fields:") {
				t.Fatalf("custom fields shown without the flag:\n%s", out)
			}
		})
	}
}
