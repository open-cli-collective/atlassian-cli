package automation

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

func newGetCmd(opts *root.Options) *cobra.Command {
	var showComponents bool

	cmd := &cobra.Command{
		Use:   "get <rule-id>",
		Short: "Get automation rule details",
		Long: `Retrieve and display details for a specific automation rule.

Shows rule identifier, name, state, components summary, and description.
Use --show-components to see component type details.
Use --extended for additional fields (labels, tags, author, scope, timestamps).

For the exact JSON needed for editing, use 'jtk auto export' instead.`,
		Example: `  jtk automation get 12345
  jtk auto get 12345 --show-components
  jtk auto get 12345 --extended`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(cmd.Context(), opts, args[0], showComponents)
		},
	}

	cmd.Flags().BoolVar(&showComponents, "show-components", false, "Show component type details")

	return cmd
}

func runGet(ctx context.Context, opts *root.Options, ruleID string, showComponents bool) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	rule, err := client.GetAutomationRule(ctx, ruleID)
	if err != nil {
		return err
	}

	if opts.EmitIDOnly() {
		return jtkpresent.EmitIDs(opts, []string{rule.Identifier()})
	}

	if v.Format == view.FormatJSON {
		return v.RenderArtifact(jtkartifact.ProjectAutomationRule(rule, opts.ArtifactMode()))
	}

	presenter := jtkpresent.AutomationPresenter{}

	if opts.IsExtended() {
		authorName := ""
		if rule.AuthorAccountID != "" {
			user, err := client.GetUser(ctx, rule.AuthorAccountID, "")
			if err == nil && user.DisplayName != "" {
				authorName = user.DisplayName
			} else {
				authorName = rule.AuthorAccountID
			}
		}
		return jtkpresent.Emit(opts, presenter.PresentGetDetailExtended(rule, showComponents, authorName))
	}

	return jtkpresent.Emit(opts, presenter.PresentGetDetail(rule, showComponents))
}
