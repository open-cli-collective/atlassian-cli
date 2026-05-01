package cache

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

var testSprints = map[int][]api.Sprint{
	1: {
		{ID: 100, Name: "Sprint 1", State: "closed", OriginBoardID: 1},
		{ID: 101, Name: "Sprint 2", State: "active", OriginBoardID: 1},
		{ID: 102, Name: "Sprint 3", State: "future", OriginBoardID: 1},
	},
	2: {
		{ID: 200, Name: "Sprint A", State: "active", OriginBoardID: 2},
	},
}

func stubFetcher(liveCalled *bool, result []api.Sprint) func(context.Context, *api.Client, int, string) ([]api.Sprint, error) {
	return func(_ context.Context, _ *api.Client, _ int, _ string) ([]api.Sprint, error) {
		if liveCalled != nil {
			*liveCalled = true
		}
		return result, nil
	}
}

func stubFetcherErr(liveCalled *bool) func(context.Context, *api.Client, int, string) ([]api.Sprint, error) {
	return func(_ context.Context, _ *api.Client, _ int, _ string) ([]api.Sprint, error) {
		if liveCalled != nil {
			*liveCalled = true
		}
		return nil, fmt.Errorf("live error")
	}
}

func newSprintsTestClient(t *testing.T, serverURL string) *api.Client {
	t.Helper()
	client, err := api.New(api.ClientConfig{URL: serverURL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)
	return client
}

func TestGetSprintsCacheFirst_FreshCacheSkipsLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("sprints", "24h", testSprints))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when cache is fresh")
	}))
	defer server.Close()

	client := newSprintsTestClient(t, server.URL)
	liveCalled := false
	got, err := GetSprintsCacheFirst(context.Background(), client, 1, "", stubFetcher(&liveCalled, nil))
	testutil.RequireNoError(t, err)
	if liveCalled {
		t.Fatal("live fetcher should not be called when cache is fresh")
	}
	testutil.Equal(t, len(got), 3)
}

func TestGetSprintsCacheFirst_FreshCacheWithStateFilter(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("sprints", "24h", testSprints))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when cache is fresh")
	}))
	defer server.Close()

	client := newSprintsTestClient(t, server.URL)
	got, err := GetSprintsCacheFirst(context.Background(), client, 1, "active", stubFetcher(nil, nil))
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(got), 1)
	testutil.Equal(t, got[0].Name, "Sprint 2")
}

func TestGetSprintsCacheFirst_CommaStateFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("sprints", "24h", testSprints))

	liveCalled := false
	liveResult := []api.Sprint{{ID: 101, Name: "Sprint 2", State: "active"}, {ID: 102, Name: "Sprint 3", State: "future"}}

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not hit httptest server")
	}))
	defer server.Close()

	client := newSprintsTestClient(t, server.URL)
	got, err := GetSprintsCacheFirst(context.Background(), client, 1, "active,future", stubFetcher(&liveCalled, liveResult))
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live fetcher call for comma-separated state")
	}
	testutil.Equal(t, len(got), 2)
}

func TestGetSprintsCacheFirst_MixedCaseStateServedFromCache(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("sprints", "24h", testSprints))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when cache is fresh")
	}))
	defer server.Close()

	client := newSprintsTestClient(t, server.URL)
	got, err := GetSprintsCacheFirst(context.Background(), client, 1, "Active", stubFetcher(nil, nil))
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(got), 1)
	testutil.Equal(t, got[0].Name, "Sprint 2")
}

func TestGetSprintsCacheFirst_UnknownStateFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("sprints", "24h", testSprints))

	liveCalled := false
	liveResult := []api.Sprint{{ID: 101, Name: "Sprint 2", State: "active"}}

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not hit httptest server")
	}))
	defer server.Close()

	client := newSprintsTestClient(t, server.URL)
	got, err := GetSprintsCacheFirst(context.Background(), client, 1, "typo", stubFetcher(&liveCalled, liveResult))
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live fetcher call for unknown state value")
	}
	testutil.Equal(t, len(got), 1)
}

func TestGetSprintsCacheFirst_FreshCacheMissingBoardFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("sprints", "24h", testSprints))

	liveCalled := false
	liveResult := []api.Sprint{{ID: 999, Name: "Live Sprint"}}

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not hit httptest server")
	}))
	defer server.Close()

	client := newSprintsTestClient(t, server.URL)
	got, err := GetSprintsCacheFirst(context.Background(), client, 999, "", stubFetcher(&liveCalled, liveResult))
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live fetcher call when board ID not in cache")
	}
	testutil.Equal(t, got[0].Name, "Live Sprint")
}

func TestGetSprintsCacheFirst_FreshCacheEmptyBoardReturnsEmpty(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	cacheData := map[int][]api.Sprint{
		99: {},
	}
	testutil.RequireNoError(t, WriteResource("sprints", "24h", cacheData))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when board exists in fresh cache (even if empty)")
	}))
	defer server.Close()

	client := newSprintsTestClient(t, server.URL)
	got, err := GetSprintsCacheFirst(context.Background(), client, 99, "", stubFetcher(nil, nil))
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(got), 0)
}

func TestGetSprintsCacheFirst_ManualRegistryTTL_SkipsLive(t *testing.T) {
	t.Cleanup(SetEntriesForTest([]Entry{
		{Name: "sprints", TTL: "manual"},
	}))
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("sprints", "24h", testSprints))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when registry TTL is manual")
	}))
	defer server.Close()

	client := newSprintsTestClient(t, server.URL)
	got, err := GetSprintsCacheFirst(context.Background(), client, 1, "", stubFetcher(nil, nil))
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(got), 3)
}

func TestGetSprintsCacheFirst_StaleCacheFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	staleEnv := Envelope[map[int][]api.Sprint]{
		Resource:  "sprints",
		Instance:  "test.atlassian.net",
		FetchedAt: time.Now().Add(-48 * time.Hour),
		TTL:       "1s",
		Version:   Version,
		Data:      testSprints,
	}
	testutil.RequireNoError(t, atomicWriteEnvelope("sprints", staleEnv))

	liveCalled := false
	liveResult := []api.Sprint{{ID: 101, Name: "Live Sprint"}}

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not hit httptest server")
	}))
	defer server.Close()

	client := newSprintsTestClient(t, server.URL)
	got, err := GetSprintsCacheFirst(context.Background(), client, 1, "", stubFetcher(&liveCalled, liveResult))
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live fetcher call for stale cache")
	}
	testutil.Equal(t, got[0].Name, "Live Sprint")
}

func TestGetSprintsCacheFirst_MissFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	liveCalled := false
	liveResult := []api.Sprint{{ID: 101, Name: "Live Sprint"}}

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not hit httptest server")
	}))
	defer server.Close()

	client := newSprintsTestClient(t, server.URL)
	got, err := GetSprintsCacheFirst(context.Background(), client, 1, "", stubFetcher(&liveCalled, liveResult))
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live fetcher call on cache miss")
	}
	testutil.Equal(t, got[0].Name, "Live Sprint")
}

func TestGetSprintsCacheFirst_NoInstanceFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Setenv("JIRA_URL", "")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("JIRA_CLOUD_ID", "")
	t.Setenv("ATLASSIAN_CLOUD_ID", "")

	liveCalled := false
	liveResult := []api.Sprint{{ID: 101, Name: "Live Sprint"}}

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not hit httptest server")
	}))
	defer server.Close()

	client := newSprintsTestClient(t, server.URL)
	got, err := GetSprintsCacheFirst(context.Background(), client, 1, "", stubFetcher(&liveCalled, liveResult))
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live fetcher call when no instance is configured")
	}
	testutil.Equal(t, got[0].Name, "Live Sprint")
}

func TestGetSprintsCacheFirst_LiveErrorPropagated(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not hit httptest server")
	}))
	defer server.Close()

	client := newSprintsTestClient(t, server.URL)
	liveCalled := false
	_, err := GetSprintsCacheFirst(context.Background(), client, 1, "", stubFetcherErr(&liveCalled))
	if err == nil {
		t.Fatal("expected error from live fetcher failure, got nil")
	}
	if !liveCalled {
		t.Fatal("expected live fetcher to be called")
	}
}
