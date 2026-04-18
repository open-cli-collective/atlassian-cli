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

func seedBoardsCache(t *testing.T, boards []api.Board) {
	t.Helper()
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Setenv("JIRA_URL", "https://test.atlassian.net")
	t.Setenv("JIRA_EMAIL", "t@example.com")
	t.Setenv("JIRA_API_TOKEN", "tok")
	testutil.RequireNoError(t, cache.WriteResource("boards", "24h", boards))
}

func TestBoard_NumericCacheMatch(t *testing.T) {
	seedBoardsCache(t, []api.Board{
		{ID: 23, Name: "MON board", Type: "scrum"},
	})
	b, err := New(nil).Board(context.Background(), "23")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, b.Name, "MON board")
}

func TestBoard_NameMatch(t *testing.T) {
	seedBoardsCache(t, []api.Board{
		{ID: 23, Name: "MON board", Type: "scrum"},
		{ID: 24, Name: "ON board", Type: "kanban"},
	})
	b, err := New(nil).Board(context.Background(), "mon board")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, b.ID, 23)
}

func TestBoard_NumericPassThroughWhenNotCached(t *testing.T) {
	// Cache present but doesn't contain the requested ID. Numeric shape
	// should short-circuit refresh and return a synthetic.
	seedBoardsCache(t, []api.Board{{ID: 23, Name: "MON", Type: "scrum"}})
	b, err := New(nil).Board(context.Background(), "999")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, b.ID, 999)
	testutil.Equal(t, b.Name, "")
}

func TestBoard_NotFoundCarriesRefreshHint(t *testing.T) {
	seedBoardsCache(t, []api.Board{{ID: 23, Name: "MON", Type: "scrum"}})
	// Refresh stub returns the same (no new board) → retry misses.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.BoardsResponse{IsLast: true, Values: []api.Board{
			{ID: 23, Name: "MON", Type: "scrum"},
		}})
	}))
	defer server.Close()
	client := newTestClient(t, server)

	_, err := New(client).Board(context.Background(), "Unknown Board")
	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("expected NotFoundError, got %T: %v", err, err)
	}
	testutil.Equal(t, nf.Entity, "board")
	if !strings.Contains(err.Error(), "jtk refresh boards") {
		t.Fatalf("missing refresh hint: %q", err.Error())
	}
}

func TestBoard_AmbiguousName(t *testing.T) {
	seedBoardsCache(t, []api.Board{
		{ID: 1, Name: "Dev", Type: "scrum"},
		{ID: 2, Name: "Dev", Type: "kanban"},
	})
	_, err := New(nil).Board(context.Background(), "Dev")
	var amb *AmbiguousMatchError
	if !errors.As(err, &amb) {
		t.Fatalf("expected AmbiguousMatchError, got %T: %v", err, err)
	}
	testutil.Equal(t, len(amb.Matches), 2)
}
