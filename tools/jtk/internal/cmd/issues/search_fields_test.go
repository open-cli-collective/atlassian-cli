package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

func TestDeriveFetchFields(t *testing.T) {
	t.Parallel()
	selected := []projection.ColumnSpec{
		{Header: "KEY", Identity: true},
		{Header: "SUMMARY", FieldID: "summary"},
	}

	tests := []struct {
		name      string
		projected bool
		extended  bool
		allFields bool
		want      []string
	}{
		{
			name:      "projected → union of selected specs",
			projected: true,
			want:      []string{"summary"},
		},
		{
			name:      "projected wins over extended",
			projected: true, extended: true,
			want: []string{"summary"},
		},
		{
			name:      "projected wins over allFields",
			projected: true, allFields: true,
			want: []string{"summary"},
		},
		{
			name:     "extended without projection → DefaultSearchFields",
			extended: true,
			want:     api.DefaultSearchFields,
		},
		{
			name:      "allFields without projection → DefaultSearchFields",
			allFields: true,
			want:      api.DefaultSearchFields,
		},
		{
			name:     "extended and allFields are idempotent",
			extended: true, allFields: true,
			want: api.DefaultSearchFields,
		},
		{
			name: "default → ListSearchFields",
			want: api.ListSearchFields,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveFetchFields(selected, tt.projected, tt.extended, tt.allFields)
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

// newPaginatedSearchServer creates a server that returns pageSize issues per request
// across multiple pages, up to totalIssues total.
