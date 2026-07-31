package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestGetDevelopment_DeduplicatesPullRequests(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/issue/PROJ-123":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "10001", "key": "PROJ-123"})
		case "/rest/dev-status/1.0/issue/summary":
			testutil.Equal(t, r.URL.Query().Get("issueId"), "10001")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"summary": map[string]any{
					"pullrequest": map[string]any{
						"overall": map[string]any{"count": 4, "state": "OPEN"},
						"byInstanceType": map[string]any{
							"provider-a": map[string]any{"name": "GitHub"},
							"provider-b": map[string]any{"name": "GitHub"},
						},
					},
					"repository": map[string]any{"overall": map[string]any{"count": 5}},
					"build":      map[string]any{"overall": map[string]any{"count": 3, "successfulBuildCount": 2}},
				},
			})
		case "/rest/dev-status/1.0/issue/detail":
			testutil.Equal(t, r.URL.Query().Get("issueId"), "10001")
			testutil.Equal(t, r.URL.Query().Get("dataType"), "pullrequest")
			switch r.URL.Query().Get("applicationType") {
			case "provider-a":
				writeDevelopmentDetail(t, w, "repo-a", "owner/repo-a", []map[string]any{
					{"id": "42", "name": "Older duplicate", "url": "HTTPS://GitHub.com/owner/repo-a/pull/42/#fragment", "status": "OPEN", "lastUpdate": "2026-07-30T12:00:00Z"},
					{"id": "7", "name": "Repo A seven", "url": "https://github.com/owner/repo-a/pull/7", "status": "OPEN", "lastUpdate": "2026-07-29T12:00:00Z"},
				})
			case "provider-b":
				writeDevelopmentDetail(t, w, "repo-b", "owner/repo-b", []map[string]any{
					{"id": "42", "name": "Newer duplicate", "url": "https://github.com/owner/repo-a/pull/42", "status": "MERGED", "lastUpdate": "2026-07-31T12:00:00Z"},
					{"id": "7", "name": "Repo B seven", "url": "https://github.com/owner/repo-b/pull/7", "status": "OPEN", "lastUpdate": "2026-07-28T12:00:00Z"},
				})
			default:
				http.Error(w, "unexpected provider", http.StatusBadRequest)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)
	client.HTTPClient = server.Client()

	development, err := client.GetDevelopment(context.Background(), "PROJ-123")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, development.IssueKey, "PROJ-123")
	testutil.Equal(t, development.Commits, 5)
	testutil.Equal(t, development.Builds, 3)
	testutil.Equal(t, development.SuccessfulBuilds, 2)
	testutil.Len(t, development.PullRequests, 3)
	testutil.Equal(t, development.PullRequests[0].Title, "Newer duplicate")
	testutil.Equal(t, development.PullRequests[0].URL, "https://github.com/owner/repo-a/pull/42")
	testutil.Equal(t, development.PullRequests[0].Repository, "owner/repo-b")
	testutil.Equal(t, development.PullRequests[1].URL, "https://github.com/owner/repo-a/pull/7")
	testutil.Equal(t, development.PullRequests[2].URL, "https://github.com/owner/repo-b/pull/7")
	testutil.Len(t, development.Providers, 1)
}

func TestDevelopmentRepositoryFromURL(t *testing.T) {
	t.Parallel()
	testutil.Equal(t, developmentRepositoryFromURL("https://github.com/owner/repo/pull/42"), "owner/repo")
	testutil.Equal(t, developmentRepositoryFromURL("https://gitlab.example/owner/repo/pull/42"), "")
}

func writeDevelopmentDetail(t *testing.T, w http.ResponseWriter, repositoryID, repositoryName string, pullRequests []map[string]any) {
	t.Helper()
	_ = json.NewEncoder(w).Encode(map[string]any{
		"detail": []map[string]any{{
			"pullRequests": pullRequests,
			"repositories": []map[string]any{{"id": repositoryID, "name": repositoryName}},
			"_instance":    map[string]any{"name": "GitHub"},
		}},
	})
}
