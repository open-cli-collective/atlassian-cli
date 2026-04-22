package present

import (
	"fmt"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// SprintPresenter creates presentation models for sprint data.
type SprintPresenter struct{}

// SprintListSpec declares the columns emitted by PresentList. Default order
// per #230 is ID|STATE|START|END|NAME; extended adds COMPLETED, BOARD, GOAL.
var SprintListSpec = projection.Registry{
	{Header: "ID", Identity: true},
	{Header: "STATE"},
	{Header: "START"},
	{Header: "END"},
	{Header: "COMPLETED", Extended: true},
	{Header: "BOARD", Extended: true},
	{Header: "GOAL", Extended: true},
	{Header: "NAME"},
}

// SprintDetailSpec declares the fields emitted by PresentDetailProjection.
var SprintDetailSpec = projection.Registry{
	{Header: "ID", Identity: true},
	{Header: "NAME"},
	{Header: "STATE"},
	{Header: "START"},
	{Header: "END"},
	{Header: "BOARD"},
	{Header: "GOAL", Extended: true},
	{Header: "ORIGIN_BOARD", Extended: true},
}

// PresentList renders `sprints list` output as a table. BOARD column uses
// each sprint's OriginBoardID (per-row), not the request boardID.
func (SprintPresenter) PresentList(sprints []api.Sprint, extended bool) *present.OutputModel {
	var headers []string
	if extended {
		headers = []string{"ID", "STATE", "START", "END", "COMPLETED", "BOARD", "GOAL", "NAME"}
	} else {
		headers = []string{"ID", "STATE", "START", "END", "NAME"}
	}

	rows := make([]present.Row, len(sprints))
	for i, s := range sprints {
		var cells []string
		if extended {
			boardVal := "-"
			if s.OriginBoardID != 0 {
				boardVal = FormatInt(s.OriginBoardID)
			}
			cells = []string{
				FormatInt(s.ID),
				OrDash(s.State),
				FormatDateOrDash(s.StartDate),
				FormatDateOrDash(s.EndDate),
				FormatDateOrDash(s.CompleteDate),
				boardVal,
				OrDash(s.Goal),
				s.Name,
			}
		} else {
			cells = []string{
				FormatInt(s.ID),
				OrDash(s.State),
				FormatDateOrDash(s.StartDate),
				FormatDateOrDash(s.EndDate),
				s.Name,
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

// PresentDetail builds the spec-shaped output for `sprints current`.
// Board name degrades gracefully: "Board: 23 (MON board)" when known,
// "Board: 23" when synthetic pass-through.
func (SprintPresenter) PresentDetail(sprint *api.Sprint, board *api.Board, extended bool) *present.OutputModel {
	sections := []present.Section{
		msg(fmt.Sprintf("%d  %s", sprint.ID, sprint.Name)),
	}

	if extended {
		sections = append(sections,
			msg(fmt.Sprintf("State: %s   Start: %s   End: %s",
				OrDash(sprint.State),
				FormatTimestampOrDash(sprint.StartDate),
				FormatTimestampOrDash(sprint.EndDate))),
		)
	} else {
		sections = append(sections,
			msg(fmt.Sprintf("State: %s   Start: %s   End: %s",
				OrDash(sprint.State),
				FormatDateOrDash(sprint.StartDate),
				FormatDateOrDash(sprint.EndDate))),
		)
	}

	sections = append(sections, msg("Board: "+formatBoardRef(board)))

	if extended {
		sections = append(sections, msg("Goal: "+OrDash(sprint.Goal)))
		originBoard := "-"
		if sprint.OriginBoardID != 0 {
			originBoard = FormatInt(sprint.OriginBoardID)
		}
		sections = append(sections, msg("Origin Board: "+originBoard))
	}

	return &present.OutputModel{Sections: sections}
}

// PresentDetailProjection builds a DetailSection view for `sprints current --fields`.
func (SprintPresenter) PresentDetailProjection(sprint *api.Sprint, board *api.Board) *present.OutputModel {
	originBoard := "-"
	if sprint.OriginBoardID != 0 {
		originBoard = FormatInt(sprint.OriginBoardID)
	}

	fields := []present.Field{
		{Label: "ID", Value: FormatInt(sprint.ID)},
		{Label: "NAME", Value: sprint.Name},
		{Label: "STATE", Value: OrDash(sprint.State)},
		{Label: "START", Value: FormatDateOrDash(sprint.StartDate)},
		{Label: "END", Value: FormatDateOrDash(sprint.EndDate)},
		{Label: "BOARD", Value: formatBoardRef(board)},
		{Label: "GOAL", Value: OrDash(sprint.Goal)},
		{Label: "ORIGIN_BOARD", Value: originBoard},
	}
	return &present.OutputModel{
		Sections: []present.Section{&present.DetailSection{Fields: fields}},
	}
}

// PresentMoved creates a success message for moving issues to a sprint.
func (SprintPresenter) PresentMoved(issueKeys []string, sprintID int) *present.OutputModel {
	var msg string
	if len(issueKeys) == 1 {
		msg = fmt.Sprintf("Moved %s to sprint %d", issueKeys[0], sprintID)
	} else {
		msg = fmt.Sprintf("Moved %d issues to sprint %d", len(issueKeys), sprintID)
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: msg,
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentEmpty creates an info message when no sprints are found.
func (SprintPresenter) PresentEmpty() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "No sprints found",
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentNoIssues creates an info message when no issues are in a sprint.
func (SprintPresenter) PresentNoIssues() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "No issues in sprint",
				Stream:  present.StreamStdout,
			},
		},
	}
}

// formatBoardRef renders a board reference: "23 (MON board)" when name is
// known, "23" when synthetic pass-through (cold cache / numeric-only).
func formatBoardRef(board *api.Board) string {
	if board == nil {
		return "-"
	}
	if board.Name != "" {
		return fmt.Sprintf("%d (%s)", board.ID, board.Name)
	}
	return FormatInt(board.ID)
}
