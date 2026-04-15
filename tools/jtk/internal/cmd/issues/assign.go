// Package issues provides CLI commands for managing Jira issues.
package issues

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

func newAssignCmd(opts *root.Options) *cobra.Command {
	var unassign bool

	cmd := &cobra.Command{
		Use:   "assign <issue-key> [account-id]",
		Short: "Assign an issue to a user",
		Long:  "Assign an issue to a user by their account ID, or unassign it.",
		Example: `  # Assign to a user
  jtk issues assign PROJ-123 5b10ac8d82e05b22cc7d4ef5

  # Unassign an issue
  jtk issues assign PROJ-123 --unassign`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			accountID := ""
			if len(args) > 1 {
				accountID = args[1]
			}
			return runAssign(cmd.Context(), opts, args[0], accountID, unassign)
		},
	}

	cmd.Flags().BoolVar(&unassign, "unassign", false, "Remove current assignee")

	return cmd
}

func runAssign(ctx context.Context, opts *root.Options, issueKey, accountID string, unassign bool) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if unassign {
		accountID = ""
	}

	if err := client.AssignIssue(ctx, issueKey, accountID); err != nil {
		return err
	}

	// Resolve display name for a friendlier message
	displayName := ""
	if !unassign && accountID != "" {
		displayName = accountID
		if user, err := client.GetUser(ctx, accountID); err == nil && user.DisplayName != "" {
			displayName = user.DisplayName
		}
	}

	model := jtkpresent.IssuePresenter{}.PresentAssigned(issueKey, displayName)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}
