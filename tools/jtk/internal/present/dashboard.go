package present

import (
	"fmt"
	"strings"

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

// PresentList creates a table view: ID | GADGETS | OWNER | FAVOURITE | NAME.
// gadgetCounts maps dashboard ID → gadget count; nil map or missing key renders "-".
func (DashboardPresenter) PresentList(dashboards []api.Dashboard, gadgetCounts map[string]int) *present.OutputModel {
	rows := make([]present.Row, len(dashboards))
	for i, d := range dashboards {
		owner := ""
		if d.Owner != nil {
			owner = d.Owner.DisplayName
		}
		rows[i] = present.Row{
			Cells: []string{d.ID, formatGadgetCount(d.ID, gadgetCounts), owner, BoolString(d.IsFavourite), d.Name},
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "GADGETS", "OWNER", "FAVOURITE", "NAME"},
				Rows:    rows,
			},
		},
	}
}

// PresentListExtended creates an extended table: ID | GADGETS | OWNER | FAVOURITE | RANK | PERMISSIONS | NAME.
func (DashboardPresenter) PresentListExtended(dashboards []api.Dashboard, gadgetCounts map[string]int) *present.OutputModel {
	rows := make([]present.Row, len(dashboards))
	for i, d := range dashboards {
		owner := ""
		if d.Owner != nil {
			owner = d.Owner.DisplayName
		}
		rows[i] = present.Row{
			Cells: []string{
				d.ID,
				formatGadgetCount(d.ID, gadgetCounts),
				owner,
				BoolString(d.IsFavourite),
				FormatInt(d.Popularity),
				formatPermissions(d.SharePerm),
				d.Name,
			},
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "GADGETS", "OWNER", "FAVOURITE", "RANK", "PERMISSIONS", "NAME"},
				Rows:    rows,
			},
		},
	}
}

// PresentGadgets creates a table view: ID | POSITION | TITLE | TYPE.
func (DashboardPresenter) PresentGadgets(gadgets []api.DashboardGadget) *present.OutputModel {
	rows := make([]present.Row, len(gadgets))
	for i, g := range gadgets {
		pos := fmt.Sprintf("%d,%d", g.Position.Row, g.Position.Column)
		rows[i] = present.Row{
			Cells: []string{FormatInt(g.ID), pos, g.Title, g.ModuleID},
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "POSITION", "TITLE", "TYPE"},
				Rows:    rows,
			},
		},
	}
}

// PresentCreated creates a success message for dashboard creation.
func (DashboardPresenter) PresentCreated(name, id string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Created dashboard %s (%s)", name, id),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentCreatedWithURL creates a success message with URL for dashboard creation.
func (DashboardPresenter) PresentCreatedWithURL(name, id, url string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Created dashboard %s (%s)", name, id),
				Stream:  present.StreamStdout,
			},
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("URL: %s", url),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentDeleted creates a success message for dashboard deletion.
func (DashboardPresenter) PresentDeleted(dashboardID string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Deleted dashboard %s", dashboardID),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentGadgetRemoved creates a success message for gadget removal.
func (DashboardPresenter) PresentGadgetRemoved(gadgetID int, dashboardID string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Removed gadget %d from dashboard %s", gadgetID, dashboardID),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentEmpty creates an info message when no dashboards are found.
func (DashboardPresenter) PresentEmpty() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "No dashboards found",
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentNoGadgets creates an info message when no gadgets are on a dashboard.
func (DashboardPresenter) PresentNoGadgets(dashboardID string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("No gadgets on dashboard %s", dashboardID),
				Stream:  present.StreamStdout,
			},
		},
	}
}

func formatGadgetCount(id string, counts map[string]int) string {
	if counts == nil {
		return "-"
	}
	n, ok := counts[id]
	if !ok {
		return "-"
	}
	return FormatInt(n)
}

func formatPermissions(perms []api.SharePerm) string {
	if len(perms) == 0 {
		return "private"
	}
	parts := make([]string, len(perms))
	for i, p := range perms {
		switch p.Type {
		case "group":
			if p.Group != nil && p.Group.Name != "" {
				parts[i] = "group:" + p.Group.Name
			} else {
				parts[i] = "group"
			}
		case "project":
			if p.Project != nil && p.Project.Key != "" {
				parts[i] = "project:" + p.Project.Key
			} else {
				parts[i] = "project"
			}
		case "loggedin":
			parts[i] = "logged-in"
		default:
			parts[i] = p.Type
		}
	}
	return strings.Join(parts, ", ")
}
