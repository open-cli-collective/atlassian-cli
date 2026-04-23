// Package present provides presenters that map domain types to presentation models.
package present

import (
	"fmt"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// LinkPresenter creates presentation models for issue links.
type LinkPresenter struct{}

// LinkListSpec declares the columns emitted by PresentList. Default:
// LINK_ID|TYPE|DIRECTION|ISSUE|SUMMARY. Extended adds TYPE_ID and STATUS.
var LinkListSpec = projection.Registry{
	{Header: "LINK_ID", Identity: true},
	{Header: "TYPE"},
	{Header: "DIRECTION"},
	{Header: "ISSUE"},
	{Header: "SUMMARY"},
	{Header: "TYPE_ID", Extended: true},
	{Header: "STATUS", Extended: true},
}

// LinkTypesSpec declares the columns for link types. All default.
var LinkTypesSpec = projection.Registry{
	{Header: "ID", Identity: true},
	{Header: "NAME"},
	{Header: "INWARD"},
	{Header: "OUTWARD"},
}

// PresentList creates a table presentation of issue links. Extended
// adds TYPE_ID and the linked issue's current STATUS.
func (LinkPresenter) PresentList(links []api.IssueLink, extended bool) *present.OutputModel {
	var headers []string
	if extended {
		headers = []string{"LINK_ID", "TYPE", "DIRECTION", "ISSUE", "SUMMARY", "TYPE_ID", "STATUS"}
	} else {
		headers = []string{"LINK_ID", "TYPE", "DIRECTION", "ISSUE", "SUMMARY"}
	}

	rows := make([]present.Row, len(links))
	for i, l := range links {
		var direction, key, summary, status string

		if l.OutwardIssue != nil {
			direction = l.Type.Inward
			key = l.OutwardIssue.Key
			summary = l.OutwardIssue.Fields.Summary
			if l.OutwardIssue.Fields.Status != nil {
				status = l.OutwardIssue.Fields.Status.Name
			}
		} else if l.InwardIssue != nil {
			direction = l.Type.Outward
			key = l.InwardIssue.Key
			summary = l.InwardIssue.Fields.Summary
			if l.InwardIssue.Fields.Status != nil {
				status = l.InwardIssue.Fields.Status.Name
			}
		}

		if extended {
			rows[i] = present.Row{
				Cells: []string{l.ID, l.Type.Name, direction, key, summary, OrDash(l.Type.ID), OrDash(status)},
			}
		} else {
			rows[i] = present.Row{
				Cells: []string{l.ID, l.Type.Name, direction, key, summary},
			}
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{Headers: headers, Rows: rows},
		},
	}
}

// PresentTypes creates a table presentation of issue link types.
func (LinkPresenter) PresentTypes(types []api.IssueLinkType) *present.OutputModel {
	rows := make([]present.Row, len(types))
	for i, t := range types {
		rows[i] = present.Row{
			Cells: []string{t.ID, t.Name, t.Inward, t.Outward},
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "NAME", "INWARD", "OUTWARD"},
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
