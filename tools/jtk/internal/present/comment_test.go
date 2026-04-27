package present

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func singleComment() []api.Comment {
	return []api.Comment{
		{
			ID:     "42",
			Author: api.User{DisplayName: "Alice"},
			Body: &api.ADFDocument{
				Type:    "doc",
				Version: 1,
				Content: []*api.ADFNode{
					{Type: "paragraph", Content: []*api.ADFNode{{Type: "text", Text: "body text"}}},
				},
			},
			Created: "2024-01-15T10:00:00.000Z",
		},
	}
}

// TestCommentListSpec_MatchesPresentListHeaders locks CommentListSpec against
// the hardcoded headers in PresentList and PresentListWithPagination.
// ProjectTable is header-string-driven; silent drift between the spec and the
// presenter would break --fields at runtime.
func TestCommentListSpec_MatchesPresentListHeaders(t *testing.T) {
	t.Parallel()
	comments := singleComment()

	cases := []struct {
		name  string
		model *present.OutputModel
	}{
		{"PresentList", CommentPresenter{}.PresentList(comments, false)},
		{"PresentListWithPagination_NoMore", CommentPresenter{}.PresentListWithPagination(comments, false, false)},
		{"PresentListWithPagination_HasMore", CommentPresenter{}.PresentListWithPagination(comments, false, true)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var table *present.TableSection
			for _, s := range tc.model.Sections {
				if ts, ok := s.(*present.TableSection); ok {
					table = ts
					break
				}
			}
			if table == nil {
				t.Fatalf("no TableSection in %s output", tc.name)
			}
			defaultSpec := CommentListSpec.ForMode(false)
			if len(table.Headers) != len(defaultSpec) {
				t.Fatalf("header count mismatch: spec has %d, table has %d", len(defaultSpec), len(table.Headers))
			}
			for i, spec := range defaultSpec {
				if table.Headers[i] != spec.Header {
					t.Errorf("index %d: spec Header=%q, table header=%q", i, spec.Header, table.Headers[i])
				}
			}
		})
	}
}

// TestCommentDetailSpec_MatchesPresentDetailLabels locks CommentDetailSpec
// against the Field labels emitted by PresentListFull, both directions:
//   - Every spec entry must appear as a rendered Field label.
//   - Every rendered Field label must have a matching spec entry — otherwise
//     --fields projection would silently drop that field.
//
// Order is checked too: ProjectDetail relies on the spec order being the same
// as the presenter's Field order for deterministic projection output.
func TestCommentDetailSpec_MatchesPresentDetailLabels(t *testing.T) {
	t.Parallel()
	comments := singleComment()

	cases := []struct {
		name  string
		model *present.OutputModel
	}{
		{"PresentListFull", CommentPresenter{}.PresentListFull(comments, false)},
		{"PresentListFullWithPagination_NoMore", CommentPresenter{}.PresentListFullWithPagination(comments, false, false)},
		{"PresentListFullWithPagination_HasMore", CommentPresenter{}.PresentListFullWithPagination(comments, false, true)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var detail *present.DetailSection
			for _, s := range tc.model.Sections {
				if ds, ok := s.(*present.DetailSection); ok {
					detail = ds
					break
				}
			}
			if detail == nil {
				t.Fatalf("no DetailSection in %s output", tc.name)
			}

			activeSpec := CommentDetailSpec.ForMode(false)

			renderedLabels := make(map[string]bool, len(detail.Fields))
			for _, f := range detail.Fields {
				renderedLabels[f.Label] = true
			}
			for _, spec := range activeSpec {
				if !renderedLabels[spec.Header] {
					t.Errorf("spec Header %q not emitted by %s", spec.Header, tc.name)
				}
			}

			specLabels := make(map[string]bool, len(activeSpec))
			for _, spec := range activeSpec {
				specLabels[spec.Header] = true
			}
			for _, f := range detail.Fields {
				if !specLabels[f.Label] {
					t.Errorf("rendered field %q has no matching CommentDetailSpec entry", f.Label)
				}
			}

			specOrder := make([]string, 0, len(activeSpec))
			for _, spec := range activeSpec {
				specOrder = append(specOrder, spec.Header)
			}
			renderedOrder := make([]string, 0, len(detail.Fields))
			for _, f := range detail.Fields {
				renderedOrder = append(renderedOrder, f.Label)
			}
			testutil.Equal(t, len(specOrder), len(renderedOrder))
			for i := range specOrder {
				if specOrder[i] != renderedOrder[i] {
					t.Errorf("order mismatch at index %d: spec=%q rendered=%q", i, specOrder[i], renderedOrder[i])
				}
			}
		})
	}
}

func TestCommentPresenter_PresentList_ExtendedVisibility(t *testing.T) {
	t.Parallel()
	comments := []api.Comment{
		{
			ID:     "100",
			Author: api.User{DisplayName: "Alice"},
			Body: &api.ADFDocument{
				Type: "doc", Version: 1,
				Content: []*api.ADFNode{{Type: "paragraph", Content: []*api.ADFNode{{Type: "text", Text: "public"}}}},
			},
			Created: "2024-01-15T10:00:00.000Z",
		},
		{
			ID:     "101",
			Author: api.User{DisplayName: "Bob"},
			Body: &api.ADFDocument{
				Type: "doc", Version: 1,
				Content: []*api.ADFNode{{Type: "paragraph", Content: []*api.ADFNode{{Type: "text", Text: "restricted"}}}},
			},
			Created:    "2024-01-15T11:00:00.000Z",
			Visibility: &api.CommentVisibility{Type: "role", Value: "Administrators"},
		},
		{
			ID:     "102",
			Author: api.User{DisplayName: "Carol"},
			Body: &api.ADFDocument{
				Type: "doc", Version: 1,
				Content: []*api.ADFNode{{Type: "paragraph", Content: []*api.ADFNode{{Type: "text", Text: "empty vis"}}}},
			},
			Created:    "2024-01-15T12:00:00.000Z",
			Visibility: &api.CommentVisibility{Type: "role", Value: ""},
		},
	}

	model := CommentPresenter{}.PresentList(comments, true)
	table := model.Sections[0].(*present.TableSection)

	expectedHeaders := []string{"ID", "AUTHOR", "CREATED", "UPDATED", "VISIBILITY", "BODY"}
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

	if table.Rows[0].Cells[4] != "-" {
		t.Errorf("row 0 VISIBILITY: expected '-' (nil), got %q", table.Rows[0].Cells[4])
	}
	if table.Rows[1].Cells[4] != "Administrators" {
		t.Errorf("row 1 VISIBILITY: expected 'Administrators', got %q", table.Rows[1].Cells[4])
	}
	if table.Rows[2].Cells[4] != "-" {
		t.Errorf("row 2 VISIBILITY: expected '-' (empty value), got %q", table.Rows[2].Cells[4])
	}
}

func TestCommentPresenter_PresentListFull_ExtendedVisibility(t *testing.T) {
	t.Parallel()
	comments := []api.Comment{
		{
			ID:     "100",
			Author: api.User{DisplayName: "Alice"},
			Body: &api.ADFDocument{
				Type: "doc", Version: 1,
				Content: []*api.ADFNode{{Type: "paragraph", Content: []*api.ADFNode{{Type: "text", Text: "public"}}}},
			},
			Created: "2024-01-15T10:00:00.000Z",
		},
		{
			ID:     "101",
			Author: api.User{DisplayName: "Bob"},
			Body: &api.ADFDocument{
				Type: "doc", Version: 1,
				Content: []*api.ADFNode{{Type: "paragraph", Content: []*api.ADFNode{{Type: "text", Text: "restricted"}}}},
			},
			Created:    "2024-01-15T11:00:00.000Z",
			Visibility: &api.CommentVisibility{Type: "role", Value: "Administrators"},
		},
	}

	model := CommentPresenter{}.PresentListFull(comments, true)

	ds0 := model.Sections[0].(*present.DetailSection)
	ds1 := model.Sections[1].(*present.DetailSection)

	var vis0, vis1 string
	for _, f := range ds0.Fields {
		if f.Label == "Visibility" {
			vis0 = f.Value
		}
	}
	for _, f := range ds1.Fields {
		if f.Label == "Visibility" {
			vis1 = f.Value
		}
	}
	if vis0 != "-" {
		t.Errorf("comment 0 Visibility: expected '-', got %q", vis0)
	}
	if vis1 != "Administrators" {
		t.Errorf("comment 1 Visibility: expected 'Administrators', got %q", vis1)
	}
}

func TestCommentListSpec_ExtendedMatchesPresentListHeaders(t *testing.T) {
	t.Parallel()
	comments := singleComment()

	extendedSpec := CommentListSpec.ForMode(true)
	model := CommentPresenter{}.PresentList(comments, true)

	var table *present.TableSection
	for _, s := range model.Sections {
		if ts, ok := s.(*present.TableSection); ok {
			table = ts
			break
		}
	}
	if table == nil {
		t.Fatal("no TableSection in extended PresentList output")
	}
	if len(table.Headers) != len(extendedSpec) {
		t.Fatalf("header count mismatch: spec has %d, table has %d", len(extendedSpec), len(table.Headers))
	}
	for i, spec := range extendedSpec {
		if table.Headers[i] != spec.Header {
			t.Errorf("index %d: spec Header=%q, table header=%q", i, spec.Header, table.Headers[i])
		}
	}
}

func TestCommentDetailSpec_ExtendedMatchesPresentDetailLabels(t *testing.T) {
	t.Parallel()
	comments := singleComment()

	extendedSpec := CommentDetailSpec.ForMode(true)
	model := CommentPresenter{}.PresentListFull(comments, true)

	var detail *present.DetailSection
	for _, s := range model.Sections {
		if ds, ok := s.(*present.DetailSection); ok {
			detail = ds
			break
		}
	}
	if detail == nil {
		t.Fatal("no DetailSection in extended PresentListFull output")
	}

	renderedLabels := make(map[string]bool, len(detail.Fields))
	for _, f := range detail.Fields {
		renderedLabels[f.Label] = true
	}
	for _, spec := range extendedSpec {
		if !renderedLabels[spec.Header] {
			t.Errorf("spec Header %q not emitted in extended mode", spec.Header)
		}
	}

	specLabels := make(map[string]bool, len(extendedSpec))
	for _, spec := range extendedSpec {
		specLabels[spec.Header] = true
	}
	mismatch := false
	for _, f := range detail.Fields {
		if !specLabels[f.Label] {
			t.Errorf("rendered field %q has no matching CommentDetailSpec entry in extended mode", f.Label)
			mismatch = true
		}
	}
	if mismatch {
		t.Fatal("reverse-direction check failed; skipping order check")
	}

	specOrder := make([]string, 0, len(extendedSpec))
	for _, spec := range extendedSpec {
		specOrder = append(specOrder, spec.Header)
	}
	renderedOrder := make([]string, 0, len(detail.Fields))
	for _, f := range detail.Fields {
		renderedOrder = append(renderedOrder, f.Label)
	}
	if len(specOrder) != len(renderedOrder) {
		t.Fatalf("extended spec has %d entries, rendered has %d", len(specOrder), len(renderedOrder))
	}
	for i := range specOrder {
		if specOrder[i] != renderedOrder[i] {
			t.Errorf("extended order mismatch at index %d: spec=%q rendered=%q", i, specOrder[i], renderedOrder[i])
		}
	}
}
