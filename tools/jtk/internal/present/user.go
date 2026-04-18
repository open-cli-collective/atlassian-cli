// Package present provides presenters that map domain types to presentation models.
package present

import (
	"fmt"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// UserPresenter creates presentation models for user data.
type UserPresenter struct{}

// UserDetailSpec declares the logical fields a `me` / `users get` response
// carries. Used by `--fields` projection on `users get`. `me` itself does not
// expose `--fields` (scope decision: too few fields to justify the flag), but
// the Registry is kept single-sourced so spec parity with `users get` is
// mechanical to lock.
var UserDetailSpec = projection.Registry{
	{Header: "ACCOUNT_ID", Identity: true},
	{Header: "NAME"},
	{Header: "EMAIL"},
	{Header: "TIMEZONE", Extended: true},
	{Header: "LOCALE", Extended: true},
	{Header: "ACTIVE", Extended: true},
	{Header: "GROUPS", Extended: true},
	{Header: "APPLICATION_ROLES", Aliases: []string{"APP_ROLES"}, Extended: true},
}

// UserListSpec declares the columns emitted by PresentUserList and
// PresentUserListProjection. Order must match the hardcoded Headers in
// PresentUserList's TableSection; a parity test locks the two in sync.
var UserListSpec = projection.Registry{
	{Header: "ACCOUNT_ID", Identity: true},
	{Header: "NAME"},
	{Header: "EMAIL"},
	{Header: "ACTIVE"},
	{Header: "TIMEZONE", Extended: true},
	{Header: "LOCALE", Extended: true},
}

// PresentUserOneLiner builds the spec-shaped default output for `me` and
// `users get`: a single pipe-delimited record "{accountId} | {name} | {email}".
// Empty email renders as `-`. Emitted as a MessageSection rather than a
// TableSection because the spec one-liner has no column header line.
func (UserPresenter) PresentUserOneLiner(u *api.User) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: userOneLiner(u),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentUserExtended builds the spec-shaped `--extended` output: the pipe
// one-liner followed by two compound "Key: X   Key2: Y" rows. Missing
// timeZone/locale render as `-`; missing groups/applicationRoles (i.e., the
// endpoint did not expand them) also render as `-`.
func (UserPresenter) PresentUserExtended(u *api.User) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: userOneLiner(u),
				Stream:  present.StreamStdout,
			},
			&present.MessageSection{
				Kind: present.MessageInfo,
				Message: fmt.Sprintf("Timezone: %s   Locale: %s   Active: %s",
					OrDash(u.TimeZone), OrDash(u.Locale), BoolString(u.Active)),
				Stream: present.StreamStdout,
			},
			&present.MessageSection{
				Kind: present.MessageInfo,
				Message: fmt.Sprintf("Groups: %s   Application Roles: %s",
					PresentOptionalCount(u.Groups), PresentOptionalCount(u.ApplicationRoles)),
				Stream: present.StreamStdout,
			},
		},
	}
}

// PresentUserDetailProjection builds a DetailSection view for a single user,
// keyed by the headers declared in UserDetailSpec. Commands use this only
// when `--fields` is active; projection.ProjectDetail then slices the section
// to the user-selected subset. Output flattens to "Label: Value" lines,
// matching the issues get --fields precedent.
func (UserPresenter) PresentUserDetailProjection(u *api.User) *present.OutputModel {
	fields := []present.Field{
		{Label: "ACCOUNT_ID", Value: u.AccountID},
		{Label: "NAME", Value: u.DisplayName},
		{Label: "EMAIL", Value: OrDash(u.EmailAddress)},
		{Label: "TIMEZONE", Value: OrDash(u.TimeZone)},
		{Label: "LOCALE", Value: OrDash(u.Locale)},
		{Label: "ACTIVE", Value: BoolString(u.Active)},
		{Label: "GROUPS", Value: PresentOptionalCount(u.Groups)},
		{Label: "APPLICATION_ROLES", Value: PresentOptionalCount(u.ApplicationRoles)},
	}
	return &present.OutputModel{
		Sections: []present.Section{&present.DetailSection{Fields: fields}},
	}
}

// PresentUserList builds a TableSection for `users search`. When extended is
// true, TIMEZONE/LOCALE columns are appended so the Registry and the rendered
// headers stay aligned in both modes.
// The command is responsible for appending any pagination hint via
// AppendPaginationHintWithToken; this presenter returns the plain model.
func (UserPresenter) PresentUserList(users []api.User, extended bool) *present.OutputModel {
	headers := []string{"ACCOUNT_ID", "NAME", "EMAIL", "ACTIVE"}
	if extended {
		headers = append(headers, "TIMEZONE", "LOCALE")
	}
	rows := make([]present.Row, len(users))
	for i, u := range users {
		cells := []string{
			u.AccountID,
			u.DisplayName,
			OrDash(u.EmailAddress),
			BoolString(u.Active),
		}
		if extended {
			cells = append(cells, OrDash(u.TimeZone), OrDash(u.Locale))
		}
		rows[i] = present.Row{Cells: cells}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{Headers: headers, Rows: rows},
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

// userOneLiner formats the default identity record used by `me` and
// `users get`. "-" substitutes for an empty email address.
func userOneLiner(u *api.User) string {
	return fmt.Sprintf("%s | %s | %s", u.AccountID, u.DisplayName, OrDash(u.EmailAddress))
}
