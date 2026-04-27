package present

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestFieldPresenter_PresentList_Default(t *testing.T) {
	t.Parallel()
	fields := []api.Field{
		{ID: "summary", Name: "Summary", Schema: api.FieldSchema{Type: "string"}, Searchable: true},
		{ID: "customfield_10035", Name: "Story Points", Schema: api.FieldSchema{Type: "number"}, Custom: true},
	}

	model := FieldPresenter{}.PresentList(fields, false)

	table := model.Sections[0].(*present.TableSection)
	wantHeaders := []string{"ID", "TYPE", "NAME"}
	if len(table.Headers) != len(wantHeaders) {
		t.Fatalf("expected %d headers, got %d", len(wantHeaders), len(table.Headers))
	}
	for i, h := range wantHeaders {
		if table.Headers[i] != h {
			t.Errorf("header[%d] = %q, want %q", i, table.Headers[i], h)
		}
	}
	if len(table.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(table.Rows))
	}
	if table.Rows[0].Cells[0] != "summary" {
		t.Errorf("row 0 cell 0 (ID) = %q, want %q", table.Rows[0].Cells[0], "summary")
	}
	if table.Rows[0].Cells[1] != "string" {
		t.Errorf("row 0 cell 1 (TYPE) = %q, want %q", table.Rows[0].Cells[1], "string")
	}
	if table.Rows[0].Cells[2] != "Summary" {
		t.Errorf("row 0 cell 2 (NAME) = %q, want %q", table.Rows[0].Cells[2], "Summary")
	}
}

func TestFieldPresenter_PresentList_Extended(t *testing.T) {
	t.Parallel()
	fields := []api.Field{
		{
			ID:          "summary",
			Name:        "Summary",
			Schema:      api.FieldSchema{Type: "string"},
			Searchable:  true,
			Navigable:   true,
			Orderable:   true,
			ClauseNames: []string{"summary"},
		},
		{
			ID:     "customfield_10035",
			Name:   "Story Points",
			Schema: api.FieldSchema{Type: "number"},
			Custom: true,
		},
	}

	model := FieldPresenter{}.PresentList(fields, true)

	table := model.Sections[0].(*present.TableSection)
	wantHeaders := []string{"ID", "TYPE", "SEARCHABLE", "NAVIGABLE", "ORDERABLE", "CLAUSE_NAMES", "NAME"}
	if len(table.Headers) != len(wantHeaders) {
		t.Fatalf("expected %d headers, got %d", len(wantHeaders), len(table.Headers))
	}
	for i, h := range wantHeaders {
		if table.Headers[i] != h {
			t.Errorf("header[%d] = %q, want %q", i, table.Headers[i], h)
		}
	}

	// Row 0: summary
	if table.Rows[0].Cells[0] != "summary" {
		t.Errorf("row 0 cell 0 (ID) = %q, want %q", table.Rows[0].Cells[0], "summary")
	}
	if table.Rows[0].Cells[1] != "string" {
		t.Errorf("row 0 cell 1 (TYPE) = %q, want %q", table.Rows[0].Cells[1], "string")
	}
	if table.Rows[0].Cells[2] != "yes" {
		t.Errorf("row 0 cell 2 (SEARCHABLE) = %q, want %q", table.Rows[0].Cells[2], "yes")
	}
	if table.Rows[0].Cells[3] != "yes" {
		t.Errorf("row 0 cell 3 (NAVIGABLE) = %q, want %q", table.Rows[0].Cells[3], "yes")
	}
	if table.Rows[0].Cells[4] != "yes" {
		t.Errorf("row 0 cell 4 (ORDERABLE) = %q, want %q", table.Rows[0].Cells[4], "yes")
	}
	if table.Rows[0].Cells[5] != "summary" {
		t.Errorf("row 0 cell 5 (CLAUSE_NAMES) = %q, want %q", table.Rows[0].Cells[5], "summary")
	}
	if table.Rows[0].Cells[6] != "Summary" {
		t.Errorf("row 0 cell 6 (NAME) = %q, want %q", table.Rows[0].Cells[6], "Summary")
	}

	// Row 1: Story Points (no clause names → dash)
	if table.Rows[1].Cells[5] != "-" {
		t.Errorf("row 1 cell 5 (CLAUSE_NAMES) = %q, want %q", table.Rows[1].Cells[5], "-")
	}
}

func TestFieldPresenter_PresentFieldShow(t *testing.T) {
	t.Parallel()
	rows := []FieldShowRow{
		{ContextID: "10100", Context: "Default Context", Projects: "(global)", OptionID: "20001", OptionValue: "Platform"},
		{ContextID: "10100", Context: "Default Context", Projects: "(global)", OptionID: "20002", OptionValue: "Integration"},
		{ContextID: "10101", Context: "MON Project Context", Projects: "10001", OptionID: "20010", OptionValue: "CapOne"},
		{ContextID: "10102", Context: "ON Project Context", Projects: "10002", OptionID: "-", OptionValue: "-"},
	}

	model := FieldPresenter{}.PresentFieldShow(rows)

	table := model.Sections[0].(*present.TableSection)
	wantHeaders := []string{"CONTEXT_ID", "CONTEXT", "PROJECTS", "OPTION_ID", "OPTION_VALUE"}
	for i, h := range wantHeaders {
		if table.Headers[i] != h {
			t.Errorf("header[%d] = %q, want %q", i, table.Headers[i], h)
		}
	}

	if len(table.Rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(table.Rows))
	}
	if table.Rows[0].Cells[2] != "(global)" {
		t.Errorf("row 0 PROJECTS = %q, want %q", table.Rows[0].Cells[2], "(global)")
	}
	if table.Rows[3].Cells[3] != "-" {
		t.Errorf("row 3 OPTION_ID = %q, want %q", table.Rows[3].Cells[3], "-")
	}
	if table.Rows[3].Cells[4] != "-" {
		t.Errorf("row 3 OPTION_VALUE = %q, want %q", table.Rows[3].Cells[4], "-")
	}
}
