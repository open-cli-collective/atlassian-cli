package issues

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

func TestNewGetCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newGetCmd(opts)

	assert.Equal(t, "get <issue-key>", cmd.Use)
	assert.Equal(t, "Get issue details", cmd.Short)

	// Check that full flag exists
	fullFlag := cmd.Flags().Lookup("full")
	require.NotNil(t, fullFlag)
	assert.Equal(t, "false", fullFlag.DefValue)
}

func newTestIssueServer(t *testing.T, issue api.Issue) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(issue)
	}))
}

func TestRunGet_TruncatesDescription(t *testing.T) {
	longText := strings.Repeat("A", 300)
	issue := api.Issue{
		Key: "TEST-1",
		Fields: api.IssueFields{
			Summary:     "Test issue",
			Description: &api.Description{Text: longText},
			Status:      &api.Status{Name: "Open"},
			IssueType:   &api.IssueType{Name: "Task"},
		},
	}

	server := newTestIssueServer(t, issue)
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

	err = runGet(opts, "TEST-1", false)
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "TEST-1")
	assert.Contains(t, output, "[truncated, use --full for complete text]")
	assert.NotContains(t, output, longText)
}

func TestRunGet_FullDescription(t *testing.T) {
	longText := strings.Repeat("A", 300)
	issue := api.Issue{
		Key: "TEST-1",
		Fields: api.IssueFields{
			Summary:     "Test issue",
			Description: &api.Description{Text: longText},
			Status:      &api.Status{Name: "Open"},
			IssueType:   &api.IssueType{Name: "Task"},
		},
	}

	server := newTestIssueServer(t, issue)
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

	err = runGet(opts, "TEST-1", true)
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, longText)
	assert.NotContains(t, output, "[truncated")
}

func TestRunGet_ShortDescriptionNotTruncated(t *testing.T) {
	issue := api.Issue{
		Key: "TEST-1",
		Fields: api.IssueFields{
			Summary:     "Test issue",
			Description: &api.Description{Text: "Short description"},
			Status:      &api.Status{Name: "Open"},
			IssueType:   &api.IssueType{Name: "Task"},
		},
	}

	server := newTestIssueServer(t, issue)
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

	err = runGet(opts, "TEST-1", false)
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "Short description")
	assert.NotContains(t, output, "[truncated")
}

func TestRunGet_JSONOutputIgnoresFullFlag(t *testing.T) {
	issue := api.Issue{
		Key: "TEST-1",
		Fields: api.IssueFields{
			Summary:   "Test issue",
			Status:    &api.Status{Name: "Open"},
			IssueType: &api.IssueType{Name: "Task"},
		},
	}

	server := newTestIssueServer(t, issue)
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "json",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runGet(opts, "TEST-1", true)
	require.NoError(t, err)

	// Should be valid JSON
	var result api.Issue
	err = json.Unmarshal(stdout.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "TEST-1", result.Key)
}
