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
	attachments := []api.Attachment{
		{
			ID:       "10234",
			Filename: "audit-notes.md",
			Size:     4301,
			MimeType: "text/markdown",
			Created:  "2026-04-16T09:00:00+0000",
			Author:   api.User{DisplayName: "Alice"},
		},
		{
			ID:       "10235",
			Filename: "mystery.bin",
			Size:     100,
			MimeType: "",
			Created:  "2026-04-16T10:00:00+0000",
			Author:   api.User{DisplayName: "Bob"},
		},
	}

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

	for i, row := range table.Rows {
		if len(row.Cells) != len(expectedHeaders) {
			t.Fatalf("row %d: expected %d cells, got %d", i, len(expectedHeaders), len(row.Cells))
		}
	}

	want0 := []string{"10234", "audit-notes.md", "4.2 KB", "4301", "text/markdown", "Alice", "2026-04-16T09:00:00+0000"}
	for i, w := range want0 {
		if table.Rows[0].Cells[i] != w {
			t.Errorf("row0[%d] (%s): expected %q, got %q", i, expectedHeaders[i], w, table.Rows[0].Cells[i])
		}
	}

	// Row 1: empty MimeType renders as "-"
	if table.Rows[1].Cells[4] != "-" {
		t.Errorf("row1 MIME_TYPE: expected '-' for empty, got %q", table.Rows[1].Cells[4])
	}
}

func TestAttachmentPresenter_PresentList_Default_CellOrder(t *testing.T) {
	t.Parallel()
	attachments := []api.Attachment{{
		ID:       "10234",
		Filename: "audit-notes.md",
		Size:     4301,
		MimeType: "text/markdown",
		Created:  "2026-04-16T09:00:00+0000",
		Author:   api.User{DisplayName: "Alice"},
	}}

	model := AttachmentPresenter{}.PresentList(attachments, false)
	table := model.Sections[0].(*present.TableSection)

	row := table.Rows[0]
	if len(row.Cells) != 5 {
		t.Fatalf("expected 5 cells, got %d", len(row.Cells))
	}
	if row.Cells[0] != "10234" {
		t.Errorf("ID: expected '10234', got %q", row.Cells[0])
	}
	if row.Cells[1] != "audit-notes.md" {
		t.Errorf("FILENAME: expected 'audit-notes.md', got %q", row.Cells[1])
	}
	if row.Cells[3] != "Alice" {
		t.Errorf("AUTHOR: expected 'Alice', got %q", row.Cells[3])
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
