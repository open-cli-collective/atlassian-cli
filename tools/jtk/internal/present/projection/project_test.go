package projection

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestProjectTable_KeepsSelectedColumns_InUserOrder(t *testing.T) {
	t.Parallel()
	section := &present.TableSection{
		Headers: []string{"KEY", "SUMMARY", "STATUS", "ASSIGNEE", "TYPE"},
		Rows: []present.Row{
			{Cells: []string{"MON-1", "First", "Backlog", "Aaron", "SDLC"}},
			{Cells: []string{"MON-2", "Second", "Done", "Rian", "Kanban"}},
		},
	}
	selected := []ColumnSpec{
		{Header: "KEY", Identity: true},
		{Header: "STATUS", FieldID: "status"},
		{Header: "SUMMARY", FieldID: "summary"},
	}

	out := ProjectTable(section, selected)

	testutil.Equal(t, []string{"KEY", "STATUS", "SUMMARY"}, out.Headers)
	testutil.Equal(t, []string{"MON-1", "Backlog", "First"}, out.Rows[0].Cells)
	testutil.Equal(t, []string{"MON-2", "Done", "Second"}, out.Rows[1].Cells)

	// Input unchanged.
	testutil.Equal(t, 5, len(section.Headers))
	testutil.Equal(t, "SUMMARY", section.Headers[1])
}

func TestProjectTable_SkipsHeadersNotInSelection(t *testing.T) {
	t.Parallel()
	section := &present.TableSection{
		Headers: []string{"KEY", "SUMMARY"},
		Rows:    []present.Row{{Cells: []string{"MON-1", "One"}}},
	}
	selected := []ColumnSpec{
		{Header: "KEY", Identity: true},
		{Header: "STATUS"}, // not in section
	}
	out := ProjectTable(section, selected)
	testutil.Equal(t, []string{"KEY"}, out.Headers)
	testutil.Equal(t, []string{"MON-1"}, out.Rows[0].Cells)
}

func TestProjectDetail_KeepsSelectedFields_CaseInsensitive(t *testing.T) {
	t.Parallel()
	section := &present.DetailSection{
		Fields: []present.Field{
			{Label: "Key", Value: "MON-1"},
			{Label: "Summary", Value: "First"},
			{Label: "Status", Value: "Backlog"},
		},
	}
	selected := []ColumnSpec{
		{Header: "Key", Identity: true},
		{Header: "Status", FieldID: "status"},
	}
	out := ProjectDetail(section, selected)
	testutil.Equal(t, 2, len(out.Fields))
	testutil.Equal(t, "Key", out.Fields[0].Label)
	testutil.Equal(t, "Status", out.Fields[1].Label)
}

func TestDeriveFetchFields_UnionsAndDedups(t *testing.T) {
	t.Parallel()
	selected := []ColumnSpec{
		{Header: "KEY", Identity: true}, // synthetic (no FieldID)
		{Header: "SUMMARY", FieldID: "summary"},
		{Header: "STATUS", FieldID: "status"},
		{Header: "COMPOSITE", FieldID: "", Fetch: []string{"a", "b", "summary"}}, // dup with SUMMARY
	}
	got := DeriveFetchFields(selected)
	testutil.Equal(t, []string{"a", "b", "status", "summary"}, got)
}

func TestDeriveFetchFields_SyntheticColumnsContributeNothing(t *testing.T) {
	t.Parallel()
	selected := []ColumnSpec{
		{Header: "KEY", Identity: true},
		{Header: "URL"}, // synthetic
	}
	got := DeriveFetchFields(selected)
	testutil.Equal(t, 0, len(got))
}
