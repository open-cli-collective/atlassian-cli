package automation

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newDeleteCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <rule-id>",
		Short: "Delete an automation rule",
		Long: `Delete an automation rule permanently. If the rule is currently ENABLED,
it will be automatically disabled before deletion.

This action cannot be undone.`,
		Example: `  jtk automation delete 019cd438-229b-75f4-a443-9a96e687b867
  jtk auto delete 019cd438-229b-75f4-a443-9a96e687b867`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd.Context(), opts, args[0])
		},
	}

	return cmd
}

func runDelete(ctx context.Context, opts *root.Options, ruleID string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	current, err := client.GetAutomationRule(ctx, ruleID)
	if err != nil {
		return err
	}

	// API rejects DELETE on ENABLED rules — disable first.
	wasEnabled := current.State == "ENABLED"
	if wasEnabled {
		if err := client.SetAutomationRuleState(ctx, ruleID, false); err != nil {
			return err
		}
	}

	if err := client.DeleteAutomationRule(ctx, ruleID); err != nil {
		if wasEnabled {
			return fmt.Errorf("rule was disabled but delete failed: %w — re-enable with: jtk auto enable %s", err, ruleID)
		}
		return err
	}

	if opts.Output == "json" {
		return v.JSON(map[string]string{"status": "deleted", "ruleId": ruleID, "name": current.Name})
	}

	v.Success("Deleted automation rule %q (%s)", current.Name, ruleID)
	return nil
}
