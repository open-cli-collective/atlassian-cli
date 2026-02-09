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

	err = runCreate(opts, "MYPROJECT", "Task", "Fix login bug", "Users cannot log in with SSO credentials", nil)
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

	err = runCreate(opts, "PROJ", "Bug", `Error: "unexpected token" in parser`, "", nil)
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
