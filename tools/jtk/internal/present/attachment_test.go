package present

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestAttachmentListSpec_MatchesPresentListHeaders(t *testing.T) {
	t.Parallel()
	attachments := []api.Attachment{{
		ID:       "10234",
		Filename: "test.md",
		Size:     4301,
		MimeType: "text/markdown",
		Created:  "2026-04-16",
		Author:   api.User{DisplayName: "Alice"},
	}}

	for _, extended := range []bool{false, true} {
		name := "default"
		if extended {
			name = "extended"
		}
		t.Run(name, func(t *testing.T) {
			specs := AttachmentListSpec.ForMode(extended)
			model := AttachmentPresenter{}.PresentList(attachments, extended)
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

func TestAttachmentPresenter_PresentList_Extended(t *testing.T) {
	t.Parallel()
	attachments := []api.Attachment{{
		ID:       "10234",
		Filename: "audit-notes.md",
		Size:     4301,
		MimeType: "text/markdown",
		Created:  "2026-04-16T09:00:00+0000",
		Author:   api.User{DisplayName: "Alice"},
	}}

	model := AttachmentPresenter{}.PresentList(attachments, true)
	table := model.Sections[0].(*present.TableSection)

	if table.Rows[0].Cells[5] != "4301" {
		t.Errorf("BYTES: expected '4301', got %q", table.Rows[0].Cells[5])
	}
	if table.Rows[0].Cells[6] != "text/markdown" {
		t.Errorf("MIME_TYPE: expected 'text/markdown', got %q", table.Rows[0].Cells[6])
	}
}
