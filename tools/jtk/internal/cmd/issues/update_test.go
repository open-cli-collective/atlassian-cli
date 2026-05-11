package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestRunUpdate_RequestBodyNoDoubleQuoting(t *testing.T) {
	t.Parallel()
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(context.Background(), opts, "PROJ-123", "Updated summary", "Updated description", "", "", "", "", nil)
	testutil.RequireNoError(t, err)

	testutil.NotEmpty(t, capturedBody)

	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)

	// Summary must be the exact string without extra quotes
	summary := fields["summary"].(string)
	testutil.Equal(t, summary, "Updated summary")
	testutil.NotContains(t, summary, `"`)

	// Description should be ADF format
	desc := fields["description"].(map[string]any)
	testutil.Equal(t, desc["type"], "doc")
	content := desc["content"].([]any)
	testutil.NotEmpty(t, content)

	firstPara := content[0].(map[string]any)
	paraContent := firstPara["content"].([]any)
	firstTextNode := paraContent[0].(map[string]any)
	descText := firstTextNode["text"].(string)
	testutil.Equal(t, descText, "Updated description")
}

func TestNewUpdateCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newUpdateCmd(opts)

	testutil.Equal(t, cmd.Use, "update <issue-key>")
	testutil.Equal(t, cmd.Short, "Update an issue")

	summaryFlag := cmd.Flags().Lookup("summary")
	testutil.NotNil(t, summaryFlag)
	testutil.Equal(t, summaryFlag.Shorthand, "s")

	descFlag := cmd.Flags().Lookup("description")
	testutil.NotNil(t, descFlag)
	testutil.Equal(t, descFlag.Shorthand, "d")

	parentFlag := cmd.Flags().Lookup("parent")
	testutil.NotNil(t, parentFlag)
	testutil.Equal(t, parentFlag.Shorthand, "")

	assigneeFlag := cmd.Flags().Lookup("assignee")
	testutil.NotNil(t, assigneeFlag)
	testutil.Equal(t, assigneeFlag.Shorthand, "a")

	typeFlag := cmd.Flags().Lookup("type")
	testutil.NotNil(t, typeFlag)
	testutil.Equal(t, typeFlag.Shorthand, "t")
}

func TestRunUpdate_TypeChange(t *testing.T) {
	seedCacheForIssues(t)
	// Override the generic Task=10000 seed with the mapping this test expects
	// end-to-end: PROJ's Task type has ID 10001.
	testutil.RequireNoError(t, cache.WriteResource("issuetypes", "24h", map[string][]api.IssueType{
		"PROJ": {
			{ID: "10000", Name: "Epic"},
			{ID: "10001", Name: "Task"},
			{ID: "10002", Name: "Story"},
		},
	}))
	var moveBody []byte
	moveCompleted := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/api/3/issue/PROJ-123" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(api.Issue{
				Key: "PROJ-123",
				ID:  "10001",
				Fields: api.IssueFields{
					Project:   &api.Project{Key: "PROJ"},
					IssueType: &api.IssueType{ID: "10000", Name: "Epic"},
				},
			})
		case r.URL.Path == "/rest/api/3/project/PROJ" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(struct {
				IssueTypes []api.IssueType `json:"issueTypes"`
			}{
				IssueTypes: []api.IssueType{
					{ID: "10000", Name: "Epic"},
					{ID: "10001", Name: "Task"},
					{ID: "10002", Name: "Story"},
				},
			})
		case r.URL.Path == "/rest/api/3/bulk/issues/move" && r.Method == "POST":
			moveBody, _ = io.ReadAll(r.Body)
			moveCompleted = true
			_ = json.NewEncoder(w).Encode(api.MoveIssuesResponse{TaskID: "task-123"})
		case r.URL.Path == "/rest/api/3/bulk/queue/task-123" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(api.MoveTaskStatus{
				TaskID:   "task-123",
				Status:   "COMPLETE",
				Progress: 100,
				Result:   &api.MoveTaskResult{Successful: []string{"PROJ-123"}},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(context.Background(), opts, "PROJ-123", "", "", "", "", "Task", "", nil)
	testutil.RequireNoError(t, err)
	testutil.True(t, moveCompleted, "should have called the move API")

	// Verify move request body
	var moveReq api.MoveIssuesRequest
	err = json.Unmarshal(moveBody, &moveReq)
	testutil.RequireNoError(t, err)

	// The target key should be "PROJ,10001" (project key, Task type ID)
	spec, ok := moveReq.TargetToSourcesMapping["PROJ,10001"]
	testutil.True(t, ok, "should have mapping for PROJ,10001")
	testutil.Equal(t, spec.IssueIdsOrKeys, []string{"PROJ-123"})
}

func TestRunUpdate_TypeAlreadyCorrect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue/PROJ-123" && r.Method == "GET" {
			issue := map[string]any{
				"key": "PROJ-123",
				"id":  "10001",
				"fields": map[string]any{
					"summary":   "Test issue",
					"status":    map[string]any{"name": "Backlog"},
					"issuetype": map[string]any{"id": "10001", "name": "Task"},
					"priority":  map[string]any{"name": "Medium"},
					"project":   map[string]any{"key": "PROJ"},
					"updated":   "2026-04-16T00:00:00.000+0000",
				},
			}
			_ = json.NewEncoder(w).Encode(issue)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	// Should succeed without calling move API since it's already the right type.
	// The silent changeIssueType returns nil (no-op), then WriteAndPresent
	// re-fetches and shows post-state detail.
	err = runUpdate(context.Background(), opts, "PROJ-123", "", "", "", "", "Task", "", nil)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "PROJ-123")
}

func TestRunUpdate_SummaryOnly(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(context.Background(), opts, "PROJ-123", "New summary", "", "", "", "", "", nil)
	testutil.RequireNoError(t, err)

	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)
	testutil.Equal(t, fields["summary"], "New summary")
	testutil.Nil(t, fields["description"])
	testutil.Nil(t, fields["parent"])
}

func TestRunUpdate_IDOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "e@x", APIToken: "t"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runUpdate(context.Background(), opts, "PROJ-123", "New summary", "", "", "", "", "", nil)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "PROJ-123\n")
}

func TestRunUpdate_NoFieldsError(t *testing.T) {
	opts := &root.Options{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	err := runUpdate(context.Background(), opts, "PROJ-123", "", "", "", "", "", "", nil)
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "no fields specified")
}

func TestRunUpdate_ParentOnly(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(context.Background(), opts, "PROJ-456", "", "", "PROJ-100", "", "", "", nil)
	testutil.RequireNoError(t, err)

	testutil.NotEmpty(t, capturedBody)
	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)
	parentField := fields["parent"].(map[string]any)
	testutil.Equal(t, parentField["key"], "PROJ-100")
	testutil.Nil(t, fields["summary"])
	testutil.Nil(t, fields["description"])
}

func TestRunUpdate_ParentWithSummary(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(context.Background(), opts, "PROJ-456", "Updated title", "", "PROJ-200", "", "", "", nil)
	testutil.RequireNoError(t, err)

	testutil.NotEmpty(t, capturedBody)
	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)
	testutil.Equal(t, fields["summary"], "Updated title")
	parentField := fields["parent"].(map[string]any)
	testutil.Equal(t, parentField["key"], "PROJ-200")
}

func TestUpdateCmd_CobraExecution_WithParent(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	cmd := newUpdateCmd(opts)
	cmd.SetArgs([]string{
		"PROJ-456",
		"--parent", "PROJ-100",
	})

	err = cmd.Execute()
	testutil.RequireNoError(t, err)

	testutil.NotEmpty(t, capturedBody)
	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)
	parentField := fields["parent"].(map[string]any)
	testutil.Equal(t, parentField["key"], "PROJ-100")
}

func TestRunUpdate_AssigneeOnly(t *testing.T) {
	// The assignee resolver reads the cache before falling through to
	// accountId shape pass-through, so InstanceKey() must resolve.
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(context.Background(), opts, "PROJ-789", "", "", "", "61292e4c4f29230069621c5f", "", "", nil)
	testutil.RequireNoError(t, err)

	testutil.NotEmpty(t, capturedBody)
	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)
	assigneeField := fields["assignee"].(map[string]any)
	testutil.Equal(t, assigneeField["accountId"], "61292e4c4f29230069621c5f")
	testutil.Nil(t, fields["summary"])
}

func TestRunUpdate_AssigneeMe(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/myself" && r.Method == "GET" {
			_ = json.NewEncoder(w).Encode(api.User{
				AccountID:   "myself-account-id",
				DisplayName: "Test User",
			})
			return
		}
		if r.Method == "PUT" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(context.Background(), opts, "PROJ-789", "", "", "", "me", "", "", nil)
	testutil.RequireNoError(t, err)

	testutil.NotEmpty(t, capturedBody)
	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)
	assigneeField := fields["assignee"].(map[string]any)
	testutil.Equal(t, assigneeField["accountId"], "myself-account-id")
}

func TestUpdateCmd_CobraExecution_WithAssignee(t *testing.T) {
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	cmd := newUpdateCmd(opts)
	cmd.SetArgs([]string{
		"PROJ-789",
		"--assignee", "61292e4c4f29230069621c5f",
	})

	err = cmd.Execute()
	testutil.RequireNoError(t, err)

	testutil.NotEmpty(t, capturedBody)
	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)
	assigneeField := fields["assignee"].(map[string]any)
	testutil.Equal(t, assigneeField["accountId"], "61292e4c4f29230069621c5f")
}

func TestRunUpdate_TypeChange_MoveNotFound_ServerDCError(t *testing.T) {
	seedCacheForIssues(t)
	testutil.RequireNoError(t, cache.WriteResource("issuetypes", "24h", map[string][]api.IssueType{
		"PROJ": {
			{ID: "10000", Name: "Epic"},
			{ID: "10001", Name: "Task"},
		},
	}))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/api/3/issue/PROJ-123" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(api.Issue{
				Key: "PROJ-123",
				ID:  "10001",
				Fields: api.IssueFields{
					Project:   &api.Project{Key: "PROJ"},
					IssueType: &api.IssueType{ID: "10000", Name: "Epic"},
				},
			})
		case r.URL.Path == "/rest/api/3/bulk/issues/move" && r.Method == "POST":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errorMessages":["not found"]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	opts := &root.Options{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runUpdate(context.Background(), opts, "PROJ-123", "", "", "", "", "Task", "", nil)
	if err == nil {
		t.Fatal("expected error for Server/DC detection")
	}
	testutil.Contains(t, err.Error(), "requires Jira Cloud")
}

func TestRunUpdate_TypeChange_PollNotFound_ContinuesFieldUpdates(t *testing.T) {
	seedCacheForIssues(t)
	testutil.RequireNoError(t, cache.WriteResource("issuetypes", "24h", map[string][]api.IssueType{
		"PROJ": {
			{ID: "10000", Name: "Epic"},
			{ID: "10001", Name: "Task"},
		},
	}))

	var putCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/api/3/issue/PROJ-123" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(api.Issue{
				Key: "PROJ-123",
				ID:  "10001",
				Fields: api.IssueFields{
					Summary:   "Original",
					Project:   &api.Project{Key: "PROJ"},
					IssueType: &api.IssueType{ID: "10000", Name: "Epic"},
					Status:    &api.Status{Name: "Open"},
				},
			})
		case r.URL.Path == "/rest/api/3/issue/PROJ-123" && r.Method == "PUT":
			putCalled = true
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/rest/api/3/bulk/issues/move" && r.Method == "POST":
			_ = json.NewEncoder(w).Encode(api.MoveIssuesResponse{TaskID: "task-1"})
		case strings.HasPrefix(r.URL.Path, "/rest/api/3/bulk/queue/"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errorMessages":["not found"]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)

	err = runUpdate(context.Background(), opts, "PROJ-123", "New summary", "", "", "", "Task", "", nil)
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stderr.String(), "could not be verified")
	testutil.True(t, putCalled, "field update PUT should still have been called")
}

// --- Tests for --status (#358) ----------------------------------------------

// statusServerConfig drives the mock server used by --status tests.
type statusServerConfig struct {
	currentStatus     string           // status name returned on the initial issue GET
	transitions       []api.Transition // transitions returned by GET /transitions
	transitionsErr    int              // if non-zero, /transitions returns this HTTP status
	postTransitionErr int              // if non-zero, POST /transitions returns this HTTP status
}

type statusServerRecord struct {
	getIssueCalls  int
	getTransitions int
	postTransition int
	putIssue       int
	transitionBody []byte
}

func newStatusServer(t *testing.T, cfg statusServerConfig) (*httptest.Server, *statusServerRecord) {
	t.Helper()
	rec := &statusServerRecord{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/api/3/issue/PROJ-123" && r.Method == "GET":
			rec.getIssueCalls++
			// After a successful transition POST the issue reflects the new
			// status, so subsequent GETs return it. This lets the IsFresh
			// contract actually be tested rather than always reading the
			// pre-write status. Only safe when there is a single transition
			// (unambiguous target).
			currentStatus := cfg.currentStatus
			if rec.postTransition > 0 && cfg.postTransitionErr == 0 && len(cfg.transitions) == 1 {
				currentStatus = cfg.transitions[0].To.Name
			}
			issue := map[string]any{
				"key": "PROJ-123",
				"id":  "10001",
				"fields": map[string]any{
					"summary":   "Test",
					"status":    map[string]any{"name": currentStatus},
					"project":   map[string]any{"key": "PROJ"},
					"issuetype": map[string]any{"id": "10001", "name": "Task"},
				},
			}
			_ = json.NewEncoder(w).Encode(issue)
		case r.URL.Path == "/rest/api/3/issue/PROJ-123/transitions" && r.Method == "GET":
			rec.getTransitions++
			if cfg.transitionsErr != 0 {
				w.WriteHeader(cfg.transitionsErr)
				return
			}
			_ = json.NewEncoder(w).Encode(api.TransitionsResponse{Transitions: cfg.transitions})
		case r.URL.Path == "/rest/api/3/issue/PROJ-123/transitions" && r.Method == "POST":
			rec.postTransition++
			rec.transitionBody, _ = io.ReadAll(r.Body)
			if cfg.postTransitionErr != 0 {
				w.WriteHeader(cfg.postTransitionErr)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/rest/api/3/issue/PROJ-123" && r.Method == "PUT":
			rec.putIssue++
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

func newOptsForServer(t *testing.T, url string) (*root.Options, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	client, err := api.New(api.ClientConfig{URL: url, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	opts := &root.Options{Stdout: stdout, Stderr: stderr}
	opts.SetAPIClient(client)
	return opts, stdout, stderr
}

func TestRunUpdate_StatusOnly_HappyPath(t *testing.T) {
	srv, rec := newStatusServer(t, statusServerConfig{
		currentStatus: "To Do",
		transitions: []api.Transition{
			{ID: "31", Name: "Complete", To: api.Status{Name: "Done"}},
		},
	})
	opts, stdout, _ := newOptsForServer(t, srv.URL)

	err := runUpdate(context.Background(), opts, "PROJ-123", "", "", "", "", "", "Done", nil)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, rec.postTransition, 1)
	testutil.Equal(t, rec.putIssue, 0)

	var body api.TransitionRequest
	testutil.RequireNoError(t, json.Unmarshal(rec.transitionBody, &body))
	testutil.Equal(t, body.Transition.ID, "31")
	testutil.Contains(t, stdout.String(), "PROJ-123")
	// Freshness: post-write fetch must surface the new status, not the
	// pre-write one. (newStatusServer flips the status after a successful POST.)
	testutil.Contains(t, stdout.String(), "Done")
}

// TestRunUpdate_Status_FreshnessRetries proves IsFresh is wired to the
// resolved target status. The first post-write fetch returns the stale status
// ("To Do") and subsequent fetches return the fresh status ("Done"). With
// IsFresh correctly wired, the renderer retries past the stale read and
// emits a model that contains "Done". Without it, the first stale model
// would be emitted immediately and "To Do" would appear in stdout.
func TestRunUpdate_Status_FreshnessRetries(t *testing.T) {
	var (
		postCount int
		getCount  int
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/api/3/issue/PROJ-123" && r.Method == "GET":
			getCount++
			// First GET is the preflight; second is the first post-write fetch
			// (stale); third+ are the freshness retries (fresh).
			status := "To Do"
			if postCount > 0 && getCount > 2 {
				status = "Done"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"key": "PROJ-123",
				"fields": map[string]any{
					"status":  map[string]any{"name": status},
					"project": map[string]any{"key": "PROJ"},
				},
			})
		case r.URL.Path == "/rest/api/3/issue/PROJ-123/transitions" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(api.TransitionsResponse{Transitions: []api.Transition{
				{ID: "31", Name: "Complete", To: api.Status{Name: "Done"}},
			}})
		case r.URL.Path == "/rest/api/3/issue/PROJ-123/transitions" && r.Method == "POST":
			postCount++
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	opts, stdout, _ := newOptsForServer(t, srv.URL)
	err := runUpdate(context.Background(), opts, "PROJ-123", "", "", "", "", "", "Done", nil)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, postCount, 1)
	testutil.Contains(t, stdout.String(), "Done")
}

func TestRunUpdate_StatusOnly_AlreadyCurrent(t *testing.T) {
	srv, rec := newStatusServer(t, statusServerConfig{
		currentStatus: "Done",
		transitions:   []api.Transition{{ID: "31", Name: "Complete", To: api.Status{Name: "Done"}}},
	})
	opts, _, stderr := newOptsForServer(t, srv.URL)

	err := runUpdate(context.Background(), opts, "PROJ-123", "", "", "", "", "", "done", nil)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, rec.postTransition, 0)
	testutil.Equal(t, rec.getTransitions, 0)
	testutil.Contains(t, stderr.String(), "status is already")
}

func TestRunUpdate_StatusAndSummary_AlreadyCurrent(t *testing.T) {
	srv, rec := newStatusServer(t, statusServerConfig{currentStatus: "Done"})
	opts, _, _ := newOptsForServer(t, srv.URL)

	err := runUpdate(context.Background(), opts, "PROJ-123", "New summary", "", "", "", "", "Done", nil)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, rec.postTransition, 0)
	testutil.Equal(t, rec.putIssue, 1)
}

func TestRunUpdate_Status_NotFound(t *testing.T) {
	srv, rec := newStatusServer(t, statusServerConfig{
		currentStatus: "To Do",
		transitions: []api.Transition{
			{ID: "11", Name: "Start", To: api.Status{Name: "In Progress"}},
		},
	})
	opts, _, stderr := newOptsForServer(t, srv.URL)

	err := runUpdate(context.Background(), opts, "PROJ-123", "", "", "", "", "", "Bogus", nil)
	testutil.Error(t, err)
	testutil.True(t, errors.Is(err, root.ErrAlreadyReported), "expected ErrAlreadyReported sentinel")
	testutil.Equal(t, rec.postTransition, 0)
	testutil.Equal(t, rec.putIssue, 0)
	testutil.Contains(t, stderr.String(), "no transition to status 'Bogus'")
	testutil.Contains(t, stderr.String(), "In Progress")
}

func TestRunUpdate_Status_NotFound_DoesNotWriteSummary(t *testing.T) {
	// Preflight failure must not perform any field writes.
	srv, rec := newStatusServer(t, statusServerConfig{
		currentStatus: "To Do",
		transitions: []api.Transition{
			{ID: "11", Name: "Start", To: api.Status{Name: "In Progress"}},
		},
	})
	opts, _, _ := newOptsForServer(t, srv.URL)

	err := runUpdate(context.Background(), opts, "PROJ-123", "New summary", "", "", "", "", "Bogus", nil)
	testutil.Error(t, err)
	testutil.Equal(t, rec.putIssue, 0)
}

func TestRunUpdate_Status_Ambiguous(t *testing.T) {
	srv, rec := newStatusServer(t, statusServerConfig{
		currentStatus: "To Do",
		transitions: []api.Transition{
			{ID: "31", Name: "Resolve", To: api.Status{Name: "Done"}},
			{ID: "41", Name: "Close", To: api.Status{Name: "Done"}},
		},
	})
	opts, _, stderr := newOptsForServer(t, srv.URL)

	err := runUpdate(context.Background(), opts, "PROJ-123", "", "", "", "", "", "Done", nil)
	testutil.Error(t, err)
	testutil.True(t, errors.Is(err, root.ErrAlreadyReported))
	testutil.Equal(t, rec.postTransition, 0)
	testutil.Contains(t, stderr.String(), "multiple transitions")
	testutil.Contains(t, stderr.String(), "jtk transitions do PROJ-123")
}

func TestRunUpdate_StatusAndSummary_HappyPath_OrdersWritesPutBeforePost(t *testing.T) {
	var order []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/api/3/issue/PROJ-123" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"key": "PROJ-123",
				"fields": map[string]any{
					"status":  map[string]any{"name": "To Do"},
					"project": map[string]any{"key": "PROJ"},
				},
			})
		case r.URL.Path == "/rest/api/3/issue/PROJ-123/transitions" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(api.TransitionsResponse{Transitions: []api.Transition{
				{ID: "31", Name: "Done", To: api.Status{Name: "Done"}},
			}})
		case r.URL.Path == "/rest/api/3/issue/PROJ-123" && r.Method == "PUT":
			order = append(order, "PUT")
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/rest/api/3/issue/PROJ-123/transitions" && r.Method == "POST":
			order = append(order, "POST")
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	opts, _, _ := newOptsForServer(t, srv.URL)
	err := runUpdate(context.Background(), opts, "PROJ-123", "New summary", "", "", "", "", "Done", nil)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, len(order), 2)
	testutil.Equal(t, order[0], "PUT")
	testutil.Equal(t, order[1], "POST")
}

func TestRunUpdate_Status_TransitionPostFailsAfterPut(t *testing.T) {
	// Documented non-rollback semantics: if the transition POST fails after a
	// successful field PUT, the PUT stands and the error surfaces.
	srv, rec := newStatusServer(t, statusServerConfig{
		currentStatus: "To Do",
		transitions: []api.Transition{
			{ID: "31", Name: "Done", To: api.Status{Name: "Done"}},
		},
		postTransitionErr: http.StatusInternalServerError,
	})
	opts, _, _ := newOptsForServer(t, srv.URL)

	err := runUpdate(context.Background(), opts, "PROJ-123", "New summary", "", "", "", "", "Done", nil)
	testutil.Error(t, err)
	testutil.Equal(t, rec.putIssue, 1)
	testutil.Equal(t, rec.postTransition, 1)
}

func TestRunUpdate_Status_IDOnly(t *testing.T) {
	srv, rec := newStatusServer(t, statusServerConfig{
		currentStatus: "To Do",
		transitions: []api.Transition{
			{ID: "31", Name: "Done", To: api.Status{Name: "Done"}},
		},
	})
	opts, stdout, _ := newOptsForServer(t, srv.URL)
	opts.IDOnly = true

	err := runUpdate(context.Background(), opts, "PROJ-123", "", "", "", "", "", "Done", nil)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, rec.postTransition, 1)
	testutil.Equal(t, stdout.String(), "PROJ-123\n")
}

func TestRunUpdate_Status_IDOnly_AlreadyCurrent_EmitsKeyNoAdvisory(t *testing.T) {
	srv, rec := newStatusServer(t, statusServerConfig{currentStatus: "Done"})
	opts, stdout, stderr := newOptsForServer(t, srv.URL)
	opts.IDOnly = true

	err := runUpdate(context.Background(), opts, "PROJ-123", "", "", "", "", "", "Done", nil)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, rec.postTransition, 0)
	testutil.Equal(t, rec.getTransitions, 0)
	testutil.Equal(t, stdout.String(), "PROJ-123\n")
	testutil.Equal(t, stderr.String(), "")
}

// TestRunUpdate_TypeAndStatus_HappyPath exercises the riskiest combinable
// path: --status is resolved against the issue's pre-move workflow, then the
// type change runs (bulk-move API), then the pre-move transition POST runs.
// Documented contract: the pre-move transition ID is what gets POSTed.
func TestRunUpdate_TypeAndStatus_HappyPath(t *testing.T) {
	seedCacheForIssues(t)
	testutil.RequireNoError(t, cache.WriteResource("issuetypes", "24h", map[string][]api.IssueType{
		"PROJ": {
			{ID: "10000", Name: "Epic"},
			{ID: "10001", Name: "Task"},
		},
	}))
	var (
		moveCalled       bool
		transitionPosted bool
		transitionBody   []byte
		order            []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/api/3/issue/PROJ-123" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"key": "PROJ-123",
				"id":  "10001",
				"fields": map[string]any{
					"status":    map[string]any{"name": "To Do"},
					"project":   map[string]any{"key": "PROJ"},
					"issuetype": map[string]any{"id": "10000", "name": "Epic"},
				},
			})
		case r.URL.Path == "/rest/api/3/project/PROJ" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(struct {
				IssueTypes []api.IssueType `json:"issueTypes"`
			}{IssueTypes: []api.IssueType{{ID: "10001", Name: "Task"}}})
		case r.URL.Path == "/rest/api/3/issue/PROJ-123/transitions" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(api.TransitionsResponse{Transitions: []api.Transition{
				{ID: "31", Name: "Complete", To: api.Status{Name: "Done"}},
			}})
		case r.URL.Path == "/rest/api/3/bulk/issues/move" && r.Method == "POST":
			order = append(order, "MOVE")
			moveCalled = true
			_ = json.NewEncoder(w).Encode(api.MoveIssuesResponse{TaskID: "task-1"})
		case r.URL.Path == "/rest/api/3/bulk/queue/task-1" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(api.MoveTaskStatus{
				TaskID: "task-1", Status: "COMPLETE", Progress: 100,
				Result: &api.MoveTaskResult{Successful: []string{"PROJ-123"}},
			})
		case r.URL.Path == "/rest/api/3/issue/PROJ-123/transitions" && r.Method == "POST":
			order = append(order, "TRANSITION")
			transitionPosted = true
			transitionBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	opts, _, _ := newOptsForServer(t, srv.URL)
	err := runUpdate(context.Background(), opts, "PROJ-123", "", "", "", "", "Task", "Done", nil)
	testutil.RequireNoError(t, err)
	testutil.True(t, moveCalled, "bulk move must be called")
	testutil.True(t, transitionPosted, "transition POST must follow the move")
	testutil.Equal(t, len(order), 2)
	testutil.Equal(t, order[0], "MOVE")
	testutil.Equal(t, order[1], "TRANSITION")

	var body api.TransitionRequest
	testutil.RequireNoError(t, json.Unmarshal(transitionBody, &body))
	testutil.Equal(t, body.Transition.ID, "31")
}

// TestRunUpdate_TypeAndStatus_TransitionPostFailsAfterMove documents the
// non-rollback contract: if the pre-move transition becomes invalid after the
// type change (e.g. the new workflow doesn't have it), the type change is
// already done and the transition error is surfaced.
func TestRunUpdate_TypeAndStatus_TransitionPostFailsAfterMove(t *testing.T) {
	seedCacheForIssues(t)
	testutil.RequireNoError(t, cache.WriteResource("issuetypes", "24h", map[string][]api.IssueType{
		"PROJ": {{ID: "10001", Name: "Task"}},
	}))
	var moveCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/api/3/issue/PROJ-123" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"key": "PROJ-123",
				"fields": map[string]any{
					"status":    map[string]any{"name": "To Do"},
					"project":   map[string]any{"key": "PROJ"},
					"issuetype": map[string]any{"id": "10000", "name": "Epic"},
				},
			})
		case r.URL.Path == "/rest/api/3/project/PROJ" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(struct {
				IssueTypes []api.IssueType `json:"issueTypes"`
			}{IssueTypes: []api.IssueType{{ID: "10001", Name: "Task"}}})
		case r.URL.Path == "/rest/api/3/issue/PROJ-123/transitions" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(api.TransitionsResponse{Transitions: []api.Transition{
				{ID: "31", Name: "Complete", To: api.Status{Name: "Done"}},
			}})
		case r.URL.Path == "/rest/api/3/bulk/issues/move" && r.Method == "POST":
			moveCalled = true
			_ = json.NewEncoder(w).Encode(api.MoveIssuesResponse{TaskID: "task-1"})
		case r.URL.Path == "/rest/api/3/bulk/queue/task-1" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(api.MoveTaskStatus{
				TaskID: "task-1", Status: "COMPLETE", Progress: 100,
				Result: &api.MoveTaskResult{Successful: []string{"PROJ-123"}},
			})
		case r.URL.Path == "/rest/api/3/issue/PROJ-123/transitions" && r.Method == "POST":
			// New workflow rejects the pre-move transition.
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"errorMessages":["transition not valid"]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	opts, _, _ := newOptsForServer(t, srv.URL)
	err := runUpdate(context.Background(), opts, "PROJ-123", "", "", "", "", "Task", "Done", nil)
	testutil.Error(t, err)
	testutil.True(t, moveCalled, "type change happened before the transition failed (non-rollback)")
}

// TestRunUpdate_Status_GetTransitionsError covers the preflight error path
// where listing transitions fails; no writes should happen and the error
// must be wrapped so the caller sees "failed to get transitions".
func TestRunUpdate_Status_GetTransitionsError(t *testing.T) {
	srv, rec := newStatusServer(t, statusServerConfig{
		currentStatus:  "To Do",
		transitionsErr: http.StatusInternalServerError,
	})
	opts, _, _ := newOptsForServer(t, srv.URL)

	err := runUpdate(context.Background(), opts, "PROJ-123", "", "", "", "", "", "Done", nil)
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "failed to get transitions")
	testutil.Equal(t, rec.postTransition, 0)
	testutil.Equal(t, rec.putIssue, 0)
}

// TestRunUpdate_Status_GetIssueError covers the preflight error path where
// the initial issue fetch fails. With --summary also requested, we assert
// that no field PUT happens — preflight protects field writes.
func TestRunUpdate_Status_GetIssueError(t *testing.T) {
	var putCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/api/3/issue/PROJ-123" && r.Method == "GET":
			w.WriteHeader(http.StatusInternalServerError)
		case r.URL.Path == "/rest/api/3/issue/PROJ-123" && r.Method == "PUT":
			putCalled = true
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	opts, _, _ := newOptsForServer(t, srv.URL)
	err := runUpdate(context.Background(), opts, "PROJ-123", "New summary", "", "", "", "", "Done", nil)
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "failed to get issue")
	testutil.False(t, putCalled, "preflight failure must block the field PUT")
}
