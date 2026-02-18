package issues

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestRunUpdate_RequestBodyNoDoubleQuoting(t *testing.T) {
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
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(opts, "PROJ-123", "Updated summary", "Updated description", "", "", "", nil)
	require.NoError(t, err)

	require.NotEmpty(t, capturedBody)

	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})

	// Summary must be the exact string without extra quotes
	summary := fields["summary"].(string)
	assert.Equal(t, "Updated summary", summary, "summary should not have extra quotes")
	assert.NotContains(t, summary, `"`, "summary should not contain literal quote characters")

	// Description should be ADF format
	desc := fields["description"].(map[string]interface{})
	assert.Equal(t, "doc", desc["type"], "description should be ADF document")
	content := desc["content"].([]interface{})
	require.NotEmpty(t, content)

	firstPara := content[0].(map[string]interface{})
	paraContent := firstPara["content"].([]interface{})
	firstTextNode := paraContent[0].(map[string]interface{})
	descText := firstTextNode["text"].(string)
	assert.Equal(t, "Updated description", descText,
		"description text should not have extra quotes")
}

func TestNewUpdateCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newUpdateCmd(opts)

	assert.Equal(t, "update <issue-key>", cmd.Use)
	assert.Equal(t, "Update an issue", cmd.Short)

	summaryFlag := cmd.Flags().Lookup("summary")
	require.NotNil(t, summaryFlag)
	assert.Equal(t, "s", summaryFlag.Shorthand)

	descFlag := cmd.Flags().Lookup("description")
	require.NotNil(t, descFlag)
	assert.Equal(t, "d", descFlag.Shorthand)

	parentFlag := cmd.Flags().Lookup("parent")
	require.NotNil(t, parentFlag)
	assert.Equal(t, "", parentFlag.Shorthand, "parent flag should have no shorthand")

	assigneeFlag := cmd.Flags().Lookup("assignee")
	require.NotNil(t, assigneeFlag)
	assert.Equal(t, "a", assigneeFlag.Shorthand)

	typeFlag := cmd.Flags().Lookup("type")
	require.NotNil(t, typeFlag)
	assert.Equal(t, "t", typeFlag.Shorthand)
}

func TestRunUpdate_TypeChange(t *testing.T) {
	var moveBody []byte
	moveCompleted := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/api/3/issue/PROJ-123" && r.Method == "GET":
			json.NewEncoder(w).Encode(api.Issue{
				Key: "PROJ-123",
				ID:  "10001",
				Fields: api.IssueFields{
					Project:   &api.Project{Key: "PROJ"},
					IssueType: &api.IssueType{ID: "10000", Name: "Epic"},
				},
			})
		case r.URL.Path == "/rest/api/3/project/PROJ" && r.Method == "GET":
			json.NewEncoder(w).Encode(struct {
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
			json.NewEncoder(w).Encode(api.MoveIssuesResponse{TaskID: "task-123"})
		case r.URL.Path == "/rest/api/3/bulk/queue/task-123" && r.Method == "GET":
			json.NewEncoder(w).Encode(api.MoveTaskStatus{
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
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(opts, "PROJ-123", "", "", "", "", "Task", nil)
	require.NoError(t, err)
	assert.True(t, moveCompleted, "should have called the move API")

	// Verify move request body
	var moveReq api.MoveIssuesRequest
	err = json.Unmarshal(moveBody, &moveReq)
	require.NoError(t, err)

	// The target key should be "PROJ,10001" (project key, Task type ID)
	spec, ok := moveReq.TargetToSourcesMapping["PROJ,10001"]
	require.True(t, ok, "should have mapping for PROJ,10001")
	assert.Equal(t, []string{"PROJ-123"}, spec.IssueIdsOrKeys)
}

func TestRunUpdate_TypeAlreadyCorrect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue/PROJ-123" && r.Method == "GET" {
			json.NewEncoder(w).Encode(api.Issue{
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
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	// Should succeed without calling move API since it's already the right type
	err = runUpdate(opts, "PROJ-123", "", "", "", "", "Task", nil)
	require.NoError(t, err)
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
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(opts, "PROJ-123", "New summary", "", "", "", "", nil)
	require.NoError(t, err)

	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})
	assert.Equal(t, "New summary", fields["summary"])
	assert.Nil(t, fields["description"], "description should not be present when empty")
	assert.Nil(t, fields["parent"], "parent should not be present when empty")
}

func TestRunUpdate_NoFieldsError(t *testing.T) {
	opts := &root.Options{
		Output: "table",
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	err := runUpdate(opts, "PROJ-123", "", "", "", "", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no fields specified")
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
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(opts, "PROJ-456", "", "", "PROJ-100", "", "", nil)
	require.NoError(t, err)

	require.NotEmpty(t, capturedBody)
	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})
	parentField := fields["parent"].(map[string]interface{})
	assert.Equal(t, "PROJ-100", parentField["key"], "parent key should match")
	assert.Nil(t, fields["summary"], "summary should not be present when empty")
	assert.Nil(t, fields["description"], "description should not be present when empty")
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
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(opts, "PROJ-456", "Updated title", "", "PROJ-200", "", "", nil)
	require.NoError(t, err)

	require.NotEmpty(t, capturedBody)
	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})
	assert.Equal(t, "Updated title", fields["summary"])
	parentField := fields["parent"].(map[string]interface{})
	assert.Equal(t, "PROJ-200", parentField["key"])
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
	require.NoError(t, err)

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
	require.NoError(t, err)

	require.NotEmpty(t, capturedBody)
	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})
	parentField := fields["parent"].(map[string]interface{})
	assert.Equal(t, "PROJ-100", parentField["key"])
}

func TestRunUpdate_AssigneeOnly(t *testing.T) {
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
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(opts, "PROJ-789", "", "", "", "61292e4c4f29230069621c5f", "", nil)
	require.NoError(t, err)

	require.NotEmpty(t, capturedBody)
	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})
	assigneeField := fields["assignee"].(map[string]interface{})
	assert.Equal(t, "61292e4c4f29230069621c5f", assigneeField["accountId"])
	assert.Nil(t, fields["summary"], "summary should not be present when empty")
}

func TestRunUpdate_AssigneeMe(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/myself" && r.Method == "GET" {
			json.NewEncoder(w).Encode(api.User{
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
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(opts, "PROJ-789", "", "", "", "me", "", nil)
	require.NoError(t, err)

	require.NotEmpty(t, capturedBody)
	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})
	assigneeField := fields["assignee"].(map[string]interface{})
	assert.Equal(t, "myself-account-id", assigneeField["accountId"])
}

func TestUpdateCmd_CobraExecution_WithAssignee(t *testing.T) {
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
	require.NoError(t, err)

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
	require.NoError(t, err)

	require.NotEmpty(t, capturedBody)
	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})
	assigneeField := fields["assignee"].(map[string]interface{})
	assert.Equal(t, "61292e4c4f29230069621c5f", assigneeField["accountId"])
}
