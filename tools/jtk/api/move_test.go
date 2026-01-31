package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMoveRequest(t *testing.T) {
	req := BuildMoveRequest([]string{"PROJ-1", "PROJ-2"}, "TARGET", "10001", true)

	assert.True(t, req.SendBulkNotification)
	assert.Len(t, req.TargetToSourcesMapping, 1)

	// Key format should be "PROJECT,ISSUE_TYPE_ID" (comma-separated)
	spec, exists := req.TargetToSourcesMapping["TARGET,10001"]
	assert.True(t, exists, "target key should use comma separator")
	assert.Equal(t, []string{"PROJ-1", "PROJ-2"}, spec.IssueIdsOrKeys)
	assert.True(t, spec.InferFieldDefaults)
	assert.True(t, spec.InferStatusDefaults)
}

func TestBuildMoveRequest_NoNotify(t *testing.T) {
	req := BuildMoveRequest([]string{"PROJ-1"}, "TARGET", "10001", false)

	assert.False(t, req.SendBulkNotification)
}

func TestGetMoveTaskStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/bulk/queue/task-123", r.URL.Path)
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
	require.NoError(t, err)

	status, err := client.GetMoveTaskStatus("task-123")
	require.NoError(t, err)
	assert.Equal(t, "task-123", status.TaskID)
	assert.Equal(t, "COMPLETE", status.Status)
	assert.Equal(t, 100, status.Progress)
	require.NotNil(t, status.Result)
	assert.Equal(t, []string{"TARGET-1", "TARGET-2"}, status.Result.Successful)
	assert.Empty(t, status.Result.Failed)
}

func TestGetMoveTaskStatus_EmptyID(t *testing.T) {
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	_, err := client.GetMoveTaskStatus("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task ID is required")
}

func TestGetMoveTaskStatus_WithFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	require.NoError(t, err)

	status, err := client.GetMoveTaskStatus("task-456")
	require.NoError(t, err)
	require.NotNil(t, status.Result)
	assert.Len(t, status.Result.Successful, 1)
	assert.Len(t, status.Result.Failed, 1)
	assert.Equal(t, "PROJ-2", status.Result.Failed[0].IssueKey)
	assert.Equal(t, []string{"Field X is required", "Invalid status"}, status.Result.Failed[0].Errors)
}

func TestMoveIssues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/rest/api/3/bulk/issues/move", r.URL.Path)

		var req MoveIssuesRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.True(t, req.SendBulkNotification)
		assert.Contains(t, req.TargetToSourcesMapping, "TARGET,10001")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"taskId": "new-task-id"}`))
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	require.NoError(t, err)

	req := BuildMoveRequest([]string{"PROJ-1"}, "TARGET", "10001", true)
	resp, err := client.MoveIssues(req)
	require.NoError(t, err)
	assert.Equal(t, "new-task-id", resp.TaskID)
}

func TestGetProjectIssueTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/project/PROJ", r.URL.Path)
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
	require.NoError(t, err)

	types, err := client.GetProjectIssueTypes("PROJ")
	require.NoError(t, err)
	assert.Len(t, types, 3)
	assert.Equal(t, "Task", types[0].Name)
	assert.False(t, types[0].Subtask)
	assert.True(t, types[2].Subtask)
}

func TestGetProjectIssueTypes_EmptyProject(t *testing.T) {
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	_, err := client.GetProjectIssueTypes("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project key is required")
}

func TestGetProjectStatuses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/project/PROJ/statuses", r.URL.Path)
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
	require.NoError(t, err)

	statuses, err := client.GetProjectStatuses("PROJ")
	require.NoError(t, err)
	assert.Len(t, statuses, 1)
	assert.Equal(t, "Task", statuses[0].Name)
	assert.Len(t, statuses[0].Statuses, 3)
	assert.Equal(t, "To Do", statuses[0].Statuses[0].Name)
}

func TestGetProjectStatuses_EmptyProject(t *testing.T) {
	client, _ := New(ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})

	_, err := client.GetProjectStatuses("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project key is required")
}
