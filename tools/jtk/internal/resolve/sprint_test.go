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

func seedSprintsCache(t *testing.T, byBoard map[int][]api.Sprint, boards []api.Board) {
	t.Helper()
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Setenv("JIRA_URL", "https://test.atlassian.net")
	t.Setenv("JIRA_EMAIL", "t@example.com")
	t.Setenv("JIRA_API_TOKEN", "tok")
	testutil.RequireNoError(t, cache.WriteResource("sprints", "24h", byBoard))
	if boards != nil {
		testutil.RequireNoError(t, cache.WriteResource("boards", "24h", boards))
	}
}

func TestSprint_NumericCacheMatch(t *testing.T) {
	seedSprintsCache(t, map[int][]api.Sprint{
		23: {{ID: 125, Name: "MON Sprint 70", State: "active"}},
	}, nil)
	s, err := New(nil).Sprint(context.Background(), "125", 0)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, s.Name, "MON Sprint 70")
}

func TestSprint_BoardScopedName(t *testing.T) {
	seedSprintsCache(t, map[int][]api.Sprint{
		23: {{ID: 125, Name: "MON Sprint 70", State: "active"}},
		24: {{ID: 200, Name: "MON Sprint 70", State: "active"}}, // same name on different board
	}, nil)
	s, err := New(nil).Sprint(context.Background(), "MON Sprint 70", 23)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, s.ID, 125)
}

func TestSprint_GlobalUniqueName(t *testing.T) {
	seedSprintsCache(t, map[int][]api.Sprint{
		23: {{ID: 125, Name: "MON Sprint 70", State: "active"}},
		24: {{ID: 200, Name: "ON Sprint 1", State: "active"}},
	}, nil)
	s, err := New(nil).Sprint(context.Background(), "ON Sprint 1", 0)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, s.ID, 200)
}

func TestSprint_GlobalAmbiguousNameIncludesBoardInfo(t *testing.T) {
	seedSprintsCache(t, map[int][]api.Sprint{
		23: {{ID: 125, Name: "Sprint 1", State: "active"}},
		24: {{ID: 200, Name: "Sprint 1", State: "closed"}},
	}, []api.Board{
		{ID: 23, Name: "MON board"},
		{ID: 24, Name: "ON board"},
	})
	_, err := New(nil).Sprint(context.Background(), "Sprint 1", 0)
	var amb *AmbiguousMatchError
	if !errors.As(err, &amb) {
		t.Fatalf("expected AmbiguousMatchError, got %T: %v", err, err)
	}
	testutil.Equal(t, len(amb.Matches), 2)
	msg := err.Error()
	if !strings.Contains(msg, "MON board") || !strings.Contains(msg, "ON board") {
		t.Fatalf("expected board names in ambiguous match output: %q", msg)
	}
	if !strings.Contains(msg, "active") || !strings.Contains(msg, "closed") {
		t.Fatalf("expected sprint states in ambiguous match output: %q", msg)
	}
}

func TestSprint_NumericPassThroughWhenNotCached(t *testing.T) {
	// Cache seeded but missing the target ID, and the resolver's refresh
	// fails. Numeric shape → pass-through synthetic.
	seedSprintsCache(t, map[int][]api.Sprint{
		23: {{ID: 1, Name: "Other"}},
	}, nil)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	client := newTestClient(t, server)

	s, err := New(client).Sprint(context.Background(), "999", 0)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, s.ID, 999)
	testutil.Equal(t, s.Name, "")
}

func TestSprint_ColdCacheTriggersRefresh(t *testing.T) {
	// No sprints cache exists; refresh pulls boards + sprints.
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))
	testutil.RequireNoError(t, cache.WriteResource("boards", "24h", []api.Board{{ID: 23, Name: "MON"}}))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/agile/1.0/board" {
			_ = json.NewEncoder(w).Encode(api.BoardsResponse{IsLast: true, Values: []api.Board{{ID: 23, Name: "MON"}}})
			return
		}
		if r.URL.Path == "/rest/agile/1.0/board/23/sprint" {
			_ = json.NewEncoder(w).Encode(api.SprintsResponse{IsLast: true, Values: []api.Sprint{
				{ID: 125, Name: "MON Sprint 70", State: "active"},
			}})
			return
		}
		t.Errorf("unexpected path: %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	client := newTestClient(t, server)

	s, err := New(client).Sprint(context.Background(), "MON Sprint 70", 0)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, s.ID, 125)
}

func TestSprint_NotFoundCarriesRefreshHint(t *testing.T) {
	seedSprintsCache(t, map[int][]api.Sprint{23: {{ID: 1, Name: "Other"}}}, nil)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/board") {
			_ = json.NewEncoder(w).Encode(api.BoardsResponse{IsLast: true, Values: []api.Board{}})
			return
		}
		_ = json.NewEncoder(w).Encode(api.SprintsResponse{IsLast: true})
	}))
	defer server.Close()
	client := newTestClient(t, server)

	_, err := New(client).Sprint(context.Background(), "Nonexistent Sprint", 0)
	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("expected NotFoundError, got %T: %v", err, err)
	}
	testutil.Equal(t, nf.Entity, "sprint")
	if !strings.Contains(err.Error(), "jtk refresh sprints") {
		t.Fatalf("missing refresh hint: %q", err.Error())
	}
}

func TestSprint_BoardScopedAmbiguous(t *testing.T) {
	seedSprintsCache(t, map[int][]api.Sprint{
		23: {
			{ID: 125, Name: "Sprint 1", State: "active"},
			{ID: 126, Name: "Sprint 1", State: "future"},
		},
	}, nil)
	_, err := New(nil).Sprint(context.Background(), "Sprint 1", 23)
	var amb *AmbiguousMatchError
	if !errors.As(err, &amb) {
		t.Fatalf("expected AmbiguousMatchError, got %T: %v", err, err)
	}
	testutil.Equal(t, len(amb.Matches), 2)
}
