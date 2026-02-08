package projects

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/prompt"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
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
			return runDelete(opts, args[0], force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(opts *root.Options, keyOrID string, force bool) error {
	v := opts.View()

	if !force {
		fmt.Printf("This will delete project %s (moves to trash). It can be restored later.\n", keyOrID)
		fmt.Print("Are you sure? [y/N]: ")

		confirmed, err := prompt.Confirm(opts.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		if !confirmed {
			v.Info("Deletion cancelled.")
			return nil
		}
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if err := client.DeleteProject(keyOrID); err != nil {
		return err
	}

	v.Success("Deleted project %s (moved to trash)", keyOrID)
	return nil
}
