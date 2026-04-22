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

func TestNewGetCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newGetCmd(opts)

	testutil.Equal(t, cmd.Use, "get <issue-key>")
	testutil.Equal(t, cmd.Short, "Get issue details")

	// Check that no-truncate flag exists
	noTruncateFlag := cmd.Flags().Lookup("no-truncate")
	testutil.NotNil(t, noTruncateFlag)
	testutil.Equal(t, noTruncateFlag.DefValue, "false")
}

func newTestIssueServer(_ *testing.T, issue api.Issue) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(issue)
	}))
}

func TestRunGet_TruncatesDescription(t *testing.T) {
	t.Parallel()
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
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "TEST-1", false, "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "TEST-1")
	testutil.Contains(t, output, "[truncated — use --fulltext for complete body]")
	testutil.NotContains(t, output, longText)
}

func TestRunGet_FullDescription(t *testing.T) {
	t.Parallel()
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
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "TEST-1", true, "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, longText)
	testutil.NotContains(t, output, "[truncated")
}

// TestNewGetCmd_FullTextRoutesFromRoot verifies that when --fulltext is set on
// the root Options (as the persistent --fulltext flag does), runGet is invoked
// with noTruncate=true even though the local --no-truncate flag is not set.
func TestNewGetCmd_FullTextRoutesFromRoot(t *testing.T) {
	t.Parallel()
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
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output:   "table",
		FullText: true, // global --fulltext
		Stdout:   &stdout,
		Stderr:   &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	cmd := newGetCmd(opts)
	cmd.SetArgs([]string{"TEST-1"}) // no --no-truncate locally
	testutil.RequireNoError(t, cmd.Execute())

	output := stdout.String()
	testutil.Contains(t, output, longText)
	testutil.NotContains(t, output, "[truncated")
}

// TestNewGetCmd_NoTruncateAndFullTextBothSet guards the OR-combined path:
// both the local --no-truncate flag and the global --fulltext must produce
// the same result when set together (prevents accidental && regression).
func TestNewGetCmd_NoTruncateAndFullTextBothSet(t *testing.T) {
	t.Parallel()
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
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output:   "table",
		FullText: true,
		Stdout:   &stdout,
		Stderr:   &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	cmd := newGetCmd(opts)
	cmd.SetArgs([]string{"TEST-1", "--no-truncate"})
	testutil.RequireNoError(t, cmd.Execute())

	output := stdout.String()
	testutil.Contains(t, output, longText)
	testutil.NotContains(t, output, "[truncated")
}

func TestRunGet_IDOnly(t *testing.T) {
	t.Parallel()
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

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", IDOnly: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	testutil.RequireNoError(t, runGet(context.Background(), opts, "TEST-1", false, ""))
	testutil.Equal(t, stdout.String(), "TEST-1\n")
}

func TestRunGet_IDOnlyPrecedenceOverExtendedFullText(t *testing.T) {
	t.Parallel()
	issue := api.Issue{
		Key: "TEST-1",
		Fields: api.IssueFields{
			Summary:     "Test issue",
			Description: &api.Description{Text: strings.Repeat("A", 300)},
			Status:      &api.Status{Name: "Open"},
			IssueType:   &api.IssueType{Name: "Task"},
		},
	}

	server := newTestIssueServer(t, issue)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", IDOnly: true, Extended: true, FullText: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	// runGet receives noTruncate derived from RunE; when --id is set, the truncation
	// value doesn't matter because EmitIDOnly collapses output before presenter runs.
	testutil.RequireNoError(t, runGet(context.Background(), opts, "TEST-1", true, ""))
	testutil.Equal(t, stdout.String(), "TEST-1\n")
}

func TestRunGet_ShortDescriptionNotTruncated(t *testing.T) {
	t.Parallel()
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
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "TEST-1", false, "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Short description")
	testutil.NotContains(t, output, "[truncated")
}

func TestRunGet_JSONOutputIgnoresFullFlag(t *testing.T) {
	t.Parallel()
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
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "json",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "TEST-1", true, "")
	testutil.RequireNoError(t, err)

	// Should be valid JSON
	var result api.Issue
	err = json.Unmarshal(stdout.Bytes(), &result)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, result.Key, "TEST-1")
}
