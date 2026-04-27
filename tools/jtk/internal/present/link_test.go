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
	links := []api.IssueLink{
		{
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
		},
		{
			ID:   "17845",
			Type: api.IssueLinkType{ID: "10200", Name: "Relates", Inward: "relates to", Outward: "relates to"},
			InwardIssue: &api.LinkedIssue{
				Key: "MON-4700",
				Fields: struct {
					Summary   string         `json:"summary"`
					Status    *api.Status    `json:"status,omitempty"`
					IssueType *api.IssueType `json:"issuetype,omitempty"`
				}{Summary: "Fix ghost row"},
			},
		},
	}

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

	for i, row := range table.Rows {
		if len(row.Cells) != len(expectedHeaders) {
			t.Fatalf("row %d: expected %d cells, got %d", i, len(expectedHeaders), len(row.Cells))
		}
	}

	// Row 0: OutwardIssue with status
	r0 := table.Rows[0].Cells
	wantR0 := []string{"17844", "10100", "Blocker", "is blocked by", "MON-4819", "Backlog", "Linked issue B"}
	for i, w := range wantR0 {
		if r0[i] != w {
			t.Errorf("row0[%d] (%s): expected %q, got %q", i, expectedHeaders[i], w, r0[i])
		}
	}

	// Row 1: InwardIssue with nil status
	r1 := table.Rows[1].Cells
	wantR1 := []string{"17845", "10200", "Relates", "relates to", "MON-4700", "-", "Fix ghost row"}
	for i, w := range wantR1 {
		if r1[i] != w {
			t.Errorf("row1[%d] (%s): expected %q, got %q", i, expectedHeaders[i], w, r1[i])
		}
	}
}

func TestLinkPresenter_PresentList_Default_CellOrder(t *testing.T) {
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

	model := LinkPresenter{}.PresentList(links, false)
	table := model.Sections[0].(*present.TableSection)

	want := []string{"17844", "Blocker", "is blocked by", "MON-4819", "Linked issue B"}
	row := table.Rows[0].Cells
	if len(row) != len(want) {
		t.Fatalf("expected %d cells, got %d", len(want), len(row))
	}
	for i, w := range want {
		if row[i] != w {
			t.Errorf("cell[%d]: expected %q, got %q", i, w, row[i])
		}
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
