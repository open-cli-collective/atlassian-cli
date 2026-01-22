package issues

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newAssignCmd(opts *root.Options) *cobra.Command {
	var unassign bool

	cmd := &cobra.Command{
		Use:   "assign <issue-key> [account-id]",
		Short: "Assign an issue to a user",
		Long:  "Assign an issue to a user by their account ID, or unassign it.",
		Example: `  # Assign to a user
  jira-ticket-cli issues assign PROJ-123 5b10ac8d82e05b22cc7d4ef5

  # Unassign an issue
  jira-ticket-cli issues assign PROJ-123 --unassign`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			accountID := ""
			if len(args) > 1 {
				accountID = args[1]
			}
			return runAssign(opts, args[0], accountID, unassign)
		},
	}

	cmd.Flags().BoolVar(&unassign, "unassign", false, "Remove current assignee")

	return cmd
}

func runAssign(opts *root.Options, issueKey, accountID string, unassign bool) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if unassign {
		accountID = ""
	}

	if err := client.AssignIssue(issueKey, accountID); err != nil {
		return err
	}

	if unassign || accountID == "" {
		v.Success("Unassigned issue %s", issueKey)
	} else {
		v.Success("Assigned issue %s to %s", issueKey, accountID)
	}

	return nil
}
