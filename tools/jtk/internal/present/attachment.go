// Package present provides presenters that map domain types to presentation models.
package present

import (
	"fmt"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// AttachmentPresenter creates presentation models for issue attachments.
type AttachmentPresenter struct{}

// AttachmentListSpec declares the columns emitted by PresentList. Default:
// ID|FILENAME|SIZE|AUTHOR|CREATED. Extended:
// ID|FILENAME|SIZE|BYTES|MIME_TYPE|AUTHOR|CREATED.
var AttachmentListSpec = projection.Registry{
	{Header: "ID", Identity: true},
	{Header: "FILENAME"},
	{Header: "SIZE"},
	{Header: "BYTES", Extended: true},
	{Header: "MIME_TYPE", Extended: true},
	{Header: "AUTHOR"},
	{Header: "CREATED"},
}

// PresentList creates a table presentation of attachments. Extended:
// ID|FILENAME|SIZE|BYTES|MIME_TYPE|AUTHOR|CREATED.
func (AttachmentPresenter) PresentList(attachments []api.Attachment, extended bool) *present.OutputModel {
	var headers []string
	if extended {
		headers = []string{"ID", "FILENAME", "SIZE", "BYTES", "MIME_TYPE", "AUTHOR", "CREATED"}
	} else {
		headers = []string{"ID", "FILENAME", "SIZE", "AUTHOR", "CREATED"}
	}

	rows := make([]present.Row, len(attachments))
	for i, a := range attachments {
		if extended {
			rows[i] = present.Row{
				Cells: []string{
					a.ID.String(),
					a.Filename,
					FormatSize(a.Size),
					FormatInt(int(a.Size)),
					OrDash(a.MimeType),
					a.Author.DisplayName,
					OrDash(a.Created),
				},
			}
		} else {
			rows[i] = present.Row{
				Cells: []string{a.ID.String(), a.Filename, FormatSize(a.Size), a.Author.DisplayName, FormatTime(a.Created)},
			}
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{Headers: headers, Rows: rows},
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
func (AttachmentPresenter) PresentDownloaded(attachmentID, filename string, sizeBytes int64) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Downloaded %s → %s (%s)", attachmentID, filename, FormatSize(sizeBytes)),
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
