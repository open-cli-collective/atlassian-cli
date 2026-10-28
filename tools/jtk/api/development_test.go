package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
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
					{"id": "42", "name": "Older duplicate", "url": "HTTPS://GitHub.com/owner/repo-a/pull/42/?utm_source=provider-a#fragment", "status": "OPEN", "lastUpdate": "2026-07-30T12:00:00Z"},
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
	testutil.Equal(t, development.PullRequests[0].Repository, "owner/repo-a")
	testutil.Equal(t, development.PullRequests[1].URL, "https://github.com/owner/repo-a/pull/7")
	testutil.Equal(t, development.PullRequests[2].URL, "https://github.com/owner/repo-b/pull/7")
	testutil.Len(t, development.Providers, 1)
	testutil.False(t, development.PullRequestsTruncated)
}

func TestGetDevelopment_BearerAuthUnavailable(t *testing.T) {
	t.Parallel()

	client, err := New(ClientConfig{
		URL:        "https://example.atlassian.net",
		APIToken:   "scoped-token",
		AuthMethod: "bearer",
		CloudID:    "cloud-id",
	})
	testutil.RequireNoError(t, err)

	_, err = client.GetDevelopment(context.Background(), "PROJ-123")
	testutil.True(t, errors.Is(err, ErrDevelopmentUnavailable))
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

func TestGetDevelopment_PagesGraphStorePullRequests(t *testing.T) {
	t.Parallel()

	var graphRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/issue/PROJ-123":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "10001", "key": "PROJ-123"})
		case "/rest/dev-status/1.0/issue/summary":
			writeDevelopmentSummary(t, w, 100)
		case "/_edge/tenant_info":
			_ = json.NewEncoder(w).Encode(map[string]any{"cloudId": "cloud-1"})
		case "/gateway/api/graphql":
			graphRequests.Add(1)
			testutil.Equal(t, r.Method, http.MethodPost)
			testutil.Equal(t, r.Header.Get("X-Query-Context"), "ari:cloud:platform::site/cloud-1")

			var request struct {
				Query     string `json:"query"`
				Variables struct {
					ID    string  `json:"id"`
					First int     `json:"first"`
					After *string `json:"after"`
				} `json:"variables"`
			}
			testutil.RequireNoError(t, json.NewDecoder(r.Body).Decode(&request))
			testutil.Contains(t, request.Query, `@optIn(to: "GraphStoreIssueAssociatedPr")`)
			testutil.Equal(t, request.Variables.ID, "ari:cloud:jira:cloud-1:issue/10001")
			testutil.Equal(t, request.Variables.First, 50)

			after := ""
			if request.Variables.After != nil {
				after = *request.Variables.After
			}
			switch after {
			case "":
				writeGraphPage(t, w, 1, 50, "cursor-1")
			case "cursor-1":
				writeGraphPage(t, w, 51, 100, "cursor-2")
			case "cursor-2":
				writeGraphPage(t, w, 101, 120, "")
			default:
				t.Errorf("unexpected cursor %q", after)
			}
		case "/rest/dev-status/1.0/issue/detail":
			t.Error("dev-status detail should not be called when graph store succeeds")
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
	testutil.Equal(t, int(graphRequests.Load()), 3)
	testutil.Len(t, development.PullRequests, 120)
	testutil.False(t, development.PullRequestsTruncated)
	testutil.Equal(t, development.Providers, []string{"GitHub"})

	newest := development.PullRequests[0]
	testutil.Equal(t, newest.ID, "120")
	testutil.Equal(t, newest.URL, "https://github.com/owner/repo-0/pull/120")
	testutil.Equal(t, newest.Repository, "owner/repo-0")
	testutil.Equal(t, newest.Status, "MERGED")
	testutil.Equal(t, newest.Provider, "GitHub")
	testutil.Equal(t, development.PullRequests[119].ID, "1")
	testutil.Equal(t, development.PullRequests[119].Repository, "owner/repo-1")
}

func TestGetDevelopment_FallsBackToTruncatedDevStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		graph func(t *testing.T, w http.ResponseWriter)
	}{
		{
			name: "graphql errors",
			graph: func(_ *testing.T, w http.ResponseWriter) {
				_ = json.NewEncoder(w).Encode(map[string]any{"errors": []map[string]any{{"message": "field is experimental"}}})
			},
		},
		{
			name: "cursor does not advance",
			graph: func(t *testing.T, w http.ResponseWriter) {
				writeGraphPage(t, w, 1, 1, "same-cursor")
			},
		},
		{
			name: "gateway unavailable",
			graph: func(_ *testing.T, w http.ResponseWriter) {
				http.Error(w, "unavailable", http.StatusServiceUnavailable)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/rest/api/3/issue/PROJ-123":
					_ = json.NewEncoder(w).Encode(map[string]any{"id": "10001", "key": "PROJ-123"})
				case "/rest/dev-status/1.0/issue/summary":
					writeDevelopmentSummary(t, w, 100)
				case "/_edge/tenant_info":
					_ = json.NewEncoder(w).Encode(map[string]any{"cloudId": "cloud-1"})
				case "/gateway/api/graphql":
					tt.graph(t, w)
				case "/rest/dev-status/1.0/issue/detail":
					pullRequests := make([]map[string]any, 0, 100)
					for i := 1; i <= 100; i++ {
						pullRequests = append(pullRequests, map[string]any{
							"id":         "#" + strconv.Itoa(i),
							"name":       "PR " + strconv.Itoa(i),
							"url":        "https://github.com/owner/repo/pull/" + strconv.Itoa(i),
							"status":     "MERGED",
							"lastUpdate": "2026-07-01T12:00:00.000+0000",
						})
					}
					writeDevelopmentDetail(t, w, "repo", "owner/repo", pullRequests)
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
			testutil.Len(t, development.PullRequests, 100)
			testutil.True(t, development.PullRequestsTruncated)
		})
	}
}

func TestGetDevelopment_DevStatusRepositoryIsPerPullRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/issue/PROJ-123":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "10001", "key": "PROJ-123"})
		case "/rest/dev-status/1.0/issue/summary":
			writeDevelopmentSummary(t, w, 3)
		case "/rest/dev-status/1.0/issue/detail":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"detail": []map[string]any{{
					"pullRequests": []map[string]any{
						{"id": "#1", "url": "https://github.com/owner/first/pull/1", "lastUpdate": "2026-07-03T12:00:00Z"},
						{"id": "#2", "url": "https://github.com/owner/second/pull/2", "lastUpdate": "2026-07-02T12:00:00Z"},
						{"id": "#3", "url": "https://git.example/x/3", "repositoryName": "owner/third", "lastUpdate": "2026-07-01T12:00:00Z"},
					},
					"repositories": []map[string]any{},
					"_instance":    map[string]any{"name": "GitHub"},
				}},
			})
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
	testutil.Len(t, development.PullRequests, 3)
	testutil.Equal(t, development.PullRequests[0].Repository, "owner/first")
	testutil.Equal(t, development.PullRequests[1].Repository, "owner/second")
	testutil.Equal(t, development.PullRequests[2].Repository, "owner/third")
}

func TestDevelopmentPullRequestNumberFromURL(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"https://github.com/owner/repo/pull/42":               "42",
		"https://bitbucket.org/owner/repo/pull-requests/7":    "7",
		"https://gitlab.com/group/project/-/merge_requests/9": "9",
		"https://github.com/owner/repo/issues/42":             "",
		"https://github.com/owner":                            "",
	}
	for raw, want := range tests {
		testutil.Equal(t, developmentPullRequestNumberFromURL(raw), want)
	}
}

func writeDevelopmentSummary(t *testing.T, w http.ResponseWriter, pullRequests int) {
	t.Helper()
	_ = json.NewEncoder(w).Encode(map[string]any{
		"summary": map[string]any{
			"pullrequest": map[string]any{
				"overall":        map[string]any{"count": pullRequests, "state": "MERGED"},
				"byInstanceType": map[string]any{"github-app": map[string]any{"name": "GitHub"}},
			},
		},
	})
}

// writeGraphPage writes Graph Store pull requests numbered first..last, each
// updated a minute after the previous, plus one non-PR node that must be ignored.
func writeGraphPage(t *testing.T, w http.ResponseWriter, first, last int, endCursor string) {
	t.Helper()
	edges := []map[string]any{{"node": map[string]any{"__typename": "DevOpsBranchDetails"}}}
	for i := first; i <= last; i++ {
		repository := "owner/repo-" + strconv.Itoa(i%2)
		edges = append(edges, map[string]any{"node": map[string]any{
			"__typename":     "DevOpsPullRequestDetails",
			"url":            "https://github.com/" + repository + "/pull/" + strconv.Itoa(i),
			"title":          "PR " + strconv.Itoa(i),
			"status":         "MERGED",
			"repositoryName": repository,
			"lastUpdated":    "2026-07-01T" + minuteClock(i) + ":00.000Z",
		}})
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": map[string]any{
			"graphStore_issueAssociatedPr": map[string]any{
				"pageInfo": map[string]any{"hasNextPage": endCursor != "", "endCursor": endCursor},
				"edges":    edges,
			},
		},
	})
}

// minuteClock formats minutes as HH:MM.
func minuteClock(minutes int) string {
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}
