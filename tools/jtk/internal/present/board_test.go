package present

import (
	"fmt"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestBoardListSpec_HeaderParityWithPresenter(t *testing.T) {
	t.Parallel()
	for _, extended := range []bool{false, true} {
		model := BoardPresenter{}.PresentList(nil, extended)
		table := sectionTable(t, model, 0)
		want := registryHeadersFor(BoardListSpec, extended)
		if !equalStringSlices(table.Headers, want) {
			t.Errorf("extended=%v headers mismatch: presenter %v vs registry %v",
				extended, table.Headers, want)
		}
	}
}

func TestPresentBoardList_DefaultShape(t *testing.T) {
	t.Parallel()
	boards := []api.Board{
		{ID: 23, Name: "MON board", Type: "scrum", Location: api.BoardLocation{ProjectKey: "MON"}},
		{ID: 24, Name: "ON board", Type: "kanban", Location: api.BoardLocation{ProjectKey: "ON"}},
	}
	model := BoardPresenter{}.PresentList(boards, false)
	table := sectionTable(t, model, 0)

	wantHeaders := []string{"ID", "TYPE", "PROJECT", "NAME"}
	if !equalStringSlices(table.Headers, wantHeaders) {
		t.Errorf("headers = %v, want %v", table.Headers, wantHeaders)
	}
	if len(table.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(table.Rows))
	}
	if table.Rows[0].Cells[0] != "23" {
		t.Errorf("row 0 ID: got %q", table.Rows[0].Cells[0])
	}
}

func TestPresentBoardList_ExtendedShape(t *testing.T) {
	t.Parallel()
	boards := []api.Board{
		{ID: 23, Name: "MON board", Type: "scrum", Location: api.BoardLocation{
			ProjectKey: "MON", ProjectName: "Platform Development",
		}},
	}
	model := BoardPresenter{}.PresentList(boards, true)
	table := sectionTable(t, model, 0)

	wantHeaders := []string{"ID", "TYPE", "PROJECT", "PROJECT_NAME", "NAME"}
	if !equalStringSlices(table.Headers, wantHeaders) {
		t.Errorf("extended headers = %v, want %v", table.Headers, wantHeaders)
	}
	if table.Rows[0].Cells[3] != "Platform Development" {
		t.Errorf("PROJECT_NAME: got %q", table.Rows[0].Cells[3])
	}
}

func TestPresentBoardDetail_Default(t *testing.T) {
	t.Parallel()
	board := &api.Board{
		ID: 23, Name: "MON board", Type: "scrum",
		Location: api.BoardLocation{ProjectKey: "MON", ProjectName: "Platform Development"},
	}
	model := BoardPresenter{}.PresentDetail(board, nil, false)

	if len(model.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(model.Sections))
	}
	title := model.Sections[0].(*present.MessageSection)
	if title.Message != "23  MON board" {
		t.Errorf("title: got %q", title.Message)
	}
	typeLine := model.Sections[1].(*present.MessageSection)
	if typeLine.Message != "Type: scrum   Project: MON (Platform Development)" {
		t.Errorf("type line: got %q", typeLine.Message)
	}
}

func TestPresentBoardDetail_Extended(t *testing.T) {
	t.Parallel()
	board := &api.Board{
		ID: 23, Name: "MON board", Type: "scrum",
		Location: api.BoardLocation{ProjectKey: "MON", ProjectName: "Platform Development"},
	}
	config := &api.BoardConfiguration{
		Filter: api.BoardFilter{ID: "10084", Name: "board filter for MON board"},
		ColumnConfig: api.BoardColumnConfig{
			Columns: []api.BoardColumn{
				{Name: "Backlog"},
				{Name: "In Development"},
				{Name: "Deployed"},
			},
		},
	}
	model := BoardPresenter{}.PresentDetail(board, config, true)

	// title + type + filter + column config = 4 sections
	if len(model.Sections) != 4 {
		t.Fatalf("expected 4 sections, got %d", len(model.Sections))
	}
	filterLine := model.Sections[2].(*present.MessageSection)
	if filterLine.Message != "Filter: board filter for MON board (id: 10084)" {
		t.Errorf("filter: got %q", filterLine.Message)
	}
	colLine := model.Sections[3].(*present.MessageSection)
	if colLine.Message != "Column config: Backlog, In Development, Deployed" {
		t.Errorf("columns: got %q", colLine.Message)
	}
}

func TestFormatFilterRef(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		filter api.BoardFilter
		want   string
	}{
		{"name and id", api.BoardFilter{ID: "100", Name: "my filter"}, "my filter (id: 100)"},
		{"empty name", api.BoardFilter{ID: "10084", Name: ""}, "id: 10084"},
		{"empty id", api.BoardFilter{ID: "", Name: "orphan"}, "-"},
		{"both empty", api.BoardFilter{ID: "", Name: ""}, "-"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatFilterRef(tt.filter)
			if got != tt.want {
				t.Errorf("formatFilterRef(%+v) = %q, want %q", tt.filter, got, tt.want)
			}
		})
	}
}

func TestPresentBoardDetail_Extended_EmptyFilterName(t *testing.T) {
	t.Parallel()
	board := &api.Board{
		ID: 23, Name: "MON board", Type: "scrum",
		Location: api.BoardLocation{ProjectKey: "MON", ProjectName: "Platform Development"},
	}
	config := &api.BoardConfiguration{
		Filter: api.BoardFilter{ID: "10084", Name: ""},
		ColumnConfig: api.BoardColumnConfig{
			Columns: []api.BoardColumn{
				{Name: "Backlog"},
				{Name: "Ready for Development"},
				{Name: "In Development"},
			},
		},
	}
	model := BoardPresenter{}.PresentDetail(board, config, true)

	filterLine := model.Sections[2].(*present.MessageSection)
	if filterLine.Message != "Filter: id: 10084" {
		t.Errorf("empty filter name: expected 'Filter: id: 10084', got %q", filterLine.Message)
	}
	colLine := model.Sections[3].(*present.MessageSection)
	if colLine.Message != "Column config: Backlog, Ready for Development, In Development" {
		t.Errorf("columns: got %q", colLine.Message)
	}
}

func TestPresentBoardDetail_ExtendedStableRows_NilConfig(t *testing.T) {
	t.Parallel()
	board := &api.Board{
		ID: 23, Name: "MON board", Type: "scrum",
		Location: api.BoardLocation{ProjectKey: "MON", ProjectName: "Platform Development"},
	}
	// Extended with nil config should still show Filter and Column config rows with "-"
	model := BoardPresenter{}.PresentDetail(board, nil, true)

	// title + type + filter + column config = 4 sections
	if len(model.Sections) != 4 {
		t.Fatalf("expected 4 sections even with nil config, got %d", len(model.Sections))
	}
	filterLine := model.Sections[2].(*present.MessageSection)
	if filterLine.Message != "Filter: -" {
		t.Errorf("nil config filter: got %q", filterLine.Message)
	}
	colLine := model.Sections[3].(*present.MessageSection)
	if colLine.Message != "Column config: -" {
		t.Errorf("nil config columns: got %q", colLine.Message)
	}
}

func TestPresentBoardDetailProjection_ContainsAllSpecHeaders(t *testing.T) {
	t.Parallel()
	board := &api.Board{
		ID: 23, Name: "MON board", Type: "scrum",
		Location: api.BoardLocation{ProjectKey: "MON", ProjectName: "Platform Development"},
	}
	config := &api.BoardConfiguration{
		Filter: api.BoardFilter{ID: "10084", Name: "test filter"},
		ColumnConfig: api.BoardColumnConfig{
			Columns: []api.BoardColumn{{Name: "Col1"}},
		},
	}
	model := BoardPresenter{}.PresentDetailProjection(board, config)
	detail := model.Sections[0].(*present.DetailSection)

	specHeaders := make(map[string]bool)
	for _, c := range BoardDetailSpec {
		specHeaders[c.Header] = false
	}
	for _, f := range detail.Fields {
		if _, ok := specHeaders[f.Label]; ok {
			specHeaders[f.Label] = true
		}
	}
	for h, found := range specHeaders {
		if !found {
			t.Errorf("spec header %q not found in projection detail fields", h)
		}
	}
}

func TestPresentBoardDetailProjection_EmptyFilterName(t *testing.T) {
	t.Parallel()
	board := &api.Board{
		ID: 23, Name: "MON board", Type: "scrum",
		Location: api.BoardLocation{ProjectKey: "MON", ProjectName: "Platform Development"},
	}
	config := &api.BoardConfiguration{
		Filter: api.BoardFilter{ID: "10084", Name: ""},
		ColumnConfig: api.BoardColumnConfig{
			Columns: []api.BoardColumn{{Name: "Backlog"}},
		},
	}
	model := BoardPresenter{}.PresentDetailProjection(board, config)
	detail := model.Sections[0].(*present.DetailSection)

	for _, f := range detail.Fields {
		if f.Label == "FILTER" {
			if f.Value != "id: 10084" {
				t.Errorf("FILTER: expected 'id: 10084', got %q", f.Value)
			}
			return
		}
	}
	t.Error("FILTER field not found in projection output")
}

func TestSprintListSpec_HeaderParityWithPresenter(t *testing.T) {
	t.Parallel()
	for _, extended := range []bool{false, true} {
		model := SprintPresenter{}.PresentList(nil, extended)
		table := sectionTable(t, model, 0)
		want := registryHeadersFor(SprintListSpec, extended)
		if !equalStringSlices(table.Headers, want) {
			t.Errorf("extended=%v headers mismatch: presenter %v vs registry %v",
				extended, table.Headers, want)
		}
	}
}

func TestSprintDetailSpec_ContainsAllProjectionHeaders(t *testing.T) {
	t.Parallel()
	sprint := &api.Sprint{ID: 1, Name: "Test", State: "active", OriginBoardID: 23}
	board := &api.Board{ID: 23, Name: "Test Board"}
	model := SprintPresenter{}.PresentDetailProjection(sprint, board)
	detail := model.Sections[0].(*present.DetailSection)

	specHeaders := make(map[string]bool)
	for _, c := range SprintDetailSpec {
		specHeaders[c.Header] = false
	}
	for _, f := range detail.Fields {
		if _, ok := specHeaders[f.Label]; ok {
			specHeaders[f.Label] = true
		}
	}
	for h, found := range specHeaders {
		if !found {
			t.Errorf("spec header %q not found in projection detail fields", h)
		}
	}
}

func TestBoardPresenter_PresentListWithPagination(t *testing.T) {
	t.Parallel()
	boards := []api.Board{{ID: 1, Name: "B", Type: "scrum"}}

	t.Run("appends_hint", func(t *testing.T) {
		model := BoardPresenter{}.PresentListWithPagination(boards, false, true, "tok")
		if len(model.Sections) != 2 {
			t.Fatalf("want 2 sections, got %d", len(model.Sections))
		}
	})

	t.Run("no_hint", func(t *testing.T) {
		model := BoardPresenter{}.PresentListWithPagination(boards, false, false, "")
		if len(model.Sections) != 1 {
			t.Errorf("want 1 section, got %d", len(model.Sections))
		}
	})
}

func TestBoardPresenter_PresentConfigFetchWarning(t *testing.T) {
	t.Parallel()
	model := BoardPresenter{}.PresentConfigFetchWarning(errTest)
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

var errTest = fmt.Errorf("test error")
