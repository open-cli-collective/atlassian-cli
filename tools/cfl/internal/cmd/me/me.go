// Package me provides the me command for cfl.
package me

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	cflpresent "github.com/open-cli-collective/confluence-cli/internal/present"
)

// Register adds the me command to the root command.
func Register(rootCmd *cobra.Command, opts *root.Options) {
	rootCmd.AddCommand(newMeCmd(opts))
}

func newMeCmd(opts *root.Options) *cobra.Command {
	var idOnly bool
	cmd := &cobra.Command{
		Use:   "me",
		Short: "Show the currently authenticated user",
		Long: `Show the user authenticated by the current cfl configuration as a token-dense one-liner: accountId | displayName | email.

Missing fields render as "-" so the row is always exactly three pipe-delimited fields.`,
		Example: `  # Show current user
  cfl me

  # Show only the account ID (for scripting)
  cfl me --id`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return Run(cmd.Context(), opts, idOnly)
		},
	}
	cmd.Flags().BoolVar(&idOnly, "id", false, "Print only the account ID")
	return cmd
}

// Run fetches and renders the currently authenticated user.
func Run(ctx context.Context, opts *root.Options, idOnly bool) error {
	client, err := opts.APIClient()
	if err != nil {
		return fmt.Errorf("getting API client: %w", err)
	}
	user, err := client.GetCurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("getting current user: %w", err)
	}

	presenter := cflpresent.UserPresenter{}
	if idOnly {
		return cflpresent.Emit(opts, presenter.PresentUserIDOnly(user))
	}
	return cflpresent.Emit(opts, presenter.PresentUserOneLiner(user))
}
