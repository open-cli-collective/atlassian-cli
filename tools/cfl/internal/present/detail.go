package present

import (
	"fmt"

	sharedpresent "github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/confluence-cli/api"
	cflconfig "github.com/open-cli-collective/confluence-cli/internal/config"
	"github.com/open-cli-collective/confluence-cli/internal/pageview"
)

type ConfigShowPresenter struct{}

func (SpacePresenter) PresentDetail(space *api.Space, full bool) *sharedpresent.OutputModel {
	fields := []sharedpresent.Field{
		{Label: "Key", Value: orDash(space.Key)},
		{Label: "Name", Value: orDash(space.Name)},
		{Label: "ID", Value: orDash(space.ID)},
		{Label: "Type", Value: orDash(space.Type)},
	}

	if full && space.Status != "" {
		fields = append(fields, sharedpresent.Field{Label: "Status", Value: space.Status})
	}
	if full && hasPlainDescription(space) {
		fields = append(fields, sharedpresent.Field{Label: "Description", Value: space.Description.Plain.Value})
	}

	return &sharedpresent.OutputModel{
		Sections: []sharedpresent.Section{
			&sharedpresent.DetailSection{Fields: fields},
		},
	}
}

func (ConfigShowPresenter) PresentDetail(proj cflconfig.ShowProjection) *sharedpresent.OutputModel {
	fields := []sharedpresent.Field{
		{Label: "URL", Value: formatValueWithSource(proj.URL)},
		{Label: "Email", Value: formatValueWithSource(proj.Email)},
		{Label: "API Token", Value: formatValueWithSource(proj.APIToken)},
		{Label: "Default Space", Value: formatValueWithSource(proj.DefaultSpace)},
		{Label: "Auth Method", Value: formatValueWithSource(proj.AuthMethod)},
		{Label: "Cloud ID", Value: formatValueWithSource(proj.CloudID)},
		{Label: "Keyring Ref", Value: formatValueWithSource(proj.KeyringRef)},
	}
	if proj.HasKeyringBackend {
		fields = append(fields, sharedpresent.Field{
			Label: "Keyring Backend",
			Value: formatValueWithSource(proj.KeyringBackend),
		})
	}
	if proj.HasKeyringPassphrase {
		fields = append(fields, sharedpresent.Field{
			Label: "Keyring Passphrase",
			Value: formatValueWithSource(proj.KeyringPassphrase),
		})
	}

	stderr := fmt.Sprintf("\nConfig file: %s", proj.ConfigPath)
	if !proj.ConfigReadable {
		stderr += "\n  (file not found or unreadable)"
	}

	return &sharedpresent.OutputModel{
		Sections: []sharedpresent.Section{
			&sharedpresent.DetailSection{Fields: fields},
			stderrInfo(stderr),
		},
	}
}

func (PagePresenter) PresentView(proj pageview.Projection) *sharedpresent.OutputModel {
	sections := make([]sharedpresent.Section, 0, 3)

	if !proj.ContentOnly {
		fields := []sharedpresent.Field{
			{Label: "Title", Value: orDash(proj.Title)},
			{Label: "ID", Value: orDash(proj.ID)},
		}
		switch {
		case proj.SpaceKey != "" && proj.SpaceID != "":
			fields = append(fields, sharedpresent.Field{
				Label: "Space",
				Value: fmt.Sprintf("%s (ID: %s)", proj.SpaceKey, proj.SpaceID),
			})
		case proj.SpaceID != "":
			fields = append(fields, sharedpresent.Field{
				Label: "Space ID",
				Value: proj.SpaceID,
			})
		}
		if proj.HasVersion {
			fields = append(fields, sharedpresent.Field{
				Label: "Version",
				Value: fmt.Sprintf("%d", proj.Version),
			})
		}
		sections = append(sections, &sharedpresent.DetailSection{Fields: fields})
	}

	if advisory := pageViewAdvisory(proj.Fallback); advisory != "" {
		sections = append(sections, stderrInfo(advisory))
	}

	body := pageViewBody(proj)
	if !proj.ContentOnly {
		sections = append(sections, stdoutInfo(""))
	}
	sections = append(sections, stdoutInfo(body))

	return &sharedpresent.OutputModel{Sections: sections}
}

func pageViewBody(proj pageview.Projection) string {
	if !proj.HasContent {
		return "(No content)"
	}

	body := proj.Body
	if proj.Truncated {
		body += fmt.Sprintf("\n\n... [truncated at %d chars, use --no-truncate for complete text]", pageview.MaxChars)
	}
	return body
}

func pageViewAdvisory(fallback pageview.FallbackKind) string {
	switch fallback {
	case pageview.FallbackNone:
		return ""
	case pageview.FallbackStorageRaw:
		return "(Failed to convert to markdown, showing raw HTML)"
	case pageview.FallbackADFRaw:
		return "(Failed to convert ADF to markdown, showing raw ADF)"
	default:
		return ""
	}
}

func formatValueWithSource(v cflconfig.ShowValue) string {
	if v.Value == "" {
		return fmt.Sprintf("(source: %s)", v.Source)
	}
	return fmt.Sprintf("%s  (source: %s)", v.Value, v.Source)
}

func hasPlainDescription(space *api.Space) bool {
	return space != nil &&
		space.Description != nil &&
		space.Description.Plain != nil &&
		space.Description.Plain.Value != ""
}
