// Package present provides presenters that map domain types to presentation models.
package present

import (
	"fmt"
	"strings"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// CommentPresenter creates presentation models for comment data.
type CommentPresenter struct{}

// CommentListSpec declares the columns emitted by PresentList /
// PresentListWithPagination. Order MUST match the hardcoded Headers in
// PresentList. None of these have a Jira FieldID — comment fields are
// not Jira issue fields, so resolve.go's slow path will never find a
// match and unknown tokens correctly return UnknownFieldError.
var CommentListSpec = projection.Registry{
	{Header: "ID", Identity: true},
	{Header: "AUTHOR"},
	{Header: "CREATED"},
	{Header: "UPDATED", Extended: true},
	{Header: "BODY"},
}

// CommentDetailSpec declares the Fields emitted by PresentListFull /
// PresentListFullWithPagination. Order MUST match the per-comment field
// order in PresentListFull.
var CommentDetailSpec = projection.Registry{
	{Header: "ID", Identity: true},
	{Header: "Author"},
	{Header: "Created"},
	{Header: "Updated", Extended: true},
	{Header: "Body"},
}

// PresentList creates a table view for a list of comments. Extended
// adds UPDATED column with full timestamp.
func (CommentPresenter) PresentList(comments []api.Comment, extended bool) *present.OutputModel {
	var headers []string
	if extended {
		headers = []string{"ID", "AUTHOR", "CREATED", "UPDATED", "BODY"}
	} else {
		headers = []string{"ID", "AUTHOR", "CREATED", "BODY"}
	}

	rows := make([]present.Row, len(comments))
	for i, c := range comments {
		author := "Unknown"
		if c.Author.DisplayName != "" {
			author = c.Author.DisplayName
		}
		body := ""
		if c.Body != nil {
			body = c.Body.ToPlainText()
			if len(body) > 100 {
				body = body[:100] + "..."
			}
		}
		if extended {
			rows[i] = present.Row{
				Cells: []string{c.ID, author, OrDash(c.Created), OrDash(c.Updated), body},
			}
		} else {
			rows[i] = present.Row{
				Cells: []string{c.ID, author, FormatTime(c.Created), body},
			}
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{Headers: headers, Rows: rows},
		},
	}
}

// PresentListFull creates detail views for comments without truncation.
// Each comment becomes a DetailSection. Extended adds Updated field.
func (CommentPresenter) PresentListFull(comments []api.Comment, extended bool) *present.OutputModel {
	sections := make([]present.Section, len(comments))
	for i, c := range comments {
		author := "Unknown"
		if c.Author.DisplayName != "" {
			author = c.Author.DisplayName
		}
		body := ""
		if c.Body != nil {
			body = strings.TrimRight(c.Body.ToPlainText(), "\n")
		}
		fields := []present.Field{
			{Label: "ID", Value: c.ID},
			{Label: "Author", Value: author},
			{Label: "Created", Value: FormatTime(c.Created)},
		}
		if extended {
			fields = append(fields, present.Field{Label: "Updated", Value: OrDash(c.Updated)})
		}
		fields = append(fields, present.Field{Label: "Body", Value: body})
		sections[i] = &present.DetailSection{Fields: fields}
	}
	return &present.OutputModel{Sections: sections}
}

// PresentListWithPagination wraps PresentList and appends a stdout-bound
// pagination hint when hasMore is true.
func (p CommentPresenter) PresentListWithPagination(comments []api.Comment, extended bool, hasMore bool) *present.OutputModel {
	model := p.PresentList(comments, extended)
	model.Sections = AppendPaginationHint(model.Sections, hasMore)
	return model
}

// PresentListFullWithPagination wraps PresentListFull and appends a
// stdout-bound pagination hint when hasMore is true.
func (p CommentPresenter) PresentListFullWithPagination(comments []api.Comment, extended bool, hasMore bool) *present.OutputModel {
	model := p.PresentListFull(comments, extended)
	model.Sections = AppendPaginationHint(model.Sections, hasMore)
	return model
}

// PresentAddedDetail creates a post-state detail block for a newly added comment.
// Matches the spec shape: "ISSUE-KEY #ID — Author, Date\nBody text"
func (CommentPresenter) PresentAddedDetail(issueKey string, c *api.Comment) *present.OutputModel {
	author := "Unknown"
	if c.Author.DisplayName != "" {
		author = c.Author.DisplayName
	}
	body := ""
	if c.Body != nil {
		body = strings.TrimRight(c.Body.ToPlainText(), "\n")
	}
	header := fmt.Sprintf("%s #%s — %s, %s", issueKey, c.ID, author, FormatTime(c.Created))
	sections := []present.Section{
		&present.MessageSection{
			Kind:    present.MessageSuccess,
			Message: header,
			Stream:  present.StreamStdout,
		},
	}
	if body != "" {
		sections = append(sections, &present.MessageSection{
			Kind:    present.MessageInfo,
			Message: body,
			Stream:  present.StreamStdout,
		})
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
