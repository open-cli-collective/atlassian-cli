// Package issues provides CLI commands for managing Jira issues.
package issues

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/resolve"
)

func newAssignCmd(opts *root.Options) *cobra.Command {
	var unassign bool

	cmd := &cobra.Command{
		Use:   "assign <issue-key> [user]",
		Short: "Assign an issue to a user",
		Long:  `Assign an issue to a user, or unassign it. The <user> argument accepts an accountId, email, display name, or "me" — it is resolved via the instance cache.`,
		Example: `  # Assign by display name, email, "me", or raw accountId
  jtk issues assign PROJ-123 "Aaron Wong"
  jtk issues assign PROJ-123 aaron@example.com
  jtk issues assign PROJ-123 me
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

func runAssign(ctx context.Context, opts *root.Options, issueKey, userInput string, unassign bool) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	accountID := ""
	displayName := ""

	if !unassign && userInput != "" {
		resolvedUser, err := resolve.New(client).User(ctx, userInput)
		if err != nil {
			return err
		}
		accountID = resolvedUser.AccountID
		displayName = resolvedUser.DisplayName
		if displayName == "" {
			// Pass-through path: resolver returned synthetic api.User with only
			// AccountID populated. Fall back to the raw ID in the message.
			displayName = accountID
		}
	}

	if err := client.AssignIssue(ctx, issueKey, accountID); err != nil {
		return err
	}

	model := jtkpresent.IssuePresenter{}.PresentAssigned(issueKey, displayName)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}
