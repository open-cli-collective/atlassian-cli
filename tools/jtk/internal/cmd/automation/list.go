package automation

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"

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
  jtk auto list -o json`,
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

	if len(rules) == 0 {
		model := jtkpresent.AutomationPresenter{}.PresentEmpty()
		out := present.Render(model, opts.RenderStyle())
		fmt.Fprint(opts.Stdout, out.Stdout)
		return nil
	}

	if opts.Output == "json" {
		v := opts.View()
		return v.JSON(rules)
	}

	model := jtkpresent.AutomationPresenter{}.PresentList(rules)
	out := present.Render(model, opts.RenderStyle())
	fmt.Fprint(opts.Stdout, out.Stdout)
	fmt.Fprint(opts.Stderr, out.Stderr)

	return nil
}
