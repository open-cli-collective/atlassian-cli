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

var testFields = []api.Field{
	{ID: "summary", Name: "Summary", Schema: api.FieldSchema{Type: "string"}},
	{ID: "customfield_10100", Name: "Story Points", Custom: true, Schema: api.FieldSchema{Type: "number"}},
}

// newFieldsTestClient builds a client pointed at server. The server call count
// is tracked externally by the caller (via request tracking in the handler).
func newFieldsTestClient(t *testing.T, serverURL string) *api.Client {
	t.Helper()
	client, err := api.New(api.ClientConfig{URL: serverURL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)
	return client
}

func TestGetFieldsCacheFirst_FreshCacheSkipsLive(t *testing.T) {
	// Non-parallel: SetRootForTest / SetInstanceKeyForTest are process-globals.
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("fields", "24h", testFields))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when cache is fresh")
	}))
	defer server.Close()

	client := newFieldsTestClient(t, server.URL)
	got, err := GetFieldsCacheFirst(context.Background(), client)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(got), len(testFields))
	testutil.Equal(t, got[0].ID, testFields[0].ID)
}

func TestGetFieldsCacheFirst_ManualRegistryTTL_SkipsLive(t *testing.T) {
	// GetFieldsCacheFirst uses registry TTL (not env.TTL) for Classify. Override
	// the "fields" entry to TTL "manual" so this test exercises StatusManual.
	t.Cleanup(SetEntriesForTest([]Entry{
		{Name: "fields", TTL: "manual"},
	}))
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("fields", "24h", testFields))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when registry TTL is manual")
	}))
	defer server.Close()

	client := newFieldsTestClient(t, server.URL)
	got, err := GetFieldsCacheFirst(context.Background(), client)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(got), len(testFields))
}

func TestGetFieldsCacheFirst_UninitializedEnvelopeFallsBackToLive(t *testing.T) {
	// StatusUninitialized is returned by Classify when FetchedAt.IsZero() —
	// e.g. an envelope written by a placeholder write before population.
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	uninitEnv := Envelope[[]api.Field]{
		Resource:  "fields",
		Instance:  "test.atlassian.net",
		FetchedAt: time.Time{}, // zero → StatusUninitialized
		TTL:       "24h",
		Version:   Version,
		Data:      testFields,
	}
	testutil.RequireNoError(t, atomicWriteEnvelope("fields", uninitEnv))

	liveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			liveCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"summary","name":"Summary"}]`))
		}
	}))
	defer server.Close()

	client := newFieldsTestClient(t, server.URL)
	got, err := GetFieldsCacheFirst(context.Background(), client)
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live API call for uninitialized (zero FetchedAt) envelope")
	}
	testutil.Equal(t, got[0].ID, "summary")
}

func TestGetFieldsCacheFirst_StaleCacheFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	// Write a stale envelope: FetchedAt set well in the past, TTL very short.
	staleEnv := Envelope[[]api.Field]{
		Resource:  "fields",
		Instance:  "test.atlassian.net",
		FetchedAt: time.Now().Add(-48 * time.Hour),
		TTL:       "1s",
		Version:   Version,
		Data:      []api.Field{{ID: "stale", Name: "Stale"}},
	}
	testutil.RequireNoError(t, atomicWriteEnvelope("fields", staleEnv))

	liveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			liveCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"summary","name":"Summary"}]`))
		}
	}))
	defer server.Close()

	client := newFieldsTestClient(t, server.URL)
	got, err := GetFieldsCacheFirst(context.Background(), client)
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live API call for stale cache")
	}
	testutil.Equal(t, got[0].ID, "summary")
}

func TestGetFieldsCacheFirst_MissFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))
	// No WriteResource call: cache miss.

	liveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			liveCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"summary","name":"Summary"}]`))
		}
	}))
	defer server.Close()

	client := newFieldsTestClient(t, server.URL)
	_, err := GetFieldsCacheFirst(context.Background(), client)
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live API call on cache miss")
	}
}

func TestGetFieldsCacheFirst_NoInstanceFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	// Explicitly unset JIRA_URL and related env vars so InstanceKey() reliably
	// returns ErrNoInstance regardless of the developer's shell environment.
	t.Setenv("JIRA_URL", "")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("JIRA_CLOUD_ID", "")
	t.Setenv("ATLASSIAN_CLOUD_ID", "")

	liveCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			liveCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"summary","name":"Summary"}]`))
		}
	}))
	defer server.Close()

	client := newFieldsTestClient(t, server.URL)
	_, err := GetFieldsCacheFirst(context.Background(), client)
	testutil.RequireNoError(t, err)
	if !liveCalled {
		t.Fatal("expected live API call when no instance is configured")
	}
}

func TestGetFieldsCacheFirst_LiveErrorPropagated(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))
	// Cache miss.

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"errorMessages":["server error"]}`)
	}))
	defer server.Close()

	client := newFieldsTestClient(t, server.URL)
	_, err := GetFieldsCacheFirst(context.Background(), client)
	if err == nil {
		t.Fatal("expected error from live API failure, got nil")
	}
}
