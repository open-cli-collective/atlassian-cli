package present

import (
	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// DashboardPresenter creates presentation models for dashboard data.
type DashboardPresenter struct{}

// PresentDetail creates a detail view for a single dashboard with its gadgets.
func (DashboardPresenter) PresentDetail(d *api.Dashboard, gadgets []api.DashboardGadget) *present.OutputModel {
	fields := []present.Field{
		{Label: "ID", Value: d.ID},
		{Label: "Name", Value: d.Name},
	}
	if d.Description != "" {
		fields = append(fields, present.Field{Label: "Description", Value: d.Description})
	}
	if d.Owner != nil {
		fields = append(fields, present.Field{Label: "Owner", Value: d.Owner.DisplayName})
	}
	if d.View != "" {
		fields = append(fields, present.Field{Label: "URL", Value: d.View})
	}

	sections := []present.Section{&present.DetailSection{Fields: fields}}

	if len(gadgets) > 0 {
		rows := make([]present.Row, len(gadgets))
		for i, g := range gadgets {
			rows[i] = present.Row{
				Cells: []string{FormatInt(g.ID), g.Title, g.ModuleID},
			}
		}
		sections = append(sections, &present.TableSection{
			Headers: []string{"ID", "TITLE", "MODULE"},
			Rows:    rows,
		})
	}

	return &present.OutputModel{Sections: sections}
}

// PresentList creates a table view for a list of dashboards.
func (DashboardPresenter) PresentList(dashboards []api.Dashboard) *present.OutputModel {
	rows := make([]present.Row, len(dashboards))
	for i, d := range dashboards {
		owner := ""
		if d.Owner != nil {
			owner = d.Owner.DisplayName
		}
		fav := BoolString(d.IsFavourite)
		rows[i] = present.Row{
			Cells: []string{d.ID, d.Name, owner, fav},
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "NAME", "OWNER", "FAVOURITE"},
				Rows:    rows,
			},
		},
	}
}

// PresentGadgets creates a table view for a list of gadgets.
func (DashboardPresenter) PresentGadgets(gadgets []api.DashboardGadget) *present.OutputModel {
	rows := make([]present.Row, len(gadgets))
	for i, g := range gadgets {
		pos := ""
		if g.Position.Row > 0 || g.Position.Column > 0 {
			pos = FormatInt(g.Position.Row) + "," + FormatInt(g.Position.Column)
		}
		rows[i] = present.Row{
			Cells: []string{FormatInt(g.ID), g.Title, g.ModuleID, pos},
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "TITLE", "MODULE", "POSITION"},
				Rows:    rows,
			},
		},
	}
}
