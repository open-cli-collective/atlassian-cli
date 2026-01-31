package api

import (
	"encoding/json"
	"fmt"
)

// MoveIssuesRequest represents a request to move issues between projects
type MoveIssuesRequest struct {
	SendBulkNotification   bool                            `json:"sendBulkNotification"`
	TargetToSourcesMapping map[string]MoveIssuesSourceSpec `json:"targetToSourcesMapping"`
}

// MoveIssuesSourceSpec specifies which issues to move and how to map fields
type MoveIssuesSourceSpec struct {
	IssueIdsOrKeys        []string               `json:"issueIdsOrKeys"`
	InferFieldDefaults    bool                   `json:"inferFieldDefaults"`
	InferStatusDefaults   bool                   `json:"inferStatusDefaults"`
	TargetStatus          []StatusMapping        `json:"targetStatus,omitempty"`
	TargetMandatoryFields map[string]interface{} `json:"targetMandatoryFields,omitempty"`
}

// StatusMapping maps source status to target status
type StatusMapping struct {
	SourceStatusID string `json:"sourceStatusId"`
	TargetStatusID string `json:"targetStatusId"`
}

// MoveIssuesResponse represents the response from a bulk move operation
type MoveIssuesResponse struct {
	TaskID string `json:"taskId"`
}

// MoveTaskStatus represents the status of a move task
type MoveTaskStatus struct {
	TaskID      string          `json:"taskId"`
	Status      string          `json:"status"` // ENQUEUED, RUNNING, COMPLETE, FAILED, CANCELLED
	SubmittedAt string          `json:"submittedAt"`
	StartedAt   string          `json:"startedAt,omitempty"`
	FinishedAt  string          `json:"finishedAt,omitempty"`
	Progress    int             `json:"progress"`
	Result      *MoveTaskResult `json:"result,omitempty"`
}

// MoveTaskResult contains the result of a completed move task
type MoveTaskResult struct {
	Successful []string          `json:"successful,omitempty"`
	Failed     []MoveFailedIssue `json:"failed,omitempty"`
}

// MoveFailedIssue represents an issue that failed to move
type MoveFailedIssue struct {
	IssueKey string   `json:"issueKey"`
	Errors   []string `json:"errors"`
}

// MoveIssues moves issues to a target project/issue type (Jira Cloud only)
// This is an asynchronous operation that returns a task ID
func (c *Client) MoveIssues(req MoveIssuesRequest) (*MoveIssuesResponse, error) {
	urlStr := fmt.Sprintf("%s/bulk/issues/move", c.BaseURL)

	body, err := c.post(urlStr, req)
	if err != nil {
		return nil, err
	}

	var resp MoveIssuesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse move response: %w", err)
	}

	return &resp, nil
}

// GetMoveTaskStatus gets the status of a move task
func (c *Client) GetMoveTaskStatus(taskID string) (*MoveTaskStatus, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task ID is required")
	}

	// Status endpoint is /bulk/queue/{taskId}
	urlStr := fmt.Sprintf("%s/bulk/queue/%s", c.BaseURL, taskID)

	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var status MoveTaskStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("failed to parse task status: %w", err)
	}

	return &status, nil
}

// GetProjectIssueTypes returns the issue types available in a project
func (c *Client) GetProjectIssueTypes(projectKey string) ([]IssueType, error) {
	if projectKey == "" {
		return nil, fmt.Errorf("project key is required")
	}

	urlStr := fmt.Sprintf("%s/project/%s", c.BaseURL, projectKey)

	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var project struct {
		IssueTypes []IssueType `json:"issueTypes"`
	}
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("failed to parse project: %w", err)
	}

	return project.IssueTypes, nil
}

// GetProjectStatuses returns the statuses available in a project
func (c *Client) GetProjectStatuses(projectKey string) ([]ProjectStatus, error) {
	if projectKey == "" {
		return nil, fmt.Errorf("project key is required")
	}

	urlStr := fmt.Sprintf("%s/project/%s/statuses", c.BaseURL, projectKey)

	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var statuses []ProjectStatus
	if err := json.Unmarshal(body, &statuses); err != nil {
		return nil, fmt.Errorf("failed to parse statuses: %w", err)
	}

	return statuses, nil
}

// ProjectStatus represents an issue type's available statuses
type ProjectStatus struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Subtask  bool     `json:"subtask"`
	Statuses []Status `json:"statuses"`
}

// BuildMoveRequest creates a move request for a simple move operation
func BuildMoveRequest(issueKeys []string, targetProject, targetIssueTypeID string, notify bool) MoveIssuesRequest {
	// Target key format: "PROJECT_KEY,ISSUE_TYPE_ID" (comma-separated per Jira API docs)
	targetKey := fmt.Sprintf("%s,%s", targetProject, targetIssueTypeID)

	return MoveIssuesRequest{
		SendBulkNotification: notify,
		TargetToSourcesMapping: map[string]MoveIssuesSourceSpec{
			targetKey: {
				IssueIdsOrKeys:      issueKeys,
				InferFieldDefaults:  true,
				InferStatusDefaults: true,
			},
		},
	}
}
