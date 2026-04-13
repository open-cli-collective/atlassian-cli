// Package me provides the CLI command for displaying the current user.
package me

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// Register registers the me command
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:   "me",
		Short: "Show current user",
		Long:  "Show information about the currently authenticated Jira user.",
		Example: `  # Show current user info
  jtk me

  # Show just the account ID (for scripting)
  jtk me -o plain`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.Context(), opts)
		},
	}

	parent.AddCommand(cmd)
}

func run(ctx context.Context, opts *root.Options) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	user, err := client.GetCurrentUser(ctx)
	if err != nil {
		return err
	}

	if v.Format == view.FormatJSON {
		return v.RenderArtifact(jtkartifact.ProjectUser(user, opts.ArtifactMode()))
	}

	if v.Format == view.FormatPlain {
		v.Println("%s", user.AccountID)
		return nil
	}

	v.Println("Account ID:   %s", user.AccountID)
	v.Println("Display Name: %s", user.DisplayName)
	if user.EmailAddress != "" {
		v.Println("Email:        %s", user.EmailAddress)
	}
	v.Println("Active:       %t", user.Active)

	return nil
}
