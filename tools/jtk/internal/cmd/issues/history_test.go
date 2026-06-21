package issues

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

func TestNewHistoryCmd(t *testing.T) {
	t.Parallel()

	cmd := newHistoryCmd(&root.Options{})
	testutil.Equal(t, "history <issue-key>", cmd.Use)
	testutil.Equal(t, "List issue changelog history", cmd.Short)
	testutil.NotNil(t, cmd.Flags().Lookup("max"))
	testutil.NotNil(t, cmd.Flags().Lookup("next-page-token"))
	testutil.NotNil(t, cmd.Flags().Lookup("fields"))
}

func issueHistoryServer(t *testing.T, body string, assert func(*http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if assert != nil {
			assert(r)
		}
		if !strings.HasSuffix(r.URL.EscapedPath(), "/changelog") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
}

func historyOpts(t *testing.T, server *httptest.Server) (*root.Options, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@example.com", APIToken: "token"})
	testutil.RequireNoError(t, err)
	var stdout, stderr bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)
	return opts, &stdout, &stderr
}

func historyPageJSON(startAt, maxResults, total int) string {
	return fmt.Sprintf(`{
		"startAt": %d,
		"maxResults": %d,
		"total": %d,
		"values": [
			{
				"id": "10001",
				"created": "2026-06-20T15:04:05.000+0000",
				"author": {"accountId": "acct-1", "displayName": "Alice"},
				"items": [
					{"field": "status", "fieldtype": "jira", "fieldId": "status", "from": "1", "fromString": "Open", "to": "3", "toString": "Done"},
					{"field": "summary", "fieldtype": "jira", "fieldId": "summary", "fromString": "Old", "toString": "New"}
				]
			}
		]
	}`, startAt, maxResults, total)
}

func multiHistoryPageJSON(startAt, maxResults, total int) string {
	return fmt.Sprintf(`{
		"startAt": %d,
		"maxResults": %d,
		"total": %d,
		"values": [
			{
				"id": "10001",
				"created": "2026-06-20T15:04:05.000+0000",
				"author": {"accountId": "acct-1", "displayName": "Alice"},
				"items": [
					{"field": "status", "fieldtype": "jira", "fieldId": "status", "from": "1", "fromString": "Open", "to": "3", "toString": "Done"},
					{"field": "summary", "fieldtype": "jira", "fieldId": "summary", "fromString": "Old", "toString": "New"}
				]
			},
			{
				"id": "10002",
				"created": "2026-06-21T16:05:06.000+0000",
				"author": {"accountId": "acct-2", "displayName": "Bob"},
				"items": [
					{"field": "assignee", "fieldtype": "jira", "fieldId": "assignee", "fromString": "Alice", "toString": "Bob"}
				]
			}
		]
	}`, startAt, maxResults, total)
}

func TestRunHistory_DefaultOutputAndPagination(t *testing.T) {
	t.Parallel()

	server := issueHistoryServer(t, historyPageJSON(0, 1, 2), func(r *http.Request) {
		testutil.Equal(t, "/rest/api/3/issue/TEST-1/changelog", r.URL.EscapedPath())
		testutil.Equal(t, "1", r.URL.Query().Get("maxResults"))
	})
	defer server.Close()

	opts, stdout, stderr := historyOpts(t, server)
	err := runHistory(context.Background(), opts, "TEST-1", 1, "", "")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "ID | CREATED | AUTHOR | FIELD | FROM | TO")
	testutil.Contains(t, out, "10001 | 2026-06-20 | Alice | status | Open | Done")
	testutil.Contains(t, out, "10001 | 2026-06-20 | Alice | summary | Old | New")
	testutil.Contains(t, out, "More results available (next: 1)")
	testutil.Equal(t, "", stderr.String())
}

func TestRunHistory_PreservesMultipleGroupOrderAndIDs(t *testing.T) {
	t.Parallel()

	server := issueHistoryServer(t, multiHistoryPageJSON(0, 2, 2), nil)
	defer server.Close()

	opts, stdout, stderr := historyOpts(t, server)
	err := runHistory(context.Background(), opts, "TEST-1", 2, "", "")
	testutil.RequireNoError(t, err)

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	testutil.Equal(t, []string{
		"ID | CREATED | AUTHOR | FIELD | FROM | TO",
		"10001 | 2026-06-20 | Alice | status | Open | Done",
		"10001 | 2026-06-20 | Alice | summary | Old | New",
		"10002 | 2026-06-21 | Bob | assignee | Alice | Bob",
	}, lines)
	testutil.Equal(t, "", stderr.String())
}

func TestRunHistory_IDOnlyEmitsGroupIDs(t *testing.T) {
	t.Parallel()

	server := issueHistoryServer(t, historyPageJSON(0, 1, 2), nil)
	defer server.Close()

	opts, stdout, stderr := historyOpts(t, server)
	opts.IDOnly = true
	err := runHistory(context.Background(), opts, "TEST-1", 1, "", "bogus")
	testutil.RequireNoError(t, err)

	testutil.Equal(t, "10001\nMore results available (next: 1)\n", stdout.String())
	testutil.Equal(t, "", stderr.String())
}

func TestRunHistory_ExtendedImpliesFullText(t *testing.T) {
	t.Parallel()

	longValue := strings.Repeat("A", 120)
	body := fmt.Sprintf(`{
		"startAt": 0,
		"maxResults": 1,
		"total": 1,
		"values": [
			{
				"id": "10001",
				"created": "2026-06-20T15:04:05.000+0000",
				"author": {"accountId": "acct-1", "displayName": "Alice"},
				"items": [
					{"field": "description", "fieldtype": "jira", "fieldId": "description", "fromString": "%s", "toString": "Short"}
				]
			}
		]
	}`, longValue)
	server := issueHistoryServer(t, body, nil)
	defer server.Close()

	opts, stdout, stderr := historyOpts(t, server)
	opts.Extended = true
	err := runHistory(context.Background(), opts, "TEST-1", 1, "", "")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "ID | CREATED | AUTHOR | ACCOUNT_ID | FIELD | FIELD_ID | TYPE | FROM_ID | FROM | TO_ID | TO")
	testutil.Contains(t, out, longValue)
	testutil.NotContains(t, out, "...")
	testutil.Equal(t, "", stderr.String())
}

func TestNewHistoryCmd_FieldsProjectionViaCobra(t *testing.T) {
	t.Parallel()

	server := issueHistoryServer(t, historyPageJSON(0, 1, 2), nil)
	defer server.Close()

	opts, stdout, _ := historyOpts(t, server)
	cmd := newHistoryCmd(opts)
	cmd.SetArgs([]string{"TEST-1", "--fields", "CREATED,FIELD,TO", "--max", "1"})
	testutil.RequireNoError(t, cmd.Execute())

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	if len(lines) == 0 {
		t.Fatalf("empty output")
	}
	testutil.Equal(t, "ID | CREATED | FIELD | TO", lines[0])
	testutil.Contains(t, stdout.String(), "More results available (next: 1)")
}

func TestHistoryCommand_ExtendedFullTextAndNextPageTokenViaRoot(t *testing.T) {
	longValue := strings.Repeat("B", 120)
	body := fmt.Sprintf(`{
		"startAt": 5,
		"maxResults": 2,
		"total": 9,
		"values": [
			{
				"id": "10005",
				"created": "2026-06-20T15:04:05.000+0000",
				"author": {"accountId": "acct-1", "displayName": "Alice"},
				"items": [
					{"field": "description", "fieldtype": "jira", "fieldId": "description", "fromString": "%s", "toString": "Short"}
				]
			}
		]
	}`, longValue)
	server := issueHistoryServer(t, body, func(r *http.Request) {
		testutil.Equal(t, "/rest/api/3/issue/TEST-1/changelog", r.URL.EscapedPath())
		testutil.Equal(t, "5", r.URL.Query().Get("startAt"))
		testutil.Equal(t, "2", r.URL.Query().Get("maxResults"))
	})
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@example.com", APIToken: "token"})
	testutil.RequireNoError(t, err)
	var stdout, stderr bytes.Buffer
	rootCmd, opts := root.NewCmd()
	opts.Stdout = &stdout
	opts.Stderr = &stderr
	opts.SetAPIClient(client)
	Register(rootCmd, opts)
	rootCmd.SetArgs([]string{"--extended", "--fulltext", "issues", "history", "TEST-1", "--max", "2", "--next-page-token", "5"})

	testutil.RequireNoError(t, rootCmd.Execute())

	out := stdout.String()
	testutil.Contains(t, out, "ID | CREATED | AUTHOR | ACCOUNT_ID | FIELD | FIELD_ID | TYPE | FROM_ID | FROM | TO_ID | TO")
	testutil.Contains(t, out, "10005 | 2026-06-20T15:04:05.000+0000 | Alice | acct-1 | description | description | jira")
	testutil.Contains(t, out, longValue)
	testutil.Contains(t, out, "More results available (next: 6)")
	testutil.Equal(t, "", stderr.String())
}

func TestRunHistory_FieldsExtendedOnlyError(t *testing.T) {
	t.Parallel()

	server := issueHistoryServer(t, historyPageJSON(0, 1, 1), nil)
	defer server.Close()

	opts, _, _ := historyOpts(t, server)
	err := runHistory(context.Background(), opts, "TEST-1", 1, "", "ACCOUNT_ID")
	var extendedOnly *projection.ExtendedOnlyError
	if !errors.As(err, &extendedOnly) {
		t.Fatalf("got %v, want ExtendedOnlyError", err)
	}
}

func TestRunHistory_BadNextPageToken(t *testing.T) {
	t.Parallel()

	server := issueHistoryServer(t, historyPageJSON(0, 1, 1), nil)
	defer server.Close()

	opts, _, _ := historyOpts(t, server)
	err := runHistory(context.Background(), opts, "TEST-1", 1, "not-a-number", "")
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "invalid --next-page-token")
}

func TestRunHistory_EmptyWithMoreResultsEmitsOnlyPagination(t *testing.T) {
	t.Parallel()

	server := issueHistoryServer(t, `{"startAt":0,"maxResults":1,"total":2,"values":[]}`, nil)
	defer server.Close()

	opts, stdout, stderr := historyOpts(t, server)
	err := runHistory(context.Background(), opts, "TEST-1", 1, "", "")
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), "More results available (next: 1)")
	testutil.NotContains(t, stdout.String(), "No history found")
	testutil.Equal(t, "", stderr.String())
}

func TestRunHistory_EmptyWithoutMoreResults(t *testing.T) {
	t.Parallel()

	server := issueHistoryServer(t, `{"startAt":0,"maxResults":50,"total":0,"values":[]}`, nil)
	defer server.Close()

	opts, stdout, stderr := historyOpts(t, server)
	err := runHistory(context.Background(), opts, "TEST-1", 50, "", "")
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), "No history found for TEST-1")
	testutil.Equal(t, "", stderr.String())
}
