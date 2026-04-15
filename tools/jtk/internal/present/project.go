// Package present provides presenters that map domain types to presentation models.
package present

import (
	"fmt"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// ProjectPresenter creates presentation models for project data.
type ProjectPresenter struct{}

// Present creates a presentation model for text output.
// Content normalization (if any) happens here, not in the renderer.
func (ProjectPresenter) Present(p *api.ProjectDetail) *present.OutputModel {
	lead := "Unassigned"
	if p.Lead != nil {
		lead = p.Lead.DisplayName
	}

	fields := []present.Field{
		{Label: "Key", Value: p.Key},
		{Label: "Name", Value: p.Name},
		{Label: "ID", Value: p.ID.String()},
		{Label: "Type", Value: p.ProjectTypeKey},
		{Label: "Lead", Value: lead},
	}

	if p.Description != "" {
		fields = append(fields, present.Field{Label: "Description", Value: p.Description})
	}

	if len(p.IssueTypes) > 0 {
		var names []string
		for _, it := range p.IssueTypes {
			names = append(names, it.Name)
		}
		fields = append(fields, present.Field{Label: "Issue Types", Value: fmt.Sprintf("%s", names)})
	}

	if p.URL != "" {
		fields = append(fields, present.Field{Label: "URL", Value: p.URL})
	}

	return &present.OutputModel{
		Sections: []present.Section{&present.DetailSection{Fields: fields}},
	}
}

// PresentList creates a table view for a list of projects.
func (ProjectPresenter) PresentList(projects []api.ProjectDetail) *present.OutputModel {
	rows := make([]present.Row, len(projects))
	for i, p := range projects {
		lead := ""
		if p.Lead != nil {
			lead = p.Lead.DisplayName
		}
		rows[i] = present.Row{
			Cells: []string{p.Key, p.Name, p.ProjectTypeKey, lead},
		}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"KEY", "NAME", "TYPE", "LEAD"},
				Rows:    rows,
			},
		},
	}
}

// PresentTypes creates a table view for a list of project types.
func (ProjectPresenter) PresentTypes(types []api.ProjectType) *present.OutputModel {
	rows := make([]present.Row, len(types))
	for i, t := range types {
		rows[i] = present.Row{
			Cells: []string{t.Key, t.FormattedKey},
		}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"KEY", "FORMATTED"},
				Rows:    rows,
			},
		},
	}
}
