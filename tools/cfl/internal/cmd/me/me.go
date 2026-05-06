// Package me provides the me command for cfl.
package me

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
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
		return err
	}
	user, err := client.GetCurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("getting current user: %w", err)
	}

	v := opts.View()
	if idOnly {
		v.Println("%s", user.AccountID)
		return nil
	}
	RenderUserOneLiner(v, user)
	return nil
}

// RenderUserOneLiner writes the canonical 3-field user one-liner to v.
// Exported so cfl init can render the same output after a successful save
// without re-fetching the user or going through opts.APIClient().
func RenderUserOneLiner(v *view.View, user *api.User) {
	name := dashIfEmpty(user.DisplayName)
	email := dashIfEmpty(user.Email)
	v.Println("%s | %s | %s", user.AccountID, name, email)
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
