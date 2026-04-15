package present

import (
	"fmt"
	"strings"

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
		subtask := "no"
		if t.Subtask {
			subtask = "yes"
		}
		rows[i] = present.Row{
			Cells: []string{
				t.ID,
				t.Name,
				subtask,
				TruncateText(t.Description, 60),
			},
		}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "NAME", "SUBTASK", "DESCRIPTION"},
				Rows:    rows,
			},
		},
	}
}

// PresentMoveStatus creates a detail view for a move task status.
func (IssuePresenter) PresentMoveStatus(status *api.MoveTaskStatus) *present.OutputModel {
	fields := []present.Field{
		{Label: "Task ID", Value: status.TaskID},
		{Label: "Status", Value: status.Status},
		{Label: "Progress", Value: fmt.Sprintf("%d%%", status.Progress)},
		{Label: "Submitted", Value: status.SubmittedAt},
	}

	if status.StartedAt != "" {
		fields = append(fields, present.Field{Label: "Started", Value: status.StartedAt})
	}
	if status.FinishedAt != "" {
		fields = append(fields, present.Field{Label: "Finished", Value: status.FinishedAt})
	}

	sections := []present.Section{&present.DetailSection{Fields: fields}}

	// Append result messages if available
	if status.Result != nil {
		if len(status.Result.Successful) > 0 {
			sections = append(sections, &present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Successful: %s", strings.Join(status.Result.Successful, ", ")),
			})
		}
		if len(status.Result.Failed) > 0 {
			sections = append(sections, &present.MessageSection{
				Kind:    present.MessageError,
				Message: "Failed:",
				Stream:  present.StreamStderr,
			})
			for _, failed := range status.Result.Failed {
				sections = append(sections, &present.MessageSection{
					Kind:    present.MessageError,
					Message: fmt.Sprintf("  %s: %s", failed.IssueKey, strings.Join(failed.Errors, ", ")),
					Stream:  present.StreamStderr,
				})
			}
		}
	}

	return &present.OutputModel{Sections: sections}
}

// --- Mutation result methods ---

// PresentCreated creates a success message for issue creation.
func (IssuePresenter) PresentCreated(key, url string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Created issue %s (%s)", key, url),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentUpdated creates a success message for issue update.
func (IssuePresenter) PresentUpdated(key string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Updated issue %s", key),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentDeleted creates a success message for issue deletion.
func (IssuePresenter) PresentDeleted(key string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Deleted issue %s", key),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentAssigned creates a success message for assignment.
// If assignee is empty, presents as unassignment.
func (IssuePresenter) PresentAssigned(key, assignee string) *present.OutputModel {
	msg := fmt.Sprintf("Unassigned issue %s", key)
	if assignee != "" {
		msg = fmt.Sprintf("Assigned issue %s to %s", key, assignee)
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: msg,
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentTypeChanged creates a success message for type change.
func (IssuePresenter) PresentTypeChanged(key, newType string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Changed %s type to %s", key, newType),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// --- No-change/idempotent methods (route to stderr) ---

// PresentTypeAlreadyCurrent creates an advisory when type is already current.
func (IssuePresenter) PresentTypeAlreadyCurrent(key, typeName string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("Issue %s is already type %s", key, typeName),
				Stream:  present.StreamStderr,
			},
		},
	}
}

// --- Empty state methods ---

// PresentEmpty creates an info message for empty issue list.
func (IssuePresenter) PresentEmpty() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "No issues found",
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentNoEditableFields creates an info message for no editable fields.
func (IssuePresenter) PresentNoEditableFields(key string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("No editable fields found for %s", key),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentNoTypes creates an info message for no issue types found.
func (IssuePresenter) PresentNoTypes(project string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("No issue types found for project %s", project),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// --- Cancellation methods ---

// PresentDeleteCancelled creates an info message for cancelled deletion.
func (IssuePresenter) PresentDeleteCancelled() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "Deletion cancelled.",
				Stream:  present.StreamStdout,
			},
		},
	}
}

// --- Advisory methods (route to stderr) ---

// PresentPaginationHint creates an advisory about more results.
func (IssuePresenter) PresentPaginationHint() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "More results available (use --next-page-token to fetch next page)",
				Stream:  present.StreamStderr,
			},
		},
	}
}

// PresentTypeChangeProgress creates an advisory about type change in progress.
func (IssuePresenter) PresentTypeChangeProgress(key, typeName string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("Changing %s type to %s...", key, typeName),
				Stream:  present.StreamStderr,
			},
		},
	}
}

// --- Move operations ---

// PresentTypeNotFound creates a multi-section error for type not found with available types.
func (IssuePresenter) PresentTypeNotFound(targetType, project string, availableTypes []string) *present.OutputModel {
	sections := []present.Section{
		&present.MessageSection{
			Kind:    present.MessageError,
			Message: fmt.Sprintf("Issue type %q not found in target project", targetType),
			Stream:  present.StreamStderr,
		},
		&present.MessageSection{
			Kind:    present.MessageInfo,
			Message: fmt.Sprintf("Available types in %s:", project),
			Stream:  present.StreamStderr,
		},
	}

	for _, t := range availableTypes {
		sections = append(sections, &present.MessageSection{
			Kind:    present.MessageInfo,
			Message: fmt.Sprintf("  - %s", t),
			Stream:  present.StreamStderr,
		})
	}

	return &present.OutputModel{Sections: sections}
}

// PresentMoveProgress creates an advisory about move in progress.
func (IssuePresenter) PresentMoveProgress(count int, project, typeName string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("Moving %d issue(s) to %s (%s)...", count, project, typeName),
				Stream:  present.StreamStderr,
			},
		},
	}
}

// PresentMoveInitiated creates success + hint for async move (no-wait mode).
func (IssuePresenter) PresentMoveInitiated(taskID string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Move initiated (Task ID: %s)", taskID),
				Stream:  present.StreamStdout,
			},
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("Check status with: jtk issues move-status %s", taskID),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentMoveWaiting creates an advisory about waiting for completion.
func (IssuePresenter) PresentMoveWaiting() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "Waiting for move to complete...",
				Stream:  present.StreamStderr,
			},
		},
	}
}

// PresentMovePartialFailure creates warning + errors + successes for partial failure.
func (IssuePresenter) PresentMovePartialFailure(successful []string, failed []api.MoveFailedIssue) *present.OutputModel {
	sections := []present.Section{
		&present.MessageSection{
			Kind:    present.MessageWarning,
			Message: "Move completed with errors",
			Stream:  present.StreamStderr,
		},
	}

	for _, f := range failed {
		sections = append(sections, &present.MessageSection{
			Kind:    present.MessageError,
			Message: fmt.Sprintf("  %s: %s", f.IssueKey, strings.Join(f.Errors, ", ")),
			Stream:  present.StreamStderr,
		})
	}

	if len(successful) > 0 {
		sections = append(sections, &present.MessageSection{
			Kind:    present.MessageSuccess,
			Message: fmt.Sprintf("Successfully moved: %s", strings.Join(successful, ", ")),
			Stream:  present.StreamStdout,
		})
	}

	return &present.OutputModel{Sections: sections}
}

// PresentMoved creates a success message for completed move.
func (IssuePresenter) PresentMoved(count int, project string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Moved %d issue(s) to %s", count, project),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// --- List with pagination ---

// PresentListWithPagination creates a table with optional pagination hint.
func (p IssuePresenter) PresentListWithPagination(issues []api.Issue, hasMore bool) *present.OutputModel {
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

	sections := []present.Section{
		&present.TableSection{
			Headers: []string{"KEY", "SUMMARY", "STATUS", "ASSIGNEE", "TYPE"},
			Rows:    rows,
		},
	}

	if hasMore {
		sections = append(sections, &present.MessageSection{
			Kind:    present.MessageInfo,
			Message: "More results available (use --next-page-token to fetch next page)",
			Stream:  present.StreamStderr,
		})
	}

	return &present.OutputModel{Sections: sections}
}
