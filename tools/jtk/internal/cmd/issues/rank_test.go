package issues

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// rankServer accepts PUT /rest/agile/1.0/issue/rank, records each body and
// answers with status + body.
func rankServer(t *testing.T, calls *[]map[string]any, status int, body string) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/rest/agile/1.0/issue/rank" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		var c map[string]any
		_ = json.Unmarshal(raw, &c)
		mu.Lock()
		*calls = append(*calls, c)
		mu.Unlock()
		w.WriteHeader(status)
		if body != "" {
			_, _ = w.Write([]byte(body))
		}
	}))
}

func rankOpts(t *testing.T, server *httptest.Server) (*root.Options, *bytes.Buffer) {
	t.Helper()
	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)
	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)
	return opts, &stdout
}

func TestRankCmd_Before(t *testing.T) {
	var calls []map[string]any
	server := rankServer(t, &calls, http.StatusNoContent, "")
	defer server.Close()
	opts, stdout := rankOpts(t, server)

	cmd := newRankCmd(opts)
	cmd.SetArgs([]string{"PROJ-2", "PROJ-3", "--before", "PROJ-1"})
	testutil.RequireNoError(t, cmd.Execute())

	testutil.Equal(t, len(calls), 1)
	testutil.Equal(t, calls[0]["rankBeforeIssue"], "PROJ-1")
	testutil.Contains(t, stdout.String(), "Ranked PROJ-2, PROJ-3 before PROJ-1")
}

func TestRankCmd_After(t *testing.T) {
	var calls []map[string]any
	server := rankServer(t, &calls, http.StatusNoContent, "")
	defer server.Close()
	opts, stdout := rankOpts(t, server)

	cmd := newRankCmd(opts)
	cmd.SetArgs([]string{"PROJ-2", "--after", "PROJ-9"})
	testutil.RequireNoError(t, cmd.Execute())

	testutil.Equal(t, calls[0]["rankAfterIssue"], "PROJ-9")
	testutil.Contains(t, stdout.String(), "Ranked PROJ-2 after PROJ-9")
}

func TestRankCmd_IDOnlyEmitsTheRankedKeys(t *testing.T) {
	var calls []map[string]any
	server := rankServer(t, &calls, http.StatusNoContent, "")
	defer server.Close()
	opts, stdout := rankOpts(t, server)
	opts.IDOnly = true

	cmd := newRankCmd(opts)
	cmd.SetArgs([]string{"PROJ-2", "PROJ-3", "--before", "PROJ-1"})
	testutil.RequireNoError(t, cmd.Execute())
	testutil.Equal(t, stdout.String(), "PROJ-2\nPROJ-3\n")
}

func TestRankCmd_NeedsExactlyOneOfBeforeAndAfter(t *testing.T) {
	var calls []map[string]any
	server := rankServer(t, &calls, http.StatusNoContent, "")
	defer server.Close()

	for _, args := range [][]string{
		{"PROJ-2"},
		{"PROJ-2", "--before", "PROJ-1", "--after", "PROJ-3"},
	} {
		opts, _ := rankOpts(t, server)
		cmd := newRankCmd(opts)
		cmd.SetArgs(args)
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		testutil.Error(t, cmd.Execute())
	}
	testutil.Equal(t, len(calls), 0)
}

func TestRankCmd_RefusesTheAnchorAmongTheIssues(t *testing.T) {
	var calls []map[string]any
	server := rankServer(t, &calls, http.StatusNoContent, "")
	defer server.Close()
	opts, _ := rankOpts(t, server)

	cmd := newRankCmd(opts)
	cmd.SetArgs([]string{"PROJ-2", "proj-1", "--before", "PROJ-1"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "proj-1")
	testutil.Equal(t, len(calls), 0)
}

func TestRankCmd_ReportsTheIssuesJiraRefused(t *testing.T) {
	var calls []map[string]any
	server := rankServer(t, &calls, http.StatusMultiStatus,
		`{"entries":[{"issueKey":"PROJ-2","status":200},{"issueKey":"PROJ-3","status":400,"errors":["Issue is not on the board"]}]}`)
	defer server.Close()
	opts, _ := rankOpts(t, server)

	cmd := newRankCmd(opts)
	cmd.SetArgs([]string{"PROJ-2", "PROJ-3", "--before", "PROJ-1"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "PROJ-3")
	testutil.Contains(t, err.Error(), "Issue is not on the board")
	testutil.NotContains(t, err.Error(), "PROJ-2")
}
