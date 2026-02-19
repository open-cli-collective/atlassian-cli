package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestNewTypesCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newTypesCmd(opts)

	testutil.Equal(t, cmd.Use, "types")
	testutil.Equal(t, cmd.Short, "List valid issue types for a project")

	// Check that project flag exists and is required
	projectFlag := cmd.Flags().Lookup("project")
	testutil.NotNil(t, projectFlag)
	testutil.Equal(t, projectFlag.Shorthand, "p")
}

func TestRunTypes_Success(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/project/TEST")

		response := api.ProjectDetail{
			ID:   json.Number("10000"),
			Key:  "TEST",
			Name: "Test Project",
			IssueTypes: []api.IssueType{
				{ID: "10001", Name: "Bug", Description: "A problem", Subtask: false},
				{ID: "10002", Name: "Task", Description: "A task to do", Subtask: false},
				{ID: "10003", Name: "Sub-task", Description: "A subtask", Subtask: true},
			},
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
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

	err = runTypes(context.Background(), opts, "TEST")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Bug")
	testutil.Contains(t, output, "Task")
	testutil.Contains(t, output, "Sub-task")
	testutil.Contains(t, output, "yes") // subtask column
}

func TestRunTypes_ProjectNotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errorMessages":["No project could be found with key 'INVALID'."]}`))
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	opts := &root.Options{
		Output: "table",
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runTypes(context.Background(), opts, "INVALID")
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "not found")
}

func TestRunTypes_EmptyIssueTypes(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := api.ProjectDetail{
			ID:         json.Number("10000"),
			Key:        "EMPTY",
			Name:       "Empty Project",
			IssueTypes: []api.IssueType{},
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
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

	err = runTypes(context.Background(), opts, "EMPTY")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "No issue types found")
}

func TestRunTypes_JSONOutput(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := api.ProjectDetail{
			ID:   json.Number("10000"),
			Key:  "TEST",
			Name: "Test Project",
			IssueTypes: []api.IssueType{
				{ID: "10001", Name: "Bug", Description: "A bug", Subtask: false},
				{ID: "10002", Name: "Story", Description: "A user story", Subtask: false},
			},
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
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
		Output: "json",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runTypes(context.Background(), opts, "TEST")
	testutil.RequireNoError(t, err)

	// Verify JSON output
	output := stdout.String()
	testutil.True(t, strings.HasPrefix(strings.TrimSpace(output), "["))

	var issueTypes []api.IssueType
	err = json.Unmarshal([]byte(output), &issueTypes)
	testutil.RequireNoError(t, err)
	testutil.Len(t, issueTypes, 2)
	testutil.Equal(t, issueTypes[0].Name, "Bug")
	testutil.Equal(t, issueTypes[1].Name, "Story")
}

func TestRunTypes_DescriptionTruncation(t *testing.T) {
	t.Parallel()
	longDesc := strings.Repeat("A", 100) // 100 character description

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := api.ProjectDetail{
			ID:   json.Number("10000"),
			Key:  "TEST",
			Name: "Test Project",
			IssueTypes: []api.IssueType{
				{ID: "10001", Name: "Bug", Description: longDesc, Subtask: false},
			},
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
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

	err = runTypes(context.Background(), opts, "TEST")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	// Description should be truncated to 60 chars
	testutil.NotContains(t, output, longDesc)
	testutil.Contains(t, output, "...")
}
