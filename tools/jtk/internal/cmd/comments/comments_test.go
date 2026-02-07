package comments

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestNewListCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newListCmd(opts)

	assert.Equal(t, "list <issue-key>", cmd.Use)

	// Check that full flag exists
	fullFlag := cmd.Flags().Lookup("full")
	require.NotNil(t, fullFlag)
	assert.Equal(t, "false", fullFlag.DefValue)

	// Check that max flag exists
	maxFlag := cmd.Flags().Lookup("max")
	require.NotNil(t, maxFlag)
	assert.Equal(t, "50", maxFlag.DefValue)
}

func newTestCommentsServer(t *testing.T, comments []api.Comment) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := api.CommentsResponse{
			StartAt:    0,
			MaxResults: 50,
			Total:      len(comments),
			Comments:   comments,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
}

func TestRunList_TruncatesCommentBody(t *testing.T) {
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
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runList(opts, "TEST-1", 50, false)
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "Alice")
	assert.Contains(t, output, "[truncated, use --full for complete text]")
	assert.NotContains(t, output, longText)
}

func TestRunList_FullCommentBody(t *testing.T) {
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
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runList(opts, "TEST-1", 50, true)
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, longText)
	assert.NotContains(t, output, "[truncated")
	// Full mode uses key-value layout
	assert.Contains(t, output, "ID:")
	assert.Contains(t, output, "Author:")
	assert.Contains(t, output, "Body:")
}

func TestRunList_ShortCommentNotTruncated(t *testing.T) {
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
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runList(opts, "TEST-1", 50, false)
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "Short comment")
	assert.NotContains(t, output, "[truncated")
}

func TestRunList_NoComments(t *testing.T) {
	server := newTestCommentsServer(t, []api.Comment{})
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &stderr,
	}
	opts.SetAPIClient(client)

	err = runList(opts, "TEST-1", 50, false)
	require.NoError(t, err)

	combined := stdout.String() + stderr.String()
	assert.Contains(t, combined, "No comments")
}

func TestRunList_MultipleCommentsFullMode(t *testing.T) {
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
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runList(opts, "TEST-1", 50, true)
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "First comment")
	assert.Contains(t, output, "Second comment")
	assert.Contains(t, output, "---") // separator between comments
}
