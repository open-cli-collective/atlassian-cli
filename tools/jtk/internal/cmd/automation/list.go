package automation

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
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
  jtk automation list --id
  jtk automation list --extended`,
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

	v := opts.View()
	if v.Format == view.FormatJSON {
		return v.JSON(rules)
	}

	if opts.IsExtended() {
		authorNames := resolveAuthorNames(ctx, client, automationSummaryAuthorIDs(rules))
		return jtkpresent.Emit(opts, jtkpresent.AutomationPresenter{}.PresentListExtended(rules, authorNames))
	}

	return jtkpresent.Emit(opts, jtkpresent.AutomationPresenter{}.PresentList(rules))
}

func automationSummaryAuthorIDs(rules []api.AutomationRuleSummary) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, r := range rules {
		if r.AuthorAccountID != "" && !seen[r.AuthorAccountID] {
			seen[r.AuthorAccountID] = true
			ids = append(ids, r.AuthorAccountID)
		}
	}
	return ids
}

func resolveAuthorNames(ctx context.Context, client *api.Client, accountIDs []string) map[string]string {
	names := make(map[string]string, len(accountIDs))
	for _, id := range accountIDs {
		user, err := client.GetUser(ctx, id, "")
		if err != nil {
			continue
		}
		if user.DisplayName != "" {
			names[id] = user.DisplayName
		}
	}
	return names
}
