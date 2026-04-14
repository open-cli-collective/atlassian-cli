// Package present provides presenters that map domain types to presentation models.
package present

import (
	"fmt"

	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// UserPresenter creates presentation models for user data.
type UserPresenter struct{}

// Present creates a presentation model for text output.
// Content normalization (if any) happens here, not in the renderer.
func (UserPresenter) Present(user *api.User) *present.OutputModel {
	fields := []present.Field{
		{Label: "Account ID", Value: user.AccountID},
		{Label: "Display Name", Value: user.DisplayName},
	}
	if user.EmailAddress != "" {
		fields = append(fields, present.Field{Label: "Email", Value: user.EmailAddress})
	}
	fields = append(fields, present.Field{
		Label: "Active", Value: fmt.Sprintf("%t", user.Active),
	})
	return &present.OutputModel{
		Sections: []present.Section{&present.DetailSection{Fields: fields}},
	}
}
