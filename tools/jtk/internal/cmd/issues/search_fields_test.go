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
)

func TestResolveFields(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		fieldsFlag string
		output     string
		full       bool
		want       []string
	}{
		{
			name:       "explicit fields flag takes precedence",
			fieldsFlag: "summary,customfield_10005",
			output:     "json",
			full:       true,
			want:       []string{"summary", "customfield_10005"},
		},
		{
			name:       "json output without fields flag returns all",
			fieldsFlag: "",
			output:     "json",
			full:       false,
			want:       []string{"*all"},
		},
		{
			name:       "json output with full flag still returns all",
			fieldsFlag: "",
			output:     "json",
			full:       true,
			want:       []string{"*all"},
		},
		{
			name:       "full flag returns DefaultSearchFields",
			fieldsFlag: "",
			output:     "",
			full:       true,
			want:       api.DefaultSearchFields,
		},
		{
			name:       "default returns ListSearchFields",
			fieldsFlag: "",
			output:     "",
			full:       false,
			want:       api.ListSearchFields,
		},
		{
			name:       "table output returns ListSearchFields",
			fieldsFlag: "",
			output:     "table",
			full:       false,
			want:       api.ListSearchFields,
		},
		{
			name:       "single field",
			fieldsFlag: "summary",
			output:     "",
			full:       false,
			want:       []string{"summary"},
		},
		{
			name:       "trims whitespace around fields",
			fieldsFlag: "summary , customfield_10005 , status",
			output:     "",
			full:       false,
			want:       []string{"summary", "customfield_10005", "status"},
		},
		{
			name:       "drops empty segments from trailing comma",
			fieldsFlag: "summary,status,",
			output:     "",
			full:       false,
			want:       []string{"summary", "status"},
		},
		{
			name:       "all empty tokens falls through to json default",
			fieldsFlag: ",, ",
			output:     "json",
			full:       false,
			want:       []string{"*all"},
		},
		{
			name:       "all empty tokens falls through to list default",
			fieldsFlag: ",, ",
			output:     "",
			full:       false,
			want:       api.ListSearchFields,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveFields(tt.fieldsFlag, tt.output, tt.full)
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

func TestRunSearch_FieldsFlagOverridesJSONDefault(t *testing.T) {
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

	err = runSearch(context.Background(), opts, "project = TEST", 25, "", false, "summary,customfield_10005")
	testutil.RequireNoError(t, err)

	testutil.Equal(t, 2, len(captured.Fields))
	testutil.Equal(t, "summary", captured.Fields[0])
	testutil.Equal(t, "customfield_10005", captured.Fields[1])
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

func TestRunList_FieldsFlagOverridesJSONDefault(t *testing.T) {
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

	err = runList(context.Background(), opts, "TEST", "", 25, "", false, "summary,customfield_10035")
	testutil.RequireNoError(t, err)

	testutil.Equal(t, 2, len(captured.Fields))
	testutil.Equal(t, "summary", captured.Fields[0])
	testutil.Equal(t, "customfield_10035", captured.Fields[1])
}
