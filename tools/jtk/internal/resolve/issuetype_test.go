package resolve

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
)

func seedIssueTypesCache(t *testing.T, byProject map[string][]api.IssueType) {
	t.Helper()
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Setenv("JIRA_URL", "https://test.atlassian.net")
	t.Setenv("JIRA_EMAIL", "t@example.com")
	t.Setenv("JIRA_API_TOKEN", "tok")
	testutil.RequireNoError(t, cache.WriteResource("issuetypes", "24h", byProject))
}

func TestIssueType_NameMatchInProject(t *testing.T) {
	seedIssueTypesCache(t, map[string][]api.IssueType{
		"MON": {{ID: "10025", Name: "SDLC"}, {ID: "10000", Name: "Epic"}},
		"ON":  {{ID: "10001", Name: "Task"}},
	})
	it, err := New(nil).IssueType(context.Background(), "MON", "SDLC")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, it.ID, "10025")
}

func TestIssueType_CaseInsensitive(t *testing.T) {
	seedIssueTypesCache(t, map[string][]api.IssueType{
		"MON": {{ID: "10025", Name: "SDLC"}},
	})
	it, err := New(nil).IssueType(context.Background(), "MON", "sdlc")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, it.ID, "10025")
}

func TestIssueType_IDMatch(t *testing.T) {
	seedIssueTypesCache(t, map[string][]api.IssueType{
		"MON": {{ID: "10025", Name: "SDLC"}},
	})
	it, err := New(nil).IssueType(context.Background(), "MON", "10025")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, it.Name, "SDLC")
}

func TestIssueType_EmptyProjectKey(t *testing.T) {
	seedIssueTypesCache(t, map[string][]api.IssueType{"MON": {{ID: "1", Name: "Task"}}})
	_, err := New(nil).IssueType(context.Background(), "", "Task")
	if err == nil {
		t.Fatalf("expected error for empty projectKey")
	}
	if !strings.Contains(err.Error(), "projectKey") {
		t.Fatalf("expected error to mention projectKey, got %q", err.Error())
	}
}

func TestIssueType_ColdCacheTriggersRefresh(t *testing.T) {
	// No cache file exists → resolver refreshes once. The refresh pulls
	// projects + issuetypes (dep chain), then the retry hits.
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/project":
			_ = json.NewEncoder(w).Encode([]api.Project{{Key: "MON", Name: "Platform"}})
		case "/rest/api/3/project/MON":
			_ = json.NewEncoder(w).Encode(struct {
				IssueTypes []api.IssueType `json:"issueTypes"`
			}{
				IssueTypes: []api.IssueType{{ID: "10025", Name: "SDLC"}},
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client := newTestClient(t, server)

	it, err := New(client).IssueType(context.Background(), "MON", "SDLC")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, it.ID, "10025")
}

func TestIssueType_ColdCacheOfflineFallsBackToRawName(t *testing.T) {
	// No cache + refresh unreachable — the coldFallback should let the
	// caller proceed with the raw name, matching pre-#236 behavior for
	// uninitialized installs.
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	client := newTestClient(t, server)

	it, err := New(client).IssueType(context.Background(), "MON", "Task")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, it.Name, "Task")
	testutil.Equal(t, it.ID, "") // synthetic
}

func TestIssueType_ProjectMissingFromCacheRefreshes(t *testing.T) {
	// Issuetypes envelope exists but doesn't list our project. Resolver
	// should treat that as a cache miss and attempt a refresh.
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))
	testutil.RequireNoError(t, cache.WriteResource("issuetypes", "24h", map[string][]api.IssueType{
		"OTHER": {{ID: "1", Name: "Task"}},
	}))
	testutil.RequireNoError(t, cache.WriteResource("projects", "24h", []api.Project{
		{Key: "MON", Name: "Platform"},
	}))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/project":
			_ = json.NewEncoder(w).Encode([]api.Project{{Key: "MON", Name: "Platform"}})
		case "/rest/api/3/project/MON":
			_ = json.NewEncoder(w).Encode(struct {
				IssueTypes []api.IssueType `json:"issueTypes"`
			}{
				IssueTypes: []api.IssueType{{ID: "10025", Name: "SDLC"}},
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client := newTestClient(t, server)

	it, err := New(client).IssueType(context.Background(), "MON", "SDLC")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, it.ID, "10025")
}

func TestIssueType_AmbiguousName(t *testing.T) {
	seedIssueTypesCache(t, map[string][]api.IssueType{
		"MON": {
			{ID: "1", Name: "Task"},
			{ID: "2", Name: "task"}, // case mismatch but equals case-insensitive
		},
	})
	_, err := New(nil).IssueType(context.Background(), "MON", "Task")
	var amb *AmbiguousMatchError
	if !errors.As(err, &amb) {
		t.Fatalf("expected AmbiguousMatchError, got %T: %v", err, err)
	}
	testutil.Equal(t, len(amb.Matches), 2)
}
