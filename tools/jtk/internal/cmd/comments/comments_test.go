package comments

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

func TestNewListCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newListCmd(opts)

	testutil.Equal(t, cmd.Use, "list <issue-key>")

	// Check that no-truncate flag exists
	noTruncateFlag := cmd.Flags().Lookup("no-truncate")
	testutil.NotNil(t, noTruncateFlag)
	testutil.Equal(t, noTruncateFlag.DefValue, "false")

	// Check that max flag exists
	maxFlag := cmd.Flags().Lookup("max")
	testutil.NotNil(t, maxFlag)
	testutil.Equal(t, maxFlag.DefValue, "50")

	// Check that fields flag exists
	fieldsFlag := cmd.Flags().Lookup("fields")
	testutil.NotNil(t, fieldsFlag)
	testutil.Equal(t, fieldsFlag.DefValue, "")
}

func newTestCommentsServer(_ *testing.T, comments []api.Comment) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := api.CommentsResponse{
			StartAt:    0,
			MaxResults: 50,
			Total:      len(comments),
			Comments:   comments,
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
}

func TestRunList_TruncatesCommentBody(t *testing.T) {
	t.Parallel()
	longText := strings.Repeat("B", 200)
	comments := []api.Comment{
		{
			ID:     "1",
			Author: api.User{DisplayName: "Alice"},
			Body: &api.ADFDocument{
				Type:    "doc",
				Version: 1,
				Content: []*api.ADFNode{
					{
						Type: "paragraph",
						Content: []*api.ADFNode{
							{Type: "text", Text: longText},
						},
					},
				},
			},
			Created: "2024-01-15T10:00:00.000Z",
		},
	}

	server := newTestCommentsServer(t, comments)
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

	err = runList(context.Background(), opts, "TEST-1", 50, false, "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Alice")
	testutil.Contains(t, output, "...")
	testutil.NotContains(t, output, longText)
}

func TestRunList_FullCommentBody(t *testing.T) {
	t.Parallel()
	longText := strings.Repeat("B", 200)
	comments := []api.Comment{
		{
			ID:     "1",
			Author: api.User{DisplayName: "Alice"},
			Body: &api.ADFDocument{
				Type:    "doc",
				Version: 1,
				Content: []*api.ADFNode{
					{
						Type: "paragraph",
						Content: []*api.ADFNode{
							{Type: "text", Text: longText},
						},
					},
				},
			},
			Created: "2024-01-15T10:00:00.000Z",
		},
	}

	server := newTestCommentsServer(t, comments)
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

	err = runList(context.Background(), opts, "TEST-1", 50, true, "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, longText)
	testutil.NotContains(t, output, "[truncated")
	// Full mode uses key-value layout
	testutil.Contains(t, output, "ID:")
	testutil.Contains(t, output, "Author:")
	testutil.Contains(t, output, "Body:")
}

// TestNewListCmd_FullTextRoutesFromRoot verifies that --fulltext on the root
// Options flows through the RunE wrapper to disable truncation, even when the
// local --no-truncate flag is not set.
func TestNewListCmd_FullTextRoutesFromRoot(t *testing.T) {
	t.Parallel()
	longText := strings.Repeat("B", 200)
	comments := []api.Comment{
		{
			ID:     "1",
			Author: api.User{DisplayName: "Alice"},
			Body: &api.ADFDocument{
				Type:    "doc",
				Version: 1,
				Content: []*api.ADFNode{
					{
						Type: "paragraph",
						Content: []*api.ADFNode{
							{Type: "text", Text: longText},
						},
					},
				},
			},
			Created: "2024-01-15T10:00:00.000Z",
		},
	}

	server := newTestCommentsServer(t, comments)
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output:   "table",
		FullText: true,
		Stdout:   &stdout,
		Stderr:   &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	cmd := newListCmd(opts)
	cmd.SetArgs([]string{"TEST-1"}) // no --no-truncate locally
	testutil.RequireNoError(t, cmd.Execute())

	output := stdout.String()
	testutil.Contains(t, output, longText)
	testutil.NotContains(t, output, "[truncated")
}

// TestNewListCmd_NoTruncateAndFullTextBothSet guards the OR-combined path:
// both the local --no-truncate flag and the global --fulltext must produce
// the same result when set together (prevents accidental && regression).
func TestNewListCmd_NoTruncateAndFullTextBothSet(t *testing.T) {
	t.Parallel()
	longText := strings.Repeat("B", 200)
	comments := []api.Comment{
		{
			ID:     "1",
			Author: api.User{DisplayName: "Alice"},
			Body: &api.ADFDocument{
				Type:    "doc",
				Version: 1,
				Content: []*api.ADFNode{
					{
						Type: "paragraph",
						Content: []*api.ADFNode{
							{Type: "text", Text: longText},
						},
					},
				},
			},
			Created: "2024-01-15T10:00:00.000Z",
		},
	}

	server := newTestCommentsServer(t, comments)
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output:   "table",
		FullText: true,
		Stdout:   &stdout,
		Stderr:   &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	cmd := newListCmd(opts)
	cmd.SetArgs([]string{"TEST-1", "--no-truncate"})
	testutil.RequireNoError(t, cmd.Execute())

	output := stdout.String()
	testutil.Contains(t, output, longText)
	testutil.NotContains(t, output, "[truncated")
}

func TestRunList_ShortCommentNotTruncated(t *testing.T) {
	t.Parallel()
	comments := []api.Comment{
		{
			ID:     "1",
			Author: api.User{DisplayName: "Bob"},
			Body: &api.ADFDocument{
				Type:    "doc",
				Version: 1,
				Content: []*api.ADFNode{
					{
						Type: "paragraph",
						Content: []*api.ADFNode{
							{Type: "text", Text: "Short comment"},
						},
					},
				},
			},
			Created: "2024-01-15T10:00:00.000Z",
		},
	}

	server := newTestCommentsServer(t, comments)
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

	err = runList(context.Background(), opts, "TEST-1", 50, false, "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Short comment")
	testutil.NotContains(t, output, "[truncated")
}

func TestRunList_NoComments(t *testing.T) {
	t.Parallel()
	server := newTestCommentsServer(t, []api.Comment{})
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &stderr,
	}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "TEST-1", 50, false, "")
	testutil.RequireNoError(t, err)

	combined := stdout.String() + stderr.String()
	testutil.Contains(t, combined, "No comments")
}

// commentsServerWithTotal is like newTestCommentsServer but lets the caller
// set Total independently from len(comments), so pagination hasMore can be
// exercised explicitly.
func commentsServerWithTotal(comments []api.Comment, total int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := api.CommentsResponse{
			StartAt:    0,
			MaxResults: 50,
			Total:      total,
			Comments:   comments,
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
}

func plainComment(id, author, text string) api.Comment {
	return api.Comment{
		ID:     id,
		Author: api.User{DisplayName: author},
		Body: &api.ADFDocument{
			Type: "doc", Version: 1,
			Content: []*api.ADFNode{
				{Type: "paragraph", Content: []*api.ADFNode{{Type: "text", Text: text}}},
			},
		},
		Created: "2024-01-15T10:00:00.000Z",
	}
}

func newCommentsOpts(t *testing.T, server *httptest.Server) (*root.Options, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "e@x", APIToken: "t"})
	testutil.RequireNoError(t, err)
	var stdout, stderr bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)
	return opts, &stdout, &stderr
}

func TestRunList_FullTextBlockSpacing(t *testing.T) {
	t.Parallel()
	comments := []api.Comment{
		plainComment("11", "Alice", "First comment body"),
		plainComment("22", "Bob", "Second comment body"),
	}
	server := commentsServerWithTotal(comments, 2)
	defer server.Close()

	opts, stdout, _ := newCommentsOpts(t, server)
	err := runList(context.Background(), opts, "TEST-1", 50, true, "")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	// Each comment block ends with "Body: <text>\n" and the second block starts with "ID: 22".
	// A blank line between blocks means "Body: First comment body\n\nID: 22" appears.
	if !strings.Contains(out, "First comment body\n\nID: 22") {
		t.Errorf("expected blank line between comment blocks; got:\n%s", out)
	}
}

func TestRunList_FullTextPaginationOnStdout(t *testing.T) {
	t.Parallel()
	comments := []api.Comment{plainComment("1", "Alice", "hello")}
	// Total=5 means there are more pages after this one.
	server := commentsServerWithTotal(comments, 5)
	defer server.Close()

	opts, stdout, stderr := newCommentsOpts(t, server)
	err := runList(context.Background(), opts, "TEST-1", 1, true, "")
	testutil.RequireNoError(t, err)

	if !strings.Contains(stdout.String(), "More results available") {
		t.Errorf("pagination hint should appear on stdout, got stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if strings.Contains(stderr.String(), "More results available") {
		t.Errorf("pagination hint should NOT appear on stderr: %q", stderr.String())
	}
}

func TestRunList_IDOnlyEmitsIDsOnePerLine(t *testing.T) {
	t.Parallel()
	comments := []api.Comment{
		plainComment("11", "Alice", "first"),
		plainComment("22", "Bob", "second"),
	}
	server := commentsServerWithTotal(comments, 2)
	defer server.Close()

	opts, stdout, stderr := newCommentsOpts(t, server)
	opts.IDOnly = true
	err := runList(context.Background(), opts, "TEST-1", 50, false, "")
	testutil.RequireNoError(t, err)

	want := "11\n22\n"
	if stdout.String() != want {
		t.Errorf("stdout:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
	if stderr.String() != "" {
		t.Errorf("stderr should be empty, got: %q", stderr.String())
	}
}

func TestRunList_IDOnlyWithMoreResultsAppendsContinuation(t *testing.T) {
	t.Parallel()
	comments := []api.Comment{plainComment("1", "Alice", "a")}
	server := commentsServerWithTotal(comments, 5)
	defer server.Close()

	opts, stdout, _ := newCommentsOpts(t, server)
	opts.IDOnly = true
	err := runList(context.Background(), opts, "TEST-1", 1, false, "")
	testutil.RequireNoError(t, err)

	want := "1\nMore results available (use --next-page-token to fetch next page)\n"
	if stdout.String() != want {
		t.Errorf("stdout:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
}

func TestRunList_EmptyNeverEmitsSpuriousPaginationHint(t *testing.T) {
	t.Parallel()
	// Even when the API reports Total>0, an empty page means we cannot
	// meaningfully continue paging (no cursor advances). commentsHasMore's
	// got==0 guard ensures the pagination hint is NOT emitted — no false
	// "there are more" signal on stdout or stderr.
	server := commentsServerWithTotal(nil, 5)
	defer server.Close()

	opts, stdout, stderr := newCommentsOpts(t, server)
	err := runList(context.Background(), opts, "TEST-1", 50, false, "")
	testutil.RequireNoError(t, err)

	if strings.Contains(stdout.String(), "More results available") {
		t.Errorf("pagination hint should NOT appear for empty page; got stdout=%q", stdout.String())
	}
	if strings.Contains(stderr.String(), "More results available") {
		t.Errorf("pagination hint should NOT appear on stderr either; got stderr=%q", stderr.String())
	}
}

func TestRunList_EmptyWithIDOnly_EmitsNothing(t *testing.T) {
	t.Parallel()
	server := commentsServerWithTotal(nil, 0)
	defer server.Close()

	opts, stdout, stderr := newCommentsOpts(t, server)
	opts.IDOnly = true
	err := runList(context.Background(), opts, "TEST-1", 50, false, "")
	testutil.RequireNoError(t, err)

	if stdout.String() != "" {
		t.Errorf("stdout should be empty under --id with zero comments, got: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Errorf("stderr should be empty, got: %q", stderr.String())
	}
}

func TestRunList_EmptyDefaultGoesToStdout(t *testing.T) {
	t.Parallel()
	server := commentsServerWithTotal(nil, 0)
	defer server.Close()

	opts, stdout, stderr := newCommentsOpts(t, server)
	err := runList(context.Background(), opts, "TEST-1", 50, false, "")
	testutil.RequireNoError(t, err)

	if !strings.Contains(stdout.String(), "No comments on TEST-1") {
		t.Errorf("expected 'No comments on TEST-1' on stdout, got: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Errorf("stderr should be empty, got: %q", stderr.String())
	}
}

func TestCommentsHasMore(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                        string
		total, startAt, got, maxRes int
		want                        bool
	}{
		{"total exceeds page", 10, 0, 5, 5, true},
		{"total reached", 5, 0, 5, 5, false},
		{"total zero and full page (heuristic true)", 0, 0, 5, 5, true},
		{"total zero and partial page", 0, 0, 3, 5, false},
		{"later page, more remain", 20, 10, 5, 5, true},
		{"empty page, total zero (no more)", 0, 0, 0, 5, false},
		{"empty page, total nonzero (no more)", 10, 10, 0, 5, false},
		{"degenerate all-zeros", 0, 0, 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := commentsHasMore(tc.total, tc.startAt, tc.got, tc.maxRes)
			if got != tc.want {
				t.Errorf("commentsHasMore(total=%d startAt=%d got=%d maxRes=%d) = %v, want %v",
					tc.total, tc.startAt, tc.got, tc.maxRes, got, tc.want)
			}
		})
	}
}

func TestRunList_MultipleCommentsFullMode(t *testing.T) {
	t.Parallel()
	comments := []api.Comment{
		{
			ID:     "1",
			Author: api.User{DisplayName: "Alice"},
			Body: &api.ADFDocument{
				Type:    "doc",
				Version: 1,
				Content: []*api.ADFNode{
					{Type: "paragraph", Content: []*api.ADFNode{{Type: "text", Text: "First comment"}}},
				},
			},
			Created: "2024-01-15T10:00:00.000Z",
		},
		{
			ID:     "2",
			Author: api.User{DisplayName: "Bob"},
			Body: &api.ADFDocument{
				Type:    "doc",
				Version: 1,
				Content: []*api.ADFNode{
					{Type: "paragraph", Content: []*api.ADFNode{{Type: "text", Text: "Second comment"}}},
				},
			},
			Created: "2024-01-16T10:00:00.000Z",
		},
	}

	server := newTestCommentsServer(t, comments)
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

	err = runList(context.Background(), opts, "TEST-1", 50, true, "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "First comment")
	testutil.Contains(t, output, "Second comment")
	// Comments are now rendered as DetailSections with blank line separators (renderer-owned)
}

func TestRunAdd_IDOnly(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(api.Comment{
				ID:     "21276",
				Author: api.User{DisplayName: "Alice"},
			})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runAdd(context.Background(), opts, "TEST-1", "test body")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "21276\n")
}

func TestRunAdd_PostState(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(api.Comment{
				ID:      "21276",
				Author:  api.User{DisplayName: "Alice"},
				Created: "2024-06-15T10:00:00Z",
			})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runAdd(context.Background(), opts, "TEST-1", "test body")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "TEST-1 #21276")
	testutil.Contains(t, stdout.String(), "Alice")
}
