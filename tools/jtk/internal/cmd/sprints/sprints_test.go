package sprints

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// seedBoardsAndSprints seeds the instance-scoped caches used by the Cobra
// entry-point tests below. Pairs with cache.SetRootForTest for full
// isolation; returns nothing (cleanup runs via t.Cleanup).
func seedBoardsAndSprints(t *testing.T, boards []api.Board, byBoard map[int][]api.Sprint) {
	t.Helper()
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))
	testutil.RequireNoError(t, cache.WriteResource("boards", "24h", boards))
	if byBoard != nil {
		testutil.RequireNoError(t, cache.WriteResource("sprints", "24h", byBoard))
	}
}

// newAgileClient builds an api.Client pointed at the given server with a
// URL that triggers SupportsAgile() so the boards/sprints PersistentPreRunE
// guard passes.
func newAgileClient(t *testing.T, server *httptest.Server) *api.Client {
	t.Helper()
	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)
	return client
}

// --- list subcommand ---

func TestNewListCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newListCmd(opts)

	testutil.Equal(t, cmd.Use, "list")
	testutil.NotEmpty(t, cmd.Short)

	boardFlag := cmd.Flags().Lookup("board")
	testutil.NotNil(t, boardFlag)
	testutil.Equal(t, boardFlag.DefValue, "")

	stateFlag := cmd.Flags().Lookup("state")
	testutil.NotNil(t, stateFlag)
	testutil.Equal(t, stateFlag.DefValue, "")

	maxFlag := cmd.Flags().Lookup("max")
	testutil.NotNil(t, maxFlag)
	testutil.Equal(t, maxFlag.DefValue, "50")

	nextPageTokenFlag := cmd.Flags().Lookup("next-page-token")
	testutil.NotNil(t, nextPageTokenFlag)

	fieldsFlag := cmd.Flags().Lookup("fields")
	testutil.NotNil(t, fieldsFlag)
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

	err = runList(context.Background(), opts, client, 123, "", 50, "", "")
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

func TestRunList_IDOnly(t *testing.T) {
	t.Parallel()
	sprints := []api.Sprint{
		{ID: 10, Name: "Sprint 1", State: "active"},
		{ID: 11, Name: "Sprint 2", State: "future"},
	}

	server := newTestSprintsServer(t, sprints)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, client, 123, "", 50, "", "")
	testutil.RequireNoError(t, err)

	testutil.Equal(t, stdout.String(), "10\n11\n")
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

	err = runList(context.Background(), opts, client, 123, "", 50, "", "")
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

	err = runList(context.Background(), opts, client, 123, "", 50, "", "")
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

	err = runList(context.Background(), opts, client, 123, "", 50, "", "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Sprint Future")
	testutil.NotContains(t, output, "0001-01-01")
}

func TestRunList_FieldsWithJSONError(t *testing.T) {
	t.Parallel()
	server := newTestSprintsServer(t, []api.Sprint{{ID: 1}})
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	opts := &root.Options{Output: "json", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, client, 123, "", 50, "", "ID,NAME")
	testutil.NotNil(t, err)
	testutil.Contains(t, err.Error(), "--fields is not supported")
}

func TestRunList_InvalidNextPageToken(t *testing.T) {
	t.Parallel()
	client, err := api.New(api.ClientConfig{URL: "http://localhost", Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	opts := &root.Options{Output: "table", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, client, 123, "", 50, "abc", "")
	testutil.NotNil(t, err)
	testutil.Contains(t, err.Error(), "--next-page-token")
}

func TestRunList_Pagination(t *testing.T) {
	t.Parallel()

	// Server returns 3 sprints across 2 pages.
	allSprints := []api.Sprint{
		{ID: 10, Name: "S1", State: "active"},
		{ID: 11, Name: "S2", State: "future"},
		{ID: 12, Name: "S3", State: "closed"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startAt := 0
		if v := r.URL.Query().Get("startAt"); v != "" {
			startAt, _ = strconv.Atoi(v)
		}
		end := startAt + 2
		if end > len(allSprints) {
			end = len(allSprints)
		}
		page := allSprints[startAt:end]
		_ = json.NewEncoder(w).Encode(api.SprintsResponse{
			Values: page,
			IsLast: end >= len(allSprints),
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	// Request max=2: client-side pagination over 3 sorted sprints.
	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, client, 123, "", 2, "", "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "More results available (next: 2)")
}

// --- current subcommand ---

func TestNewCurrentCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newCurrentCmd(opts)

	testutil.Equal(t, cmd.Use, "current")
	testutil.NotEmpty(t, cmd.Short)

	boardFlag := cmd.Flags().Lookup("board")
	testutil.NotNil(t, boardFlag)
	testutil.Equal(t, boardFlag.DefValue, "")

	fieldsFlag := cmd.Flags().Lookup("fields")
	testutil.NotNil(t, fieldsFlag)
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

	board := &api.Board{ID: 123, Name: "Test Board"}
	err = runCurrent(context.Background(), opts, client, board, "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "42")
	testutil.Contains(t, output, "Sprint Active")
	testutil.Contains(t, output, "active")
	testutil.Contains(t, output, "2025-02-01")
	testutil.Contains(t, output, "2025-02-14")
	testutil.Contains(t, output, "Board: 123 (Test Board)")
}

func TestRunCurrent_IDOnly(t *testing.T) {
	t.Parallel()
	sprints := []api.Sprint{
		{ID: 42, Name: "Sprint Active", State: "active"},
	}

	server := newTestSprintsServer(t, sprints)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	board := &api.Board{ID: 123}
	err = runCurrent(context.Background(), opts, client, board, "")
	testutil.RequireNoError(t, err)

	testutil.Equal(t, stdout.String(), "42\n")
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

	board := &api.Board{ID: 123}
	err = runCurrent(context.Background(), opts, client, board, "")
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
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, Extended: true}
	opts.SetAPIClient(client)

	board := &api.Board{ID: 123, Name: "Test Board"}
	err = runCurrent(context.Background(), opts, client, board, "")
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

	board := &api.Board{ID: 123}
	err = runCurrent(context.Background(), opts, client, board, "")
	testutil.NotNil(t, err)
	testutil.Contains(t, err.Error(), "no active sprint")
}

func TestRunCurrent_SyntheticBoard(t *testing.T) {
	t.Parallel()
	sprints := []api.Sprint{
		{ID: 42, Name: "Sprint Active", State: "active"},
	}

	server := newTestSprintsServer(t, sprints)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	// Synthetic board with no name (cold cache / numeric pass-through)
	board := &api.Board{ID: 123}
	err = runCurrent(context.Background(), opts, client, board, "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Board: 123")
	testutil.NotContains(t, output, "Board: 123 (")
}

// --- issues subcommand ---

func TestNewIssuesCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newIssuesCmd(opts)

	testutil.Equal(t, cmd.Use, "issues <sprint>")
	testutil.NotEmpty(t, cmd.Short)

	maxFlag := cmd.Flags().Lookup("max")
	testutil.NotNil(t, maxFlag)
	testutil.Equal(t, maxFlag.DefValue, "50")

	nextPageTokenFlag := cmd.Flags().Lookup("next-page-token")
	testutil.NotNil(t, nextPageTokenFlag)
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

	err = runIssues(context.Background(), opts, 456, 50, "", "")
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

func TestRunIssues_IDOnly(t *testing.T) {
	issues := []api.Issue{
		{Key: "PROJ-101", Fields: api.IssueFields{Summary: "Fix login bug"}},
		{Key: "PROJ-102", Fields: api.IssueFields{Summary: "Add search"}},
	}

	server := newTestSprintIssuesServer(t, issues)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runIssues(context.Background(), opts, 456, 50, "", "")
	testutil.RequireNoError(t, err)

	testutil.Equal(t, stdout.String(), "PROJ-101\nPROJ-102\n")
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

	err = runIssues(context.Background(), opts, 456, 50, "", "")
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

	err = runIssues(context.Background(), opts, 456, 50, "", "")
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), "No issues in sprint")
}

func TestRunIssues_Fields_DynamicCustomField(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/field") {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]api.Field{
				{ID: "customfield_99999", Name: "Phantom"},
			})
			return
		}
		if strings.Contains(r.URL.Path, "/sprint/") && strings.Contains(r.URL.Path, "/issue") {
			resp := map[string]any{
				"startAt":    0,
				"maxResults": 50,
				"total":      1,
				"issues": []map[string]any{{
					"id":  "1",
					"key": "PROJ-101",
					"fields": map[string]any{
						"summary":           "Fix bug",
						"status":            map[string]any{"name": "Open"},
						"issuetype":         map[string]any{"name": "Task"},
						"customfield_99999": "phantom-val",
					},
				}},
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runIssues(context.Background(), opts, 456, 50, "", "KEY,STATUS,customfield_99999")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "PROJ-101")
	testutil.Contains(t, output, "Phantom")
	testutil.Contains(t, output, "phantom-val")
}

// --- add subcommand ---

func TestNewAddCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newAddCmd(opts)

	testutil.Equal(t, cmd.Use, "add <sprint> <issue-key>...")
	testutil.NotEmpty(t, cmd.Short)
}

func TestRunAdd_Success(t *testing.T) {
	postDone := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			postDone = true
			var body map[string]any
			err := json.NewDecoder(r.Body).Decode(&body)
			testutil.RequireNoError(t, err)

			issues, ok := body["issues"].([]any)
			testutil.True(t, ok)
			testutil.Len(t, issues, 2)

			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method == http.MethodGet && postDone {
			_ = json.NewEncoder(w).Encode(api.SearchResult{
				Total: 2,
				Issues: []api.Issue{
					{Key: "PROJ-101", Fields: api.IssueFields{Summary: "Issue 1"}},
					{Key: "PROJ-102", Fields: api.IssueFields{Summary: "Issue 2"}},
				},
			})
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, NoColor: true}
	opts.SetAPIClient(client)

	err = runAdd(context.Background(), opts, client, 123, []string{"PROJ-101", "PROJ-102"})
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), "PROJ-101")
	testutil.Contains(t, stdout.String(), "PROJ-102")
}

func TestRunAdd_SingleIssue(t *testing.T) {
	postDone := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			postDone = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method == http.MethodGet && postDone {
			_ = json.NewEncoder(w).Encode(api.SearchResult{
				Total:  1,
				Issues: []api.Issue{{Key: "PROJ-101", Fields: api.IssueFields{Summary: "Issue 1"}}},
			})
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, NoColor: true}
	opts.SetAPIClient(client)

	err = runAdd(context.Background(), opts, client, 123, []string{"PROJ-101"})
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), "PROJ-101")
}

// --- Cobra entry-point tests that exercise the resolver ---

func TestListCmd_ResolvesBoardByName(t *testing.T) {
	seedBoardsAndSprints(t,
		[]api.Board{{ID: 23, Name: "MON board", Type: "scrum"}},
		nil,
	)

	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(api.SprintsResponse{IsLast: true, Values: []api.Sprint{}})
	}))
	defer server.Close()

	client := newAgileClient(t, server)

	rootCmd, opts := root.NewCmd()
	opts.SetAPIClient(client)
	opts.Stdout = &bytes.Buffer{}
	opts.Stderr = &bytes.Buffer{}
	Register(rootCmd, opts)

	rootCmd.SetArgs([]string{"sprints", "list", "--board", "MON board"})
	err := rootCmd.Execute()
	testutil.RequireNoError(t, err)
	testutil.Equal(t, capturedPath, "/rest/agile/1.0/board/23/sprint")
}

func TestCurrentCmd_ResolvesBoardByName(t *testing.T) {
	seedBoardsAndSprints(t,
		[]api.Board{{ID: 23, Name: "MON board", Type: "scrum"}},
		nil,
	)

	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(api.SprintsResponse{
			IsLast: true,
			Values: []api.Sprint{{ID: 125, Name: "MON Sprint 70", State: "active"}},
		})
	}))
	defer server.Close()

	client := newAgileClient(t, server)

	rootCmd, opts := root.NewCmd()
	opts.SetAPIClient(client)
	opts.Stdout = &bytes.Buffer{}
	opts.Stderr = &bytes.Buffer{}
	Register(rootCmd, opts)

	rootCmd.SetArgs([]string{"sprints", "current", "--board", "MON board"})
	err := rootCmd.Execute()
	testutil.RequireNoError(t, err)
	testutil.Equal(t, capturedPath, "/rest/agile/1.0/board/23/sprint")
}

func TestIssuesCmd_ResolvesSprintByName(t *testing.T) {
	seedBoardsAndSprints(t,
		[]api.Board{{ID: 23, Name: "MON board"}},
		map[int][]api.Sprint{
			23: {{ID: 125, Name: "MON Sprint 70", State: "active"}},
		},
	)

	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(api.SearchResult{
			Total:  0,
			Issues: []api.Issue{},
		})
	}))
	defer server.Close()

	client := newAgileClient(t, server)

	rootCmd, opts := root.NewCmd()
	opts.SetAPIClient(client)
	opts.Stdout = &bytes.Buffer{}
	opts.Stderr = &bytes.Buffer{}
	Register(rootCmd, opts)

	rootCmd.SetArgs([]string{"sprints", "issues", "MON Sprint 70"})
	err := rootCmd.Execute()
	testutil.RequireNoError(t, err)
	testutil.Equal(t, capturedPath, "/rest/agile/1.0/sprint/125/issue")
}

func TestAddCmd_ResolvesSprintByName(t *testing.T) {
	seedBoardsAndSprints(t,
		[]api.Board{{ID: 23, Name: "MON board"}},
		map[int][]api.Sprint{
			23: {{ID: 125, Name: "MON Sprint 70", State: "active"}},
		},
	)

	var capturedPath string
	var capturedIssues []any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if v, ok := body["issues"].([]any); ok {
			capturedIssues = v
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newAgileClient(t, server)

	rootCmd, opts := root.NewCmd()
	opts.SetAPIClient(client)
	opts.Stdout = &bytes.Buffer{}
	opts.Stderr = &bytes.Buffer{}
	opts.NoColor = true
	Register(rootCmd, opts)

	rootCmd.SetArgs([]string{"sprints", "add", "MON Sprint 70", "PROJ-1"})
	err := rootCmd.Execute()
	testutil.RequireNoError(t, err)
	testutil.Equal(t, capturedPath, "/rest/agile/1.0/sprint/125/issue")
	testutil.Len(t, capturedIssues, 1)
}

func TestAddCmd_AmbiguousSprintAcrossBoardsErrors(t *testing.T) {
	seedBoardsAndSprints(t,
		[]api.Board{
			{ID: 23, Name: "MON board"},
			{ID: 24, Name: "ON board"},
		},
		map[int][]api.Sprint{
			23: {{ID: 125, Name: "Sprint 1", State: "active"}},
			24: {{ID: 200, Name: "Sprint 1", State: "closed"}},
		},
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("sprints add must fail before hitting the API on ambiguous input")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newAgileClient(t, server)

	rootCmd, opts := root.NewCmd()
	opts.SetAPIClient(client)
	opts.Stdout = &bytes.Buffer{}
	opts.Stderr = &bytes.Buffer{}
	opts.NoColor = true
	Register(rootCmd, opts)

	rootCmd.SetArgs([]string{"sprints", "add", "Sprint 1", "PROJ-1"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected ambiguous-match error, got nil")
	}
	if !strings.Contains(err.Error(), "Ambiguous sprint") {
		t.Fatalf("expected 'Ambiguous sprint' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "MON board") || !strings.Contains(err.Error(), "ON board") {
		t.Fatalf("expected both board names in error, got: %v", err)
	}
}

func TestAddCmd_NumericSprintPassThrough(t *testing.T) {
	seedBoardsAndSprints(t,
		[]api.Board{{ID: 23, Name: "MON"}},
		map[int][]api.Sprint{23: {{ID: 1, Name: "Other"}}},
	)

	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newAgileClient(t, server)

	rootCmd, opts := root.NewCmd()
	opts.SetAPIClient(client)
	opts.Stdout = &bytes.Buffer{}
	opts.Stderr = &bytes.Buffer{}
	opts.NoColor = true
	Register(rootCmd, opts)

	rootCmd.SetArgs([]string{"sprints", "add", "999", "PROJ-1"})
	err := rootCmd.Execute()
	testutil.RequireNoError(t, err)
	testutil.Equal(t, capturedPath, "/rest/agile/1.0/sprint/999/issue")
}

func TestRunList_SortOrder(t *testing.T) {
	t.Parallel()

	d := func(year, month, day int) *time.Time {
		tt := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		return &tt
	}

	sprints := []api.Sprint{
		{ID: 1, Name: "Old Closed", State: "closed", StartDate: d(2025, 1, 1), EndDate: d(2025, 1, 14), CompleteDate: d(2025, 1, 14)},
		{ID: 2, Name: "Future", State: "future"},
		{ID: 3, Name: "Active", State: "active", StartDate: d(2025, 4, 1), EndDate: d(2025, 4, 14)},
		{ID: 4, Name: "Recent Closed", State: "closed", StartDate: d(2025, 3, 1), EndDate: d(2025, 3, 14), CompleteDate: d(2025, 3, 14)},
	}

	server := newTestSprintsServer(t, sprints)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, client, 123, "", 50, "", "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	// lines[0] is the header; data rows start at lines[1]
	if len(lines) < 5 {
		t.Fatalf("expected header + 4 data rows, got %d lines:\n%s", len(lines), output)
	}
	testutil.True(t, strings.Contains(lines[1], "Active"))
	testutil.True(t, strings.Contains(lines[2], "Future"))
	testutil.True(t, strings.Contains(lines[3], "Recent Closed"))
	testutil.True(t, strings.Contains(lines[4], "Old Closed"))
}

func TestRunList_SortOrder_IDOnly(t *testing.T) {
	t.Parallel()

	d := func(year, month, day int) *time.Time {
		tt := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		return &tt
	}

	sprints := []api.Sprint{
		{ID: 1, Name: "Closed", State: "closed", CompleteDate: d(2025, 1, 14)},
		{ID: 2, Name: "Future", State: "future"},
		{ID: 3, Name: "Active", State: "active", StartDate: d(2025, 4, 1)},
	}

	server := newTestSprintsServer(t, sprints)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, client, 123, "", 50, "", "")
	testutil.RequireNoError(t, err)

	testutil.Equal(t, stdout.String(), "3\n2\n1\n")
}

func TestRunList_SortOrder_JSON(t *testing.T) {
	t.Parallel()

	d := func(year, month, day int) *time.Time {
		tt := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		return &tt
	}

	sprints := []api.Sprint{
		{ID: 1, Name: "Closed", State: "closed", CompleteDate: d(2025, 1, 14)},
		{ID: 2, Name: "Future", State: "future"},
		{ID: 3, Name: "Active", State: "active", StartDate: d(2025, 4, 1)},
	}

	server := newTestSprintsServer(t, sprints)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, client, 123, "", 50, "", "")
	testutil.RequireNoError(t, err)

	// Verify sort order: Active (ID=3) before Future (ID=2) before Closed (ID=1)
	output := stdout.String()
	posActive := strings.Index(output, "Active")
	posFuture := strings.Index(output, "Future")
	posClosed := strings.Index(output, "Closed")
	if posActive < 0 || posFuture < 0 || posClosed < 0 {
		t.Fatalf("expected all three sprints in JSON output:\n%s", output)
	}
	if posActive >= posFuture || posFuture >= posClosed {
		t.Errorf("expected Active before Future before Closed in JSON, got positions %d, %d, %d", posActive, posFuture, posClosed)
	}
}

func TestRunList_Extended_GoalColumn(t *testing.T) {
	t.Parallel()

	d := func(year, month, day int) *time.Time {
		tt := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		return &tt
	}

	sprints := []api.Sprint{
		{ID: 1, Name: "Closed Sprint", State: "closed", StartDate: d(2025, 1, 1), EndDate: d(2025, 1, 14), CompleteDate: d(2025, 1, 14), Goal: "Complete Q1 milestone"},
		{ID: 2, Name: "Active Sprint", State: "active", StartDate: d(2025, 4, 1), EndDate: d(2025, 4, 14), Goal: "Ship CapOne a11y fixes", OriginBoardID: 23},
	}

	server := newTestSprintsServer(t, sprints)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, Extended: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, client, 123, "", 50, "", "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	// lines[0] is the header; data rows start at lines[1]
	if len(lines) < 3 {
		t.Fatalf("expected header + 2 data rows, got %d lines:\n%s", len(lines), output)
	}
	testutil.True(t, strings.Contains(lines[1], "Ship CapOne a11y fixes"))
	testutil.True(t, strings.Contains(lines[2], "Complete Q1 milestone"))
}

func TestRunList_ClientSidePagination(t *testing.T) {
	t.Parallel()

	d := func(year, month, day int) *time.Time {
		tt := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		return &tt
	}

	sprints := []api.Sprint{
		{ID: 1, Name: "Closed A", State: "closed", CompleteDate: d(2025, 1, 14)},
		{ID: 2, Name: "Closed B", State: "closed", CompleteDate: d(2025, 2, 14)},
		{ID: 3, Name: "Active", State: "active", StartDate: d(2025, 4, 1)},
	}

	server := newTestSprintsServer(t, sprints)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	// Page 1: max=2, sorted order is Active(3), Closed B(2), Closed A(1)
	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, client, 123, "", 2, "", "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "3\n2\n")
	testutil.Contains(t, output, "More results available (next: 2)")

	// Page 2: startAt=2, max=2
	stdout.Reset()
	err = runList(context.Background(), opts, client, 123, "", 2, "2", "")
	testutil.RequireNoError(t, err)

	output = stdout.String()
	testutil.Equal(t, output, "1\n")
}

func TestRunList_ClientSidePagination_Table(t *testing.T) {
	t.Parallel()

	d := func(year, month, day int) *time.Time {
		tt := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		return &tt
	}

	sprints := []api.Sprint{
		{ID: 1, Name: "Closed", State: "closed", CompleteDate: d(2025, 1, 14)},
		{ID: 2, Name: "Future", State: "future"},
		{ID: 3, Name: "Active", State: "active", StartDate: d(2025, 4, 1)},
	}

	server := newTestSprintsServer(t, sprints)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, client, 123, "", 2, "", "")
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), "More results available (next: 2)")
}

func TestRunCurrent_BoardEnrichment(t *testing.T) {
	t.Parallel()

	seedBoardsAndSprints(t, nil, nil) // empty cache — board resolves synthetic

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/board/23/sprint"):
			_ = json.NewEncoder(w).Encode(api.SprintsResponse{
				IsLast: true,
				Values: []api.Sprint{{ID: 125, Name: "MON Sprint 70", State: "active"}},
			})
		case strings.Contains(r.URL.Path, "/board/23"):
			_ = json.NewEncoder(w).Encode(api.Board{ID: 23, Name: "MON board", Type: "scrum"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newAgileClient(t, server)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	board := &api.Board{ID: 23} // synthetic — no name
	err := runCurrent(context.Background(), opts, client, board, "")
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), "Board: 23 (MON board)")
}

func TestRunCurrent_BoardEnrichmentFailsGracefully(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/board/23/sprint"):
			_ = json.NewEncoder(w).Encode(api.SprintsResponse{
				IsLast: true,
				Values: []api.Sprint{{ID: 125, Name: "MON Sprint 70", State: "active"}},
			})
		case strings.Contains(r.URL.Path, "/board/23"):
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newAgileClient(t, server)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	board := &api.Board{ID: 23} // synthetic — no name
	err := runCurrent(context.Background(), opts, client, board, "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Board: 23")
	testutil.NotContains(t, output, "Board: 23 (")
}

func TestFetchAllSprints_Error(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	_, err = fetchAllSprints(context.Background(), client, 123, "")
	testutil.NotNil(t, err)
}

func TestFetchAllSprints_EmptyWithNotLast(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.SprintsResponse{
			Values: []api.Sprint{},
			IsLast: false,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	_, err = fetchAllSprints(context.Background(), client, 123, "")
	testutil.NotNil(t, err)
	testutil.Contains(t, err.Error(), "unexpected paginated response")
}

func TestFetchAllSprints_StatePassthrough(t *testing.T) {
	t.Parallel()
	var capturedStates []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedStates = append(capturedStates, r.URL.Query().Get("state"))
		_ = json.NewEncoder(w).Encode(api.SprintsResponse{
			Values: []api.Sprint{{ID: 1, State: "active"}},
			IsLast: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	_, err = fetchAllSprints(context.Background(), client, 123, "active")
	testutil.RequireNoError(t, err)
	if len(capturedStates) != 1 || capturedStates[0] != "active" {
		t.Errorf("expected state=active to be passed through, got %v", capturedStates)
	}
}

func TestRunList_StartAtBeyondTotal(t *testing.T) {
	t.Parallel()
	sprints := []api.Sprint{{ID: 1, Name: "S1", State: "active"}}

	server := newTestSprintsServer(t, sprints)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	// startAt=100 is beyond the 1-sprint total
	err = runList(context.Background(), opts, client, 123, "", 50, "100", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "")
}

func TestCurrentCmd_BoardEnrichmentCobraLevel(t *testing.T) {
	t.Parallel()
	seedBoardsAndSprints(t, nil, nil) // empty cache — numeric board resolves synthetic

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/board/23/sprint"):
			_ = json.NewEncoder(w).Encode(api.SprintsResponse{
				IsLast: true,
				Values: []api.Sprint{{ID: 125, Name: "MON Sprint 70", State: "active"}},
			})
		case r.URL.Path == "/rest/agile/1.0/board/23":
			_ = json.NewEncoder(w).Encode(api.Board{ID: 23, Name: "MON board", Type: "scrum"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newAgileClient(t, server)

	rootCmd, opts := root.NewCmd()
	opts.SetAPIClient(client)
	var stdout bytes.Buffer
	opts.Stdout = &stdout
	opts.Stderr = &bytes.Buffer{}
	Register(rootCmd, opts)

	rootCmd.SetArgs([]string{"sprints", "current", "--board", "23"})
	err := rootCmd.Execute()
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Board: 23 (MON board)")
}

func TestRunAdd_AgentOutput(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, NoColor: true}
	opts.SetAPIClient(client)

	err = runAdd(context.Background(), opts, client, 456, []string{"MON-123"})
	testutil.RequireNoError(t, err)

	want := "Moved MON-123 to sprint 456\n"
	if stdout.String() != want {
		t.Errorf("mutation output:\ngot: %q\nwant: %q", stdout.String(), want)
	}
	if strings.Contains(stdout.String(), "✓") {
		t.Error("agent policy should not have checkmark")
	}
}
