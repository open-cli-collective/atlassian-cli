package present

import (
	"fmt"
	"strings"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// BoardPresenter creates presentation models for board data.
type BoardPresenter struct{}

// BoardListSpec declares the columns emitted by PresentList. Default order
// per #230 is ID|TYPE|PROJECT|NAME; extended adds PROJECT_NAME between
// PROJECT and NAME.
var BoardListSpec = projection.Registry{
	{Header: "ID", Identity: true},
	{Header: "TYPE"},
	{Header: "PROJECT"},
	{Header: "PROJECT_NAME", Extended: true},
	{Header: "NAME"},
}

// BoardDetailSpec declares the fields emitted by PresentDetailProjection.
var BoardDetailSpec = projection.Registry{
	{Header: "ID", Identity: true},
	{Header: "NAME"},
	{Header: "TYPE"},
	{Header: "PROJECT"},
	{Header: "PROJECT_NAME"},
	{Header: "FILTER", Extended: true},
	{Header: "COLUMN_CONFIG", Extended: true},
}

// PresentList renders `boards list` output as a table. Default order is
// ID|TYPE|PROJECT|NAME; --extended adds PROJECT_NAME.
func (BoardPresenter) PresentList(boards []api.Board, extended bool) *present.OutputModel {
	var headers []string
	if extended {
		headers = []string{"ID", "TYPE", "PROJECT", "PROJECT_NAME", "NAME"}
	} else {
		headers = []string{"ID", "TYPE", "PROJECT", "NAME"}
	}

	rows := make([]present.Row, len(boards))
	for i, b := range boards {
		var cells []string
		if extended {
			cells = []string{
				FormatInt(b.ID),
				OrDash(b.Type),
				OrDash(b.Location.ProjectKey),
				OrDash(b.Location.ProjectName),
				b.Name,
			}
		} else {
			cells = []string{
				FormatInt(b.ID),
				OrDash(b.Type),
				OrDash(b.Location.ProjectKey),
				b.Name,
			}
		}
		rows[i] = present.Row{Cells: cells}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{Headers: headers, Rows: rows},
		},
	}
}

// PresentDetail builds the spec-shaped output for `boards get`. Default:
// title line + Type/Project row. Extended adds Filter and Column config.
func (BoardPresenter) PresentDetail(board *api.Board, config *api.BoardConfiguration, extended bool) *present.OutputModel {
	projectRef := OrDash(board.Location.ProjectKey)
	if board.Location.ProjectName != "" {
		projectRef = fmt.Sprintf("%s (%s)", board.Location.ProjectKey, board.Location.ProjectName)
	}

	sections := []present.Section{
		msg(fmt.Sprintf("%d  %s", board.ID, board.Name)),
		msg(fmt.Sprintf("Type: %s   Project: %s", OrDash(board.Type), projectRef)),
	}

	if extended {
		filterVal := "-"
		columnVal := "-"
		if config != nil {
			filterVal = fmt.Sprintf("%s (id: %s)", config.Filter.Name, config.Filter.ID)
			colNames := make([]string, len(config.ColumnConfig.Columns))
			for i, c := range config.ColumnConfig.Columns {
				colNames[i] = c.Name
			}
			columnVal = OrDash(strings.Join(colNames, ", "))
		}
		sections = append(sections, msg("Filter: "+filterVal))
		sections = append(sections, msg("Column config: "+columnVal))
	}

	return &present.OutputModel{Sections: sections}
}

// PresentDetailProjection builds a DetailSection view for `boards get --fields`.
func (BoardPresenter) PresentDetailProjection(board *api.Board, config *api.BoardConfiguration) *present.OutputModel {
	filterName := "-"
	columnConfig := "-"
	if config != nil {
		filterName = fmt.Sprintf("%s (id: %s)", config.Filter.Name, config.Filter.ID)
		colNames := make([]string, len(config.ColumnConfig.Columns))
		for i, c := range config.ColumnConfig.Columns {
			colNames[i] = c.Name
		}
		columnConfig = OrDash(strings.Join(colNames, ", "))
	}

	fields := []present.Field{
		{Label: "ID", Value: FormatInt(board.ID)},
		{Label: "NAME", Value: board.Name},
		{Label: "TYPE", Value: OrDash(board.Type)},
		{Label: "PROJECT", Value: OrDash(board.Location.ProjectKey)},
		{Label: "PROJECT_NAME", Value: OrDash(board.Location.ProjectName)},
		{Label: "FILTER", Value: filterName},
		{Label: "COLUMN_CONFIG", Value: columnConfig},
	}
	return &present.OutputModel{
		Sections: []present.Section{&present.DetailSection{Fields: fields}},
	}
}

// PresentEmpty creates an info message when no boards are found.
func (BoardPresenter) PresentEmpty() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "No boards found",
				Stream:  present.StreamStdout,
			},
		},
	}
}
