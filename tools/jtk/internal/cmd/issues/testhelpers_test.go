package issues

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
)

// seedCacheForIssues isolates the cache root to a temp dir and populates it
// with the minimum projects + issuetypes + users + linktypes entries needed
// to satisfy resolver lookups for the project keys and types commonly used
// across `issues` package tests.
//
// Tests that don't care about resolver behavior should call this to avoid
// accidental refresh attempts against the httptest server.
func seedCacheForIssues(t *testing.T) {
	t.Helper()
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, cache.WriteResource("projects", "24h", []api.Project{
		{Key: "PROJ", Name: "Project"},
		{Key: "MYPROJECT", Name: "My Project"},
		{Key: "MYPROJ", Name: "My Proj"},
		{Key: "TEST", Name: "Test"},
	}))

	stdTypes := []api.IssueType{
		{ID: "10000", Name: "Task"},
		{ID: "10001", Name: "Bug"},
		{ID: "10002", Name: "Story"},
		{ID: "10003", Name: "Epic"},
	}
	testutil.RequireNoError(t, cache.WriteResource("issuetypes", "24h", map[string][]api.IssueType{
		"PROJ":      stdTypes,
		"MYPROJECT": stdTypes,
		"MYPROJ":    stdTypes,
		"TEST":      stdTypes,
	}))

	testutil.RequireNoError(t, cache.WriteResource("users", "24h", []api.User{
		{AccountID: "self-me", DisplayName: "Test User", EmailAddress: "test@example.com"},
		{AccountID: "abc123", DisplayName: "User One", EmailAddress: "user1@example.com"},
		{AccountID: "61292e4c4f29230069621c5f", DisplayName: "Account User"},
	}))

	testutil.RequireNoError(t, cache.WriteResource("linktypes", "24h", []api.IssueLinkType{
		{ID: "10100", Name: "Blocks", Inward: "is blocked by", Outward: "blocks"},
		{ID: "10200", Name: "Relates", Inward: "relates to", Outward: "relates to"},
	}))
}
