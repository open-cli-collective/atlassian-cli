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
		Label: "Active", Value: BoolString(user.Active),
	})
	return &present.OutputModel{
		Sections: []present.Section{&present.DetailSection{Fields: fields}},
	}
}

// PresentList creates a table view for a list of users.
func (UserPresenter) PresentList(users []api.User) *present.OutputModel {
	rows := make([]present.Row, len(users))
	for i, u := range users {
		active := "yes"
		if !u.Active {
			active = "no"
		}
		rows[i] = present.Row{
			Cells: []string{u.AccountID, u.DisplayName, u.EmailAddress, active},
		}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ACCOUNT ID", "NAME", "EMAIL", "ACTIVE"},
				Rows:    rows,
			},
		},
	}
}

// PresentEmpty creates an info message when no users are found.
func (UserPresenter) PresentEmpty(query string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("No users found matching '%s'", query),
				Stream:  present.StreamStdout,
			},
		},
	}
}
