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
	SentLen     int
	StoredLen   int
	// DiffOffset is where the two bodies first diverge, or -1 when they do
	// not. SentExcerpt and StoredExcerpt are the text from that point.
	DiffOffset    int
	SentExcerpt   string
	StoredExcerpt string
	DroppedAttrs  []string
	AddedAttrs    []string
}

// PresentWriteDrift reports what Confluence stored versus what was sent.
// Losing content and normalizing attributes are worded differently because
// they oblige the reader differently: one means the change did not land, the
// other means it landed in a document the server tidied.
func (PagePresenter) PresentWriteDrift(d WriteDrift) *sharedpresent.OutputModel {
	var lines []string
	if d.TextChanged {
		lines = append(lines,
			fmt.Sprintf("Stored %s body does not match what was sent: content differs (%d chars sent, %d stored).", d.BodyFormat, d.SentLen, d.StoredLen),
			"The page was updated, but it does not hold the content supplied. Re-read the page before treating the change as applied.",
		)
		if d.DiffOffset >= 0 {
			lines = append(lines, fmt.Sprintf("  first difference at offset %d — sent %q, stored %q", d.DiffOffset, d.SentExcerpt, d.StoredExcerpt))
		}
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
