// Package present provides presenters that map domain types to presentation models.
package present

import (
	"fmt"
	"strconv"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// RemoteLinkPresenter creates presentation models for issue remote (web) links.
type RemoteLinkPresenter struct{}

// RemoteLinkListSpec declares the columns emitted by PresentList. Default:
// ID|TITLE|URL. Extended: ID|RELATIONSHIP|TITLE|URL|SUMMARY. None of these
// map to Jira issue fields, so unknown --fields tokens correctly resolve to
// UnknownFieldError rather than a real /rest/api/3/field lookup.
var RemoteLinkListSpec = projection.Registry{
	{Header: "ID", Identity: true},
	{Header: "RELATIONSHIP", Extended: true},
	{Header: "TITLE"},
	{Header: "URL"},
	{Header: "SUMMARY", Extended: true},
}

// PresentList creates a table presentation of remote links. Extended adds the
// RELATIONSHIP and SUMMARY columns.
func (RemoteLinkPresenter) PresentList(links []api.RemoteLink, extended bool) *present.OutputModel {
	var headers []string
	if extended {
		headers = []string{"ID", "RELATIONSHIP", "TITLE", "URL", "SUMMARY"}
	} else {
		headers = []string{"ID", "TITLE", "URL"}
	}

	rows := make([]present.Row, len(links))
	for i, l := range links {
		id := strconv.Itoa(l.ID)
		if extended {
			rows[i] = present.Row{
				Cells: []string{id, OrDash(l.Relationship), OrDash(l.Object.Title), l.Object.URL, OrDash(l.Object.Summary)},
			}
		} else {
			rows[i] = present.Row{
				Cells: []string{id, OrDash(l.Object.Title), l.Object.URL},
			}
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{Headers: headers, Rows: rows},
		},
	}
}

// PresentAddedDetail creates a post-state detail block for a newly added
// remote link, mirroring the `get`-style shape used by other mutations.
func (RemoteLinkPresenter) PresentAddedDetail(issueKey string, l *api.RemoteLink) *present.OutputModel {
	fields := []present.Field{
		{Label: "ID", Value: strconv.Itoa(l.ID)},
		{Label: "Issue", Value: issueKey},
		{Label: "Title", Value: OrDash(l.Object.Title)},
		{Label: "URL", Value: l.Object.URL},
	}
	if l.Relationship != "" {
		fields = append(fields, present.Field{Label: "Relationship", Value: l.Relationship})
	}
	if l.Object.Summary != "" {
		fields = append(fields, present.Field{Label: "Summary", Value: l.Object.Summary})
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Added remote link %d to %s", l.ID, issueKey),
				Stream:  present.StreamStdout,
			},
			&present.DetailSection{Fields: fields},
		},
	}
}

// PresentRemoved creates a success message for remote link removal.
func (RemoteLinkPresenter) PresentRemoved(linkID, issueKey string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Removed remote link %s from %s", linkID, issueKey),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentEmpty creates an info message when no remote links are found.
func (RemoteLinkPresenter) PresentEmpty(issueKey string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("No remote links on %s", issueKey),
				Stream:  present.StreamStdout,
			},
		},
	}
}
