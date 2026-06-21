package present

import (
	"fmt"
	"strings"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// IssueHistoryPresenter creates presentation models for issue changelog data.
type IssueHistoryPresenter struct{}

// IssueHistorySpec declares the columns emitted by PresentIssueHistory.
// Default: ID|CREATED|AUTHOR|FIELD|FROM|TO. Extended adds raw audit fields.
var IssueHistorySpec = projection.Registry{
	{Header: "ID", Aliases: []string{"id"}, Identity: true},
	{Header: "CREATED", Aliases: []string{"created"}},
	{Header: "AUTHOR", Aliases: []string{"author"}},
	{Header: "ACCOUNT_ID", Aliases: []string{"accountId", "account_id"}, Extended: true},
	{Header: "FIELD", Aliases: []string{"field"}},
	{Header: "FIELD_ID", Aliases: []string{"fieldId", "field_id"}, Extended: true},
	{Header: "TYPE", Aliases: []string{"fieldtype", "fieldType", "type"}, Extended: true},
	{Header: "FROM_ID", Aliases: []string{"fromId", "from_id"}, Extended: true},
	{Header: "FROM", Aliases: []string{"from"}},
	{Header: "TO_ID", Aliases: []string{"toId", "to_id"}, Extended: true},
	{Header: "TO", Aliases: []string{"to"}},
}

// IssueHistoryRow is one flattened changelog item row.
type IssueHistoryRow struct {
	ID        string
	Created   string
	Author    string
	AccountID string
	Field     string
	FieldID   string
	FieldType string
	FromID    string
	From      string
	ToID      string
	To        string
}

// FlattenIssueHistory flattens Jira changelog groups into one row per changed field.
func FlattenIssueHistory(histories []api.IssueChangelogHistory) []IssueHistoryRow {
	var rows []IssueHistoryRow
	for _, h := range histories {
		author := "Unknown"
		accountID := ""
		if h.Author != nil {
			if h.Author.DisplayName != "" {
				author = h.Author.DisplayName
			}
			accountID = h.Author.AccountID
		}
		for _, item := range h.Items {
			rows = append(rows, IssueHistoryRow{
				ID:        h.ID,
				Created:   h.Created,
				Author:    author,
				AccountID: accountID,
				Field:     item.Field,
				FieldID:   item.FieldID,
				FieldType: item.FieldType,
				FromID:    item.From,
				From:      historyDisplayValue(item.FromString, item.From),
				ToID:      item.To,
				To:        historyDisplayValue(item.ToString, item.To),
			})
		}
	}
	return rows
}

// PresentIssueHistory creates a table view for issue changelog rows.
func (IssueHistoryPresenter) PresentIssueHistory(rows []IssueHistoryRow, extended bool, fulltext bool) *present.OutputModel {
	var headers []string
	if extended {
		headers = []string{"ID", "CREATED", "AUTHOR", "ACCOUNT_ID", "FIELD", "FIELD_ID", "TYPE", "FROM_ID", "FROM", "TO_ID", "TO"}
	} else {
		headers = []string{"ID", "CREATED", "AUTHOR", "FIELD", "FROM", "TO"}
	}

	outRows := make([]present.Row, len(rows))
	for i, row := range rows {
		created := FormatTime(row.Created)
		if extended {
			created = row.Created
		}
		if extended {
			outRows[i] = present.Row{Cells: []string{
				OrDash(row.ID),
				OrDash(created),
				OrDash(row.Author),
				OrDash(row.AccountID),
				OrDash(row.Field),
				OrDash(row.FieldID),
				OrDash(row.FieldType),
				OrDash(row.FromID),
				formatHistoryCell(row.From, fulltext),
				OrDash(row.ToID),
				formatHistoryCell(row.To, fulltext),
			}}
		} else {
			outRows[i] = present.Row{Cells: []string{
				OrDash(row.ID),
				OrDash(created),
				OrDash(row.Author),
				OrDash(row.Field),
				formatHistoryCell(row.From, fulltext),
				formatHistoryCell(row.To, fulltext),
			}}
		}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{Headers: headers, Rows: outRows},
		},
	}
}

// PresentIssueHistoryWithPagination wraps PresentIssueHistory and appends a continuation token when needed.
func (p IssueHistoryPresenter) PresentIssueHistoryWithPagination(rows []IssueHistoryRow, extended bool, fulltext bool, hasMore bool, nextToken string) *present.OutputModel {
	model := p.PresentIssueHistory(rows, extended, fulltext)
	model.Sections = AppendPaginationHintWithToken(model.Sections, hasMore, nextToken)
	return model
}

// PresentNoIssueHistory creates an info message when no changelog items are found.
func (IssueHistoryPresenter) PresentNoIssueHistory(issueKey string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("No history found for %s", issueKey),
				Stream:  present.StreamStdout,
			},
		},
	}
}

func historyDisplayValue(display, raw string) string {
	if display != "" {
		return display
	}
	return raw
}

func formatHistoryCell(s string, fulltext bool) string {
	// Jira changelog values can contain embedded newlines and tabs that break table output.
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return "-"
	}
	if fulltext {
		return s
	}
	return OrDash(TruncateText(s, 80))
}
