package issues

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestNewCreateCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newCreateCmd(opts)

	assert.Equal(t, "create", cmd.Use)

	// Check that assignee flag exists
	assigneeFlag := cmd.Flags().Lookup("assignee")
	require.NotNil(t, assigneeFlag)
	assert.Equal(t, "a", assigneeFlag.Shorthand)
	assert.Equal(t, "", assigneeFlag.DefValue)
}

func TestRunCreate_WithAssigneeMe(t *testing.T) {
	var capturedBody api.CreateIssueRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/myself":
			json.NewEncoder(w).Encode(api.User{
				AccountID:   "me-account-id",
				DisplayName: "Current User",
			})
		case "/rest/api/3/issue":
			assert.Equal(t, http.MethodPost, r.Method)
			json.NewDecoder(r.Body).Decode(&capturedBody)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.Issue{Key: "PROJ-1"})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
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

	err = runCreate(opts, "PROJ", "Task", "My task", "", "me", nil)
	require.NoError(t, err)

	// Verify assignee was set in the request
	assignee, ok := capturedBody.Fields["assignee"]
	require.True(t, ok, "assignee field should be present")
	assigneeMap, ok := assignee.(map[string]interface{})
	require.True(t, ok, "assignee should be a map")
	assert.Equal(t, "me-account-id", assigneeMap["accountId"])

	assert.Contains(t, stdout.String(), "PROJ-1")
}

func TestRunCreate_WithAssigneeAccountID(t *testing.T) {
	var capturedBody api.CreateIssueRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" {
			json.NewDecoder(r.Body).Decode(&capturedBody)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.Issue{Key: "PROJ-2"})
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

	err = runCreate(opts, "PROJ", "Task", "Their task", "", "5b10ac8d82e05b22cc7d4ef5", nil)
	require.NoError(t, err)

	assignee, ok := capturedBody.Fields["assignee"]
	require.True(t, ok)
	assigneeMap, ok := assignee.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "5b10ac8d82e05b22cc7d4ef5", assigneeMap["accountId"])
}

func TestRunCreate_WithoutAssignee(t *testing.T) {
	var capturedBody api.CreateIssueRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" {
			json.NewDecoder(r.Body).Decode(&capturedBody)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.Issue{Key: "PROJ-3"})
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

	err = runCreate(opts, "PROJ", "Task", "Unassigned task", "", "", nil)
	require.NoError(t, err)

	_, hasAssignee := capturedBody.Fields["assignee"]
	assert.False(t, hasAssignee, "assignee should not be present when not specified")
}

func TestRunCreate_WithAssigneeJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/myself":
			json.NewEncoder(w).Encode(api.User{AccountID: "me-id"})
		case "/rest/api/3/issue":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.Issue{Key: "PROJ-4"})
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
		Output: "json",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runCreate(opts, "PROJ", "Task", "JSON task", "", "me", nil)
	require.NoError(t, err)

	var result api.Issue
	err = json.Unmarshal(stdout.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "PROJ-4", result.Key)
}
