package cache

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestSeedUsers_PaginatesAndStops(t *testing.T) {
	cleanup := SetRootForTest(t.TempDir())
	defer cleanup()
	t.Setenv("JIRA_URL", "https://test.atlassian.net")
	t.Setenv("JIRA_EMAIL", "t@example.com")
	t.Setenv("JIRA_API_TOKEN", "tok")

	// Fixture: 250 users across 3 pages (100, 100, 50).
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/users")
		startAt, _ := strconv.Atoi(r.URL.Query().Get("startAt"))
		maxResults, _ := strconv.Atoi(r.URL.Query().Get("maxResults"))
		testutil.Equal(t, maxResults, seedUsersPageSize)

		var page []api.User
		end := startAt + maxResults
		if end > 250 {
			end = 250
		}
		for i := startAt; i < end; i++ {
			page = append(page, api.User{AccountID: "a" + strconv.Itoa(i), DisplayName: "u" + strconv.Itoa(i), Active: true})
		}
		calls++
		_ = json.NewEncoder(w).Encode(page)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: "https://test.atlassian.net", Email: "t@example.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)
	client.BaseURL = server.URL + "/rest/api/3"

	count, err := SeedUsers(context.Background(), client)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, count, 250)
	testutil.Equal(t, calls, 3) // 100 + 100 + 50 (50 < pageSize => stop)

	// Envelope was written correctly.
	env, err := ReadResource[[]api.User]("users")
	testutil.RequireNoError(t, err)
	testutil.Len(t, env.Data, 250)
	testutil.Equal(t, env.TTL, "24h")
	testutil.Equal(t, env.Data[0].AccountID, "a0")
	testutil.Equal(t, env.Data[249].AccountID, "a249")
}

func TestSeedUsers_EmptyInstance(t *testing.T) {
	cleanup := SetRootForTest(t.TempDir())
	defer cleanup()
	t.Setenv("JIRA_URL", "https://test.atlassian.net")
	t.Setenv("JIRA_EMAIL", "t@example.com")
	t.Setenv("JIRA_API_TOKEN", "tok")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("[]"))
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: "https://test.atlassian.net", Email: "t@example.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)
	client.BaseURL = server.URL + "/rest/api/3"

	count, err := SeedUsers(context.Background(), client)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, count, 0)

	env, err := ReadResource[[]api.User]("users")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(env.Data), 0)
}
