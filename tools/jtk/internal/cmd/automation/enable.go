package automation

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

func newEnableCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable <rule-id>",
		Short: "Enable an automation rule",
		Long:  "Enable a disabled automation rule. This is a safe operation that does not modify the rule definition.",
		Example: `  jtk automation enable 12345
  jtk auto enable 12345`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetState(cmd.Context(), opts, args[0], true)
		},
	}

	return cmd
}

func runSetState(ctx context.Context, opts *root.Options, ruleID string, enabled bool) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// Fetch current rule to show context
	current, err := client.GetAutomationRule(ctx, ruleID)
	if err != nil {
		return err
	}

	newState := "DISABLED"
	if enabled {
		newState = "ENABLED"
	}

	if current.State == newState {
		model := jtkpresent.AutomationPresenter{}.PresentNoChange(current.Name, newState)
		out := present.Render(model, opts.RenderStyle())
		fmt.Fprint(opts.Stdout, out.Stdout)
		fmt.Fprint(opts.Stderr, out.Stderr)
		return nil
	}

	if err := client.SetAutomationRuleState(ctx, ruleID, enabled); err != nil {
		return err
	}

	model := jtkpresent.AutomationPresenter{}.PresentStateChanged(current.Name, current.State, newState)
	out := present.Render(model, opts.RenderStyle())
	fmt.Fprint(opts.Stdout, out.Stdout)
	fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}
