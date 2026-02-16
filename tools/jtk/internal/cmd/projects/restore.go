package projects

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newRestoreCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:     "restore <project-key>",
		Short:   "Restore a deleted project",
		Long:    "Restore a project from the trash.",
		Example: `  jtk projects restore MYPROJ`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestore(cmd.Context(), opts, args[0])
		},
	}
}

func runRestore(ctx context.Context, opts *root.Options, keyOrID string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	project, err := client.RestoreProject(ctx, keyOrID)
	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(project)
	}

	v.Success("Restored project %s (%s)", project.Key, project.Name)

	return nil
}
