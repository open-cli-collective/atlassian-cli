package development

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestRegister_DevelopmentGet(t *testing.T) {
	t.Parallel()

	server := developmentServer(t)
	defer server.Close()
	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	rootCmd, opts := root.NewCmd()
	var stdout bytes.Buffer
	opts.Stdout = &stdout
	opts.Stderr = &bytes.Buffer{}
	opts.SetAPIClient(client)
	Register(rootCmd, opts)
	rootCmd.SetArgs([]string{"development", "get", "PROJ-123"})

	err = rootCmd.Execute()
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "PROJ-123  Development")
	testutil.Contains(t, stdout.String(), "Pull Requests: 1 (MERGED)")
	testutil.Contains(t, stdout.String(), "https://github.com/owner/repo/pull/42")
}

func TestRunGet_IDOnly(t *testing.T) {
	t.Parallel()

	server := developmentServer(t)
	defer server.Close()
	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runGet(t.Context(), opts, "PROJ-123")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "https://github.com/owner/repo/pull/42\n")
}

func developmentServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/issue/PROJ-123":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "10001", "key": "PROJ-123"})
		case "/rest/dev-status/1.0/issue/summary":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"summary": map[string]any{
					"pullrequest": map[string]any{
						"overall":        map[string]any{"count": 1, "state": "MERGED"},
						"byInstanceType": map[string]any{"github": map[string]any{"name": "GitHub"}},
					},
				},
			})
		case "/rest/dev-status/1.0/issue/detail":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"detail": []map[string]any{{
					"pullRequests": []map[string]any{{
						"id": "42", "name": "Add feature", "url": "https://github.com/owner/repo/pull/42",
						"status": "MERGED", "lastUpdate": "2026-07-31T12:00:00Z",
					}},
					"repositories": []map[string]any{{"id": "repo", "name": "owner/repo"}},
					"_instance":    map[string]any{"name": "GitHub"},
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
}
