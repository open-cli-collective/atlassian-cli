package automation

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

func newListCmd(opts *root.Options) *cobra.Command {
	var state string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List automation rules",
		Long:  "List all automation rules with optional state filtering.",
		Example: `  jtk automation list
  jtk automation list --state ENABLED
  jtk automation list --id`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts, strings.ToUpper(state))
		},
	}

	cmd.Flags().StringVar(&state, "state", "", "Filter by state (ENABLED or DISABLED)")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, state string) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	rules, err := client.ListAutomationRulesFiltered(ctx, state)
	if err != nil {
		return err
	}

	if opts.EmitIDOnly() {
		ids := make([]string, len(rules))
		for i, r := range rules {
			ids[i] = r.Identifier()
		}
		return jtkpresent.EmitIDs(opts, ids)
	}

	if len(rules) == 0 {
		return jtkpresent.Emit(opts, jtkpresent.AutomationPresenter{}.PresentEmpty())
	}

	return jtkpresent.Emit(opts, jtkpresent.AutomationPresenter{}.PresentList(rules))
}
