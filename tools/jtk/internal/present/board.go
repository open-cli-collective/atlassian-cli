package present

import (
	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// BoardPresenter creates presentation models for board data.
type BoardPresenter struct{}

// PresentDetail creates a detail view for a single board.
func (BoardPresenter) PresentDetail(board *api.Board) *present.OutputModel {
	fields := []present.Field{
		{Label: "ID", Value: FormatInt(board.ID)},
		{Label: "Name", Value: board.Name},
		{Label: "Type", Value: board.Type},
		{Label: "Project", Value: board.Location.ProjectKey},
	}

	return &present.OutputModel{
		Sections: []present.Section{&present.DetailSection{Fields: fields}},
	}
}

// PresentList creates a table view for a list of boards.
func (BoardPresenter) PresentList(boards []api.Board) *present.OutputModel {
	rows := make([]present.Row, len(boards))
	for i, b := range boards {
		rows[i] = present.Row{
			Cells: []string{
				FormatInt(b.ID),
				b.Name,
				b.Type,
				b.Location.ProjectKey,
			},
		}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "NAME", "TYPE", "PROJECT"},
				Rows:    rows,
			},
		},
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
