// Package present provides presenters that map domain types to presentation models.
package present

import (
	"fmt"
	"strings"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// componentPreviewLimit caps the enumerated component list in `projects get
// --extended` before a "... [N more]" truncation line. 4 mirrors the example
// in the #230 specification.
const componentPreviewLimit = 4

// ProjectPresenter creates presentation models for project data.
type ProjectPresenter struct{}

// ProjectDetailSpec declares the logical fields rendered by
// PresentProjectDetailProjection. Identity is KEY (projection always retains
// it on the first line). Extended fields correspond to the admin/schema rows
// the spec puts behind `--extended`.
var ProjectDetailSpec = projection.Registry{
	{Header: "KEY", Identity: true},
	{Header: "NAME"},
	{Header: "TYPE"},
	{Header: "LEAD"},
	{Header: "STYLE"},
	{Header: "ISSUE_TYPES"},
	{Header: "COMPONENTS"},
	{Header: "VERSIONS"},
	{Header: "DESCRIPTION", Extended: true},
	{Header: "LEAD_ID", Extended: true},
	{Header: "ISSUE_TYPE_IDS", Extended: true},
	{Header: "COMPONENT_IDS", Extended: true},
	{Header: "SIMPLIFIED", Extended: true},
	{Header: "PRIVATE", Extended: true},
}

// ProjectListSpec declares the columns emitted by PresentProjectList. Order
// matches the Headers slice inside the presenter; a parity test locks the two.
// Default order per #230 is KEY|TYPE|LEAD|NAME; extended order interleaves
// STYLE between TYPE and LEAD and ISSUE_TYPES/COMPONENTS before NAME:
// KEY|TYPE|STYLE|LEAD|ISSUE_TYPES|COMPONENTS|NAME. Registry.ForMode preserves
// declaration order when expanding, so this file declares the interleaved
// extended order and marks the three non-default entries Extended:true.
var ProjectListSpec = projection.Registry{
	{Header: "KEY", Identity: true},
	{Header: "TYPE"},
	{Header: "STYLE", Extended: true},
	{Header: "LEAD"},
	{Header: "ISSUE_TYPES", Extended: true},
	{Header: "COMPONENTS", Extended: true},
	{Header: "NAME"},
}

// ProjectTypeSpec declares the columns for `projects types`. The default NAME
// column sources from ProjectType.FormattedKey (matches the spec's user-facing
// wording); extended adds the raw i18n description key.
var ProjectTypeSpec = projection.Registry{
	{Header: "KEY", Identity: true},
	{Header: "NAME"},
	{Header: "DESCRIPTION_KEY", Extended: true},
}

// PresentProjectDetail builds the spec-shaped default or extended output for
// `projects get`. Default output: title line + three compound rows. Extended
// output: title line + extended compound rows with lead/issue-type IDs, an
// enumerated component preview with truncation, a Versions line, Simplified
// / Private flags, and an optional Description block.
func (ProjectPresenter) PresentProjectDetail(p *api.ProjectDetail, extended bool) *present.OutputModel {
	sections := []present.Section{
		msg(fmt.Sprintf("%s  %s", p.Key, p.Name)),
	}
	if extended {
		sections = append(sections, extendedProjectDetailSections(p)...)
	} else {
		sections = append(sections, defaultProjectDetailSections(p)...)
	}
	return &present.OutputModel{Sections: sections}
}

// defaultProjectDetailSections returns the compound-KV rows that follow the
// title line in default-mode output. Types/Lead/Style, Issue Types list,
// Components / Versions counts. The Issue Types row is always emitted so the
// line count stays stable across projects — "-" when empty per #230's
// parseable-text-shape goal.
func defaultProjectDetailSections(p *api.ProjectDetail) []present.Section {
	return []present.Section{
		msg(fmt.Sprintf("Type: %s   Lead: %s   Style: %s",
			OrDash(p.ProjectTypeKey), leadDisplayName(p.Lead), OrDash(p.Style))),
		msg("Issue Types: " + OrDash(issueTypeNames(p.IssueTypes))),
		msg(fmt.Sprintf("Components: %d   Versions: %d", len(p.Components), len(p.Versions))),
	}
}

// extendedProjectDetailSections returns the expanded post-title sections for
// --extended mode. Components are enumerated with IDs, truncated after
// componentPreviewLimit entries. Issue Types is always emitted (empty →
// "Issue Types: -") to keep the rendered row count independent of data.
func extendedProjectDetailSections(p *api.ProjectDetail) []present.Section {
	out := []present.Section{
		msg(fmt.Sprintf("Type: %s   Lead: %s   Style: %s",
			OrDash(p.ProjectTypeKey), leadDisplayNameWithID(p.Lead), OrDash(p.Style))),
		msg("Issue Types: " + OrDash(issueTypeNamesWithIDs(p.IssueTypes))),
	}
	out = append(out, msg(fmt.Sprintf("Components: %d", len(p.Components))))
	for i, c := range p.Components {
		if i >= componentPreviewLimit {
			remaining := len(p.Components) - componentPreviewLimit
			out = append(out, msg(fmt.Sprintf("  ... [%d more]", remaining)))
			break
		}
		out = append(out, msg(fmt.Sprintf("  %s | %s", c.ID, c.Name)))
	}
	out = append(out, msg(fmt.Sprintf("Versions: %d", len(p.Versions))))
	out = append(out, msg(fmt.Sprintf("Simplified: %s   Private: %s",
		PresentOptionalBool(p.Simplified), PresentOptionalBool(p.IsPrivate))))
	if strings.TrimSpace(p.Description) != "" {
		out = append(out,
			msg("Description:"),
			msg(p.Description),
		)
	}
	return out
}

// PresentProjectDetailProjection builds a DetailSection view for `projects
// get --fields`, keyed by the headers declared in ProjectDetailSpec. Output
// flattens to "Label: Value" lines; projection.ProjectDetail then slices the
// section to the user-selected subset.
func (ProjectPresenter) PresentProjectDetailProjection(p *api.ProjectDetail) *present.OutputModel {
	fields := []present.Field{
		{Label: "KEY", Value: p.Key},
		{Label: "NAME", Value: p.Name},
		{Label: "TYPE", Value: OrDash(p.ProjectTypeKey)},
		{Label: "LEAD", Value: leadDisplayName(p.Lead)},
		{Label: "STYLE", Value: OrDash(p.Style)},
		{Label: "ISSUE_TYPES", Value: OrDash(issueTypeNames(p.IssueTypes))},
		{Label: "COMPONENTS", Value: fmt.Sprintf("%d", len(p.Components))},
		{Label: "VERSIONS", Value: fmt.Sprintf("%d", len(p.Versions))},
		{Label: "DESCRIPTION", Value: OrDash(p.Description)},
		{Label: "LEAD_ID", Value: leadAccountID(p.Lead)},
		{Label: "ISSUE_TYPE_IDS", Value: OrDash(issueTypeIDs(p.IssueTypes))},
		{Label: "COMPONENT_IDS", Value: OrDash(componentIDs(p.Components))},
		{Label: "SIMPLIFIED", Value: PresentOptionalBool(p.Simplified)},
		{Label: "PRIVATE", Value: PresentOptionalBool(p.IsPrivate)},
	}
	return &present.OutputModel{
		Sections: []present.Section{&present.DetailSection{Fields: fields}},
	}
}

// PresentProjectList renders `projects list` output as a table. Default order
// is KEY | TYPE | LEAD | NAME; --extended interleaves STYLE between TYPE and
// LEAD and ISSUE_TYPES/COMPONENTS before NAME, producing
// KEY | TYPE | STYLE | LEAD | ISSUE_TYPES | COMPONENTS | NAME per #230.
// ISSUE_TYPES renders as the comma-joined issue-type names (not a count);
// COMPONENTS renders as the count. Pagination is appended by the command via
// AppendPaginationHintWithToken.
func (ProjectPresenter) PresentProjectList(projects []api.ProjectDetail, extended bool) *present.OutputModel {
	var headers []string
	if extended {
		headers = []string{"KEY", "TYPE", "STYLE", "LEAD", "ISSUE_TYPES", "COMPONENTS", "NAME"}
	} else {
		headers = []string{"KEY", "TYPE", "LEAD", "NAME"}
	}

	rows := make([]present.Row, len(projects))
	for i, p := range projects {
		var cells []string
		if extended {
			cells = []string{
				p.Key,
				OrDash(p.ProjectTypeKey),
				OrDash(p.Style),
				leadDisplayName(p.Lead),
				OrDash(issueTypeNames(p.IssueTypes)),
				fmt.Sprintf("%d", len(p.Components)),
				p.Name,
			}
		} else {
			cells = []string{
				p.Key,
				OrDash(p.ProjectTypeKey),
				leadDisplayName(p.Lead),
				p.Name,
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

// PresentProjectTypes renders `projects types` output. Header is KEY | NAME
// (spec #230 names the column NAME even though the API field is FormattedKey);
// --extended adds DESCRIPTION_KEY from DescriptionI18nKey.
func (ProjectPresenter) PresentProjectTypes(types []api.ProjectType, extended bool) *present.OutputModel {
	headers := []string{"KEY", "NAME"}
	if extended {
		headers = append(headers, "DESCRIPTION_KEY")
	}
	rows := make([]present.Row, len(types))
	for i, t := range types {
		cells := []string{t.Key, t.FormattedKey}
		if extended {
			cells = append(cells, OrDash(t.DescriptionI18nKey))
		}
		rows[i] = present.Row{Cells: cells}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{Headers: headers, Rows: rows},
		},
	}
}

// PresentCreated creates a success message for project creation.
func (ProjectPresenter) PresentCreated(key, name string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Created project %s (%s)", key, name),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentUpdated creates a success message for project update.
func (ProjectPresenter) PresentUpdated(key string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Updated project %s", key),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentDeleted creates a success message for project deletion.
func (ProjectPresenter) PresentDeleted(key string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Deleted project %s (moved to trash — recoverable for 60 days via projects restore)", key),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentRestored creates a success message for project restoration.
func (ProjectPresenter) PresentRestored(key, name string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Restored project %s (%s)", key, name),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentEmpty creates an info message when no projects are found.
func (ProjectPresenter) PresentEmpty() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "No projects found",
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentNoTypes creates an info message when no project types are found.
func (ProjectPresenter) PresentNoTypes() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "No project types found",
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentDeleteCancelled creates an info message for cancelled deletion.
func (ProjectPresenter) PresentDeleteCancelled() *present.OutputModel {
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

// msg constructs a stdout-routed info MessageSection from a plaintext body.
// Used throughout the spec-shaped project presenters to keep each output line
// as its own addressable section.
func msg(body string) *present.MessageSection {
	return &present.MessageSection{
		Kind:    present.MessageInfo,
		Message: body,
		Stream:  present.StreamStdout,
	}
}

func leadDisplayName(u *api.User) string {
	if u == nil || u.DisplayName == "" {
		return "-"
	}
	return u.DisplayName
}

func leadDisplayNameWithID(u *api.User) string {
	if u == nil || u.DisplayName == "" {
		return "-"
	}
	if u.AccountID == "" {
		return u.DisplayName
	}
	return fmt.Sprintf("%s (%s)", u.DisplayName, u.AccountID)
}

func leadAccountID(u *api.User) string {
	if u == nil || u.AccountID == "" {
		return "-"
	}
	return u.AccountID
}

func issueTypeNames(types []api.IssueType) string {
	names := make([]string, 0, len(types))
	for _, t := range types {
		names = append(names, t.Name)
	}
	return strings.Join(names, ", ")
}

func issueTypeNamesWithIDs(types []api.IssueType) string {
	parts := make([]string, 0, len(types))
	for _, t := range types {
		if t.ID == "" {
			parts = append(parts, t.Name)
			continue
		}
		parts = append(parts, fmt.Sprintf("%s (%s)", t.Name, t.ID))
	}
	return strings.Join(parts, ", ")
}

func issueTypeIDs(types []api.IssueType) string {
	ids := make([]string, 0, len(types))
	for _, t := range types {
		if t.ID != "" {
			ids = append(ids, t.ID)
		}
	}
	return strings.Join(ids, ", ")
}

func componentIDs(comps []api.Component) string {
	ids := make([]string, 0, len(comps))
	for _, c := range comps {
		if c.ID != "" {
			ids = append(ids, c.ID)
		}
	}
	return strings.Join(ids, ", ")
}
