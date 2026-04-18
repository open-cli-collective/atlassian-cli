package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(context.Background(), opts, "PROJ-123", "Updated summary", "Updated description", "", "", "", nil)
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
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(context.Background(), opts, "PROJ-123", "", "", "", "", "Task", nil)
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
			_ = json.NewEncoder(w).Encode(api.Issue{
				Key: "PROJ-123",
				ID:  "10001",
				Fields: api.IssueFields{
					Project:   &api.Project{Key: "PROJ"},
					IssueType: &api.IssueType{ID: "10001", Name: "Task"},
				},
			})
			return
		}
		// No move API should be called
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
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	// Should succeed without calling move API since it's already the right type
	err = runUpdate(context.Background(), opts, "PROJ-123", "", "", "", "", "Task", nil)
	testutil.RequireNoError(t, err)
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
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(context.Background(), opts, "PROJ-123", "New summary", "", "", "", "", nil)
	testutil.RequireNoError(t, err)

	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)
	testutil.Equal(t, fields["summary"], "New summary")
	testutil.Nil(t, fields["description"])
	testutil.Nil(t, fields["parent"])
}

func TestRunUpdate_NoFieldsError(t *testing.T) {
	opts := &root.Options{
		Output: "table",
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	err := runUpdate(context.Background(), opts, "PROJ-123", "", "", "", "", "", nil)
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
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(context.Background(), opts, "PROJ-456", "", "", "PROJ-100", "", "", nil)
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
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(context.Background(), opts, "PROJ-456", "Updated title", "", "PROJ-200", "", "", nil)
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
		Output: "table",
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
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(context.Background(), opts, "PROJ-789", "", "", "", "61292e4c4f29230069621c5f", "", nil)
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
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(context.Background(), opts, "PROJ-789", "", "", "", "me", "", nil)
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
		Output: "table",
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
