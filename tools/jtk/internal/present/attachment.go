// Package present provides presenters that map domain types to presentation models.
package present

import (
	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// AttachmentPresenter creates presentation models for issue attachments.
type AttachmentPresenter struct{}

// PresentList creates a table presentation of attachments.
func (AttachmentPresenter) PresentList(attachments []api.Attachment) *present.OutputModel {
	rows := make([]present.Row, len(attachments))
	for i, a := range attachments {
		rows[i] = present.Row{
			Cells: []string{a.ID.String(), a.Filename, FormatSize(a.Size), FormatTime(a.Created), a.Author.DisplayName},
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "FILENAME", "SIZE", "CREATED", "AUTHOR"},
				Rows:    rows,
			},
		},
	}
}
