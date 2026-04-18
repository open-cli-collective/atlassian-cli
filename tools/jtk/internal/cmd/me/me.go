// Package me provides the CLI command for displaying the current user.
package me

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
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
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// Only fetch groups/applicationRoles when --extended actually renders
	// them; default, --id, and JSON paths don't care about those blocks.
	expand := ""
	if opts.IsExtended() {
		expand = api.UserExtendedExpand
	}
	user, err := client.GetCurrentUser(ctx, expand)
	if err != nil {
		return err
	}

	// --id: collapse output to the primary identifier.
	if opts.EmitIDOnly() {
		return jtkpresent.EmitIDs(opts, []string{user.AccountID})
	}

	// JSON path: use existing artifact layer (unchanged)
	if v.Format == view.FormatJSON {
		return v.RenderArtifact(jtkartifact.ProjectUser(user, opts.ArtifactMode()))
	}

	// Plain path: preserve the legacy contract (bare account ID). This
	// predates --id and is kept for backwards compatibility per CLAUDE.md
	// ("--output / -o ... retained for compatibility but hidden from --help").
	// --id is the preferred surface; `-o plain` stays working for scripts
	// that haven't migrated.
	if v.Format == view.FormatPlain {
		_, _ = fmt.Fprintln(opts.Stdout, user.AccountID)
		return nil
	}

	presenter := jtkpresent.UserPresenter{}
	var model = presenter.PresentUserOneLiner(user)
	if opts.IsExtended() {
		model = presenter.PresentUserExtended(user)
	}
	return jtkpresent.Emit(opts, model)
}
