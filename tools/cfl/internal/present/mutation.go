package present

import (
	"fmt"
	"strings"

	sharedpresent "github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/confluence-cli/api"
)

func (SpacePresenter) PresentCreate(space *api.Space, baseURL string) *sharedpresent.OutputModel {
	fields := []sharedpresent.Field{{Label: "Key", Value: orDash(space.Key)}}
	if space.Links.WebUI != "" {
		fields = append(fields, sharedpresent.Field{
			Label: "URL",
			Value: baseURL + space.Links.WebUI,
		})
	}
	return successWithFields(fmt.Sprintf("Created space: %s", orDash(space.Name)), fields...)
}

func (SpacePresenter) PresentUpdate(space *api.Space) *sharedpresent.OutputModel {
	return successMessage(fmt.Sprintf("Updated space: %s (%s)", orDash(space.Name), orDash(space.Key)))
}

func (SpacePresenter) PresentDelete(space *api.Space) *sharedpresent.OutputModel {
	return successMessage(fmt.Sprintf("Deleted space: %s (%s)", orDash(space.Name), orDash(space.Key)))
}

func (PagePresenter) PresentCreate(page *api.Page, baseURL string) *sharedpresent.OutputModel {
	return successWithFields(
		fmt.Sprintf("Created page: %s", orDash(page.Title)),
		sharedpresent.Field{Label: "ID", Value: orDash(page.ID)},
		sharedpresent.Field{Label: "URL", Value: baseURL + page.Links.WebUI},
	)
}

func (PagePresenter) PresentEdit(page *api.Page, baseURL string, showLegacyWarning bool) *sharedpresent.OutputModel {
	sections := make([]sharedpresent.Section, 0, 3)
	if showLegacyWarning {
		sections = append(sections, &sharedpresent.MessageSection{
			Kind:    sharedpresent.MessageWarning,
			Message: "Using --legacy flag. If this page uses the cloud editor, it may switch to the legacy editor.",
			Stream:  sharedpresent.StreamStderr,
		})
	}
	sections = append(sections, successSection(fmt.Sprintf("Updated page: %s", orDash(page.Title))))
	sections = append(sections, &sharedpresent.DetailSection{Fields: []sharedpresent.Field{
		{Label: "ID", Value: orDash(page.ID)},
		{Label: "Version", Value: pageVersionValue(page.Version)},
		{Label: "URL", Value: baseURL + page.Links.WebUI},
	}})
	return &sharedpresent.OutputModel{Sections: sections}
}

func (PagePresenter) PresentCopy(page *api.Page) *sharedpresent.OutputModel {
	fields := []sharedpresent.Field{
		{Label: "ID", Value: orDash(page.ID)},
		{Label: "Space", Value: orDash(page.SpaceID)},
	}
	if page.Version != nil {
		fields = append(fields, sharedpresent.Field{
			Label: "Version",
			Value: pageVersionValue(page.Version),
		})
	}
	return successWithFields(fmt.Sprintf("Copied page: %s", orDash(page.Title)), fields...)
}

func (PagePresenter) PresentDelete(page *api.Page) *sharedpresent.OutputModel {
	return successMessage(fmt.Sprintf("Deleted page: %s (ID: %s)", orDash(page.Title), orDash(page.ID)))
}

func (AttachmentPresenter) PresentUpload(filename string, attachment *api.Attachment, sizeBytes int64) *sharedpresent.OutputModel {
	return successWithFields(
		fmt.Sprintf("Uploaded: %s", filename),
		sharedpresent.Field{Label: "ID", Value: orDash(attachment.ID)},
		sharedpresent.Field{Label: "Title", Value: orDash(attachment.Title)},
		sharedpresent.Field{Label: "Size", Value: formatAttachmentFileSize(sizeBytes)},
	)
}

func (AttachmentPresenter) PresentDownload(outputPath string, sizeBytes int64) *sharedpresent.OutputModel {
	return successWithFields(
		fmt.Sprintf("Downloaded: %s", outputPath),
		sharedpresent.Field{Label: "Size", Value: formatAttachmentFileSize(sizeBytes)},
	)
}

func (AttachmentPresenter) PresentDelete(attachment *api.Attachment) *sharedpresent.OutputModel {
	return successMessage(fmt.Sprintf("Deleted attachment: %s (ID: %s)", orDash(attachment.Title), orDash(attachment.ID)))
}

func PresentDeletionCancelled() *sharedpresent.OutputModel {
	return &sharedpresent.OutputModel{Sections: []sharedpresent.Section{stderrInfo("Deletion cancelled.")}}
}

func successWithFields(summary string, fields ...sharedpresent.Field) *sharedpresent.OutputModel {
	return &sharedpresent.OutputModel{
		Sections: []sharedpresent.Section{
			successSection(summary),
			&sharedpresent.DetailSection{Fields: fields},
		},
	}
}

func successMessage(summary string) *sharedpresent.OutputModel {
	return &sharedpresent.OutputModel{Sections: []sharedpresent.Section{successSection(summary)}}
}

func successSection(summary string) *sharedpresent.MessageSection {
	return &sharedpresent.MessageSection{
		Kind:    sharedpresent.MessageInfo,
		Message: summary,
		Stream:  sharedpresent.StreamStdout,
	}
}

func pageVersionValue(v *api.Version) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%d", v.Number)
}

// WriteDrift describes how a stored page body differed from the body that
// was submitted. Commands supply the finding; the wording is owned here.
type WriteDrift struct {
	BodyFormat string
	// TextChanged reports that content differs, not merely formatting.
	TextChanged bool
	// VisibleSent and VisibleStored count characters a reader sees.
	VisibleSent   int
	VisibleStored int
	// AtomsChanged reports embedded content differing where the text does not.
	AtomsChanged bool
	// DiffOffset is where the two bodies first diverge, or -1 when they do
	// not. SentExcerpt and StoredExcerpt are the text from that point.
	DiffOffset    int
	SentExcerpt   string
	StoredExcerpt string
	// AtomChanges names embedded node types whose counts moved.
	AtomChanges  []string
	DroppedAttrs []string
	AddedAttrs   []string
	// ParamChanges names macro parameters the server did not store as sent,
	// each already prefixed with - or + for dropped or added.
	ParamChanges []string
	// LostElements names storage element types that became less frequent
	// across the write.
	LostElements []string
	// StorageUncomparable reports that the check could not run; silence
	// would otherwise be indistinguishable from a clean write.
	StorageUncomparable bool
	StorageReason       string
	// LossCause explains why formatting can vanish for the format in use.
	LossCause string
}

// attrLines lists attribute changes, which identify the embedded content
// involved when the visible text cannot.
func attrLines(d WriteDrift) []string {
	var lines []string
	if len(d.AtomChanges) > 0 {
		lines = append(lines, "  embedded content changed:")
		for _, a := range d.AtomChanges {
			lines = append(lines, "    ~ "+a)
		}
	}
	if len(d.DroppedAttrs) > 0 {
		lines = append(lines, "  attributes dropped:")
		for _, a := range d.DroppedAttrs {
			lines = append(lines, "    - "+a)
		}
	}
	if len(d.AddedAttrs) > 0 {
		lines = append(lines, "  attributes added:")
		for _, a := range d.AddedAttrs {
			lines = append(lines, "    + "+a)
		}
	}
	if len(d.ParamChanges) > 0 {
		lines = append(lines, "  macro parameters changed:")
		for _, p := range d.ParamChanges {
			lines = append(lines, "    "+p)
		}
	}
	return lines
}

// describeContentChange states what moved in terms of the document: the
// characters a reader sees, and whether embedded content changed underneath
// unchanged text.
func describeContentChange(d WriteDrift) string {
	if d.AtomsChanged {
		return fmt.Sprintf("visible text is unchanged at %d characters, but embedded content differs", d.VisibleSent)
	}
	if d.VisibleSent == d.VisibleStored {
		return fmt.Sprintf("content differs at the same length of %d characters", d.VisibleSent)
	}
	return fmt.Sprintf("visible text went from %d to %d characters", d.VisibleSent, d.VisibleStored)
}

// PresentWriteDrift reports what Confluence stored versus what was sent.
// Losing content and normalizing attributes are worded differently because
// they oblige the reader differently: one means the change did not land, the
// other means it landed in a document the server tidied.
func (PagePresenter) PresentWriteDrift(d WriteDrift) *sharedpresent.OutputModel {
	var lines []string
	if d.StorageUncomparable {
		msg := "Could not compare the stored page against its state before the write, so formatting loss would not have been noticed."
		if d.StorageReason != "" {
			msg += " (" + d.StorageReason + ")"
		}
		lines = append(lines, msg, "")
	}
	if len(d.LostElements) > 0 {
		lines = append(lines, "The stored page lost formatting that was present before this write:")
		for _, e := range d.LostElements {
			lines = append(lines, "  - "+e)
		}
		tail := "Compare against the storage body before assuming the change was clean."
		if d.LossCause != "" {
			tail = d.LossCause + " " + tail
		}
		lines = append(lines, tail, "")
	}
	if d.TextChanged {
		lines = append(lines,
			fmt.Sprintf("Stored %s body does not match what was sent: %s.", d.BodyFormat, describeContentChange(d)),
			"The page was updated, but it does not hold the content supplied. Re-read the page before treating the change as applied.",
		)
		if d.DiffOffset >= 0 {
			lines = append(lines, fmt.Sprintf("  first difference at offset %d — sent %q, stored %q", d.DiffOffset, d.SentExcerpt, d.StoredExcerpt))
		}
		// Name what vanished. When only embedded content moved there is no
		// text position to point at, so this is the only detail available.
		lines = append(lines, attrLines(d)...)
	} else {
		if len(d.DroppedAttrs) > 0 {
			lines = append(lines, fmt.Sprintf("Confluence normalized the stored %s body. Content is intact; these attributes were dropped:", d.BodyFormat))
			for _, a := range d.DroppedAttrs {
				lines = append(lines, "  - "+a)
			}
		}
		if len(d.AddedAttrs) > 0 {
			lines = append(lines, "Confluence added attributes that were not sent:")
			for _, a := range d.AddedAttrs {
				lines = append(lines, "  + "+a)
			}
		}
		if len(d.ParamChanges) > 0 {
			lines = append(lines, fmt.Sprintf("Stored %s body does not hold the macro parameters that were sent. Page text is intact; these parameters differ:", d.BodyFormat))
			for _, p := range d.ParamChanges {
				lines = append(lines, "  "+p)
			}
		}
	}
	if len(lines) == 0 {
		return &sharedpresent.OutputModel{}
	}
	return &sharedpresent.OutputModel{Sections: []sharedpresent.Section{
		&sharedpresent.MessageSection{
			Kind:    sharedpresent.MessageWarning,
			Message: strings.Join(lines, "\n"),
			Stream:  sharedpresent.StreamStderr,
		},
	}}
}
