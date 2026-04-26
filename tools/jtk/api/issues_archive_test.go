package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestArchiveIssues_Success(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, "PUT")
		testutil.Equal(t, r.URL.Path, "/rest/api/3/issue/archive")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ArchiveResult{NumberUpdated: 2})
	}))
	defer server.Close()

	client, _ := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	result, err := client.ArchiveIssues(context.Background(), []string{"TEST-1", "TEST-2"})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, result.NumberUpdated, 2)
	testutil.Equal(t, len(result.Errors), 0)
}

func TestArchiveIssues_PartialFailure(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ArchiveResult{
			NumberUpdated: 1,
			Errors: map[string]ArchiveError{
				"PERM": {Count: 1, IssueIdsOrKeys: []string{"TEST-2"}, Message: "denied"},
			},
		})
	}))
	defer server.Close()

	client, _ := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	result, err := client.ArchiveIssues(context.Background(), []string{"TEST-1", "TEST-2"})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, result.NumberUpdated, 1)
	testutil.Equal(t, len(result.Errors), 1)
	testutil.Equal(t, result.Errors["PERM"].IssueIdsOrKeys[0], "TEST-2")
}

func TestArchiveIssues_EmptyKeys(t *testing.T) {
	t.Parallel()
	client, _ := New(ClientConfig{URL: "http://unused", Email: "t@t.com", APIToken: "tok"})
	_, err := client.ArchiveIssues(context.Background(), nil)
	testutil.NotNil(t, err)
}

func TestGetWatchers_Success(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(WatchersInfo{WatchCount: 3, IsWatching: true})
	}))
	defer server.Close()

	client, _ := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	info, err := client.GetWatchers(context.Background(), "TEST-1")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, info.WatchCount, 3)
	testutil.Equal(t, info.IsWatching, true)
}

func TestGetWatchers_EmptyKey(t *testing.T) {
	t.Parallel()
	client, _ := New(ClientConfig{URL: "http://unused", Email: "t@t.com", APIToken: "tok"})
	_, err := client.GetWatchers(context.Background(), "")
	testutil.NotNil(t, err)
}
