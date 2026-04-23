package present

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestTransitionPresenter_PresentList_Default(t *testing.T) {
	t.Parallel()
	transitions := []api.Transition{
		{ID: "11", Name: "Backlog", To: api.Status{Name: "Backlog"}},
		{ID: "21", Name: "In Progress", To: api.Status{Name: "In Progress"}},
	}

	model := TransitionPresenter{}.PresentList(transitions, false)
	table := model.Sections[0].(*present.TableSection)

	expectedHeaders := []string{"ID", "NAME", "TO_STATUS"}
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
	if table.Rows[0].Cells[0] != "11" {
		t.Errorf("row 0 ID: expected '11', got %q", table.Rows[0].Cells[0])
	}
	if table.Rows[0].Cells[2] != "Backlog" {
		t.Errorf("row 0 TO_STATUS: expected 'Backlog', got %q", table.Rows[0].Cells[2])
	}
}

func TestTransitionPresenter_PresentList_Extended(t *testing.T) {
	t.Parallel()
	transitions := []api.Transition{
		{
			ID:   "71",
			Name: "Deployed",
			To: api.Status{
				Name:           "Deployed",
				StatusCategory: api.StatusCategory{Name: "Done"},
			},
			Fields: map[string]api.TransitionField{
				"resolution": {Required: true, Name: "Resolution"},
			},
		},
		{
			ID:   "11",
			Name: "Backlog",
			To: api.Status{
				Name:           "Backlog",
				StatusCategory: api.StatusCategory{Name: "To Do"},
			},
		},
	}

	model := TransitionPresenter{}.PresentList(transitions, true)
	table := model.Sections[0].(*present.TableSection)

	expectedHeaders := []string{"ID", "NAME", "TO_STATUS", "STATUS_CATEGORY", "REQUIRED_FIELDS"}
	if len(table.Headers) != len(expectedHeaders) {
		t.Fatalf("expected %d headers, got %d", len(expectedHeaders), len(table.Headers))
	}
	for i, h := range expectedHeaders {
		if table.Headers[i] != h {
			t.Errorf("header[%d]: expected %q, got %q", i, h, table.Headers[i])
		}
	}

	if table.Rows[0].Cells[3] != "Done" {
		t.Errorf("row 0 STATUS_CATEGORY: expected 'Done', got %q", table.Rows[0].Cells[3])
	}
	if table.Rows[0].Cells[4] != "Resolution" {
		t.Errorf("row 0 REQUIRED_FIELDS: expected 'Resolution', got %q", table.Rows[0].Cells[4])
	}
	if table.Rows[1].Cells[4] != "-" {
		t.Errorf("row 1 REQUIRED_FIELDS: expected '-', got %q", table.Rows[1].Cells[4])
	}
}

// TestTransitionListSpec_MatchesPresentListHeaders locks the spec against
// PresentList headers for both default and extended modes.
func TestTransitionListSpec_MatchesPresentListHeaders(t *testing.T) {
	t.Parallel()
	transitions := []api.Transition{{ID: "1", Name: "x", To: api.Status{Name: "y"}}}

	for _, extended := range []bool{false, true} {
		name := "default"
		if extended {
			name = "extended"
		}
		t.Run(name, func(t *testing.T) {
			specs := TransitionListSpec.ForMode(extended)
			model := TransitionPresenter{}.PresentList(transitions, extended)
			table := model.Sections[0].(*present.TableSection)

			if len(table.Headers) != len(specs) {
				t.Fatalf("header count mismatch: spec has %d, table has %d", len(specs), len(table.Headers))
			}
			for i, spec := range specs {
				if table.Headers[i] != spec.Header {
					t.Errorf("index %d: spec Header=%q, table header=%q", i, spec.Header, table.Headers[i])
				}
			}
		})
	}
}
