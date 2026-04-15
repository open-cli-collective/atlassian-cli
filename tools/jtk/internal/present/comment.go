// Package present provides presenters that map domain types to presentation models.
package present

import (
	"fmt"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// CommentPresenter creates presentation models for comment data.
type CommentPresenter struct{}

// PresentList creates a table view for a list of comments.
func (CommentPresenter) PresentList(comments []api.Comment) *present.OutputModel {
	rows := make([]present.Row, len(comments))
	for i, c := range comments {
		author := "Unknown"
		if c.Author.DisplayName != "" {
			author = c.Author.DisplayName
		}
		// Truncate body for table display
		body := ""
		if c.Body != nil {
			body = c.Body.ToPlainText()
			if len(body) > 100 {
				body = body[:100] + "... [truncated, use --no-truncate for complete text]"
			}
		}
		rows[i] = present.Row{
			Cells: []string{c.ID, author, FormatTime(c.Created), body},
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "AUTHOR", "CREATED", "BODY"},
				Rows:    rows,
			},
		},
	}
}

// PresentListFull creates detail views for comments without truncation.
// Each comment becomes a DetailSection; the renderer owns spacing between sections.
func (CommentPresenter) PresentListFull(comments []api.Comment) *present.OutputModel {
	sections := make([]present.Section, len(comments))
	for i, c := range comments {
		author := "Unknown"
		if c.Author.DisplayName != "" {
			author = c.Author.DisplayName
		}
		body := ""
		if c.Body != nil {
			body = c.Body.ToPlainText()
		}
		sections[i] = &present.DetailSection{
			Fields: []present.Field{
				{Label: "ID", Value: c.ID},
				{Label: "Author", Value: author},
				{Label: "Created", Value: FormatTime(c.Created)},
				{Label: "Body", Value: body},
			},
		}
	}
	return &present.OutputModel{Sections: sections}
}

// PresentAdded creates a success message for comment addition.
func (CommentPresenter) PresentAdded(commentID, issueKey string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Added comment %s to %s", commentID, issueKey),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentDeleted creates a success message for comment deletion.
func (CommentPresenter) PresentDeleted(commentID, issueKey string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Deleted comment %s from %s", commentID, issueKey),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentEmpty creates an info message when no comments are found.
func (CommentPresenter) PresentEmpty(issueKey string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("No comments on %s", issueKey),
				Stream:  present.StreamStdout,
			},
		},
	}
}
