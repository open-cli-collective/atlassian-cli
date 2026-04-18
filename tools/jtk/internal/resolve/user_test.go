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

// seedUsersCache stands up a temp cache root, points JIRA_URL at a valid
// instance so InstanceKey resolves, and writes the given users to disk.
func seedUsersCache(t *testing.T, users []api.User) {
	t.Helper()
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Setenv("JIRA_URL", "https://test.atlassian.net")
	t.Setenv("JIRA_EMAIL", "t@example.com")
	t.Setenv("JIRA_API_TOKEN", "tok")
	testutil.RequireNoError(t, cache.WriteResource("users", "24h", users))
}

func newTestClient(t *testing.T, server *httptest.Server) *api.Client {
	t.Helper()
	t.Setenv("JIRA_URL", "https://test.atlassian.net")
	t.Setenv("JIRA_EMAIL", "t@example.com")
	t.Setenv("JIRA_API_TOKEN", "tok")
	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@example.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)
	return client
}

func TestUser_MeCallsCurrentUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/myself") {
			_ = json.NewEncoder(w).Encode(api.User{AccountID: "self-id", DisplayName: "Self"})
			return
		}
		t.Errorf("unexpected path: %s", r.URL.Path)
	}))
	defer server.Close()

	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	client := newTestClient(t, server)

	u, err := New(client).User(context.Background(), "me")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, u.AccountID, "self-id")
}

func TestUser_ExactAccountIDFromCache(t *testing.T) {
	seedUsersCache(t, []api.User{
		{AccountID: "aaa", DisplayName: "Alice", EmailAddress: "alice@x.io"},
		{AccountID: "bbb", DisplayName: "Bob", EmailAddress: "bob@x.io"},
	})
	u, err := New(nil).User(context.Background(), "aaa")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, u.DisplayName, "Alice")
}

func TestUser_ExactEmailMatch(t *testing.T) {
	seedUsersCache(t, []api.User{
		{AccountID: "aaa", DisplayName: "Alice", EmailAddress: "alice@x.io"},
	})
	u, err := New(nil).User(context.Background(), "ALICE@X.IO")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, u.AccountID, "aaa")
}

func TestUser_DisplayNameMatch(t *testing.T) {
	seedUsersCache(t, []api.User{
		{AccountID: "aaa", DisplayName: "Alice Wonderland", EmailAddress: "alice@x.io"},
	})
	u, err := New(nil).User(context.Background(), "alice wonderland")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, u.AccountID, "aaa")
}

func TestUser_AmbiguousByName(t *testing.T) {
	seedUsersCache(t, []api.User{
		{AccountID: "111", DisplayName: "John Smith", EmailAddress: "john@a.io"},
		{AccountID: "222", DisplayName: "John Smith", EmailAddress: "john@b.io"},
	})
	_, err := New(nil).User(context.Background(), "John Smith")
	testutil.Error(t, err)

	var amb *AmbiguousMatchError
	if !errors.As(err, &amb) {
		t.Fatalf("expected AmbiguousMatchError, got %T: %v", err, err)
	}
	testutil.Equal(t, amb.Entity, "user")
	testutil.Equal(t, len(amb.Matches), 2)
	if !strings.Contains(err.Error(), "Ambiguous user") {
		t.Fatalf("error text missing preamble: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "account ID or email") {
		t.Fatalf("error text missing disambiguation hint: %q", err.Error())
	}
}

func TestUser_AccountIDPassThroughWhenNotInCache(t *testing.T) {
	// Cache populated but doesn't contain the input. Two-phase retry runs,
	// refresh fails (no client), caller expects pass-through via shape.
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Setenv("JIRA_URL", "https://test.atlassian.net")
	t.Setenv("JIRA_EMAIL", "t@example.com")
	t.Setenv("JIRA_API_TOKEN", "tok")
	testutil.RequireNoError(t, cache.WriteResource("users", "24h", []api.User{
		{AccountID: "someone-else", DisplayName: "Someone"},
	}))

	// Stub refresh: return a server that answers the users endpoint with an
	// empty page so the resolver concludes "no match".
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.User{})
	}))
	defer server.Close()
	client := newTestClient(t, server)

	input := "60e09bae7fcd820073089249" // 24-char hex, accountId shape
	u, err := New(client).User(context.Background(), input)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, u.AccountID, input)
}

func TestUser_NotFoundWithRefreshHint(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Setenv("JIRA_URL", "https://test.atlassian.net")
	t.Setenv("JIRA_EMAIL", "t@example.com")
	t.Setenv("JIRA_API_TOKEN", "tok")
	testutil.RequireNoError(t, cache.WriteResource("users", "24h", []api.User{
		{AccountID: "aaa", DisplayName: "Alice"},
	}))

	// Refresh returns no users — the cache stays empty after retry.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.User{})
	}))
	defer server.Close()
	client := newTestClient(t, server)

	_, err := New(client).User(context.Background(), "Zzznonexistent")
	testutil.Error(t, err)

	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("expected NotFoundError, got %T: %v", err, err)
	}
	testutil.Equal(t, nf.Entity, "user")
	if !strings.Contains(err.Error(), "jtk refresh users") {
		t.Fatalf("error text missing refresh hint: %q", err.Error())
	}
}

func TestUser_ColdCacheTriggersRefresh(t *testing.T) {
	// No cache file exists; resolver should auto-refresh once and find Alice.
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Setenv("JIRA_URL", "https://test.atlassian.net")
	t.Setenv("JIRA_EMAIL", "t@example.com")
	t.Setenv("JIRA_API_TOKEN", "tok")

	refreshCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/rest/api/3/users") {
			refreshCalls++
			_ = json.NewEncoder(w).Encode([]api.User{
				{AccountID: "aaa", DisplayName: "Alice Wonderland"},
			})
			return
		}
		t.Errorf("unexpected path: %s", r.URL.Path)
	}))
	defer server.Close()
	client := newTestClient(t, server)

	u, err := New(client).User(context.Background(), "Alice Wonderland")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, u.AccountID, "aaa")
	if refreshCalls == 0 {
		t.Fatalf("expected at least one refresh call")
	}
}
