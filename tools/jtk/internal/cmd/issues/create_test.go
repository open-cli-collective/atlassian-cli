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

func TestRunCreate_RequestBodyNoDoubleQuoting(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.Issue{
				Key: "TEST-1",
				ID:  "10001",
			})
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

	err = runCreate(opts, "MYPROJECT", "Task", "Fix login bug", "Users cannot log in with SSO credentials", "", "", nil)
	require.NoError(t, err)

	// Parse the captured request body
	require.NotEmpty(t, capturedBody, "request body should have been captured")

	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})

	// Summary must be the exact string without extra quotes
	summary := fields["summary"].(string)
	assert.Equal(t, "Fix login bug", summary, "summary should not have extra quotes")
	assert.NotContains(t, summary, `"`, "summary should not contain literal quote characters")

	// Description should be ADF format, extract text from first paragraph
	desc := fields["description"].(map[string]interface{})
	assert.Equal(t, "doc", desc["type"], "description should be ADF document")
	content := desc["content"].([]interface{})
	require.NotEmpty(t, content)

	// Walk ADF to extract text
	firstPara := content[0].(map[string]interface{})
	paraContent := firstPara["content"].([]interface{})
	firstTextNode := paraContent[0].(map[string]interface{})
	descText := firstTextNode["text"].(string)
	assert.Equal(t, "Users cannot log in with SSO credentials", descText,
		"description text should not have extra quotes")
	assert.NotContains(t, descText, `"`, "description text should not contain literal quote characters")
}

func TestRunCreate_SummaryWithSpecialCharacters(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.Issue{Key: "TEST-2", ID: "10002"})
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

	err = runCreate(opts, "PROJ", "Bug", `Error: "unexpected token" in parser`, "", "", "", nil)
	require.NoError(t, err)

	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})
	summary := fields["summary"].(string)
	assert.Equal(t, `Error: "unexpected token" in parser`, summary,
		"summary with embedded quotes should be preserved exactly")
}

func TestNewCreateCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newCreateCmd(opts)

	assert.Equal(t, "create", cmd.Use)
	assert.Equal(t, "Create a new issue", cmd.Short)

	// Check required flags
	summaryFlag := cmd.Flags().Lookup("summary")
	require.NotNil(t, summaryFlag)
	assert.Equal(t, "s", summaryFlag.Shorthand)

	projectFlag := cmd.Flags().Lookup("project")
	require.NotNil(t, projectFlag)
	assert.Equal(t, "p", projectFlag.Shorthand)

	descFlag := cmd.Flags().Lookup("description")
	require.NotNil(t, descFlag)
	assert.Equal(t, "d", descFlag.Shorthand)

	parentFlag := cmd.Flags().Lookup("parent")
	require.NotNil(t, parentFlag)
	assert.Equal(t, "", parentFlag.Shorthand, "parent flag should have no shorthand")

	assigneeFlag := cmd.Flags().Lookup("assignee")
	require.NotNil(t, assigneeFlag)
	assert.Equal(t, "a", assigneeFlag.Shorthand)
}

func TestCreateCmd_CobraExecution_NoDoubleQuoting(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.Issue{Key: "TEST-1", ID: "10001"})
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

	cmd := newCreateCmd(opts)
	cmd.SetArgs([]string{
		"--project", "PROJ",
		"--type", "Task",
		"--summary", "Fix login bug",
		"--description", "Users cannot log in with SSO credentials",
	})

	err = cmd.Execute()
	require.NoError(t, err)

	require.NotEmpty(t, capturedBody)
	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})

	// Verify no double-quoting via Cobra flag parsing
	summary := fields["summary"].(string)
	assert.Equal(t, "Fix login bug", summary)
	assert.False(t, summary[0] == '"', "summary must not start with a literal quote")
}

func TestRunCreate_WithParent(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.Issue{Key: "PROJ-456", ID: "10456"})
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

	err = runCreate(opts, "PROJ", "Task", "Child task", "", "PROJ-100", "", nil)
	require.NoError(t, err)

	require.NotEmpty(t, capturedBody)
	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})

	// Parent should be an object with "key" field
	parentField := fields["parent"].(map[string]interface{})
	assert.Equal(t, "PROJ-100", parentField["key"], "parent key should match")
}

func TestRunCreate_WithoutParent(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.Issue{Key: "PROJ-789", ID: "10789"})
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

	err = runCreate(opts, "PROJ", "Task", "Standalone task", "", "", "", nil)
	require.NoError(t, err)

	require.NotEmpty(t, capturedBody)
	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})
	assert.Nil(t, fields["parent"], "parent should not be present when empty")
}

func TestCreateCmd_CobraExecution_WithParent(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.Issue{Key: "PROJ-456", ID: "10456"})
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

	cmd := newCreateCmd(opts)
	cmd.SetArgs([]string{
		"--project", "PROJ",
		"--type", "Task",
		"--summary", "Child task",
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

func TestRunCreate_WithAssigneeAccountID(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.Issue{Key: "PROJ-500", ID: "10500"})
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

	err = runCreate(opts, "PROJ", "Task", "Assigned task", "", "", "61292e4c4f29230069621c5f", nil)
	require.NoError(t, err)

	require.NotEmpty(t, capturedBody)
	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})
	assigneeField := fields["assignee"].(map[string]interface{})
	assert.Equal(t, "61292e4c4f29230069621c5f", assigneeField["accountId"])
}

func TestRunCreate_WithAssigneeMe(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/myself" && r.Method == "GET" {
			json.NewEncoder(w).Encode(api.User{
				AccountID:   "myself-account-id",
				DisplayName: "Test User",
			})
			return
		}
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.Issue{Key: "PROJ-501", ID: "10501"})
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

	err = runCreate(opts, "PROJ", "Task", "My task", "", "", "me", nil)
	require.NoError(t, err)

	require.NotEmpty(t, capturedBody)
	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})
	assigneeField := fields["assignee"].(map[string]interface{})
	assert.Equal(t, "myself-account-id", assigneeField["accountId"])
}

func TestRunCreate_WithAssigneeEmail(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/user/search" && r.Method == "GET" {
			json.NewEncoder(w).Encode([]api.User{
				{AccountID: "found-account-id", DisplayName: "Found User"},
			})
			return
		}
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.Issue{Key: "PROJ-502", ID: "10502"})
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

	err = runCreate(opts, "PROJ", "Task", "Their task", "", "", "user@example.com", nil)
	require.NoError(t, err)

	require.NotEmpty(t, capturedBody)
	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})
	assigneeField := fields["assignee"].(map[string]interface{})
	assert.Equal(t, "found-account-id", assigneeField["accountId"])
}

func TestRunCreate_DescriptionEscapeSequences(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.Issue{Key: "TEST-10", ID: "10010"})
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

	// Simulate what the shell passes when user types: --description "First paragraph.\n\nSecond paragraph."
	// The shell delivers literal backslash-n, not actual newlines.
	err = runCreate(opts, "PROJ", "Task", "Test", `First paragraph.\n\nSecond paragraph.`, "", "", nil)
	require.NoError(t, err)

	require.NotEmpty(t, capturedBody)
	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})
	desc := fields["description"].(map[string]interface{})
	assert.Equal(t, "doc", desc["type"])

	// With escape interpretation, the description should produce multiple paragraphs
	// (not a single paragraph with literal \n text)
	content := desc["content"].([]interface{})
	assert.GreaterOrEqual(t, len(content), 2, "escaped newlines should produce multiple ADF nodes, not one paragraph with literal \\n")
}

func TestRunCreate_WithoutAssignee(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.Issue{Key: "PROJ-503", ID: "10503"})
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

	err = runCreate(opts, "PROJ", "Task", "Unassigned task", "", "", "", nil)
	require.NoError(t, err)

	require.NotEmpty(t, capturedBody)
	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})
	assert.Nil(t, fields["assignee"], "assignee should not be present when empty")
}
