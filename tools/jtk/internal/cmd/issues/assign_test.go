package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/resolve"
)

// stubAssignServer returns a server that accepts the /assignee PUT and
// captures the accountId body (or nil body on unassign). Any other request
// is a test failure.
func stubAssignServer(t *testing.T, capture *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/assignee") || r.Method != http.MethodPut {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		switch v := payload["accountId"].(type) {
		case string:
			*capture = v
		case nil:
			*capture = "<null>"
		}
		w.WriteHeader(http.StatusNoContent)
	}))
}

func TestRunAssign_ResolvesDisplayName(t *testing.T) {
	seedCacheForIssues(t)

	var captured string
	server := stubAssignServer(t, &captured)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	// "User One" is seeded with AccountID=abc123 in seedCacheForIssues.
	err = runAssign(context.Background(), opts, "PROJ-123", "User One", false)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, captured, "abc123")
	testutil.Contains(t, stdout.String(), "User One")
}

func TestRunAssign_ResolvesByAccountID(t *testing.T) {
	seedCacheForIssues(t)

	var captured string
	server := stubAssignServer(t, &captured)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	// Raw accountId passes cache lookup directly.
	err = runAssign(context.Background(), opts, "PROJ-123", "61292e4c4f29230069621c5f", false)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, captured, "61292e4c4f29230069621c5f")
	// Display name resolved from cache for the message.
	testutil.Contains(t, stdout.String(), "Account User")
}

func TestRunAssign_SyntheticUserFallsBackToAccountID(t *testing.T) {
	// Empty users cache forces shape-based pass-through — no display-name
	// hit, so the presenter must fall back to the raw accountId.
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))

	var captured string
	server := stubAssignServer(t, &captured)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	rawID := "557058:295fe89c-10c2-4b0c-ba84-a4dd14ea7729"
	err = runAssign(context.Background(), opts, "PROJ-123", rawID, false)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, captured, rawID)
	// No cache hit → synthetic user → display name echoes the accountId.
	testutil.Contains(t, stdout.String(), rawID)
}

func TestRunAssign_Unassign(t *testing.T) {
	// Unassign must skip the resolver entirely and send a null accountId.
	// No cache seed → verifies we don't hit any resolver path.
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))

	var captured string
	server := stubAssignServer(t, &captured)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	// --unassign with an accountId arg: the flag wins and the resolver is
	// bypassed (see runAssign's guard). Confirms we don't accidentally resolve
	// the ignored positional.
	err = runAssign(context.Background(), opts, "PROJ-123", "some-ignored-id", true)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, captured, "<null>")
}

func TestRunAssign_ResolverNotFoundPropagates(t *testing.T) {
	// Non-shape input + empty users cache + refresh unreachable → resolver
	// surfaces a NotFoundError, runAssign must not silently assign.
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))
	testutil.RequireNoError(t, cache.WriteResource("users", "24h", []api.User{
		{AccountID: "aaa", DisplayName: "Alice"},
	}))

	// The bulk refresh endpoint is /rest/api/3/users (plural). Use an exact
	// path match so the assertion can't accidentally swallow a call to
	// /rest/api/3/user/search (singular) — those are distinct endpoints and
	// a live SearchUsers call for email-shaped input must not silently
	// resolve when this test claims a clean NotFoundError path.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/users" {
			_ = json.NewEncoder(w).Encode([]api.User{})
			return
		}
		t.Errorf("unexpected assign attempt: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	opts := &root.Options{Output: "table", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runAssign(context.Background(), opts, "PROJ-123", "Zzznonexistent", false)
	var nf *resolve.NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("expected NotFoundError, got %T: %v", err, err)
	}
	testutil.Equal(t, nf.Entity, "user")
}
