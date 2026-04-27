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

	expectedHeaders := []string{"ID", "FILENAME", "SIZE", "BYTES", "MIME_TYPE", "AUTHOR", "CREATED"}
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
	want := []string{"10234", "audit-notes.md", "4.2 KB", "4301", "text/markdown", "Alice", "2026-04-16T09:00:00+0000"}
	for i, w := range want {
		if row.Cells[i] != w {
			t.Errorf("cell[%d] (%s): expected %q, got %q", i, expectedHeaders[i], w, row.Cells[i])
		}
	}
}

func TestAttachmentPresenter_PresentDownloaded(t *testing.T) {
	t.Parallel()
	model := AttachmentPresenter{}.PresentDownloaded("10234", "./audit.md", 4301)
	msg := model.Sections[0].(*present.MessageSection)
	if msg.Kind != present.MessageSuccess {
		t.Errorf("want MessageSuccess, got %v", msg.Kind)
	}
	if msg.Stream != present.StreamStdout {
		t.Errorf("want StreamStdout, got %v", msg.Stream)
	}
	if msg.Message != "Downloaded 10234 → ./audit.md (4.2 KB)" {
		t.Errorf("unexpected message: %q", msg.Message)
	}
}
