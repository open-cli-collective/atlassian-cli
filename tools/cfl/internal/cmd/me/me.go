// Package me provides the me command for cfl.
package me

import (
	"context"
	"fmt"
	"strings"

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
		return fmt.Errorf("getting API client: %w", err)
	}
	user, err := client.GetCurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("getting current user: %w", err)
	}

	v := opts.View()
	if idOnly {
		v.Println("%s", normalizeField(user.AccountID))
		return nil
	}
	RenderUserOneLiner(v, user)
	return nil
}

// RenderUserOneLiner writes the canonical 3-field user one-liner to v.
// Exported so cfl init can render the same output after a successful save
// without re-fetching the user or going through opts.APIClient().
//
// Field values are normalized so the output is always exactly three
// pipe-delimited fields on a single line — newlines collapse to spaces and
// embedded pipes are escaped to "\|" so downstream parsers can rely on
// `split('|')` returning three columns.
func RenderUserOneLiner(v *view.View, user *api.User) {
	id := normalizeField(user.AccountID)
	name := normalizeField(user.DisplayName)
	email := normalizeField(user.Email)
	v.Println("%s | %s | %s", id, name, email)
}

func normalizeField(s string) string {
	if s == "" {
		return "-"
	}
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "|", `\|`)
	return s
}
