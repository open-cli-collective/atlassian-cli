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
			ID:        "71",
			Name:      "Deployed",
			HasScreen: true,
			To: api.Status{
				Name:           "Deployed",
				StatusCategory: api.StatusCategory{Name: "Done"},
			},
			Fields: map[string]api.TransitionField{
				"resolution": {Required: true, Name: "Resolution"},
			},
		},
		{
			ID:            "11",
			Name:          "Backlog",
			IsConditional: true,
			To: api.Status{
				Name:           "Backlog",
				StatusCategory: api.StatusCategory{Name: "To Do"},
			},
		},
		{
			ID:            "41",
			Name:          "In Review",
			HasScreen:     true,
			IsConditional: true,
			To: api.Status{
				Name:           "In Review",
				StatusCategory: api.StatusCategory{Name: "In Progress"},
			},
		},
	}

	model := TransitionPresenter{}.PresentList(transitions, true)
	table := model.Sections[0].(*present.TableSection)

	expectedHeaders := []string{"ID", "NAME", "TO_STATUS", "STATUS_CATEGORY", "HAS_SCREEN", "CONDITIONAL", "REQUIRED_FIELDS"}
	if len(table.Headers) != len(expectedHeaders) {
		t.Fatalf("expected %d headers, got %d", len(expectedHeaders), len(table.Headers))
	}
	for i, h := range expectedHeaders {
		if table.Headers[i] != h {
			t.Errorf("header[%d]: expected %q, got %q", i, h, table.Headers[i])
		}
	}

	for i, row := range table.Rows {
		if len(row.Cells) != len(expectedHeaders) {
			t.Fatalf("row %d: expected %d cells, got %d", i, len(expectedHeaders), len(row.Cells))
		}
	}

	if table.Rows[0].Cells[3] != "Done" {
		t.Errorf("row 0 STATUS_CATEGORY: expected 'Done', got %q", table.Rows[0].Cells[3])
	}
	if table.Rows[0].Cells[4] != "yes" {
		t.Errorf("row 0 HAS_SCREEN: expected 'yes', got %q", table.Rows[0].Cells[4])
	}
	if table.Rows[0].Cells[5] != "no" {
		t.Errorf("row 0 CONDITIONAL: expected 'no', got %q", table.Rows[0].Cells[5])
	}
	if table.Rows[0].Cells[6] != "Resolution" {
		t.Errorf("row 0 REQUIRED_FIELDS: expected 'Resolution', got %q", table.Rows[0].Cells[6])
	}
	if table.Rows[1].Cells[4] != "no" {
		t.Errorf("row 1 HAS_SCREEN: expected 'no', got %q", table.Rows[1].Cells[4])
	}
	if table.Rows[1].Cells[5] != "yes" {
		t.Errorf("row 1 CONDITIONAL: expected 'yes', got %q", table.Rows[1].Cells[5])
	}
	if table.Rows[1].Cells[6] != "-" {
		t.Errorf("row 1 REQUIRED_FIELDS: expected '-', got %q", table.Rows[1].Cells[6])
	}
	if table.Rows[2].Cells[4] != "yes" {
		t.Errorf("row 2 HAS_SCREEN: expected 'yes', got %q", table.Rows[2].Cells[4])
	}
	if table.Rows[2].Cells[5] != "yes" {
		t.Errorf("row 2 CONDITIONAL: expected 'yes', got %q", table.Rows[2].Cells[5])
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

func TestTransitionPresenter_PresentStatusNotFound(t *testing.T) {
	t.Parallel()
	transitions := []api.Transition{
		{ID: "11", Name: "Start", To: api.Status{Name: "In Progress"}},
		{ID: "21", Name: "Resume", To: api.Status{Name: "In Progress"}},
		{ID: "31", Name: "Complete", To: api.Status{Name: "Done"}},
	}
	model := TransitionPresenter{}.PresentStatusNotFound("Bogus", transitions)

	first := model.Sections[0].(*present.MessageSection)
	if first.Kind != present.MessageError {
		t.Errorf("want MessageError, got %v", first.Kind)
	}
	if first.Stream != present.StreamStderr {
		t.Errorf("want StreamStderr, got %v", first.Stream)
	}

	// Expect dedup: In Progress appears once, Done once -> header + 2 status lines = 4 sections.
	if len(model.Sections) != 4 {
		t.Fatalf("want 4 sections (error, header, 2 deduped statuses), got %d", len(model.Sections))
	}

	statuses := []string{
		model.Sections[2].(*present.MessageSection).Message,
		model.Sections[3].(*present.MessageSection).Message,
	}
	for _, want := range []string{"In Progress", "Done"} {
		found := false
		for _, s := range statuses {
			if contains(s, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected status %q in output, got %v", want, statuses)
		}
	}
}

func TestTransitionPresenter_PresentStatusAmbiguous(t *testing.T) {
	t.Parallel()
	candidates := []api.Transition{
		{ID: "31", Name: "Resolve", To: api.Status{Name: "Done"}},
		{ID: "41", Name: "Close", To: api.Status{Name: "Done"}},
	}
	model := TransitionPresenter{}.PresentStatusAmbiguous("PROJ-123", "Done", candidates)

	first := model.Sections[0].(*present.MessageSection)
	if first.Kind != present.MessageError {
		t.Errorf("want MessageError, got %v", first.Kind)
	}
	last := model.Sections[len(model.Sections)-1].(*present.MessageSection)
	if !contains(last.Message, "jtk transitions do PROJ-123") {
		t.Errorf("expected recommendation with issue key, got %q", last.Message)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
