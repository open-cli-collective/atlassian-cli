package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// listResultServer returns a fixed set of issues with configurable IsLast.
// `keys` drives which issue keys the mock returns.
func listResultServer(t *testing.T, keys []string, isLast bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		issues := make([]api.Issue, len(keys))
		for i, k := range keys {
			issues[i] = api.Issue{
				Key: k,
				Fields: api.IssueFields{
					Summary:   "summary for " + k,
					Status:    &api.Status{Name: "Open"},
					IssueType: &api.IssueType{Name: "Task"},
				},
			}
		}
		result := api.JQLSearchResult{Issues: issues, IsLast: isLast}
		if !isLast {
			result.NextPageToken = "next-token"
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	}))
}

func newListOpts(t *testing.T, server *httptest.Server) (*root.Options, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "e@x", APIToken: "t"})
	testutil.RequireNoError(t, err)
	var stdout, stderr bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)
	return opts, &stdout, &stderr
}

func TestRunList_DefaultPaginationOnStdout(t *testing.T) {
	t.Parallel()
	server := listResultServer(t, []string{"TEST-1", "TEST-2"}, false)
	defer server.Close()

	opts, stdout, stderr := newListOpts(t, server)
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "")
	testutil.RequireNoError(t, err)

	if !strings.Contains(stdout.String(), "TEST-1") {
		t.Errorf("stdout missing issue key: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "More results available") {
		t.Errorf("pagination hint should be on stdout, got stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if strings.Contains(stderr.String(), "More results available") {
		t.Errorf("pagination hint should NOT be on stderr: %q", stderr.String())
	}
}

func TestRunList_IDOnlyEmitsKeysOnePerLine(t *testing.T) {
	t.Parallel()
	server := listResultServer(t, []string{"TEST-1", "TEST-2", "TEST-3"}, true)
	defer server.Close()

	opts, stdout, stderr := newListOpts(t, server)
	opts.IDOnly = true
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "")
	testutil.RequireNoError(t, err)

	want := "TEST-1\nTEST-2\nTEST-3\n"
	if stdout.String() != want {
		t.Errorf("stdout:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
	if stderr.String() != "" {
		t.Errorf("stderr should be empty, got: %q", stderr.String())
	}
}

func TestRunList_IDOnlyWithMoreResultsAppendsContinuation(t *testing.T) {
	t.Parallel()
	server := listResultServer(t, []string{"TEST-1", "TEST-2"}, false)
	defer server.Close()

	opts, stdout, _ := newListOpts(t, server)
	opts.IDOnly = true
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "")
	testutil.RequireNoError(t, err)

	want := "TEST-1\nTEST-2\nMore results available (use --next-page-token to fetch next page)\n"
	if stdout.String() != want {
		t.Errorf("stdout:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
}

func TestRunList_EmptyDefault_NoIssuesFoundOnStdout(t *testing.T) {
	t.Parallel()
	server := listResultServer(t, nil, true)
	defer server.Close()

	opts, stdout, stderr := newListOpts(t, server)
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "")
	testutil.RequireNoError(t, err)

	if !strings.Contains(stdout.String(), "No issues found") {
		t.Errorf("expected 'No issues found' on stdout, got: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Errorf("stderr should be empty, got: %q", stderr.String())
	}
}

func TestRunList_EmptyWithMoreResults_EmitsOnlyPaginationHint(t *testing.T) {
	t.Parallel()
	// Empty page with IsLast=false (more pages exist). The continuation hint
	// alone reaches stdout so agents keep paging; the "No issues found"
	// message is suppressed because the result set is not actually empty —
	// only this page is. Emitting both would self-contradict.
	server := listResultServer(t, nil, false)
	defer server.Close()

	opts, stdout, stderr := newListOpts(t, server)
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "")
	testutil.RequireNoError(t, err)

	if !strings.Contains(stdout.String(), "More results available") {
		t.Errorf("pagination hint should appear on stdout; got %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "No issues found") {
		t.Errorf("'No issues found' must not co-occur with pagination hint; got %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Errorf("stderr should be empty, got: %q", stderr.String())
	}
}

func TestRunList_EmptyWithIDOnly_EmitsNothing(t *testing.T) {
	t.Parallel()
	server := listResultServer(t, nil, true)
	defer server.Close()

	opts, stdout, stderr := newListOpts(t, server)
	opts.IDOnly = true
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "")
	testutil.RequireNoError(t, err)

	if stdout.String() != "" {
		t.Errorf("stdout should be empty under --id with zero results, got: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Errorf("stderr should be empty, got: %q", stderr.String())
	}
}
