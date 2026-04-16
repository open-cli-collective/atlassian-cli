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

	err = runList(context.Background(), opts, "TEST-1", 50, false)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Alice")
	testutil.Contains(t, output, "[truncated, use --fulltext for complete text]")
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

	err = runList(context.Background(), opts, "TEST-1", 50, true)
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

	err = runList(context.Background(), opts, "TEST-1", 50, false)
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

	err = runList(context.Background(), opts, "TEST-1", 50, false)
	testutil.RequireNoError(t, err)

	combined := stdout.String() + stderr.String()
	testutil.Contains(t, combined, "No comments")
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

	err = runList(context.Background(), opts, "TEST-1", 50, true)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "First comment")
	testutil.Contains(t, output, "Second comment")
	// Comments are now rendered as DetailSections with blank line separators (renderer-owned)
}
