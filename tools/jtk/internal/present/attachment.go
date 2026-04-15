// Package present provides presenters that map domain types to presentation models.
package present

import (
	"fmt"

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

// PresentUploaded creates a success message for attachment upload.
func (AttachmentPresenter) PresentUploaded(filename, id, size string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Uploaded %s (ID: %s, Size: %s)", filename, id, size),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentDownloaded creates a success message for attachment download.
func (AttachmentPresenter) PresentDownloaded(filename, size string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Downloaded %s (%s)", filename, size),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentDeleted creates a success message for attachment deletion.
func (AttachmentPresenter) PresentDeleted(attachmentID string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Deleted attachment %s", attachmentID),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentEmpty creates an info message when no attachments are found.
func (AttachmentPresenter) PresentEmpty(issueKey string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("No attachments found on %s", issueKey),
				Stream:  present.StreamStdout,
			},
		},
	}
}
