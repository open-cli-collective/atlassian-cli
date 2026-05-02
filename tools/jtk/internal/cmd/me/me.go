// Package me provides the CLI command for displaying the current user.
package me

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

// Register registers the me command
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:   "me",
		Short: "Show current user",
		Long:  "Show information about the currently authenticated Jira user.",
		Example: `  # Show current user info (pipe one-liner)
  jtk me

  # Include timezone, locale, and group/application-role counts
  jtk me --extended

  # Show just the account ID (for scripting)
  jtk me --id`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.Context(), opts)
		},
	}

	parent.AddCommand(cmd)
}

func run(ctx context.Context, opts *root.Options) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	expand := ""
	if opts.IsExtended() {
		expand = api.UserExtendedExpand
	}
	user, err := client.GetCurrentUser(ctx, expand)
	if err != nil {
		return err
	}

	if opts.EmitIDOnly() {
		return jtkpresent.EmitIDs(opts, []string{user.AccountID})
	}

	presenter := jtkpresent.UserPresenter{}
	var model = presenter.PresentUserOneLiner(user)
	if opts.IsExtended() {
		model = presenter.PresentUserExtended(user)
	}
	return jtkpresent.Emit(opts, model)
}
