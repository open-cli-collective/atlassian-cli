package present

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestRemoteLinkListSpec_MatchesPresentListHeaders(t *testing.T) {
	t.Parallel()
	links := []api.RemoteLink{{
		ID:           10001,
		Relationship: "mentioned in",
		Object: api.RemoteLinkObject{
			URL:     "https://example.com",
			Title:   "Example",
			Summary: "A summary",
		},
	}}

	for _, extended := range []bool{false, true} {
		name := "default"
		if extended {
			name = "extended"
		}
		t.Run(name, func(t *testing.T) {
			specs := RemoteLinkListSpec.ForMode(extended)
			model := RemoteLinkPresenter{}.PresentList(links, extended)
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

func TestRemoteLinkPresenter_PresentList_Default_CellOrder(t *testing.T) {
	t.Parallel()
	links := []api.RemoteLink{{
		ID: 10001,
		Object: api.RemoteLinkObject{
			URL:   "https://github.com/owner/repo/issues/456",
			Title: "GitHub #456",
		},
	}}

	model := RemoteLinkPresenter{}.PresentList(links, false)
	table := model.Sections[0].(*present.TableSection)

	want := []string{"10001", "GitHub #456", "https://github.com/owner/repo/issues/456"}
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

func TestRemoteLinkPresenter_PresentList_Extended(t *testing.T) {
	t.Parallel()
	links := []api.RemoteLink{
		{
			ID:           10001,
			Relationship: "mentioned in",
			Object: api.RemoteLinkObject{
				URL:     "https://example.com",
				Title:   "Example",
				Summary: "Summary text",
			},
		},
		{
			// No title, relationship, or summary → dash placeholders.
			ID: 10002,
			Object: api.RemoteLinkObject{
				URL: "https://other.example",
			},
		},
	}

	model := RemoteLinkPresenter{}.PresentList(links, true)
	table := model.Sections[0].(*present.TableSection)

	expectedHeaders := []string{"ID", "RELATIONSHIP", "TITLE", "URL", "SUMMARY"}
	if len(table.Headers) != len(expectedHeaders) {
		t.Fatalf("expected %d headers, got %d", len(expectedHeaders), len(table.Headers))
	}
	for i, h := range expectedHeaders {
		if table.Headers[i] != h {
			t.Errorf("header[%d]: expected %q, got %q", i, h, table.Headers[i])
		}
	}

	wantR0 := []string{"10001", "mentioned in", "Example", "https://example.com", "Summary text"}
	for i, w := range wantR0 {
		if table.Rows[0].Cells[i] != w {
			t.Errorf("row0[%d] (%s): expected %q, got %q", i, expectedHeaders[i], w, table.Rows[0].Cells[i])
		}
	}

	wantR1 := []string{"10002", "-", "-", "https://other.example", "-"}
	for i, w := range wantR1 {
		if table.Rows[1].Cells[i] != w {
			t.Errorf("row1[%d] (%s): expected %q, got %q", i, expectedHeaders[i], w, table.Rows[1].Cells[i])
		}
	}
}

func TestRemoteLinkPresenter_PresentAddedDetail(t *testing.T) {
	t.Parallel()
	link := &api.RemoteLink{
		ID:           10010,
		Relationship: "mentioned in",
		Object: api.RemoteLinkObject{
			URL:     "https://example.com",
			Title:   "Example",
			Summary: "Summary",
		},
	}

	model := RemoteLinkPresenter{}.PresentAddedDetail("PROJ-123", link)

	msg := model.Sections[0].(*present.MessageSection)
	if msg.Kind != present.MessageSuccess {
		t.Errorf("want MessageSuccess, got %v", msg.Kind)
	}
	if msg.Stream != present.StreamStdout {
		t.Errorf("want StreamStdout, got %v", msg.Stream)
	}
	if msg.Message != "Added remote link 10010 to PROJ-123" {
		t.Errorf("unexpected message: %q", msg.Message)
	}

	detail := model.Sections[1].(*present.DetailSection)
	got := map[string]string{}
	for _, f := range detail.Fields {
		got[f.Label] = f.Value
	}
	for label, want := range map[string]string{
		"ID":           "10010",
		"Issue":        "PROJ-123",
		"Title":        "Example",
		"URL":          "https://example.com",
		"Relationship": "mentioned in",
		"Summary":      "Summary",
	} {
		if got[label] != want {
			t.Errorf("field %q: expected %q, got %q", label, want, got[label])
		}
	}
}

func TestRemoteLinkPresenter_PresentAddedDetail_OmitsEmptyOptional(t *testing.T) {
	t.Parallel()
	link := &api.RemoteLink{
		ID:     10011,
		Object: api.RemoteLinkObject{URL: "https://example.com", Title: "Example"},
	}

	model := RemoteLinkPresenter{}.PresentAddedDetail("PROJ-123", link)
	detail := model.Sections[1].(*present.DetailSection)
	for _, f := range detail.Fields {
		if f.Label == "Relationship" || f.Label == "Summary" {
			t.Errorf("optional field %q should be omitted when empty", f.Label)
		}
	}
}

func TestRemoteLinkPresenter_PresentDeleted(t *testing.T) {
	t.Parallel()
	model := RemoteLinkPresenter{}.PresentDeleted(10001, "PROJ-123")
	msg := model.Sections[0].(*present.MessageSection)
	if msg.Kind != present.MessageSuccess {
		t.Errorf("want MessageSuccess, got %v", msg.Kind)
	}
	if msg.Message != "Deleted remote link 10001 from PROJ-123" {
		t.Errorf("unexpected message: %q", msg.Message)
	}
}

func TestRemoteLinkPresenter_PresentEmpty(t *testing.T) {
	t.Parallel()
	model := RemoteLinkPresenter{}.PresentEmpty("PROJ-123")
	msg := model.Sections[0].(*present.MessageSection)
	if msg.Kind != present.MessageInfo {
		t.Errorf("want MessageInfo, got %v", msg.Kind)
	}
	if msg.Message != "No remote links on PROJ-123" {
		t.Errorf("unexpected message: %q", msg.Message)
	}
}
