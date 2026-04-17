package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// artifactListResult is a helper struct for parsing artifact list output in tests.
type artifactListResult struct {
	Results []*jtkartifact.IssueArtifact `json:"results"`
	Meta    struct {
		Count   int  `json:"count"`
		HasMore bool `json:"hasMore"`
	} `json:"_meta"`
}

func TestDeriveFetchFields(t *testing.T) {
	t.Parallel()
	selected := []projection.ColumnSpec{
		{Header: "KEY", Identity: true}, // synthetic — no fetch contribution
		{Header: "SUMMARY", FieldID: "summary"},
	}

	tests := []struct {
		name      string
		projected bool
		extended  bool
		allFields bool
		output    string
		want      []string
	}{
		{
			name:   "json output → *all",
			output: "json",
			want:   []string{"*all"},
		},
		{
			name:   "json + extended → still *all",
			output: "json", extended: true,
			want: []string{"*all"},
		},
		{
			name:      "projected → union of selected specs",
			projected: true,
			output:    "table",
			want:      []string{"summary"},
		},
		{
			name:      "projected wins over extended",
			projected: true, extended: true,
			output: "table",
			want:   []string{"summary"},
		},
		{
			name:      "projected wins over allFields",
			projected: true, allFields: true,
			output: "table",
			want:   []string{"summary"},
		},
		{
			name:     "extended without projection → DefaultSearchFields",
			extended: true,
			output:   "table",
			want:     api.DefaultSearchFields,
		},
		{
			name:      "allFields without projection → DefaultSearchFields",
			allFields: true,
			output:    "table",
			want:      api.DefaultSearchFields,
		},
		{
			name:     "extended and allFields are idempotent",
			extended: true, allFields: true,
			output: "table",
			want:   api.DefaultSearchFields,
		},
		{
			name:   "default → ListSearchFields",
			output: "table",
			want:   api.ListSearchFields,
		},
		{
			name:   "empty output treated as non-json",
			output: "",
			want:   api.ListSearchFields,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveFetchFields(selected, tt.projected, tt.extended, tt.allFields, tt.output)
			testutil.Equal(t, len(tt.want), len(got))
			for i := range tt.want {
				testutil.Equal(t, tt.want[i], got[i])
			}
		})
	}
}

// newSearchServer creates an httptest server that captures the request body
// and responds with a valid JQL search result.
func newSearchServer(t *testing.T, capturedBody *api.SearchRequest) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("reading request body: %v", err)
		}
		if capturedBody != nil {
			if err := json.Unmarshal(body, capturedBody); err != nil {
				t.Fatalf("parsing request body: %v", err)
			}
		}

		result := api.JQLSearchResult{
			Issues: []api.Issue{
				{
					Key: "TEST-1",
					Fields: api.IssueFields{
						Summary:   "Test issue",
						Status:    &api.Status{Name: "Open"},
						IssueType: &api.IssueType{Name: "Task"},
					},
				},
			},
			IsLast: true,
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	}))
}

func TestRunSearch_JSONOutputRequestsAllFields(t *testing.T) {
	t.Parallel()
	var captured api.SearchRequest
	server := newSearchServer(t, &captured)
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "json",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runSearch(context.Background(), opts, "project = TEST", 25, "", false, "")
	testutil.RequireNoError(t, err)

	testutil.Equal(t, 1, len(captured.Fields))
	testutil.Equal(t, "*all", captured.Fields[0])
}

func TestRunSearch_TableOutputUsesListFields(t *testing.T) {
	t.Parallel()
	var captured api.SearchRequest
	server := newSearchServer(t, &captured)
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runSearch(context.Background(), opts, "project = TEST", 25, "", false, "")
	testutil.RequireNoError(t, err)

	testutil.Equal(t, len(api.ListSearchFields), len(captured.Fields))
	for i, f := range api.ListSearchFields {
		testutil.Equal(t, f, captured.Fields[i])
	}
}

func TestRunList_JSONOutputRequestsAllFields(t *testing.T) {
	t.Parallel()
	var captured api.SearchRequest
	server := newSearchServer(t, &captured)
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "json",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "TEST", "", 25, "", false, "")
	testutil.RequireNoError(t, err)

	testutil.Equal(t, 1, len(captured.Fields))
	testutil.Equal(t, "*all", captured.Fields[0])
}

// newPaginatedSearchServer creates a server that returns pageSize issues per request
// across multiple pages, up to totalIssues total.
func newPaginatedSearchServer(t *testing.T, pageSize, totalIssues int) *httptest.Server {
	t.Helper()
	var requestCount atomic.Int32
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req api.SearchRequest
		_ = json.Unmarshal(body, &req)

		page := int(requestCount.Add(1))
		start := (page - 1) * pageSize
		remaining := totalIssues - start
		count := pageSize
		if remaining < count {
			count = remaining
		}
		if count < 0 {
			count = 0
		}

		issues := make([]api.Issue, count)
		for i := range count {
			issues[i] = api.Issue{
				Key: fmt.Sprintf("TEST-%d", start+i+1),
				Fields: api.IssueFields{
					Summary:   fmt.Sprintf("Issue %d", start+i+1),
					Status:    &api.Status{Name: "Open"},
					IssueType: &api.IssueType{Name: "Task"},
				},
			}
		}

		isLast := start+count >= totalIssues
		nextToken := ""
		if !isLast {
			nextToken = fmt.Sprintf("page%dtoken", page+1)
		}

		result := api.JQLSearchResult{
			Issues:        issues,
			IsLast:        isLast,
			NextPageToken: nextToken,
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	}))
}

func TestRunSearch_AutoPaginationJSON(t *testing.T) {
	t.Parallel()
	// Server has 150 issues, serves 75 per page
	server := newPaginatedSearchServer(t, 75, 150)
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "json",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runSearch(context.Background(), opts, "project = TEST", 150, "", false, "")
	testutil.RequireNoError(t, err)

	var result artifactListResult
	err = json.Unmarshal(stdout.Bytes(), &result)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 150, len(result.Results))
	testutil.Equal(t, 150, result.Meta.Count)
	testutil.False(t, result.Meta.HasMore) // All results fetched
	testutil.Equal(t, "TEST-1", result.Results[0].Key)
	testutil.Equal(t, "TEST-150", result.Results[149].Key)
}

func TestRunList_AutoPaginationJSON(t *testing.T) {
	t.Parallel()
	// Server has 120 issues, serves 60 per page
	server := newPaginatedSearchServer(t, 60, 120)
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "json",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "TEST", "", 120, "", false, "")
	testutil.RequireNoError(t, err)

	var result artifactListResult
	err = json.Unmarshal(stdout.Bytes(), &result)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 120, len(result.Results))
	testutil.Equal(t, 120, result.Meta.Count)
	testutil.False(t, result.Meta.HasMore) // All results fetched
	testutil.Equal(t, "TEST-1", result.Results[0].Key)
	testutil.Equal(t, "TEST-120", result.Results[119].Key)
}
