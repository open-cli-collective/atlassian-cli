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

func TestNewUpdateCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newUpdateCmd(opts)

	assert.Equal(t, "update <issue-key>", cmd.Use)

	// Check that assignee flag exists
	assigneeFlag := cmd.Flags().Lookup("assignee")
	require.NotNil(t, assigneeFlag)
	assert.Equal(t, "a", assigneeFlag.Shorthand)
	assert.Equal(t, "", assigneeFlag.DefValue)
}

func TestRunUpdate_WithAssigneeMe(t *testing.T) {
	var capturedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/myself":
			json.NewEncoder(w).Encode(api.User{
				AccountID:   "me-account-id",
				DisplayName: "Current User",
			})
		case "/rest/api/3/issue/PROJ-123":
			assert.Equal(t, http.MethodPut, r.Method)
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &capturedBody)
			w.WriteHeader(http.StatusNoContent)
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

	err = runUpdate(opts, "PROJ-123", "", "", "me", nil)
	require.NoError(t, err)

	// Verify assignee was set in the request
	fields, ok := capturedBody["fields"].(map[string]interface{})
	require.True(t, ok)
	assignee, ok := fields["assignee"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "me-account-id", assignee["accountId"])

	assert.Contains(t, stdout.String(), "PROJ-123")
}

func TestRunUpdate_WithAssigneeEmail(t *testing.T) {
	var capturedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/user/search":
			json.NewEncoder(w).Encode([]api.User{
				{AccountID: "jane456", DisplayName: "Jane Doe", EmailAddress: "jane@example.com"},
			})
		case "/rest/api/3/issue/PROJ-456":
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &capturedBody)
			w.WriteHeader(http.StatusNoContent)
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

	err = runUpdate(opts, "PROJ-456", "", "", "jane@example.com", nil)
	require.NoError(t, err)

	fields, ok := capturedBody["fields"].(map[string]interface{})
	require.True(t, ok)
	assignee, ok := fields["assignee"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "jane456", assignee["accountId"])
}

func TestRunUpdate_AssigneeOnly(t *testing.T) {
	// Verify that assignee alone counts as "fields specified"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/myself":
			json.NewEncoder(w).Encode(api.User{AccountID: "me-id"})
		case "/rest/api/3/issue/PROJ-789":
			w.WriteHeader(http.StatusNoContent)
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

	// Should not error with "no fields specified" since assignee is a field
	err = runUpdate(opts, "PROJ-789", "", "", "me", nil)
	require.NoError(t, err)
}

func TestRunUpdate_NoFieldsError(t *testing.T) {
	client, err := api.New(api.ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})
	require.NoError(t, err)

	opts := &root.Options{
		Output: "table",
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(opts, "PROJ-999", "", "", "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fields specified to update")
}
