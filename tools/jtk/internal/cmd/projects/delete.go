package projects

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/prompt"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

func newDeleteCmd(opts *root.Options) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <project-key>",
		Short: "Delete a project",
		Long: `Soft-delete a Jira project (moves it to trash).

The project can be restored from trash using 'jtk projects restore'.`,
		Example: `  # Delete a project (will prompt for confirmation)
  jtk projects delete MYPROJ

  # Delete without confirmation
  jtk projects delete MYPROJ --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd.Context(), opts, args[0], force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(ctx context.Context, opts *root.Options, keyOrID string, force bool) error {
	if !force {
		fmt.Fprintf(opts.Stderr, "This will delete project %s (moves to trash). It can be restored later.\n", keyOrID)
		fmt.Fprint(opts.Stderr, "Are you sure? [y/N]: ")

		confirmed, err := prompt.Confirm(opts.Stdin)
		if err != nil {
			return fmt.Errorf("reading confirmation: %w", err)
		}
		if !confirmed {
			model := jtkpresent.ProjectPresenter{}.PresentDeleteCancelled()
			out := present.Render(model, opts.RenderStyle())
			_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
			return nil
		}
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if err := client.DeleteProject(ctx, keyOrID); err != nil {
		return err
	}

	_ = cache.Touch(cache.ProjectDependents()...)

	model := jtkpresent.ProjectPresenter{}.PresentDeleted(keyOrID)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	return nil
}
