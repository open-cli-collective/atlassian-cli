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
	expectedHeaders := []string{"ID", "NAME", "TYPE", "CUSTOM"}
	if len(table.Headers) != len(expectedHeaders) {
		t.Fatalf("expected %d headers, got %d", len(expectedHeaders), len(table.Headers))
	}
	for i, h := range expectedHeaders {
		if table.Headers[i] != h {
			t.Errorf("header[%d]: expected %q, got %q", i, h, table.Headers[i])
		}
	}
	if len(table.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(table.Rows))
	}
	if table.Rows[0].Cells[0] != "summary" {
		t.Errorf("row 0 ID: expected 'summary', got %q", table.Rows[0].Cells[0])
	}
	if table.Rows[0].Cells[1] != "Summary" {
		t.Errorf("row 0 Name: expected 'Summary', got %q", table.Rows[0].Cells[1])
	}
	if table.Rows[0].Cells[3] != "no" {
		t.Errorf("row 0 Custom: expected 'no', got %q", table.Rows[0].Cells[3])
	}
	if table.Rows[1].Cells[3] != "yes" {
		t.Errorf("row 1 Custom: expected 'yes', got %q", table.Rows[1].Cells[3])
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
	expectedHeaders := []string{"ID", "NAME", "TYPE", "CUSTOM", "SEARCHABLE", "NAVIGABLE", "ORDERABLE", "CLAUSE_NAMES"}
	if len(table.Headers) != len(expectedHeaders) {
		t.Fatalf("expected %d headers, got %d", len(expectedHeaders), len(table.Headers))
	}
	for i, h := range expectedHeaders {
		if table.Headers[i] != h {
			t.Errorf("header[%d]: expected %q, got %q", i, h, table.Headers[i])
		}
	}

	if table.Rows[0].Cells[3] != "no" {
		t.Errorf("row 0 custom: expected 'no', got %q", table.Rows[0].Cells[3])
	}
	if table.Rows[0].Cells[4] != "yes" {
		t.Errorf("row 0 searchable: expected 'yes', got %q", table.Rows[0].Cells[4])
	}
	if table.Rows[0].Cells[7] != "summary" {
		t.Errorf("row 0 clause_names: expected 'summary', got %q", table.Rows[0].Cells[7])
	}
	if table.Rows[1].Cells[7] != "-" {
		t.Errorf("row 1 clause_names: expected '-' for no clauses, got %q", table.Rows[1].Cells[7])
	}
}
