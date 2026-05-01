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

var testUsers = []api.User{
	{AccountID: "abc123", DisplayName: "Alice", EmailAddress: "alice@example.com", Active: true},
	{AccountID: "def456", DisplayName: "Bob", EmailAddress: "bob@example.com", Active: true},
}

func newUsersTestClient(t *testing.T, serverURL string) *api.Client {
	t.Helper()
	client, err := api.New(api.ClientConfig{URL: serverURL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)
	return client
}

func TestGetUserCacheFirst_FreshCacheSkipsLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("users", "24h", testUsers))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when cache is fresh")
	}))
	defer server.Close()

	client := newUsersTestClient(t, server.URL)
	got, err := GetUserCacheFirst(context.Background(), client, "abc123", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, got.AccountID, "abc123")
	testutil.Equal(t, got.DisplayName, "Alice")
}

func TestGetUserCacheFirst_ExpandAlwaysGoesLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("users", "24h", testUsers))

	liveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/user" {
			liveCalled = true
			testutil.Equal(t, r.URL.Query().Get("expand"), "groups,applicationRoles")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.User{
				AccountID:   "abc123",
				DisplayName: "Alice",
				Groups:      &api.UserCountBlock{Size: 3},
			})
		}
	}))
	defer server.Close()

	client := newUsersTestClient(t, server.URL)
	got, err := GetUserCacheFirst(context.Background(), client, "abc123", "groups,applicationRoles")
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live API call when expand is non-empty")
	}
	testutil.Equal(t, got.Groups.Size, 3)
}

func TestGetUserCacheFirst_FreshCacheMissingUserFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("users", "24h", testUsers))

	liveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/user" {
			liveCalled = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.User{AccountID: "unknown999", DisplayName: "Unknown"})
		}
	}))
	defer server.Close()

	client := newUsersTestClient(t, server.URL)
	got, err := GetUserCacheFirst(context.Background(), client, "unknown999", "")
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live API call when user not in cache")
	}
	testutil.Equal(t, got.DisplayName, "Unknown")
}

func TestGetUserCacheFirst_ManualRegistryTTL_SkipsLive(t *testing.T) {
	t.Cleanup(SetEntriesForTest([]Entry{
		{Name: "users", TTL: "manual"},
	}))
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("users", "24h", testUsers))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when registry TTL is manual")
	}))
	defer server.Close()

	client := newUsersTestClient(t, server.URL)
	got, err := GetUserCacheFirst(context.Background(), client, "abc123", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, got.AccountID, "abc123")
}

func TestGetUserCacheFirst_StaleCacheFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	staleEnv := Envelope[[]api.User]{
		Resource:  "users",
		Instance:  "test.atlassian.net",
		FetchedAt: time.Now().Add(-48 * time.Hour),
		TTL:       "1s",
		Version:   Version,
		Data:      testUsers,
	}
	testutil.RequireNoError(t, atomicWriteEnvelope("users", staleEnv))

	liveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/user" {
			liveCalled = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.User{AccountID: "abc123", DisplayName: "Alice"})
		}
	}))
	defer server.Close()

	client := newUsersTestClient(t, server.URL)
	_, err := GetUserCacheFirst(context.Background(), client, "abc123", "")
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live API call for stale cache")
	}
}

func TestGetUserCacheFirst_UninitializedEnvelopeFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	uninitEnv := Envelope[[]api.User]{
		Resource:  "users",
		Instance:  "test.atlassian.net",
		FetchedAt: time.Time{},
		TTL:       "24h",
		Version:   Version,
		Data:      testUsers,
	}
	testutil.RequireNoError(t, atomicWriteEnvelope("users", uninitEnv))

	liveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/user" {
			liveCalled = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.User{AccountID: "abc123", DisplayName: "Alice"})
		}
	}))
	defer server.Close()

	client := newUsersTestClient(t, server.URL)
	_, err := GetUserCacheFirst(context.Background(), client, "abc123", "")
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live API call for uninitialized envelope")
	}
}

func TestGetUserCacheFirst_MissFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	liveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/user" {
			liveCalled = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.User{AccountID: "abc123", DisplayName: "Alice"})
		}
	}))
	defer server.Close()

	client := newUsersTestClient(t, server.URL)
	_, err := GetUserCacheFirst(context.Background(), client, "abc123", "")
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live API call on cache miss")
	}
}

func TestGetUserCacheFirst_NoInstanceFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Setenv("JIRA_URL", "")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("JIRA_CLOUD_ID", "")
	t.Setenv("ATLASSIAN_CLOUD_ID", "")

	liveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/user" {
			liveCalled = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.User{AccountID: "abc123", DisplayName: "Alice"})
		}
	}))
	defer server.Close()

	client := newUsersTestClient(t, server.URL)
	_, err := GetUserCacheFirst(context.Background(), client, "abc123", "")
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live API call when no instance is configured")
	}
}

func TestGetUserCacheFirst_LiveErrorPropagated(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"errorMessages":["server error"]}`)
	}))
	defer server.Close()

	client := newUsersTestClient(t, server.URL)
	_, err := GetUserCacheFirst(context.Background(), client, "abc123", "")
	if err == nil {
		t.Fatal("expected error from live API failure, got nil")
	}
}
