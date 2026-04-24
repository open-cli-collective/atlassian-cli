package automation

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/mutation"
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

	current, err := client.GetAutomationRule(ctx, ruleID)
	if err != nil {
		return err
	}

	newState := "DISABLED"
	if enabled {
		newState = "ENABLED"
	}

	if current.State == newState {
		return jtkpresent.Emit(opts, jtkpresent.AutomationPresenter{}.PresentNoChange(current.Name, newState))
	}

	if err := client.SetAutomationRuleState(ctx, ruleID, enabled); err != nil {
		return err
	}

	if opts.EmitIDOnly() {
		return jtkpresent.EmitIDs(opts, []string{ruleID})
	}

	savedName := current.Name
	savedState := current.State
	return mutation.WriteAndPresent(ctx, opts, mutation.Config{
		Write: func(_ context.Context) (string, error) {
			return ruleID, nil
		},
		Fetch: func(ctx context.Context, id string) (*present.OutputModel, error) {
			rule, err := client.GetAutomationRule(ctx, id)
			if err != nil {
				return nil, err
			}
			return jtkpresent.AutomationPresenter{}.PresentDetail(rule, false), nil
		},
		IsFresh: func(model *present.OutputModel) bool {
			return mutation.DetailFieldEquals(model, "State", newState)
		},
		Fallback: func(_ string) *present.OutputModel {
			return jtkpresent.AutomationPresenter{}.PresentStateChanged(savedName, savedState, newState)
		},
	})
}
