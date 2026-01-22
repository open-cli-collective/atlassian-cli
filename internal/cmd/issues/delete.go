package issues

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newDeleteCmd(opts *root.Options) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <issue-key>",
		Short: "Delete an issue",
		Long:  "Permanently delete a Jira issue. This action cannot be undone.",
		Example: `  # Delete an issue (will prompt for confirmation)
  jira-ticket-cli issues delete PROJ-123

  # Delete without confirmation
  jira-ticket-cli issues delete PROJ-123 --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(opts, args[0], force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(opts *root.Options, issueKey string, force bool) error {
	v := opts.View()

	if !force {
		v.Warning("This will permanently delete issue %s. This action cannot be undone.", issueKey)
		v.Info("Use --force to skip this confirmation.")
		return fmt.Errorf("deletion cancelled (use --force to confirm)")
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if err := client.DeleteIssue(issueKey); err != nil {
		return err
	}

	v.Success("Deleted issue %s", issueKey)
	return nil
}
