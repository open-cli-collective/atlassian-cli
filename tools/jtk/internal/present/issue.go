package present

import (
	"fmt"
	"sort"
	"strings"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// IssuePresenter creates presentation models for issue data.
type IssuePresenter struct{}

// IssueListSpec declares the columns emitted by PresentList and the
// metadata needed for --fields projection and minimum-fetch derivation.
// Order MUST match the hardcoded Headers in PresentList (locked by a
// parity test). Default: KEY|STATUS|TYPE|PTS|ASSIGNEE|SUMMARY.
// Extended adds REPORTER, SPRINT, PARENT, UPDATED, LABELS, COMPONENTS.
var IssueListSpec = projection.Registry{
	{Header: "KEY", Identity: true},
	{Header: "STATUS", FieldID: "status"},
	{Header: "TYPE", FieldID: "issuetype"},
	{Header: "PTS", FieldID: "customfield_10035", Fetch: []string{"customfield_10035"}},
	{Header: "ASSIGNEE", FieldID: "assignee"},
	{Header: "REPORTER", FieldID: "reporter", Extended: true},
	{Header: "SPRINT", FieldID: "sprint", Aliases: []string{"customfield_10020"}, Fetch: []string{"customfield_10020"}, Extended: true},
	{Header: "PARENT", FieldID: "parent", Extended: true},
	{Header: "UPDATED", FieldID: "updated", Extended: true},
	{Header: "LABELS", FieldID: "labels", Extended: true},
	{Header: "COMPONENTS", FieldID: "components", Extended: true},
	{Header: "SUMMARY", FieldID: "summary"},
}

// IssueDetailSpec declares the fields emitted by PresentDetail /
// PresentDetailProjection and the metadata for --fields projection.
// Default fields are those an agent needs daily; extended adds
// Reporter, Created, Fix Versions, Resolution — matching the
// "add more built-in context" semantics of --extended on list/search.
var IssueDetailSpec = projection.Registry{
	{Header: "Key", Identity: true},
	{Header: "Summary", FieldID: "summary"},
	{Header: "Status", FieldID: "status"},
	{Header: "Type", FieldID: "issuetype"},
	{Header: "Priority", FieldID: "priority"},
	{Header: "Points", FieldID: "customfield_10035", Fetch: []string{"customfield_10035"}},
	{Header: "Assignee", FieldID: "assignee"},
	{Header: "Updated", FieldID: "updated"},
	{Header: "Sprint", FieldID: "sprint"},
	{Header: "Parent", FieldID: "parent"},
	{Header: "Labels", FieldID: "labels"},
	{Header: "Components", FieldID: "components"},
	{Header: "Description", FieldID: "description"},
	{Header: "Reporter", FieldID: "reporter", Extended: true},
	{Header: "Created", FieldID: "created", Extended: true},
	{Header: "Fix_Versions", FieldID: "fixVersions", Extended: true},
	{Header: "Resolution", FieldID: "resolution", Extended: true},
}

// PresentDetail creates a spec-shaped detail view for a single issue.
// Output uses msg() sections (title line + compound KV rows) matching
// the boards/sprints/projects pattern. Labels and Components rows
// appear only when non-empty in default mode; always in extended.
func (IssuePresenter) PresentDetail(issue *api.Issue, _ string, extended bool, fulltext bool) *present.OutputModel {
	sections := []present.Section{
		msg(fmt.Sprintf("%s  %s", issue.Key, issue.Fields.Summary)),
	}

	if extended {
		sections = append(sections, issueDetailExtendedSections(issue, fulltext)...)
	} else {
		sections = append(sections, issueDetailDefaultSections(issue, fulltext)...)
	}

	return &present.OutputModel{Sections: sections}
}

func issueDetailDefaultSections(issue *api.Issue, fulltext bool) []present.Section {
	status := issueStatusName(issue)
	issueType := issueTypeName(issue)
	priority := issuePriorityName(issue)
	points := formatStoryPoints(issue)
	assignee := issueAssigneeName(issue)
	updated := FormatTime(issue.Fields.Updated)

	sections := []present.Section{
		msg(fmt.Sprintf("Status: %s   Type: %s   Priority: %s   Points: %s",
			OrDash(status), OrDash(issueType), OrDash(priority), points)),
		msg(fmt.Sprintf("Assignee: %s   Updated: %s",
			assignee, OrDash(updated))),
	}

	if issue.Fields.Sprint != nil {
		sprintRef := issue.Fields.Sprint.Name
		if issue.Fields.Sprint.State != "" {
			sprintRef += " (" + issue.Fields.Sprint.State + ")"
		}
		sections = append(sections, msg("Sprint: "+sprintRef))
	}

	if issue.Fields.Parent != nil {
		parentRef := issue.Fields.Parent.Key
		if issue.Fields.Parent.Fields.Summary != "" {
			parentRef += " — " + issue.Fields.Parent.Fields.Summary
		}
		if issue.Fields.Parent.Fields.IssueType != nil {
			parentRef += " (" + issue.Fields.Parent.Fields.IssueType.Name + ")"
		}
		sections = append(sections, msg("Parent: "+parentRef))
	}

	if len(issue.Fields.Labels) > 0 {
		sections = append(sections, msg("Labels: "+strings.Join(issue.Fields.Labels, ", ")))
	}

	if len(issue.Fields.Components) > 0 {
		names := make([]string, len(issue.Fields.Components))
		for i, c := range issue.Fields.Components {
			names[i] = c.Name
		}
		sections = append(sections, msg("Components: "+strings.Join(names, ", ")))
	}

	sections = append(sections, issueDescriptionSection(issue, fulltext)...)

	return sections
}

func issueDetailExtendedSections(issue *api.Issue, fulltext bool) []present.Section {
	status := issueStatusName(issue)
	issueType := issueTypeName(issue)
	priority := issuePriorityName(issue)
	points := formatStoryPoints(issue)
	assignee := issueAssigneeName(issue)
	updated := FormatTime(issue.Fields.Updated)
	reporter := issueReporterName(issue)
	created := FormatTime(issue.Fields.Created)

	sections := []present.Section{
		msg(fmt.Sprintf("Status: %s   Type: %s   Priority: %s   Points: %s",
			OrDash(status), OrDash(issueType), OrDash(priority), points)),
		msg(fmt.Sprintf("Assignee: %s   Reporter: %s",
			assignee, reporter)),
		msg(fmt.Sprintf("Updated: %s   Created: %s",
			OrDash(updated), OrDash(created))),
	}

	if issue.Fields.Sprint != nil {
		sprintRef := issue.Fields.Sprint.Name
		if issue.Fields.Sprint.State != "" {
			sprintRef += " (" + issue.Fields.Sprint.State + ")"
		}
		sections = append(sections, msg("Sprint: "+sprintRef))
	}

	if issue.Fields.Parent != nil {
		parentRef := issue.Fields.Parent.Key
		if issue.Fields.Parent.Fields.Summary != "" {
			parentRef += " — " + issue.Fields.Parent.Fields.Summary
		}
		if issue.Fields.Parent.Fields.IssueType != nil {
			parentRef += " (" + issue.Fields.Parent.Fields.IssueType.Name + ")"
		}
		sections = append(sections, msg("Parent: "+parentRef))
	}

	if len(issue.Fields.Labels) > 0 {
		sections = append(sections, msg("Labels: "+strings.Join(issue.Fields.Labels, ", ")))
	}

	if len(issue.Fields.Components) > 0 {
		names := make([]string, len(issue.Fields.Components))
		for i, c := range issue.Fields.Components {
			names[i] = c.Name
		}
		sections = append(sections, msg("Components: "+strings.Join(names, ", ")))
	}

	if len(issue.Fields.FixVersions) > 0 {
		names := make([]string, len(issue.Fields.FixVersions))
		for i, v := range issue.Fields.FixVersions {
			names[i] = v.Name
		}
		sections = append(sections, msg("Fix Versions: "+strings.Join(names, ", ")))
	}

	if issue.Fields.Resolution != nil {
		sections = append(sections, msg("Resolution: "+issue.Fields.Resolution.Name))
	}

	sections = append(sections, issueDescriptionSection(issue, fulltext)...)

	return sections
}

func issueDescriptionSection(issue *api.Issue, fulltext bool) []present.Section {
	if issue.Fields.Description == nil {
		return nil
	}
	desc := issue.Fields.Description.ToPlainText()
	if desc == "" {
		return nil
	}
	if !fulltext && len(desc) > 200 {
		desc = desc[:200] + "...\n[truncated — use --fulltext for complete body]"
	}
	return []present.Section{
		msg(""),
		msg("Description:"),
		msg(desc),
	}
}

// AppendCustomFieldsSection appends a "Custom Fields:" section to the model
// using pre-extracted IssueFieldEntry values (from api.ExtractIssueFieldValues).
// Only custom fields (customfield_*) are included; entries are sorted by ID.
func AppendCustomFieldsSection(model *present.OutputModel, entries []api.IssueFieldEntry) {
	if model == nil || len(entries) == 0 {
		return
	}

	var custom []api.IssueFieldEntry
	for _, e := range entries {
		if strings.HasPrefix(e.ID, "customfield_") {
			custom = append(custom, e)
		}
	}
	if len(custom) == 0 {
		return
	}

	sort.Slice(custom, func(i, j int) bool { return custom[i].ID < custom[j].ID })

	model.Sections = append(model.Sections, msg(""), msg("Custom Fields:"))
	for _, e := range custom {
		label := e.Name
		if label == "" || label == e.ID {
			label = e.ID
		}
		model.Sections = append(model.Sections, msg(fmt.Sprintf("  %s: %s", label, e.Value)))
	}
}

// PresentDetailProjection builds a DetailSection view for `issues get --fields`.
// Only fields in the IssueDetailSpec registry are included; ProjectDetail
// slices the result to the user's selected subset.
func (IssuePresenter) PresentDetailProjection(issue *api.Issue, _ string, fulltext bool) *present.OutputModel {
	fields := []present.Field{
		{Label: "Key", Value: issue.Key},
		{Label: "Summary", Value: issue.Fields.Summary},
		{Label: "Status", Value: issueStatusName(issue)},
		{Label: "Type", Value: issueTypeName(issue)},
		{Label: "Priority", Value: issuePriorityName(issue)},
		{Label: "Points", Value: formatStoryPoints(issue)},
		{Label: "Assignee", Value: issueAssigneeName(issue)},
		{Label: "Updated", Value: OrDash(FormatTime(issue.Fields.Updated))},
		{Label: "Sprint", Value: issueSprintName(issue)},
		{Label: "Parent", Value: issueParentRef(issue)},
		{Label: "Labels", Value: OrDash(strings.Join(issue.Fields.Labels, ", "))},
		{Label: "Components", Value: OrDash(issueComponentNames(issue))},
		{Label: "Description", Value: issueDescriptionText(issue, fulltext)},
		{Label: "Reporter", Value: issueReporterName(issue)},
		{Label: "Created", Value: OrDash(FormatTime(issue.Fields.Created))},
		{Label: "Fix_Versions", Value: issueFixVersions(issue)},
		{Label: "Resolution", Value: issueResolution(issue)},
	}
	return &present.OutputModel{
		Sections: []present.Section{&present.DetailSection{Fields: fields}},
	}
}

func issueStatusName(issue *api.Issue) string {
	if issue.Fields.Status != nil {
		return issue.Fields.Status.Name
	}
	return ""
}

func issueTypeName(issue *api.Issue) string {
	if issue.Fields.IssueType != nil {
		return issue.Fields.IssueType.Name
	}
	return ""
}

func issuePriorityName(issue *api.Issue) string {
	if issue.Fields.Priority != nil {
		return issue.Fields.Priority.Name
	}
	return ""
}

func issueAssigneeName(issue *api.Issue) string {
	if issue.Fields.Assignee != nil {
		return issue.Fields.Assignee.DisplayName
	}
	return "Unassigned"
}

func issueReporterName(issue *api.Issue) string {
	if issue.Fields.Reporter != nil {
		return issue.Fields.Reporter.DisplayName
	}
	return "-"
}

func issueSprintName(issue *api.Issue) string {
	if issue.Fields.Sprint != nil {
		return issue.Fields.Sprint.Name
	}
	return "-"
}

func issueParentRef(issue *api.Issue) string {
	if issue.Fields.Parent == nil {
		return "-"
	}
	ref := issue.Fields.Parent.Key
	if issue.Fields.Parent.Fields.Summary != "" {
		ref += " — " + issue.Fields.Parent.Fields.Summary
	}
	return ref
}

func issueComponentNames(issue *api.Issue) string {
	if len(issue.Fields.Components) == 0 {
		return ""
	}
	names := make([]string, len(issue.Fields.Components))
	for i, c := range issue.Fields.Components {
		names[i] = c.Name
	}
	return strings.Join(names, ", ")
}

func issueFixVersions(issue *api.Issue) string {
	if len(issue.Fields.FixVersions) == 0 {
		return "-"
	}
	names := make([]string, len(issue.Fields.FixVersions))
	for i, v := range issue.Fields.FixVersions {
		names[i] = v.Name
	}
	return strings.Join(names, ", ")
}

func issueResolution(issue *api.Issue) string {
	if issue.Fields.Resolution == nil {
		return "-"
	}
	return issue.Fields.Resolution.Name
}

func issueDescriptionText(issue *api.Issue, fulltext bool) string {
	if issue.Fields.Description == nil {
		return "-"
	}
	desc := issue.Fields.Description.ToPlainText()
	if desc == "" {
		return "-"
	}
	if !fulltext && len(desc) > 200 {
		return desc[:200] + "... [truncated]"
	}
	return desc
}

// PresentListWithPagination wraps PresentList and appends a pagination
// hint when hasMore is true.
func (p IssuePresenter) PresentListWithPagination(issues []api.Issue, extended, hasMore bool, nextToken string) *present.OutputModel {
	model := p.PresentList(issues, extended)
	model.Sections = AppendPaginationHintWithToken(model.Sections, hasMore, nextToken)
	return model
}

// PresentList creates a table view for a list of issues. Default order
// is KEY|STATUS|TYPE|PTS|ASSIGNEE|SUMMARY; --extended adds REPORTER,
// SPRINT, PARENT, UPDATED, LABELS, COMPONENTS.
func (IssuePresenter) PresentList(issues []api.Issue, extended bool) *present.OutputModel {
	var headers []string
	if extended {
		headers = []string{"KEY", "STATUS", "TYPE", "PTS", "ASSIGNEE", "REPORTER", "SPRINT", "PARENT", "UPDATED", "LABELS", "COMPONENTS", "SUMMARY"}
	} else {
		headers = []string{"KEY", "STATUS", "TYPE", "PTS", "ASSIGNEE", "SUMMARY"}
	}

	rows := make([]present.Row, len(issues))
	for i, issue := range issues {
		status := issueStatusName(&issue)
		issueType := issueTypeName(&issue)
		pts := formatStoryPoints(&issue)
		assignee := FormatAssignee(issueAssigneeNameRaw(&issue))

		var cells []string
		if extended {
			cells = []string{
				issue.Key,
				OrDash(status),
				OrDash(issueType),
				pts,
				assignee,
				OrDash(issueReporterNameRaw(&issue)),
				OrDash(issueSprintName(&issue)),
				OrDash(issueParentKey(&issue)),
				OrDash(FormatTime(issue.Fields.Updated)),
				OrDash(strings.Join(issue.Fields.Labels, ", ")),
				OrDash(issueComponentNames(&issue)),
				TruncateText(issue.Fields.Summary, 80),
			}
		} else {
			cells = []string{
				issue.Key,
				OrDash(status),
				OrDash(issueType),
				pts,
				assignee,
				TruncateText(issue.Fields.Summary, 80),
			}
		}
		rows[i] = present.Row{Cells: cells}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{Headers: headers, Rows: rows},
		},
	}
}

// AppendDynamicTableColumns adds columns for dynamic (cache-resolved) field
// specs to the first TableSection in model. Each spec becomes a new column
// with values extracted via api.ExtractFieldValue. Must be called BEFORE
// ProjectTable so the projection slice can find the headers.
func AppendDynamicTableColumns(model *present.OutputModel, issues []api.Issue, specs []projection.ColumnSpec) {
	if model == nil || len(specs) == 0 {
		return
	}
	for i, s := range model.Sections {
		ts, ok := s.(*present.TableSection)
		if !ok {
			continue
		}
		for _, spec := range specs {
			ts.Headers = append(ts.Headers, spec.Header)
		}
		for j, r := range ts.Rows {
			if j >= len(issues) {
				break
			}
			for _, spec := range specs {
				val := api.ExtractFieldValue(&issues[j], spec.FieldID)
				r.Cells = append(r.Cells, OrDash(val))
			}
			ts.Rows[j] = r
		}
		model.Sections[i] = ts
		return
	}
}

// AppendDynamicDetailFields adds fields for dynamic (cache-resolved) field
// specs to the first DetailSection in model. Must be called BEFORE
// ProjectDetail so the projection slice can find the labels.
func AppendDynamicDetailFields(model *present.OutputModel, issue *api.Issue, specs []projection.ColumnSpec) {
	if model == nil || issue == nil || len(specs) == 0 {
		return
	}
	for i, s := range model.Sections {
		ds, ok := s.(*present.DetailSection)
		if !ok {
			continue
		}
		for _, spec := range specs {
			val := api.ExtractFieldValue(issue, spec.FieldID)
			ds.Fields = append(ds.Fields, present.Field{
				Label: spec.Header,
				Value: OrDash(val),
			})
		}
		model.Sections[i] = ds
		return
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

// PresentArchived creates a confirmation message for issue archival.
func (IssuePresenter) PresentArchived(key string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Archived %s", key),
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
				Message: fmt.Sprintf("Deleted %s", key),
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
func (IssuePresenter) PresentTypeAlreadyCurrent(typeName string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("type is already %s", typeName),
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

// PresentTypeFallbackWarning creates a warning when the source issue type was
// not found in the target project and a fallback type was used.
func (IssuePresenter) PresentTypeFallbackWarning(sourceType, projectKey, fallbackType string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageWarning,
				Message: fmt.Sprintf("warning: source issue type %q not found in project %s; using %q as the target type (pass --to-type to override).", sourceType, projectKey, fallbackType),
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

// --- Issue field helpers ---

func formatStoryPoints(issue *api.Issue) string {
	if issue.Fields.CustomFields == nil {
		return "-"
	}
	v, ok := issue.Fields.CustomFields["customfield_10035"]
	if !ok || v == nil {
		return "-"
	}
	switch n := v.(type) {
	case float64:
		if n == float64(int(n)) {
			return fmt.Sprintf("%d", int(n))
		}
		return fmt.Sprintf("%.1f", n)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func issueAssigneeNameRaw(issue *api.Issue) string {
	if issue.Fields.Assignee != nil {
		return issue.Fields.Assignee.DisplayName
	}
	return ""
}

func issueReporterNameRaw(issue *api.Issue) string {
	if issue.Fields.Reporter != nil {
		return issue.Fields.Reporter.DisplayName
	}
	return ""
}

func issueParentKey(issue *api.Issue) string {
	if issue.Fields.Parent != nil {
		return issue.Fields.Parent.Key
	}
	return ""
}
