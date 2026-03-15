package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

// makeIssues creates n test issues with keys TEST-1 through TEST-n.
func makeIssues(start, count int) []Issue {
	issues := make([]Issue, count)
	for i := range count {
		issues[i] = Issue{
			Key: fmt.Sprintf("TEST-%d", start+i),
			Fields: IssueFields{
				Summary: fmt.Sprintf("Issue %d", start+i),
			},
		}
	}
	return issues
}

func TestSearchPage_SinglePage(t *testing.T) {
	t.Parallel()
	client, server := newTestClientWithServer(t, func(w http.ResponseWriter, _ *http.Request) {
		result := JQLSearchResult{
			Issues: makeIssues(1, 10),
			IsLast: true,
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	})
	defer server.Close()

	result, err := client.SearchPage(context.Background(), SearchPageOptions{
		JQL:        "project = TEST",
		MaxResults: 10,
	})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 10, len(result.Issues))
	testutil.Equal(t, 10, result.Pagination.Total)
	testutil.True(t, result.Pagination.IsLast)
	testutil.Equal(t, "", result.Pagination.NextPageToken)
}

func TestSearchPage_SinglePageDefaultMaxResults(t *testing.T) {
	t.Parallel()
	client, server := newTestClientWithServer(t, func(w http.ResponseWriter, _ *http.Request) {
		result := JQLSearchResult{
			Issues: makeIssues(1, 5),
			IsLast: true,
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	})
	defer server.Close()

	// MaxResults=0 means single page mode with default page size
	result, err := client.SearchPage(context.Background(), SearchPageOptions{
		JQL: "project = TEST",
	})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 5, len(result.Issues))
	testutil.True(t, result.Pagination.IsLast)
}

func TestSearchPage_AutoPagination_MultiplePages(t *testing.T) {
	t.Parallel()
	var requestCount atomic.Int32

	client, server := newTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req SearchRequest
		_ = json.Unmarshal(body, &req)

		page := requestCount.Add(1)
		var result JQLSearchResult
		switch page {
		case 1:
			result = JQLSearchResult{
				Issues:        makeIssues(1, 50),
				IsLast:        false,
				NextPageToken: "page2token",
			}
		case 2:
			testutil.Equal(t, "page2token", req.NextPageToken)
			result = JQLSearchResult{
				Issues: makeIssues(51, 50),
				IsLast: true,
			}
		default:
			t.Fatalf("unexpected request %d", page)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	})
	defer server.Close()

	result, err := client.SearchPage(context.Background(), SearchPageOptions{
		JQL:        "project = TEST",
		MaxResults: 100,
		PageSize:   50,
	})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, int32(2), requestCount.Load())
	testutil.Equal(t, 100, len(result.Issues))
	testutil.Equal(t, 100, result.Pagination.Total)
	testutil.True(t, result.Pagination.IsLast)
	testutil.Equal(t, "TEST-1", result.Issues[0].Key)
	testutil.Equal(t, "TEST-100", result.Issues[99].Key)
}

func TestSearchPage_AutoPagination_StopsAtMaxResults(t *testing.T) {
	t.Parallel()
	var requestCount atomic.Int32

	client, server := newTestClientWithServer(t, func(w http.ResponseWriter, _ *http.Request) {
		page := requestCount.Add(1)
		result := JQLSearchResult{
			Issues:        makeIssues(int(page-1)*100+1, 100),
			IsLast:        false,
			NextPageToken: fmt.Sprintf("page%dtoken", page+1),
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	})
	defer server.Close()

	result, err := client.SearchPage(context.Background(), SearchPageOptions{
		JQL:        "project = TEST",
		MaxResults: 200,
	})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, int32(2), requestCount.Load())
	testutil.Equal(t, 200, len(result.Issues))
	testutil.Equal(t, 200, result.Pagination.Total)
	testutil.False(t, result.Pagination.IsLast)
	testutil.NotEqual(t, "", result.Pagination.NextPageToken)
}

func TestSearchPage_AutoPagination_TrimsOvershoot(t *testing.T) {
	t.Parallel()
	var requestCount atomic.Int32

	client, server := newTestClientWithServer(t, func(w http.ResponseWriter, _ *http.Request) {
		page := requestCount.Add(1)
		result := JQLSearchResult{
			Issues:        makeIssues(int(page-1)*75+1, 75),
			IsLast:        false,
			NextPageToken: fmt.Sprintf("page%dtoken", page+1),
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	})
	defer server.Close()

	result, err := client.SearchPage(context.Background(), SearchPageOptions{
		JQL:        "project = TEST",
		MaxResults: 100,
		PageSize:   75,
	})
	testutil.RequireNoError(t, err)
	// First page: 75, second page: 75 = 150 total, trimmed to 100
	testutil.Equal(t, 100, len(result.Issues))
	testutil.Equal(t, 100, result.Pagination.Total)
	testutil.False(t, result.Pagination.IsLast)
}

func TestSearchPage_AutoPagination_PassesFields(t *testing.T) {
	t.Parallel()
	var capturedFields [][]string

	client, server := newTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req SearchRequest
		_ = json.Unmarshal(body, &req)
		capturedFields = append(capturedFields, req.Fields)

		isLast := len(capturedFields) >= 2
		result := JQLSearchResult{
			Issues:        makeIssues(len(capturedFields)*50-49, 50),
			IsLast:        isLast,
			NextPageToken: "nexttoken",
		}
		if isLast {
			result.NextPageToken = ""
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	})
	defer server.Close()

	fields := []string{"summary", "customfield_10005"}
	_, err := client.SearchPage(context.Background(), SearchPageOptions{
		JQL:        "project = TEST",
		MaxResults: 100,
		PageSize:   50,
		Fields:     fields,
	})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 2, len(capturedFields))
	for _, cf := range capturedFields {
		testutil.Equal(t, 2, len(cf))
		testutil.Equal(t, "summary", cf[0])
		testutil.Equal(t, "customfield_10005", cf[1])
	}
}

func TestSearchPage_AutoPagination_WithNextPageToken(t *testing.T) {
	t.Parallel()
	var capturedTokens []string

	client, server := newTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req SearchRequest
		_ = json.Unmarshal(body, &req)
		capturedTokens = append(capturedTokens, req.NextPageToken)

		result := JQLSearchResult{
			Issues: makeIssues(1, 50),
			IsLast: true,
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	})
	defer server.Close()

	_, err := client.SearchPage(context.Background(), SearchPageOptions{
		JQL:           "project = TEST",
		MaxResults:    200,
		NextPageToken: "starthere",
	})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 1, len(capturedTokens))
	testutil.Equal(t, "starthere", capturedTokens[0])
}

func TestSearchPage_AutoPagination_CancelledContext(t *testing.T) {
	t.Parallel()
	client, server := newTestClientWithServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.SearchPage(ctx, SearchPageOptions{
		JQL:        "project = TEST",
		MaxResults: 200,
	})
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "searching issues")
}

func TestSearchPage_AutoPagination_ErrorMidPagination(t *testing.T) {
	t.Parallel()
	var requestCount atomic.Int32

	client, server := newTestClientWithServer(t, func(w http.ResponseWriter, _ *http.Request) {
		page := requestCount.Add(1)
		if page == 1 {
			result := JQLSearchResult{
				Issues:        makeIssues(1, 50),
				IsLast:        false,
				NextPageToken: "page2token",
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(result)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errorMessages":["Internal error"]}`))
		}
	})
	defer server.Close()

	_, err := client.SearchPage(context.Background(), SearchPageOptions{
		JQL:        "project = TEST",
		MaxResults: 200,
		PageSize:   50,
	})
	testutil.Error(t, err)
}

func TestSearchPage_PageSizeCappedAt100(t *testing.T) {
	t.Parallel()
	var capturedMaxResults []int

	client, server := newTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req SearchRequest
		_ = json.Unmarshal(body, &req)
		capturedMaxResults = append(capturedMaxResults, req.MaxResults)

		result := JQLSearchResult{
			Issues: makeIssues(1, req.MaxResults),
			IsLast: true,
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	})
	defer server.Close()

	_, err := client.SearchPage(context.Background(), SearchPageOptions{
		JQL:        "project = TEST",
		MaxResults: 500,
		PageSize:   200, // Should be capped to 100
	})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 1, len(capturedMaxResults))
	testutil.True(t, capturedMaxResults[0] <= 100, "page size should be capped at 100")
}
