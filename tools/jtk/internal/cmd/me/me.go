// Package me provides the CLI command for displaying the current user.
package me

import (
	"context"
	"fmt"

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

	pairs := []view.KeyValue{
		{Key: "Account ID", Value: user.AccountID},
		{Key: "Display Name", Value: user.DisplayName},
	}
	if user.EmailAddress != "" {
		pairs = append(pairs, view.KeyValue{Key: "Email", Value: user.EmailAddress})
	}
	pairs = append(pairs, view.KeyValue{Key: "Active", Value: fmt.Sprintf("%t", user.Active)})

	return v.RenderKeyValues(pairs)
}
