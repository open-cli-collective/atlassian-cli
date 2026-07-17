package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func makeIssues(start, count int) []Issue {
	issues := make([]Issue, count)
	for i := range count {
		issues[i] = Issue{Key: fmt.Sprintf("TEST-%d", start+i)}
	}
	return issues
}

func TestSearchPageRequestsOnePage(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	client, server := newTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		var req SearchRequest
		testutil.RequireNoError(t, json.NewDecoder(r.Body).Decode(&req))
		testutil.Equal(t, 50, req.MaxResults)
		testutil.Equal(t, "", req.NextPageToken)
		_ = json.NewEncoder(w).Encode(JQLSearchResult{
			Issues: makeIssues(1, 50), NextPageToken: "page-2", IsLast: false,
		})
	})
	defer server.Close()

	result, err := client.SearchPage(context.Background(), SearchPageOptions{
		JQL: "project = TEST", MaxResults: 50,
	})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, int32(1), requests.Load())
	testutil.Equal(t, 50, len(result.Issues))
	testutil.False(t, result.Pagination.IsLast)
	testutil.Equal(t, "page-2", result.Pagination.NextPageToken)
}

func TestSearchPagePassesSubsequentPageTokenUnchanged(t *testing.T) {
	t.Parallel()
	client, server := newTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req SearchRequest
		testutil.RequireNoError(t, json.NewDecoder(r.Body).Decode(&req))
		testutil.Equal(t, "stable-token", req.NextPageToken)
		_ = json.NewEncoder(w).Encode(JQLSearchResult{
			Issues: makeIssues(51, 2), NextPageToken: "next-stable-token", IsLast: false,
		})
	})
	defer server.Close()

	result, err := client.SearchPage(context.Background(), SearchPageOptions{
		JQL: "project = TEST", MaxResults: 25, NextPageToken: "stable-token",
	})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "next-stable-token", result.Pagination.NextPageToken)
}

func TestSearchPageCapsRequestedPageSizeAt100(t *testing.T) {
	t.Parallel()
	client, server := newTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req SearchRequest
		testutil.RequireNoError(t, json.NewDecoder(r.Body).Decode(&req))
		testutil.Equal(t, 100, req.MaxResults)
		_ = json.NewEncoder(w).Encode(JQLSearchResult{IsLast: true})
	})
	defer server.Close()

	result, err := client.SearchPage(context.Background(), SearchPageOptions{
		JQL: "project = TEST", MaxResults: 500,
	})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 100, result.Pagination.PageSize)
	testutil.Equal(t, 0, len(result.Issues))
	testutil.True(t, result.Pagination.IsLast)
}

func TestSearchPagePreservesServerEnforcedCapAndToken(t *testing.T) {
	t.Parallel()
	client, server := newTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req SearchRequest
		testutil.RequireNoError(t, json.NewDecoder(r.Body).Decode(&req))
		testutil.Equal(t, 50, req.MaxResults)
		_ = json.NewEncoder(w).Encode(JQLSearchResult{
			Issues: makeIssues(1, 10), NextPageToken: "server-cap-token", IsLast: false,
		})
	})
	defer server.Close()

	result, err := client.SearchPage(context.Background(), SearchPageOptions{
		JQL: "project = TEST", MaxResults: 50,
	})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 10, len(result.Issues))
	testutil.False(t, result.Pagination.IsLast)
	testutil.Equal(t, "server-cap-token", result.Pagination.NextPageToken)
}

func TestSearchPageDefaultsAndPassesFields(t *testing.T) {
	t.Parallel()
	client, server := newTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req SearchRequest
		testutil.RequireNoError(t, json.NewDecoder(r.Body).Decode(&req))
		testutil.Equal(t, 50, req.MaxResults)
		testutil.Equal(t, []string{"summary", "customfield_10005"}, req.Fields)
		_ = json.NewEncoder(w).Encode(JQLSearchResult{IsLast: true})
	})
	defer server.Close()

	_, err := client.SearchPage(context.Background(), SearchPageOptions{
		JQL: "project = TEST", Fields: []string{"summary", "customfield_10005"},
	})
	testutil.RequireNoError(t, err)
}
