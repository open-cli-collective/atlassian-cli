package present

import (
	"fmt"
	"sort"
	"time"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// SortSprintsForDisplay sorts sprints for display: active first, then future,
// then closed by recency. Within closed, sort by CompleteDate → EndDate →
// StartDate descending. Deterministic tie-breaker: ID descending.
func SortSprintsForDisplay(sprints []api.Sprint) {
	sort.SliceStable(sprints, func(i, j int) bool {
		pi, pj := statePriority(sprints[i].State), statePriority(sprints[j].State)
		if pi != pj {
			return pi < pj
		}
		if pi == 2 { // closed: most recently completed first
			return closedLess(sprints[i], sprints[j])
		}
		if pi == 1 { // future: nearest upcoming first (ascending)
			return dateAscThenID(sprints[i].StartDate, sprints[j].StartDate, sprints[i].ID, sprints[j].ID)
		}
		// active: most recent first (descending)
		return dateDescThenID(sprints[i].StartDate, sprints[j].StartDate, sprints[i].ID, sprints[j].ID)
	})
}

func statePriority(state string) int {
	switch state {
	case "active":
		return 0
	case "future":
		return 1
	default:
		return 2
	}
}

func closedLess(a, b api.Sprint) bool {
	if cmp := compareDatesDesc(a.CompleteDate, b.CompleteDate); cmp != 0 {
		return cmp < 0
	}
	if cmp := compareDatesDesc(a.EndDate, b.EndDate); cmp != 0 {
		return cmp < 0
	}
	return dateDescThenID(a.StartDate, b.StartDate, a.ID, b.ID)
}

func dateDescThenID(a, b *time.Time, idA, idB int) bool {
	if cmp := compareDatesDesc(a, b); cmp != 0 {
		return cmp < 0
	}
	return idA > idB
}

func dateAscThenID(a, b *time.Time, idA, idB int) bool {
	if cmp := compareDatesAsc(a, b); cmp != 0 {
		return cmp < 0
	}
	return idA < idB
}

// compareDatesAsc returns -1 if a should sort before b (ascending),
// +1 if after, 0 if equal. Nil sorts after non-nil (unknown = last).
func compareDatesAsc(a, b *time.Time) int {
	aNil := a == nil || a.IsZero()
	bNil := b == nil || b.IsZero()
	if aNil && bNil {
		return 0
	}
	if aNil {
		return 1
	}
	if bNil {
		return -1
	}
	if a.Before(*b) {
		return -1
	}
	if a.After(*b) {
		return 1
	}
	return 0
}

// compareDatesDesc returns -1 if a should sort before b (descending),
// +1 if after, 0 if equal. Nil sorts after non-nil.
func compareDatesDesc(a, b *time.Time) int {
	aNil := a == nil || a.IsZero()
	bNil := b == nil || b.IsZero()
	if aNil && bNil {
		return 0
	}
	if aNil {
		return 1
	}
	if bNil {
		return -1
	}
	if a.After(*b) {
		return -1
	}
	if a.Before(*b) {
		return 1
	}
	return 0
}

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

// PresentListWithPagination wraps PresentList and appends a pagination
// hint when hasMore is true.
func (p SprintPresenter) PresentListWithPagination(sprints []api.Sprint, extended, hasMore bool, nextToken string) *present.OutputModel {
	model := p.PresentList(sprints, extended)
	model.Sections = AppendPaginationHintWithToken(model.Sections, hasMore, nextToken)
	return model
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

// PresentPostStateUnavailable creates an advisory when post-state cannot be
// fetched after a mutation, falling back to a confirmation-only output.
func (SprintPresenter) PresentPostStateUnavailable() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "post-state unavailable; showing confirmation only",
				Stream:  present.StreamStderr,
			},
		},
	}
}

// PresentResolutionAmbiguity creates a warning when a sprint name matches
// multiple cached boards during JQL resolution.
func (SprintPresenter) PresentResolutionAmbiguity(sprintName string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageWarning,
				Message: fmt.Sprintf("warning: sprint name %q matched multiple cached boards; falling back to JQL name resolution — results may span sprints on different boards.", sprintName),
				Stream:  present.StreamStderr,
			},
		},
	}
}

// PresentResolutionCacheMiss creates a warning when a sprint name is not found
// in the local cache during JQL resolution.
func (SprintPresenter) PresentResolutionCacheMiss(sprintName string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageWarning,
				Message: fmt.Sprintf("warning: sprint %q not found in cache; falling back to JQL name resolution — Jira will resolve the name or return an empty result set. Run `jtk refresh sprints` to update the cache.", sprintName),
				Stream:  present.StreamStderr,
			},
		},
	}
}

// PresentResolutionError creates a warning when the sprint resolver fails
// due to a non-cache error (network, auth, etc.).
func (SprintPresenter) PresentResolutionError(sprintName string, err error) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageWarning,
				Message: fmt.Sprintf("warning: sprint resolver failed for %q (%v); falling back to JQL name resolution.", sprintName, err),
				Stream:  present.StreamStderr,
			},
		},
	}
}

// PresentResolutionSynthetic creates a warning when the sprint resolver returns
// a synthetic result (no cached ID).
func (SprintPresenter) PresentResolutionSynthetic(sprintName string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageWarning,
				Message: fmt.Sprintf("warning: sprint %q not resolved to a cached ID; falling back to JQL name resolution.", sprintName),
				Stream:  present.StreamStderr,
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
