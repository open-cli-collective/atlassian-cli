// Package me provides the CLI command for displaying the current user.
package me

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/view"

	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
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
  jtk me --id`,
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

	// --id: collapse output to the primary identifier.
	if opts.EmitIDOnly() {
		_, _ = fmt.Fprintln(opts.Stdout, user.AccountID)
		return nil
	}

	// JSON path: use existing artifact layer (unchanged)
	if v.Format == view.FormatJSON {
		return v.RenderArtifact(jtkartifact.ProjectUser(user, opts.ArtifactMode()))
	}

	// Plain path: just the account ID (legacy; --id is the preferred surface)
	if v.Format == view.FormatPlain {
		_, _ = fmt.Fprintln(opts.Stdout, user.AccountID)
		return nil
	}

	// Text path: presenter → model → pure render → write to both streams
	model := jtkpresent.UserPresenter{}.Present(user)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}
