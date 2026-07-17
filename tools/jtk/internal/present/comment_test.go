package present

import (
	"reflect"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestCommentPresenter_FullTextShapeStable(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("body ", 30)
	comments := []api.Comment{{
		ID: "42", Author: api.User{DisplayName: "Alice"}, Created: "2024-01-15T10:00:00.000Z",
		Body: &api.ADFDocument{Type: "doc", Version: 1, Content: []*api.ADFNode{
			{Type: "paragraph", Content: []*api.ADFNode{{Type: "text", Text: long}}},
			{Type: "paragraph", Content: []*api.ADFNode{{Type: "text", Text: "second line"}}},
		}},
	}}

	compact := CommentPresenter{}.PresentList(comments, false, false).Sections[0].(*present.TableSection)
	full := CommentPresenter{}.PresentList(comments, false, true).Sections[0].(*present.TableSection)
	if !reflect.DeepEqual(compact.Headers, full.Headers) || !reflect.DeepEqual(compact.Rows[0].Cells[:3], full.Rows[0].Cells[:3]) {
		t.Fatalf("--fulltext changed table shape: compact=%#v full=%#v", compact, full)
	}
	if !strings.HasSuffix(compact.Rows[0].Cells[3], "...") {
		t.Fatalf("default body was not truncated: %q", compact.Rows[0].Cells[3])
	}
	if !strings.Contains(full.Rows[0].Cells[3], long) || !strings.Contains(full.Rows[0].Cells[3], "second line") {
		t.Fatalf("fulltext did not preserve multiline body: %q", full.Rows[0].Cells[3])
	}
}

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
		{"PresentList", CommentPresenter{}.PresentList(comments, false, false)},
		{"PresentListWithPagination_NoMore", CommentPresenter{}.PresentListWithPagination(comments, false, false, false, "token")},
		{"PresentListWithPagination_HasMore", CommentPresenter{}.PresentListWithPagination(comments, false, false, true, "token")},
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

	model := CommentPresenter{}.PresentList(comments, true, false)
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

func TestCommentListSpec_ExtendedMatchesPresentListHeaders(t *testing.T) {
	t.Parallel()
	comments := singleComment()

	extendedSpec := CommentListSpec.ForMode(true)
	model := CommentPresenter{}.PresentList(comments, true, false)

	var table *present.TableSection
	for _, s := range model.Sections {
		if ts, ok := s.(*present.TableSection); ok {
			table = ts
			break
		}
	}
	if table == nil {
		t.Fatal("no TableSection in includeOptional PresentList output")
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
