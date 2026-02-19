package sprints

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// --- list subcommand ---

func TestNewListCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newListCmd(opts)

	testutil.Equal(t, cmd.Use, "list")
	testutil.NotEmpty(t, cmd.Short)

	boardFlag := cmd.Flags().Lookup("board")
	testutil.NotNil(t, boardFlag)
	testutil.Equal(t, boardFlag.DefValue, "0")

	stateFlag := cmd.Flags().Lookup("state")
	testutil.NotNil(t, stateFlag)
	testutil.Equal(t, stateFlag.DefValue, "")

	maxFlag := cmd.Flags().Lookup("max")
	testutil.NotNil(t, maxFlag)
	testutil.Equal(t, maxFlag.DefValue, "50")
}

func newTestSprintsServer(_ *testing.T, sprints []api.Sprint) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := api.SprintsResponse{
			MaxResults: 50,
			StartAt:    0,
			IsLast:     true,
			Values:     sprints,
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
}

func TestRunList_Table(t *testing.T) {
	t.Parallel()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 14, 0, 0, 0, 0, time.UTC)
	sprints := []api.Sprint{
		{ID: 10, Name: "Sprint 1", State: "active", StartDate: &start, EndDate: &end},
		{ID: 11, Name: "Sprint 2", State: "future"},
	}

	server := newTestSprintsServer(t, sprints)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, 123, "", 50)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "10")
	testutil.Contains(t, output, "Sprint 1")
	testutil.Contains(t, output, "active")
	testutil.Contains(t, output, "2025-01-01")
	testutil.Contains(t, output, "2025-01-14")
	testutil.Contains(t, output, "11")
	testutil.Contains(t, output, "Sprint 2")
	testutil.Contains(t, output, "future")
}

func TestRunList_JSON(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sprints := []api.Sprint{
		{ID: 10, Name: "Sprint 1", State: "active", StartDate: &start},
	}

	server := newTestSprintsServer(t, sprints)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, 123, "", 50)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, `"name"`)
	testutil.Contains(t, output, "Sprint 1")
	testutil.Contains(t, output, `"state"`)
}

func TestRunList_Empty(t *testing.T) {
	server := newTestSprintsServer(t, []api.Sprint{})
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, 123, "", 50)
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), "No sprints found")
}

func TestRunList_NullDates(t *testing.T) {
	sprints := []api.Sprint{
		{ID: 10, Name: "Sprint Future", State: "future", StartDate: nil, EndDate: nil},
	}

	server := newTestSprintsServer(t, sprints)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, 123, "", 50)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Sprint Future")
	// Dates should be empty strings (not "N/A") when nil
	testutil.NotContains(t, output, "0001-01-01")
}

// --- current subcommand ---

func TestNewCurrentCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newCurrentCmd(opts)

	testutil.Equal(t, cmd.Use, "current")
	testutil.NotEmpty(t, cmd.Short)

	boardFlag := cmd.Flags().Lookup("board")
	testutil.NotNil(t, boardFlag)
	testutil.Equal(t, boardFlag.DefValue, "0")
}

func TestRunCurrent_Table(t *testing.T) {
	start := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 2, 14, 0, 0, 0, 0, time.UTC)
	sprints := []api.Sprint{
		{ID: 42, Name: "Sprint Active", State: "active", StartDate: &start, EndDate: &end},
	}

	server := newTestSprintsServer(t, sprints)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCurrent(context.Background(), opts, 123)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "42")
	testutil.Contains(t, output, "Sprint Active")
	testutil.Contains(t, output, "active")
	testutil.Contains(t, output, "2025-02-01")
	testutil.Contains(t, output, "2025-02-14")
}

func TestRunCurrent_JSON(t *testing.T) {
	start := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	sprints := []api.Sprint{
		{ID: 42, Name: "Sprint Active", State: "active", StartDate: &start},
	}

	server := newTestSprintsServer(t, sprints)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCurrent(context.Background(), opts, 123)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, `"id"`)
	testutil.Contains(t, output, `"name"`)
	testutil.Contains(t, output, "Sprint Active")
}

func TestRunCurrent_WithGoal(t *testing.T) {
	start := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	sprints := []api.Sprint{
		{ID: 42, Name: "Sprint Active", State: "active", StartDate: &start, Goal: "Ship feature X"},
	}

	server := newTestSprintsServer(t, sprints)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCurrent(context.Background(), opts, 123)
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), "Ship feature X")
}

func TestRunCurrent_NotFound(t *testing.T) {
	server := newTestSprintsServer(t, []api.Sprint{})
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCurrent(context.Background(), opts, 123)
	testutil.NotNil(t, err)
	testutil.Contains(t, err.Error(), "no active sprint")
}

// --- issues subcommand ---

func TestNewIssuesCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newIssuesCmd(opts)

	testutil.Equal(t, cmd.Use, "issues <sprint-id>")
	testutil.NotEmpty(t, cmd.Short)

	maxFlag := cmd.Flags().Lookup("max")
	testutil.NotNil(t, maxFlag)
	testutil.Equal(t, maxFlag.DefValue, "50")
}

func newTestSprintIssuesServer(_ *testing.T, issues []api.Issue) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := api.SearchResult{
			StartAt:    0,
			MaxResults: 50,
			Total:      len(issues),
			Issues:     issues,
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
}

func TestRunIssues_Table(t *testing.T) {
	issues := []api.Issue{
		{
			Key: "PROJ-101",
			Fields: api.IssueFields{
				Summary:   "Fix login bug",
				Status:    &api.Status{Name: "In Progress"},
				Assignee:  &api.User{DisplayName: "John Doe"},
				IssueType: &api.IssueType{Name: "Bug"},
			},
		},
		{
			Key: "PROJ-102",
			Fields: api.IssueFields{
				Summary:   "Add search feature",
				Status:    &api.Status{Name: "To Do"},
				IssueType: &api.IssueType{Name: "Story"},
			},
		},
	}

	server := newTestSprintIssuesServer(t, issues)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runIssues(context.Background(), opts, 456, 50)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "PROJ-101")
	testutil.Contains(t, output, "Fix login bug")
	testutil.Contains(t, output, "In Progress")
	testutil.Contains(t, output, "John Doe")
	testutil.Contains(t, output, "Bug")
	testutil.Contains(t, output, "PROJ-102")
	testutil.Contains(t, output, "Add search feature")
	testutil.Contains(t, output, "Story")
}

func TestRunIssues_JSON(t *testing.T) {
	issues := []api.Issue{
		{
			Key: "PROJ-101",
			Fields: api.IssueFields{
				Summary: "Fix login bug",
				Status:  &api.Status{Name: "In Progress"},
			},
		},
	}

	server := newTestSprintIssuesServer(t, issues)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runIssues(context.Background(), opts, 456, 50)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, `"key"`)
	testutil.Contains(t, output, "PROJ-101")
}

func TestRunIssues_Empty(t *testing.T) {
	server := newTestSprintIssuesServer(t, []api.Issue{})
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runIssues(context.Background(), opts, 456, 50)
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), "No issues in sprint")
}

// --- add subcommand ---

func TestNewAddCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newAddCmd(opts)

	testutil.Equal(t, cmd.Use, "add <sprint-id> <issue-key>...")
	testutil.NotEmpty(t, cmd.Short)
}

func TestRunAdd_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodPost)

		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		testutil.RequireNoError(t, err)

		issues, ok := body["issues"].([]any)
		testutil.True(t, ok)
		testutil.Len(t, issues, 2)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, NoColor: true}
	opts.SetAPIClient(client)

	err = runAdd(context.Background(), opts, 123, []string{"PROJ-101", "PROJ-102"})
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), fmt.Sprintf("Moved 2 issues to sprint %d", 123))
}

func TestRunAdd_SingleIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, NoColor: true}
	opts.SetAPIClient(client)

	err = runAdd(context.Background(), opts, 123, []string{"PROJ-101"})
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), "Moved PROJ-101 to sprint 123")
}
