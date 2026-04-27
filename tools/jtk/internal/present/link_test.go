package present

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestLinkListSpec_MatchesPresentListHeaders(t *testing.T) {
	t.Parallel()
	links := []api.IssueLink{{
		ID:   "1",
		Type: api.IssueLinkType{ID: "10", Name: "Blocker", Inward: "is blocked by", Outward: "blocks"},
		OutwardIssue: &api.LinkedIssue{
			Key: "PROJ-2",
			Fields: struct {
				Summary   string         `json:"summary"`
				Status    *api.Status    `json:"status,omitempty"`
				IssueType *api.IssueType `json:"issuetype,omitempty"`
			}{Summary: "Target", Status: &api.Status{Name: "Open"}},
		},
	}}

	for _, extended := range []bool{false, true} {
		name := "default"
		if extended {
			name = "extended"
		}
		t.Run(name, func(t *testing.T) {
			specs := LinkListSpec.ForMode(extended)
			model := LinkPresenter{}.PresentList(links, extended)
			table := model.Sections[0].(*present.TableSection)

			if len(table.Headers) != len(specs) {
				t.Fatalf("header count mismatch: spec has %d, table has %d", len(specs), len(table.Headers))
			}
			for i, spec := range specs {
				if table.Headers[i] != spec.Header {
					t.Errorf("index %d: spec=%q, table=%q", i, spec.Header, table.Headers[i])
				}
			}
		})
	}
}

func TestLinkTypesSpec_MatchesPresentTypesHeaders(t *testing.T) {
	t.Parallel()
	types := []api.IssueLinkType{{ID: "1", Name: "Blocker", Inward: "is blocked by", Outward: "blocks"}}

	specs := LinkTypesSpec.ForMode(false)
	model := LinkPresenter{}.PresentTypes(types)
	table := model.Sections[0].(*present.TableSection)

	if len(table.Headers) != len(specs) {
		t.Fatalf("header count mismatch: spec has %d, table has %d", len(specs), len(table.Headers))
	}
	for i, spec := range specs {
		if table.Headers[i] != spec.Header {
			t.Errorf("index %d: spec=%q, table=%q", i, spec.Header, table.Headers[i])
		}
	}
}

func TestLinkPresenter_PresentList_Extended(t *testing.T) {
	t.Parallel()
	links := []api.IssueLink{{
		ID:   "17844",
		Type: api.IssueLinkType{ID: "10100", Name: "Blocker", Inward: "is blocked by", Outward: "blocks"},
		OutwardIssue: &api.LinkedIssue{
			Key: "MON-4819",
			Fields: struct {
				Summary   string         `json:"summary"`
				Status    *api.Status    `json:"status,omitempty"`
				IssueType *api.IssueType `json:"issuetype,omitempty"`
			}{Summary: "Linked issue B", Status: &api.Status{Name: "Backlog"}},
		},
	}}

	model := LinkPresenter{}.PresentList(links, true)
	table := model.Sections[0].(*present.TableSection)

	expectedHeaders := []string{"LINK_ID", "TYPE_ID", "TYPE", "DIRECTION", "ISSUE", "STATUS", "SUMMARY"}
	if len(table.Headers) != len(expectedHeaders) {
		t.Fatalf("expected %d headers, got %d", len(expectedHeaders), len(table.Headers))
	}
	for i, h := range expectedHeaders {
		if table.Headers[i] != h {
			t.Errorf("header[%d]: expected %q, got %q", i, h, table.Headers[i])
		}
	}

	row := table.Rows[0]
	if len(row.Cells) != len(expectedHeaders) {
		t.Fatalf("expected %d cells, got %d", len(expectedHeaders), len(row.Cells))
	}
	if row.Cells[0] != "17844" {
		t.Errorf("LINK_ID: expected '17844', got %q", row.Cells[0])
	}
	if row.Cells[1] != "10100" {
		t.Errorf("TYPE_ID: expected '10100', got %q", row.Cells[1])
	}
	if row.Cells[5] != "Backlog" {
		t.Errorf("STATUS: expected 'Backlog', got %q", row.Cells[5])
	}
	if row.Cells[6] != "Linked issue B" {
		t.Errorf("SUMMARY: expected 'Linked issue B', got %q", row.Cells[6])
	}
}

func TestLinkPresenter_PresentIDUnavailable(t *testing.T) {
	t.Parallel()
	model := LinkPresenter{}.PresentIDUnavailable()
	msg := model.Sections[0].(*present.MessageSection)
	if msg.Stream != present.StreamStderr {
		t.Errorf("want StreamStderr, got %v", msg.Stream)
	}
	if msg.Message != "link ID unavailable — re-query failed" {
		t.Errorf("unexpected message: %q", msg.Message)
	}
}

func TestLinkPresenter_PresentPostStateUnavailable(t *testing.T) {
	t.Parallel()
	model := LinkPresenter{}.PresentPostStateUnavailable()
	msg := model.Sections[0].(*present.MessageSection)
	if msg.Stream != present.StreamStderr {
		t.Errorf("want StreamStderr, got %v", msg.Stream)
	}
	if msg.Message != "post-state unavailable; showing confirmation only" {
		t.Errorf("unexpected message: %q", msg.Message)
	}
}
