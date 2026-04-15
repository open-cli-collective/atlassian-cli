package present

import (
	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// IssuePresenter creates presentation models for issue data.
type IssuePresenter struct{}

// PresentDetail creates a detail view for a single issue.
func (IssuePresenter) PresentDetail(issue *api.Issue, issueURL string, noTruncate bool) *present.OutputModel {
	status := ""
	if issue.Fields.Status != nil {
		status = issue.Fields.Status.Name
	}

	issueType := ""
	if issue.Fields.IssueType != nil {
		issueType = issue.Fields.IssueType.Name
	}

	assignee := "Unassigned"
	if issue.Fields.Assignee != nil {
		assignee = issue.Fields.Assignee.DisplayName
	}

	priority := ""
	if issue.Fields.Priority != nil {
		priority = issue.Fields.Priority.Name
	}

	project := ""
	if issue.Fields.Project != nil {
		project = issue.Fields.Project.Key
	}

	description := ""
	if issue.Fields.Description != nil {
		description = issue.Fields.Description.ToPlainText()
		if !noTruncate && len(description) > 200 {
			description = description[:200] + "... [truncated, use --no-truncate for complete text]"
		}
	}

	fields := []present.Field{
		{Label: "Key", Value: issue.Key},
		{Label: "Summary", Value: issue.Fields.Summary},
		{Label: "Status", Value: status},
		{Label: "Type", Value: issueType},
		{Label: "Priority", Value: priority},
		{Label: "Assignee", Value: assignee},
		{Label: "Project", Value: project},
	}
	if description != "" {
		fields = append(fields, present.Field{Label: "Description", Value: description})
	}
	fields = append(fields, present.Field{Label: "URL", Value: issueURL})

	return &present.OutputModel{
		Sections: []present.Section{&present.DetailSection{Fields: fields}},
	}
}

// PresentList creates a table view for a list of issues.
func (IssuePresenter) PresentList(issues []api.Issue) *present.OutputModel {
	rows := make([]present.Row, len(issues))
	for i, issue := range issues {
		status := ""
		if issue.Fields.Status != nil {
			status = issue.Fields.Status.Name
		}

		assignee := ""
		if issue.Fields.Assignee != nil {
			assignee = issue.Fields.Assignee.DisplayName
		}

		issueType := ""
		if issue.Fields.IssueType != nil {
			issueType = issue.Fields.IssueType.Name
		}

		rows[i] = present.Row{
			Cells: []string{
				issue.Key,
				TruncateText(issue.Fields.Summary, 50),
				OrDash(status),
				FormatAssignee(assignee),
				OrDash(issueType),
			},
		}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"KEY", "SUMMARY", "STATUS", "ASSIGNEE", "TYPE"},
				Rows:    rows,
			},
		},
	}
}

// PresentTypes creates a table view for issue types.
func (IssuePresenter) PresentTypes(types []api.IssueType) *present.OutputModel {
	rows := make([]present.Row, len(types))
	for i, t := range types {
		rows[i] = present.Row{
			Cells: []string{
				t.ID,
				t.Name,
				BoolString(t.Subtask),
			},
		}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "NAME", "SUBTASK"},
				Rows:    rows,
			},
		},
	}
}
