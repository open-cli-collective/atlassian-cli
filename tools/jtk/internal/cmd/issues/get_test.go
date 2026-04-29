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
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
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

	err = runGet(context.Background(), opts, "TEST-1", false, "", false)
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

	err = runGet(context.Background(), opts, "TEST-1", true, "", false)
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

	testutil.RequireNoError(t, runGet(context.Background(), opts, "TEST-1", false, "", false))
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
	testutil.RequireNoError(t, runGet(context.Background(), opts, "TEST-1", true, "", false))
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

	err = runGet(context.Background(), opts, "TEST-1", false, "", false)
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

	err = runGet(context.Background(), opts, "TEST-1", true, "", false)
	testutil.RequireNoError(t, err)

	// Should be valid JSON
	var result api.Issue
	err = json.Unmarshal(stdout.Bytes(), &result)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, result.Key, "TEST-1")
}

func TestRunGet_Extended_ShowsNormalizedSections(t *testing.T) {
	t.Parallel()
	issue := api.Issue{
		Key: "TEST-1",
		Fields: api.IssueFields{
			Summary:     "Test issue",
			Status:      &api.Status{Name: "Open", StatusCategory: api.StatusCategory{Name: "To Do"}},
			IssueType:   &api.IssueType{Name: "Task"},
			Reporter:    &api.User{DisplayName: "Bob"},
			Resolution:  &api.Resolution{Name: "Done"},
			FixVersions: []api.Version{{ID: "1", Name: "v1.0"}},
			Created:     "2026-04-01T12:00:00.000+0000",
			Description: &api.Description{Text: strings.Repeat("A", 300)},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(issue)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Extended: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "TEST-1", false, "", false)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Reporter: Bob")
	testutil.Contains(t, output, "Fix Versions: v1.0")
	testutil.Contains(t, output, "Resolution: Done")
	// Extended implies fulltext — full description present
	testutil.NotContains(t, output, "[truncated")
	// Kitchen-sink items no longer present
	testutil.NotContains(t, output, "Transitions:")
	testutil.NotContains(t, output, "Watchers:")
	testutil.NotContains(t, output, "category:")
}

func TestRunGet_Extended_SprintFromCustomField(t *testing.T) {
	t.Parallel()
	issueJSON := `{
		"key": "MON-4970",
		"fields": {
			"summary": "Sprint test issue",
			"status": {"name": "In Development"},
			"issuetype": {"name": "Task"},
			"customfield_10020": [
				{"id": 100, "name": "Sprint 69", "state": "closed"},
				{"id": 125, "name": "MON Sprint 70", "state": "active", "startDate": "2026-04-10T00:00:00.000Z", "endDate": "2026-04-24T00:00:00.000Z"}
			]
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(issueJSON))
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Extended: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "MON-4970", false, "", false)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Sprint: MON Sprint 70 (active)")
}

func TestRunGet_CustomFields_AppendsSection(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))

	issueJSON := `{
		"key": "TEST-1",
		"fields": {
			"summary": "Test issue",
			"status": {"name": "Open"},
			"issuetype": {"name": "Task"},
			"customfield_10005": {"value": "Bug"}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/field") {
			_ = json.NewEncoder(w).Encode([]api.Field{
				{ID: "customfield_10005", Name: "Change type", Custom: true, Schema: api.FieldSchema{Type: "option"}},
			})
			return
		}
		_, _ = w.Write([]byte(issueJSON))
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "TEST-1", false, "", true)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Custom Fields:")
	testutil.Contains(t, output, "Change type: Bug")
}

func TestRunGet_CustomFields_WithJSON_Errors(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.Issue{Key: "TEST-1"})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	opts := &root.Options{Output: "json", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "TEST-1", false, "", true)
	testutil.NotNil(t, err)
	testutil.Contains(t, err.Error(), "not supported")
}
