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

var testIssueTypes = map[string][]api.IssueType{
	"TEST": {
		{ID: "10001", Name: "Bug", Description: "A problem", Subtask: false},
		{ID: "10002", Name: "Task", Description: "A task to do", Subtask: false},
	},
	"OTHER": {
		{ID: "10003", Name: "Story", Description: "A user story", Subtask: false},
	},
}

func newIssueTypesTestClient(t *testing.T, serverURL string) *api.Client {
	t.Helper()
	client, err := api.New(api.ClientConfig{URL: serverURL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)
	return client
}

func issueTypesLiveServer(t *testing.T, projectKey string, types []api.IssueType) (*httptest.Server, *bool) {
	t.Helper()
	liveCalled := new(bool)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := fmt.Sprintf("/rest/api/3/project/%s", projectKey)
		if r.URL.Path == expectedPath {
			*liveCalled = true
			w.Header().Set("Content-Type", "application/json")
			resp := struct {
				IssueTypes []api.IssueType `json:"issueTypes"`
			}{IssueTypes: types}
			_ = json.NewEncoder(w).Encode(resp)
		}
	}))
	return server, liveCalled
}

func TestGetIssueTypesCacheFirst_FreshCacheSkipsLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("issuetypes", "24h", testIssueTypes))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when cache is fresh")
	}))
	defer server.Close()

	client := newIssueTypesTestClient(t, server.URL)
	got, err := GetIssueTypesCacheFirst(context.Background(), client, "TEST")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(got), 2)
	testutil.Equal(t, got[0].Name, "Bug")
	testutil.Equal(t, got[1].Name, "Task")
}

func TestGetIssueTypesCacheFirst_FreshCacheMissingProjectFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("issuetypes", "24h", testIssueTypes))

	liveTypes := []api.IssueType{{ID: "99", Name: "NewType"}}
	server, liveCalled := issueTypesLiveServer(t, "NEWPROJ", liveTypes)
	defer server.Close()

	client := newIssueTypesTestClient(t, server.URL)
	got, err := GetIssueTypesCacheFirst(context.Background(), client, "NEWPROJ")
	testutil.RequireNoError(t, err)
	if !*liveCalled {
		t.Fatal("expected live API call when project key not in fresh cache")
	}
	testutil.Equal(t, len(got), 1)
	testutil.Equal(t, got[0].Name, "NewType")
}

func TestGetIssueTypesCacheFirst_FreshCacheEmptyProjectReturnsEmpty(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	cacheData := map[string][]api.IssueType{
		"EMPTY": {},
	}
	testutil.RequireNoError(t, WriteResource("issuetypes", "24h", cacheData))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when project exists in fresh cache (even if empty)")
	}))
	defer server.Close()

	client := newIssueTypesTestClient(t, server.URL)
	got, err := GetIssueTypesCacheFirst(context.Background(), client, "EMPTY")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(got), 0)
}

func TestGetIssueTypesCacheFirst_ManualRegistryTTL_SkipsLive(t *testing.T) {
	t.Cleanup(SetEntriesForTest([]Entry{
		{Name: "issuetypes", TTL: "manual"},
	}))
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, WriteResource("issuetypes", "24h", testIssueTypes))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when registry TTL is manual")
	}))
	defer server.Close()

	client := newIssueTypesTestClient(t, server.URL)
	got, err := GetIssueTypesCacheFirst(context.Background(), client, "TEST")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(got), 2)
}

func TestGetIssueTypesCacheFirst_StaleCacheFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	staleEnv := Envelope[map[string][]api.IssueType]{
		Resource:  "issuetypes",
		Instance:  "test.atlassian.net",
		FetchedAt: time.Now().Add(-48 * time.Hour),
		TTL:       "1s",
		Version:   Version,
		Data:      testIssueTypes,
	}
	testutil.RequireNoError(t, atomicWriteEnvelope("issuetypes", staleEnv))

	liveTypes := []api.IssueType{{ID: "10001", Name: "Bug"}}
	server, liveCalled := issueTypesLiveServer(t, "TEST", liveTypes)
	defer server.Close()

	client := newIssueTypesTestClient(t, server.URL)
	got, err := GetIssueTypesCacheFirst(context.Background(), client, "TEST")
	testutil.RequireNoError(t, err)
	if !*liveCalled {
		t.Fatal("expected live API call for stale cache")
	}
	testutil.Equal(t, got[0].ID, "10001")
}

func TestGetIssueTypesCacheFirst_MissFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	liveTypes := []api.IssueType{{ID: "10001", Name: "Bug"}}
	server, liveCalled := issueTypesLiveServer(t, "TEST", liveTypes)
	defer server.Close()

	client := newIssueTypesTestClient(t, server.URL)
	_, err := GetIssueTypesCacheFirst(context.Background(), client, "TEST")
	testutil.RequireNoError(t, err)
	if !*liveCalled {
		t.Fatal("expected live API call on cache miss")
	}
}

func TestGetIssueTypesCacheFirst_UninitializedEnvelopeFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	uninitEnv := Envelope[map[string][]api.IssueType]{
		Resource:  "issuetypes",
		Instance:  "test.atlassian.net",
		FetchedAt: time.Time{},
		TTL:       "24h",
		Version:   Version,
		Data:      testIssueTypes,
	}
	testutil.RequireNoError(t, atomicWriteEnvelope("issuetypes", uninitEnv))

	liveTypes := []api.IssueType{{ID: "10001", Name: "Bug"}}
	server, liveCalled := issueTypesLiveServer(t, "TEST", liveTypes)
	defer server.Close()

	client := newIssueTypesTestClient(t, server.URL)
	got, err := GetIssueTypesCacheFirst(context.Background(), client, "TEST")
	testutil.RequireNoError(t, err)
	if !*liveCalled {
		t.Fatal("expected live API call for uninitialized (zero FetchedAt) envelope")
	}
	testutil.Equal(t, got[0].ID, "10001")
}

func TestGetIssueTypesCacheFirst_NoInstanceFallsBackToLive(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Setenv("JIRA_URL", "")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("JIRA_CLOUD_ID", "")
	t.Setenv("ATLASSIAN_CLOUD_ID", "")

	liveTypes := []api.IssueType{{ID: "10001", Name: "Bug"}}
	server, liveCalled := issueTypesLiveServer(t, "TEST", liveTypes)
	defer server.Close()

	client := newIssueTypesTestClient(t, server.URL)
	_, err := GetIssueTypesCacheFirst(context.Background(), client, "TEST")
	testutil.RequireNoError(t, err)
	if !*liveCalled {
		t.Fatal("expected live API call when no instance is configured")
	}
}

func TestGetIssueTypesCacheFirst_LiveFallbackSendsExpand(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	var expandParam string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expandParam = r.URL.Query().Get("expand")
		w.Header().Set("Content-Type", "application/json")
		resp := struct {
			IssueTypes []api.IssueType `json:"issueTypes"`
		}{IssueTypes: []api.IssueType{{ID: "1", Name: "Bug"}}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newIssueTypesTestClient(t, server.URL)
	_, err := GetIssueTypesCacheFirst(context.Background(), client, "TEST")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, expandParam, "issueTypes")
}

func TestGetIssueTypesCacheFirst_LiveErrorPropagated(t *testing.T) {
	t.Cleanup(SetRootForTest(t.TempDir()))
	t.Cleanup(SetInstanceKeyForTest("test.atlassian.net"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"errorMessages":["server error"]}`)
	}))
	defer server.Close()

	client := newIssueTypesTestClient(t, server.URL)
	_, err := GetIssueTypesCacheFirst(context.Background(), client, "TEST")
	if err == nil {
		t.Fatal("expected error from live API failure, got nil")
	}
}
