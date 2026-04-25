package present

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestSprintPresenter_PresentDetail_Default(t *testing.T) {
	t.Parallel()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 14, 0, 0, 0, 0, time.UTC)

	sprint := &api.Sprint{
		ID:            42,
		Name:          "Sprint 1",
		State:         "active",
		Goal:          "Complete MVP",
		StartDate:     &startDate,
		EndDate:       &endDate,
		OriginBoardID: 23,
	}
	board := &api.Board{ID: 23, Name: "MON board"}

	p := SprintPresenter{}
	model := p.PresentDetail(sprint, board, false)

	// Default: title + state/dates + board = 3 sections
	if len(model.Sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(model.Sections))
	}

	// Title line
	titleMsg := model.Sections[0].(*present.MessageSection)
	if titleMsg.Message != "42  Sprint 1" {
		t.Errorf("title: got %q", titleMsg.Message)
	}

	// State line with short dates
	stateMsg := model.Sections[1].(*present.MessageSection)
	if stateMsg.Message != "State: active   Start: 2024-01-01   End: 2024-01-14" {
		t.Errorf("state line: got %q", stateMsg.Message)
	}

	// Board line
	boardMsg := model.Sections[2].(*present.MessageSection)
	if boardMsg.Message != "Board: 23 (MON board)" {
		t.Errorf("board line: got %q", boardMsg.Message)
	}
}

func TestSprintPresenter_PresentDetail_Extended(t *testing.T) {
	t.Parallel()
	startDate := time.Date(2024, 1, 1, 0, 0, 45, 0, time.UTC)
	endDate := time.Date(2024, 1, 14, 23, 30, 0, 0, time.UTC)

	sprint := &api.Sprint{
		ID:            42,
		Name:          "Sprint 1",
		State:         "active",
		Goal:          "Complete MVP",
		StartDate:     &startDate,
		EndDate:       &endDate,
		OriginBoardID: 23,
	}
	board := &api.Board{ID: 23, Name: "MON board"}

	p := SprintPresenter{}
	model := p.PresentDetail(sprint, board, true)

	// Extended: title + state/timestamps + board + goal + origin board = 5 sections
	if len(model.Sections) != 5 {
		t.Fatalf("expected 5 sections, got %d", len(model.Sections))
	}

	// State line with full timestamps
	stateMsg := model.Sections[1].(*present.MessageSection)
	if stateMsg.Message != "State: active   Start: 2024-01-01T00:00:45Z   End: 2024-01-14T23:30:00Z" {
		t.Errorf("state line: got %q", stateMsg.Message)
	}

	// Goal
	goalMsg := model.Sections[3].(*present.MessageSection)
	if goalMsg.Message != "Goal: Complete MVP" {
		t.Errorf("goal: got %q", goalMsg.Message)
	}

	// Origin Board
	originMsg := model.Sections[4].(*present.MessageSection)
	if originMsg.Message != "Origin Board: 23" {
		t.Errorf("origin board: got %q", originMsg.Message)
	}
}

func TestSprintPresenter_PresentDetail_MinimalFields(t *testing.T) {
	t.Parallel()
	sprint := &api.Sprint{
		ID:    1,
		Name:  "Backlog",
		State: "future",
	}
	board := &api.Board{ID: 10}

	p := SprintPresenter{}
	model := p.PresentDetail(sprint, board, false)

	// Title + state/dates + board = 3 sections
	if len(model.Sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(model.Sections))
	}

	stateMsg := model.Sections[1].(*present.MessageSection)
	if stateMsg.Message != "State: future   Start: -   End: -" {
		t.Errorf("state line: got %q", stateMsg.Message)
	}

	// Synthetic board with no name
	boardMsg := model.Sections[2].(*present.MessageSection)
	if boardMsg.Message != "Board: 10" {
		t.Errorf("board line: got %q", boardMsg.Message)
	}
}

func TestSprintPresenter_PresentDetail_ExtendedStableRows(t *testing.T) {
	t.Parallel()
	// Extended output must always have the same row count regardless of data
	sprint := &api.Sprint{
		ID:    1,
		Name:  "Backlog",
		State: "future",
	}
	board := &api.Board{ID: 10}

	p := SprintPresenter{}
	model := p.PresentDetail(sprint, board, true)

	// Extended: title + state/timestamps + board + goal + origin board = 5 sections
	if len(model.Sections) != 5 {
		t.Fatalf("expected 5 sections even with empty goal/origin, got %d", len(model.Sections))
	}
	goalMsg := model.Sections[3].(*present.MessageSection)
	if goalMsg.Message != "Goal: -" {
		t.Errorf("empty goal should show '-': got %q", goalMsg.Message)
	}
	originMsg := model.Sections[4].(*present.MessageSection)
	if originMsg.Message != "Origin Board: -" {
		t.Errorf("empty origin board should show '-': got %q", originMsg.Message)
	}
}

func TestSprintPresenter_PresentList_Default(t *testing.T) {
	t.Parallel()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 14, 0, 0, 0, 0, time.UTC)

	sprints := []api.Sprint{
		{
			ID:        1,
			Name:      "Sprint 1",
			State:     "closed",
			StartDate: &startDate,
			EndDate:   &endDate,
		},
		{
			ID:    2,
			Name:  "Sprint 2",
			State: "future",
		},
	}

	p := SprintPresenter{}
	model := p.PresentList(sprints, false)

	table, ok := model.Sections[0].(*present.TableSection)
	if !ok {
		t.Fatalf("expected TableSection, got %T", model.Sections[0])
	}

	expectedHeaders := []string{"ID", "STATE", "START", "END", "NAME"}
	if len(table.Headers) != len(expectedHeaders) {
		t.Errorf("expected %d headers, got %d", len(expectedHeaders), len(table.Headers))
	}
	for i, h := range expectedHeaders {
		if table.Headers[i] != h {
			t.Errorf("header %d: expected %q, got %q", i, h, table.Headers[i])
		}
	}

	if len(table.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(table.Rows))
	}

	// Row 1 - with dates
	if table.Rows[0].Cells[0] != "1" {
		t.Errorf("row 0 ID: expected '1', got %q", table.Rows[0].Cells[0])
	}
	if table.Rows[0].Cells[2] != "2024-01-01" {
		t.Errorf("row 0 START: expected '2024-01-01', got %q", table.Rows[0].Cells[2])
	}

	// Row 2 - no dates → "-"
	if table.Rows[1].Cells[2] != "-" {
		t.Errorf("row 1 START: expected '-' for nil date, got %q", table.Rows[1].Cells[2])
	}
}

func TestSprintPresenter_PresentList_Extended(t *testing.T) {
	t.Parallel()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 14, 0, 0, 0, 0, time.UTC)

	sprints := []api.Sprint{
		{
			ID:            1,
			Name:          "Sprint 1",
			State:         "active",
			StartDate:     &startDate,
			EndDate:       &endDate,
			OriginBoardID: 23,
			Goal:          "Ship it",
		},
	}

	p := SprintPresenter{}
	model := p.PresentList(sprints, true)

	table := model.Sections[0].(*present.TableSection)

	expectedHeaders := []string{"ID", "STATE", "START", "END", "COMPLETED", "BOARD", "GOAL", "NAME"}
	if len(table.Headers) != len(expectedHeaders) {
		t.Errorf("expected %d headers, got %d", len(expectedHeaders), len(table.Headers))
	}
	for i, h := range expectedHeaders {
		if table.Headers[i] != h {
			t.Errorf("header %d: expected %q, got %q", i, h, table.Headers[i])
		}
	}

	// BOARD column should be OriginBoardID
	if table.Rows[0].Cells[5] != "23" {
		t.Errorf("BOARD: expected '23', got %q", table.Rows[0].Cells[5])
	}
	if table.Rows[0].Cells[6] != "Ship it" {
		t.Errorf("GOAL: expected 'Ship it', got %q", table.Rows[0].Cells[6])
	}
}

func TestFormatBoardRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		board *api.Board
		want  string
	}{
		{"nil board", nil, "-"},
		{"with name", &api.Board{ID: 23, Name: "MON board"}, "23 (MON board)"},
		{"no name", &api.Board{ID: 23}, "23"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBoardRef(tt.board)
			if got != tt.want {
				t.Errorf("formatBoardRef: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSprintPresenter_PresentListWithPagination(t *testing.T) {
	t.Parallel()
	sprints := []api.Sprint{{ID: 1, Name: "S1", State: "active"}}

	t.Run("appends_hint", func(t *testing.T) {
		model := SprintPresenter{}.PresentListWithPagination(sprints, false, true, "tok")
		if len(model.Sections) != 2 {
			t.Fatalf("want 2 sections, got %d", len(model.Sections))
		}
	})

	t.Run("no_hint", func(t *testing.T) {
		model := SprintPresenter{}.PresentListWithPagination(sprints, false, false, "")
		if len(model.Sections) != 1 {
			t.Errorf("want 1 section, got %d", len(model.Sections))
		}
	})
}

func TestSprintPresenter_PresentPostStateUnavailable(t *testing.T) {
	t.Parallel()
	model := SprintPresenter{}.PresentPostStateUnavailable()
	msg := model.Sections[0].(*present.MessageSection)
	if msg.Stream != present.StreamStderr {
		t.Errorf("want StreamStderr, got %v", msg.Stream)
	}
}

func TestSprintPresenter_PresentResolutionAmbiguity(t *testing.T) {
	t.Parallel()
	model := SprintPresenter{}.PresentResolutionAmbiguity("Sprint X")
	msg := model.Sections[0].(*present.MessageSection)
	if msg.Kind != present.MessageWarning {
		t.Errorf("want MessageWarning, got %v", msg.Kind)
	}
	if msg.Stream != present.StreamStderr {
		t.Errorf("want StreamStderr, got %v", msg.Stream)
	}
	if !strings.HasPrefix(msg.Message, "warning: ") {
		t.Errorf("want warning: prefix, got %q", msg.Message)
	}
}

func TestSprintPresenter_PresentResolutionCacheMiss(t *testing.T) {
	t.Parallel()
	model := SprintPresenter{}.PresentResolutionCacheMiss("Sprint X")
	msg := model.Sections[0].(*present.MessageSection)
	if msg.Kind != present.MessageWarning {
		t.Errorf("want MessageWarning, got %v", msg.Kind)
	}
	if !strings.HasPrefix(msg.Message, "warning: ") {
		t.Errorf("want warning: prefix, got %q", msg.Message)
	}
}

func TestSprintPresenter_PresentResolutionError(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("dial tcp: connection refused")
	model := SprintPresenter{}.PresentResolutionError("Sprint X", sentinel)
	msg := model.Sections[0].(*present.MessageSection)
	if msg.Kind != present.MessageWarning {
		t.Errorf("want MessageWarning, got %v", msg.Kind)
	}
	if !strings.HasPrefix(msg.Message, "warning: ") {
		t.Errorf("want warning: prefix, got %q", msg.Message)
	}
	if !strings.Contains(msg.Message, sentinel.Error()) {
		t.Errorf("want error detail in message, got %q", msg.Message)
	}
}

func TestSprintPresenter_PresentResolutionSynthetic(t *testing.T) {
	t.Parallel()
	model := SprintPresenter{}.PresentResolutionSynthetic("Sprint X")
	msg := model.Sections[0].(*present.MessageSection)
	if msg.Kind != present.MessageWarning {
		t.Errorf("want MessageWarning, got %v", msg.Kind)
	}
	if !strings.HasPrefix(msg.Message, "warning: ") {
		t.Errorf("want warning: prefix, got %q", msg.Message)
	}
}
