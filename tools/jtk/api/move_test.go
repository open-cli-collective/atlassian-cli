package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestBuildMoveRequest(t *testing.T) {
	t.Parallel()
	req := BuildMoveRequest([]string{"PROJ-1", "PROJ-2"}, "TARGET", "10001", true)

	testutil.True(t, req.SendBulkNotification)
	testutil.Len(t, req.TargetToSourcesMapping, 1)

	// Key format should be "PROJECT,ISSUE_TYPE_ID" (comma-separated)
	spec, exists := req.TargetToSourcesMapping["TARGET,10001"]
	testutil.True(t, exists, "target key should use comma separator")
	testutil.Equal(t, spec.IssueIdsOrKeys, []string{"PROJ-1", "PROJ-2"})
	testutil.True(t, spec.InferFieldDefaults)
	testutil.True(t, spec.InferStatusDefaults)
}

func TestBuildMoveRequest_NoNotify(t *testing.T) {
	t.Parallel()
	req := BuildMoveRequest([]string{"PROJ-1"}, "TARGET", "10001", false)

	testutil.False(t, req.SendBulkNotification)
}

func TestGetMoveTaskStatus(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/bulk/queue/task-123")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"taskId": "task-123",
			"status": "COMPLETE",
			"submittedAt": "2024-01-15T10:30:00.000+0000",
			"startedAt": "2024-01-15T10:30:01.000+0000",
			"finishedAt": "2024-01-15T10:30:05.000+0000",
			"progress": 100,
			"result": {
				"successful": ["TARGET-1", "TARGET-2"],
				"failed": []
			}
		}`))
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	status, err := client.GetMoveTaskStatus(context.Background(), "task-123")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, status.TaskID, "task-123")
	testutil.Equal(t, status.Status, "COMPLETE")
	testutil.Equal(t, status.Progress, 100)
	testutil.NotNil(t, status.Result)
	testutil.Equal(t, status.Result.Successful, []string{"TARGET-1", "TARGET-2"})
	testutil.Empty(t, status.Result.Failed)
}

func TestGetMoveTaskStatus_EmptyID(t *testing.T) {
	t.Parallel()
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	_, err := client.GetMoveTaskStatus(context.Background(), "")
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "task ID is required")
}

func TestGetMoveTaskStatus_WithFailures(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"taskId": "task-456",
			"status": "COMPLETE",
			"progress": 100,
			"submittedAt": "2024-01-15T10:30:00.000+0000",
			"result": {
				"successful": ["TARGET-1"],
				"failed": [
					{"issueKey": "PROJ-2", "errors": ["Field X is required", "Invalid status"]}
				]
			}
		}`))
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	status, err := client.GetMoveTaskStatus(context.Background(), "task-456")
	testutil.RequireNoError(t, err)
	testutil.NotNil(t, status.Result)
	testutil.Len(t, status.Result.Successful, 1)
	testutil.Len(t, status.Result.Failed, 1)
	testutil.Equal(t, status.Result.Failed[0].IssueKey, "PROJ-2")
	testutil.Equal(t, status.Result.Failed[0].Errors, []string{"Field X is required", "Invalid status"})
}

func TestMoveIssues(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodPost)
		testutil.Equal(t, r.URL.Path, "/rest/api/3/bulk/issues/move")

		var req MoveIssuesRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		testutil.RequireNoError(t, err)

		testutil.True(t, req.SendBulkNotification)
		_, hasKey := req.TargetToSourcesMapping["TARGET,10001"]
		testutil.True(t, hasKey, "expected TargetToSourcesMapping to contain key TARGET,10001")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"taskId": "new-task-id"}`))
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	req := BuildMoveRequest([]string{"PROJ-1"}, "TARGET", "10001", true)
	resp, err := client.MoveIssues(context.Background(), req)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, resp.TaskID, "new-task-id")
}

func TestGetProjectIssueTypes(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/project/PROJ")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"issueTypes": [
				{"id": "10001", "name": "Task", "subtask": false},
				{"id": "10002", "name": "Bug", "subtask": false},
				{"id": "10003", "name": "Sub-task", "subtask": true}
			]
		}`))
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	types, err := client.GetProjectIssueTypes(context.Background(), "PROJ")
	testutil.RequireNoError(t, err)
	testutil.Len(t, types, 3)
	testutil.Equal(t, types[0].Name, "Task")
	testutil.False(t, types[0].Subtask)
	testutil.True(t, types[2].Subtask)
}

func TestGetProjectIssueTypes_EmptyProject(t *testing.T) {
	t.Parallel()
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	_, err := client.GetProjectIssueTypes(context.Background(), "")
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "project key is required")
}

func TestGetProjectStatuses(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/project/PROJ/statuses")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{
				"id": "10001",
				"name": "Task",
				"subtask": false,
				"statuses": [
					{"id": "1", "name": "To Do"},
					{"id": "2", "name": "In Progress"},
					{"id": "3", "name": "Done"}
				]
			}
		]`))
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	statuses, err := client.GetProjectStatuses(context.Background(), "PROJ")
	testutil.RequireNoError(t, err)
	testutil.Len(t, statuses, 1)
	testutil.Equal(t, statuses[0].Name, "Task")
	testutil.Len(t, statuses[0].Statuses, 3)
	testutil.Equal(t, statuses[0].Statuses[0].Name, "To Do")
}

func TestGetProjectStatuses_EmptyProject(t *testing.T) {
	t.Parallel()
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	_, err := client.GetProjectStatuses(context.Background(), "")
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "project key is required")
}
