package present

import (
	"fmt"

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
