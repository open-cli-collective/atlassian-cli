// Package present provides presenters that map domain types to presentation models.
package present

import (
	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// LinkPresenter creates presentation models for issue links.
type LinkPresenter struct{}

// PresentList creates a table presentation of issue links.
func (LinkPresenter) PresentList(links []api.IssueLink) *present.OutputModel {
	rows := make([]present.Row, len(links))
	for i, l := range links {
		var direction, key, summary string

		if l.OutwardIssue != nil {
			// OutwardIssue is set → current issue is the inward side
			direction = l.Type.Inward
			key = l.OutwardIssue.Key
			summary = l.OutwardIssue.Fields.Summary
		} else if l.InwardIssue != nil {
			// InwardIssue is set → current issue is the outward side
			direction = l.Type.Outward
			key = l.InwardIssue.Key
			summary = l.InwardIssue.Fields.Summary
		}

		rows[i] = present.Row{
			Cells: []string{l.ID, l.Type.Name, direction, key, summary},
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "TYPE", "DIRECTION", "ISSUE", "SUMMARY"},
				Rows:    rows,
			},
		},
	}
}

// PresentTypes creates a table presentation of issue link types.
func (LinkPresenter) PresentTypes(types []api.IssueLinkType) *present.OutputModel {
	rows := make([]present.Row, len(types))
	for i, t := range types {
		rows[i] = present.Row{
			Cells: []string{t.ID, t.Name, t.Outward, t.Inward},
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "NAME", "OUTWARD", "INWARD"},
				Rows:    rows,
			},
		},
	}
}
