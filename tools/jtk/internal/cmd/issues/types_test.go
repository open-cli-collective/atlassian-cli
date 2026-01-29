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

func TestNewTypesCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newTypesCmd(opts)

	assert.Equal(t, "types", cmd.Use)
	assert.Equal(t, "List valid issue types for a project", cmd.Short)

	// Check that project flag exists and is required
	projectFlag := cmd.Flags().Lookup("project")
	require.NotNil(t, projectFlag)
	assert.Equal(t, "p", projectFlag.Shorthand)
}

func TestRunTypes_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/project/TEST", r.URL.Path)

		response := api.ProjectDetail{
			ID:   "10000",
			Key:  "TEST",
			Name: "Test Project",
			IssueTypes: []api.IssueType{
				{ID: "10001", Name: "Bug", Description: "A problem", Subtask: false},
				{ID: "10002", Name: "Task", Description: "A task to do", Subtask: false},
				{ID: "10003", Name: "Sub-task", Description: "A subtask", Subtask: true},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &api.Client{
		BaseURL:    server.URL + "/rest/api/3",
		Email:      "test@example.com",
		APIToken:   "token",
		HTTPClient: server.Client(),
	}

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err := runTypes(opts, "TEST")
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "Bug")
	assert.Contains(t, output, "Task")
	assert.Contains(t, output, "Sub-task")
	assert.Contains(t, output, "yes") // subtask column
}

func TestRunTypes_ProjectNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errorMessages":["No project could be found with key 'INVALID'."]}`))
	}))
	defer server.Close()

	client := &api.Client{
		BaseURL:    server.URL + "/rest/api/3",
		Email:      "test@example.com",
		APIToken:   "token",
		HTTPClient: server.Client(),
	}

	opts := &root.Options{
		Output: "table",
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err := runTypes(opts, "INVALID")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunTypes_EmptyIssueTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := api.ProjectDetail{
			ID:         "10000",
			Key:        "EMPTY",
			Name:       "Empty Project",
			IssueTypes: []api.IssueType{},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &api.Client{
		BaseURL:    server.URL + "/rest/api/3",
		Email:      "test@example.com",
		APIToken:   "token",
		HTTPClient: server.Client(),
	}

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err := runTypes(opts, "EMPTY")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "No issue types found")
}

func TestRunTypes_JSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := api.ProjectDetail{
			ID:   "10000",
			Key:  "TEST",
			Name: "Test Project",
			IssueTypes: []api.IssueType{
				{ID: "10001", Name: "Bug", Description: "A bug", Subtask: false},
				{ID: "10002", Name: "Story", Description: "A user story", Subtask: false},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &api.Client{
		BaseURL:    server.URL + "/rest/api/3",
		Email:      "test@example.com",
		APIToken:   "token",
		HTTPClient: server.Client(),
	}

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "json",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err := runTypes(opts, "TEST")
	require.NoError(t, err)

	// Verify JSON output
	output := stdout.String()
	assert.True(t, strings.HasPrefix(strings.TrimSpace(output), "["))

	var issueTypes []api.IssueType
	err = json.Unmarshal([]byte(output), &issueTypes)
	require.NoError(t, err)
	assert.Len(t, issueTypes, 2)
	assert.Equal(t, "Bug", issueTypes[0].Name)
	assert.Equal(t, "Story", issueTypes[1].Name)
}

func TestRunTypes_DescriptionTruncation(t *testing.T) {
	longDesc := strings.Repeat("A", 100) // 100 character description

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := api.ProjectDetail{
			ID:   "10000",
			Key:  "TEST",
			Name: "Test Project",
			IssueTypes: []api.IssueType{
				{ID: "10001", Name: "Bug", Description: longDesc, Subtask: false},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &api.Client{
		BaseURL:    server.URL + "/rest/api/3",
		Email:      "test@example.com",
		APIToken:   "token",
		HTTPClient: server.Client(),
	}

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err := runTypes(opts, "TEST")
	require.NoError(t, err)

	output := stdout.String()
	// Description should be truncated to 60 chars
	assert.NotContains(t, output, longDesc)
	assert.Contains(t, output, "...")
}
