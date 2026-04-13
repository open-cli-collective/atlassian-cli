package artifact

import (
	"github.com/open-cli-collective/atlassian-go/artifact"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// IssueArtifact is the projected output for an issue.
// This is a minimal artifact for list contexts (e.g., sprints issues).
// Full issue artifact with more fields will be added when issues commands
// are migrated after the --full flag collision is resolved.
type IssueArtifact struct {
	// Agent fields - essential for triage
	Key      string `json:"key"`
	Summary  string `json:"summary"`
	Status   string `json:"status"`
	Type     string `json:"type,omitempty"`
	Assignee string `json:"assignee,omitempty"`

	// Full-only fields
	Priority string `json:"priority,omitempty"`
	Project  string `json:"project,omitempty"`
}

// ProjectIssue projects an api.Issue to an IssueArtifact.
func ProjectIssue(issue *api.Issue, mode artifact.Type) *IssueArtifact {
	a := &IssueArtifact{
		Key:     issue.Key,
		Summary: issue.Fields.Summary,
	}
	if issue.Fields.Status != nil {
		a.Status = issue.Fields.Status.Name
	}
	if issue.Fields.IssueType != nil {
		a.Type = issue.Fields.IssueType.Name
	}
	if issue.Fields.Assignee != nil {
		a.Assignee = issue.Fields.Assignee.DisplayName
	}
	if mode.IsFull() {
		if issue.Fields.Priority != nil {
			a.Priority = issue.Fields.Priority.Name
		}
		if issue.Fields.Project != nil {
			a.Project = issue.Fields.Project.Key
		}
	}
	return a
}

// ProjectIssues projects a slice of api.Issue to IssueArtifacts.
func ProjectIssues(issues []api.Issue, mode artifact.Type) []*IssueArtifact {
	result := make([]*IssueArtifact, len(issues))
	for i := range issues {
		result[i] = ProjectIssue(&issues[i], mode)
	}
	return result
}
