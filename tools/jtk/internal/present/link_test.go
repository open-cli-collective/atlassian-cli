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

	if table.Rows[0].Cells[5] != "10100" {
		t.Errorf("TYPE_ID: expected '10100', got %q", table.Rows[0].Cells[5])
	}
	if table.Rows[0].Cells[6] != "Backlog" {
		t.Errorf("STATUS: expected 'Backlog', got %q", table.Rows[0].Cells[6])
	}
}
