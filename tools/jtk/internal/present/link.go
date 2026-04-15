// Package present provides presenters that map domain types to presentation models.
package present

import (
	"fmt"

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

// PresentCreated creates a success message for link creation.
func (LinkPresenter) PresentCreated(linkType, outwardKey, inwardKey string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Created %s link: %s → %s", linkType, outwardKey, inwardKey),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentDeleted creates a success message for link deletion.
func (LinkPresenter) PresentDeleted(linkID string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Deleted link %s", linkID),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentEmpty creates an info message when no links are found.
func (LinkPresenter) PresentEmpty(issueKey string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("No links on %s", issueKey),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentNoTypes creates an info message when no link types are available.
func (LinkPresenter) PresentNoTypes() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "No link types available",
				Stream:  present.StreamStdout,
			},
		},
	}
}
