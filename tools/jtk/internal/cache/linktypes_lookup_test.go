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

var testLinkTypes = []api.IssueLinkType{
	{ID: "1", Name: "Blocks", Inward: "is blocked by", Outward: "blocks"},
	{ID: "2", Name: "Relates", Inward: "relates to", Outward: "relates to"},
}

func newLinkTypesTestClient(t *testing.T, serverURL string) *api.Client {
	t.Helper()
	client, err := api.New(api.ClientConfig{URL: serverURL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)
	return client
}

func TestGetLinkTypesCacheFirst_FreshCacheSkipsLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("linktypes", "24h", testLinkTypes))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when cache is fresh")
	}))
	defer server.Close()

	client := newLinkTypesTestClient(t, server.URL)
	got, err := GetLinkTypesCacheFirst(context.Background(), client)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(got), len(testLinkTypes))
	testutil.Equal(t, got[0].ID, "1")
	testutil.Equal(t, got[0].Name, "Blocks")
	testutil.Equal(t, got[1].Name, "Relates")
}

func TestGetLinkTypesCacheFirst_ManualRegistryTTL_SkipsLive(t *testing.T) {
	t.Cleanup(SetEntriesForTest([]Entry{
		{Name: "linktypes", TTL: "manual"},
	}))
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("linktypes", "24h", testLinkTypes))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when registry TTL is manual")
	}))
	defer server.Close()

	client := newLinkTypesTestClient(t, server.URL)
	got, err := GetLinkTypesCacheFirst(context.Background(), client)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(got), len(testLinkTypes))
}

func TestGetLinkTypesCacheFirst_StaleCacheFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	staleEnv := Envelope[[]api.IssueLinkType]{
		Resource:  "linktypes",
		Instance:  "test.atlassian.net",
		FetchedAt: time.Now().Add(-48 * time.Hour),
		TTL:       "1s",
		Version:   Version,
		Data:      []api.IssueLinkType{{ID: "stale", Name: "Stale"}},
	}
	testutil.RequireNoError(t, atomicWriteEnvelope("linktypes", staleEnv))

	liveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issueLinkType" {
			liveCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"issueLinkTypes":[{"id":"1","name":"Blocks","inward":"is blocked by","outward":"blocks"}]}`))
		}
	}))
	defer server.Close()

	client := newLinkTypesTestClient(t, server.URL)
	got, err := GetLinkTypesCacheFirst(context.Background(), client)
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live API call for stale cache")
	}
	testutil.Equal(t, got[0].ID, "1")
}

func TestGetLinkTypesCacheFirst_MissFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	liveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issueLinkType" {
			liveCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"issueLinkTypes":[{"id":"1","name":"Blocks","inward":"is blocked by","outward":"blocks"}]}`))
		}
	}))
	defer server.Close()

	client := newLinkTypesTestClient(t, server.URL)
	_, err := GetLinkTypesCacheFirst(context.Background(), client)
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live API call on cache miss")
	}
}

func TestGetLinkTypesCacheFirst_NoInstanceFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Setenv("JIRA_URL", "")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("JIRA_CLOUD_ID", "")
	t.Setenv("ATLASSIAN_CLOUD_ID", "")

	liveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issueLinkType" {
			liveCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"issueLinkTypes":[{"id":"1","name":"Blocks","inward":"is blocked by","outward":"blocks"}]}`))
		}
	}))
	defer server.Close()

	client := newLinkTypesTestClient(t, server.URL)
	_, err := GetLinkTypesCacheFirst(context.Background(), client)
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live API call when no instance is configured")
	}
}

func TestGetLinkTypesCacheFirst_UninitializedEnvelopeFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	uninitEnv := Envelope[[]api.IssueLinkType]{
		Resource:  "linktypes",
		Instance:  "test.atlassian.net",
		FetchedAt: time.Time{},
		TTL:       "24h",
		Version:   Version,
		Data:      testLinkTypes,
	}
	testutil.RequireNoError(t, atomicWriteEnvelope("linktypes", uninitEnv))

	liveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issueLinkType" {
			liveCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"issueLinkTypes":[{"id":"1","name":"Blocks","inward":"is blocked by","outward":"blocks"}]}`))
		}
	}))
	defer server.Close()

	client := newLinkTypesTestClient(t, server.URL)
	got, err := GetLinkTypesCacheFirst(context.Background(), client)
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live API call for uninitialized (zero FetchedAt) envelope")
	}
	testutil.Equal(t, got[0].ID, "1")
}

func TestGetLinkTypesCacheFirst_LiveErrorPropagated(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"errorMessages":["server error"]}`)
	}))
	defer server.Close()

	client := newLinkTypesTestClient(t, server.URL)
	_, err := GetLinkTypesCacheFirst(context.Background(), client)
	if err == nil {
		t.Fatal("expected error from live API failure, got nil")
	}
}
