package artifact

import (
	"github.com/open-cli-collective/atlassian-go/artifact"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// IssueArtifact is the projected output for an issue.
type IssueArtifact struct {
	// Agent fields - essential for triage
	Key      string `json:"key"`
	Summary  string `json:"summary"`
	Status   string `json:"status"`
	Type     string `json:"type,omitempty"`
	Assignee string `json:"assignee,omitempty"`

	// Full-only fields
	Priority    string   `json:"priority,omitempty"`
	Project     string   `json:"project,omitempty"`
	Created     string   `json:"created,omitempty"`
	Updated     string   `json:"updated,omitempty"`
	Reporter    string   `json:"reporter,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Description string   `json:"description,omitempty"`
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
		a.Created = formatDate(issue.Fields.Created)
		a.Updated = formatDate(issue.Fields.Updated)
		if issue.Fields.Reporter != nil {
			a.Reporter = issue.Fields.Reporter.DisplayName
		}
		if len(issue.Fields.Labels) > 0 {
			a.Labels = issue.Fields.Labels
		}
		if issue.Fields.Description != nil {
			a.Description = issue.Fields.Description.ToPlainText()
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
