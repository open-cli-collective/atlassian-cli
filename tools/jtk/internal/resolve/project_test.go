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

func seedProjectsCache(t *testing.T, projects []api.Project) {
	t.Helper()
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Setenv("JIRA_URL", "https://test.atlassian.net")
	t.Setenv("JIRA_EMAIL", "t@example.com")
	t.Setenv("JIRA_API_TOKEN", "tok")
	testutil.RequireNoError(t, cache.WriteResource("projects", "24h", projects))
}

func TestProject_ExactKeyMatch(t *testing.T) {
	seedProjectsCache(t, []api.Project{
		{Key: "MON", Name: "Platform"},
		{Key: "ON", Name: "Onboarding"},
	})
	p, err := New(nil).Project(context.Background(), "MON")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, p.Name, "Platform")
}

func TestProject_NameMatch(t *testing.T) {
	seedProjectsCache(t, []api.Project{
		{Key: "MON", Name: "Platform Development"},
	})
	p, err := New(nil).Project(context.Background(), "platform development")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, p.Key, "MON")
}

func TestProject_AmbiguousByName(t *testing.T) {
	seedProjectsCache(t, []api.Project{
		{Key: "A", Name: "Platform"},
		{Key: "B", Name: "Platform"},
	})
	_, err := New(nil).Project(context.Background(), "Platform")
	var amb *AmbiguousMatchError
	if !errors.As(err, &amb) {
		t.Fatalf("expected AmbiguousMatchError, got %T: %v", err, err)
	}
	testutil.Equal(t, len(amb.Matches), 2)
}

func TestProject_KeyShapePassThrough(t *testing.T) {
	seedProjectsCache(t, []api.Project{
		{Key: "MON", Name: "Platform"},
	})
	// Refresh stub returns same data (still no NEWPROJ); after retry, pass-through.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{Key: "MON", Name: "Platform"}})
	}))
	defer server.Close()
	client := newTestClient(t, server)

	p, err := New(client).Project(context.Background(), "NEWPROJ")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, p.Key, "NEWPROJ")
	testutil.Equal(t, p.Name, "") // synthetic: only Key is set
}

func TestProject_NotFoundForNonKeyShape(t *testing.T) {
	seedProjectsCache(t, []api.Project{
		{Key: "MON", Name: "Platform"},
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{Key: "MON", Name: "Platform"}})
	}))
	defer server.Close()
	client := newTestClient(t, server)

	_, err := New(client).Project(context.Background(), "Some Random Name")
	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("expected NotFoundError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "jtk refresh projects") {
		t.Fatalf("missing refresh hint in: %q", err.Error())
	}
}
