package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

var testBoards = []api.Board{
	{ID: 1, Name: "Board One", Type: "scrum", Location: api.BoardLocation{ProjectKey: "PROJ"}},
	{ID: 2, Name: "Board Two", Type: "kanban", Location: api.BoardLocation{ProjectKey: "PROJ"}},
	{ID: 3, Name: "Board Three", Type: "scrum", Location: api.BoardLocation{ProjectKey: "OTHER"}},
}

func newBoardsTestClient(t *testing.T, serverURL string) *api.Client {
	t.Helper()
	client, err := api.New(api.ClientConfig{URL: serverURL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)
	return client
}

func TestGetBoardsCacheFirst_FreshCacheSkipsLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("boards", "24h", testBoards))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when cache is fresh")
	}))
	defer server.Close()

	client := newBoardsTestClient(t, server.URL)
	got, err := GetBoardsCacheFirst(context.Background(), client, "", 0, 50)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(got.Values), 3)
	testutil.Equal(t, got.IsLast, true)
}

func TestGetBoardsCacheFirst_FreshCacheWithProjectFilter(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("boards", "24h", testBoards))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when cache is fresh")
	}))
	defer server.Close()

	client := newBoardsTestClient(t, server.URL)
	got, err := GetBoardsCacheFirst(context.Background(), client, "PROJ", 0, 50)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(got.Values), 2)
	testutil.Equal(t, got.Values[0].Name, "Board One")
	testutil.Equal(t, got.Values[1].Name, "Board Two")
	testutil.Equal(t, got.IsLast, true)
}

func TestGetBoardsCacheFirst_FreshCacheWithPagination(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("boards", "24h", testBoards))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when cache is fresh")
	}))
	defer server.Close()

	client := newBoardsTestClient(t, server.URL)

	// First page: 2 results
	got, err := GetBoardsCacheFirst(context.Background(), client, "", 0, 2)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(got.Values), 2)
	testutil.Equal(t, got.IsLast, false)

	// Second page: 1 result
	got2, err := GetBoardsCacheFirst(context.Background(), client, "", 2, 2)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(got2.Values), 1)
	testutil.Equal(t, got2.IsLast, true)
}

func TestGetBoardsCacheFirst_FreshCacheProjectFilterWithPagination(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("boards", "24h", testBoards))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when cache is fresh")
	}))
	defer server.Close()

	client := newBoardsTestClient(t, server.URL)
	got, err := GetBoardsCacheFirst(context.Background(), client, "PROJ", 0, 1)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(got.Values), 1)
	testutil.Equal(t, got.Values[0].Name, "Board One")
	testutil.Equal(t, got.IsLast, false)
}

func TestGetBoardsCacheFirst_FreshCacheStartAtBeyondTotal(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("boards", "24h", testBoards))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when cache is fresh")
	}))
	defer server.Close()

	client := newBoardsTestClient(t, server.URL)
	got, err := GetBoardsCacheFirst(context.Background(), client, "", 100, 50)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(got.Values), 0)
	testutil.Equal(t, got.IsLast, true)
}

func TestGetBoardsCacheFirst_FreshCacheMaxResultsZeroUsesDefault(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("boards", "24h", testBoards))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when cache is fresh")
	}))
	defer server.Close()

	client := newBoardsTestClient(t, server.URL)
	got, err := GetBoardsCacheFirst(context.Background(), client, "", 0, 0)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(got.Values), 3)
	testutil.Equal(t, got.IsLast, true)
}

func TestGetBoardsCacheFirst_ManualRegistryTTL_SkipsLive(t *testing.T) {
	t.Cleanup(SetEntriesForTest([]Entry{
		{Name: "boards", TTL: "manual"},
	}))
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("boards", "24h", testBoards))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when registry TTL is manual")
	}))
	defer server.Close()

	client := newBoardsTestClient(t, server.URL)
	got, err := GetBoardsCacheFirst(context.Background(), client, "", 0, 50)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(got.Values), 3)
}

func TestGetBoardsCacheFirst_StaleCacheFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	staleEnv := Envelope[[]api.Board]{
		Resource:  "boards",
		Instance:  "test.atlassian.net",
		FetchedAt: time.Now().Add(-48 * time.Hour),
		TTL:       "1s",
		Version:   Version,
		Data:      testBoards,
	}
	testutil.RequireNoError(t, atomicWriteEnvelope("boards", staleEnv))

	liveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/agile/1.0/board" {
			liveCalled = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.BoardsResponse{
				IsLast: true,
				Values: []api.Board{{ID: 99, Name: "Live Board"}},
			})
		}
	}))
	defer server.Close()

	client := newBoardsTestClient(t, server.URL)
	got, err := GetBoardsCacheFirst(context.Background(), client, "", 0, 50)
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live API call for stale cache")
	}
	testutil.Equal(t, got.Values[0].Name, "Live Board")
}

func TestGetBoardsCacheFirst_MissFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	liveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/agile/1.0/board" {
			liveCalled = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.BoardsResponse{IsLast: true, Values: []api.Board{{ID: 1, Name: "B"}}})
		}
	}))
	defer server.Close()

	client := newBoardsTestClient(t, server.URL)
	_, err := GetBoardsCacheFirst(context.Background(), client, "", 0, 50)
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live API call on cache miss")
	}
}

func TestGetBoardsCacheFirst_UninitializedEnvelopeFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	uninitEnv := Envelope[[]api.Board]{
		Resource:  "boards",
		Instance:  "test.atlassian.net",
		FetchedAt: time.Time{},
		TTL:       "24h",
		Version:   Version,
		Data:      testBoards,
	}
	testutil.RequireNoError(t, atomicWriteEnvelope("boards", uninitEnv))

	liveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/agile/1.0/board" {
			liveCalled = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.BoardsResponse{IsLast: true, Values: []api.Board{{ID: 1, Name: "B"}}})
		}
	}))
	defer server.Close()

	client := newBoardsTestClient(t, server.URL)
	_, err := GetBoardsCacheFirst(context.Background(), client, "", 0, 50)
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live API call for uninitialized envelope")
	}
}

func TestGetBoardsCacheFirst_NoInstanceFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Setenv("JIRA_URL", "")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("JIRA_CLOUD_ID", "")
	t.Setenv("ATLASSIAN_CLOUD_ID", "")

	liveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/agile/1.0/board" {
			liveCalled = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.BoardsResponse{IsLast: true, Values: []api.Board{{ID: 1, Name: "B"}}})
		}
	}))
	defer server.Close()

	client := newBoardsTestClient(t, server.URL)
	_, err := GetBoardsCacheFirst(context.Background(), client, "", 0, 50)
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live API call when no instance is configured")
	}
}

func TestGetBoardsCacheFirst_LiveErrorPropagated(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"errorMessages":["server error"]}`)
	}))
	defer server.Close()

	client := newBoardsTestClient(t, server.URL)
	_, err := GetBoardsCacheFirst(context.Background(), client, "", 0, 50)
	if err == nil {
		t.Fatal("expected error from live API failure, got nil")
	}
}
