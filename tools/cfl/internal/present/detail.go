package present

import (
	"fmt"

	sharedpresent "github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/confluence-cli/api"
	cflconfig "github.com/open-cli-collective/confluence-cli/internal/config"
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
	if full && space.Description != nil && space.Description.Plain != nil && space.Description.Plain.Value != "" {
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
		{Label: "URL", Value: cflconfig.FormatValueWithSource(proj.URL)},
		{Label: "Email", Value: cflconfig.FormatValueWithSource(proj.Email)},
		{Label: "API Token", Value: cflconfig.FormatValueWithSource(proj.APIToken)},
		{Label: "Default Space", Value: cflconfig.FormatValueWithSource(proj.DefaultSpace)},
		{Label: "Auth Method", Value: cflconfig.FormatValueWithSource(proj.AuthMethod)},
		{Label: "Cloud ID", Value: cflconfig.FormatValueWithSource(proj.CloudID)},
		{Label: "Keyring Ref", Value: cflconfig.FormatValueWithSource(proj.KeyringRef)},
	}
	if proj.HasKeyringBackend {
		fields = append(fields, sharedpresent.Field{
			Label: "Keyring Backend",
			Value: cflconfig.FormatValueWithSource(proj.KeyringBackend),
		})
	}
	if proj.HasKeyringPassphrase {
		fields = append(fields, sharedpresent.Field{
			Label: "Keyring Passphrase",
			Value: cflconfig.FormatValueWithSource(proj.KeyringPassphrase),
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
