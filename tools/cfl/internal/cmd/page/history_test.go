package page

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

func newHistoryTestRootOptions() *root.Options {
	return &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
}

func TestRunHistoryList_RendersTableAndTruncatesMessage(t *testing.T) {
	t.Parallel()
	longMessage := strings.Repeat("a", 90) + "tail-marker"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/api/v2/pages/12345/versions", r.URL.Path)
		testutil.Equal(t, "2", r.URL.Query().Get("limit"))
		testutil.Equal(t, "-modified-date", r.URL.Query().Get("sort"))
		testutil.Empty(t, r.URL.Query().Get("body-format"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [{
				"number": 15,
				"message": "` + longMessage + `",
				"minorEdit": true,
				"authorId": "author-1"
			}]
		}`))
	}))
	defer server.Close()

	rootOpts := newHistoryTestRootOptions()
	rootOpts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))
	opts := &historyListOptions{
		Options: rootOpts,
		limit:   2,
	}

	err := runHistoryList(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)

	stdout := rootOpts.Stdout.(*bytes.Buffer).String()
	testutil.Contains(t, stdout, "VERSION")
	testutil.Contains(t, stdout, "15")
	testutil.Contains(t, stdout, "author-1")
	testutil.Contains(t, stdout, "yes")
	testutil.Contains(t, stdout, strings.Repeat("a", 77)+"...")
	testutil.False(t, strings.Contains(stdout, "tail-marker"), "message tail should be truncated")
}

func TestRunHistoryList_IDOnly(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [
				{"number": 3},
				{"number": 2}
			]
		}`))
	}))
	defer server.Close()

	rootOpts := newHistoryTestRootOptions()
	rootOpts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))
	opts := &historyListOptions{
		Options: rootOpts,
		limit:   25,
		idOnly:  true,
	}

	err := runHistoryList(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "3\n2\n", rootOpts.Stdout.(*bytes.Buffer).String())
}

func TestExecuteHistoryListAliasWiredThroughCobra(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/api/v2/pages/12345/versions", r.URL.Path)
		testutil.Equal(t, "2", r.URL.Query().Get("limit"))
		testutil.Equal(t, "-modified-date", r.URL.Query().Get("sort"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [
				{"number": 5},
				{"number": 4}
			]
		}`))
	}))
	defer server.Close()

	rootOpts := newHistoryTestRootOptions()
	rootOpts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))
	rootCmd := &cobra.Command{Use: "cfl"}
	Register(rootCmd, rootOpts)
	rootCmd.SetArgs([]string{"page", "history", "ls", "12345", "--id", "--limit", "2"})

	err := rootCmd.Execute()
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "5\n4\n", rootOpts.Stdout.(*bytes.Buffer).String())
}

func TestRunHistoryList_LimitZeroDoesNotCallAPI(t *testing.T) {
	t.Parallel()
	rootOpts := newHistoryTestRootOptions()
	opts := &historyListOptions{
		Options: rootOpts,
		limit:   0,
	}

	err := runHistoryList(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, rootOpts.Stdout.(*bytes.Buffer).String(), "No page versions found.")
}

func TestRunHistoryList_CursorAndNextHint(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "cursor-in", r.URL.Query().Get("cursor"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [{"number": 4}],
			"_links": {"next": "/api/v2/pages/12345/versions?cursor=cursor-out"}
		}`))
	}))
	defer server.Close()

	rootOpts := newHistoryTestRootOptions()
	rootOpts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))
	opts := &historyListOptions{
		Options: rootOpts,
		limit:   25,
		cursor:  "cursor-in",
	}

	err := runHistoryList(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
	stderr := rootOpts.Stderr.(*bytes.Buffer).String()
	testutil.Contains(t, stderr, "Next page: cfl page history list 12345 --cursor \"cursor-out\"")
}

func TestRunHistoryList_NegativeLimit(t *testing.T) {
	t.Parallel()
	rootOpts := newHistoryTestRootOptions()
	opts := &historyListOptions{
		Options: rootOpts,
		limit:   -1,
	}

	err := runHistoryList(context.Background(), "12345", opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "invalid limit")
}
