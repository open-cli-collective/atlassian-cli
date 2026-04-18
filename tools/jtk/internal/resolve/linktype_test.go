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

func seedLinkTypesCache(t *testing.T, types []api.IssueLinkType) {
	t.Helper()
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Setenv("JIRA_URL", "https://test.atlassian.net")
	t.Setenv("JIRA_EMAIL", "t@example.com")
	t.Setenv("JIRA_API_TOKEN", "tok")
	testutil.RequireNoError(t, cache.WriteResource("linktypes", "24h", types))
}

func TestLinkType_NameMatch(t *testing.T) {
	seedLinkTypesCache(t, []api.IssueLinkType{
		{ID: "10100", Name: "Blocker", Inward: "is blocked by", Outward: "blocks"},
		{ID: "10200", Name: "Relates", Inward: "relates to", Outward: "relates to"},
	})
	lt, err := New(nil).LinkType(context.Background(), "blocker")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, lt.ID, "10100")
}

func TestLinkType_InwardVerb(t *testing.T) {
	seedLinkTypesCache(t, []api.IssueLinkType{
		{ID: "10100", Name: "Blocker", Inward: "is blocked by", Outward: "blocks"},
	})
	lt, err := New(nil).LinkType(context.Background(), "is blocked by")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, lt.Name, "Blocker")
}

func TestLinkType_OutwardVerb(t *testing.T) {
	seedLinkTypesCache(t, []api.IssueLinkType{
		{ID: "10100", Name: "Blocker", Inward: "is blocked by", Outward: "blocks"},
	})
	lt, err := New(nil).LinkType(context.Background(), "BLOCKS")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, lt.Name, "Blocker")
}

func TestLinkType_NotFound(t *testing.T) {
	seedLinkTypesCache(t, []api.IssueLinkType{
		{ID: "10100", Name: "Blocker", Inward: "is blocked by", Outward: "blocks"},
	})
	// Refresh stub keeps the cache unchanged — after retry, still no match.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issueLinkTypes": []api.IssueLinkType{
				{ID: "10100", Name: "Blocker", Inward: "is blocked by", Outward: "blocks"},
			},
		})
	}))
	defer server.Close()
	client := newTestClient(t, server)

	_, err := New(client).LinkType(context.Background(), "BogusLink")
	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("expected NotFoundError, got %T: %v", err, err)
	}
	testutil.Equal(t, nf.Entity, "link type")
	if !strings.Contains(err.Error(), "jtk refresh linktypes") {
		t.Fatalf("missing refresh hint in: %q", err.Error())
	}
}
