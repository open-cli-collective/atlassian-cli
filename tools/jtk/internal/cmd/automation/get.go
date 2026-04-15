package automation

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"
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

Shows rule metadata and a summary of components. Use --show-components to see
component type details. Use -o json for structured output. Use --full for
additional fields (description, labels, tags).

For the exact JSON needed for editing, use 'jtk auto export' instead.`,
		Example: `  jtk automation get 12345
  jtk auto get 12345 --show-components
  jtk auto get 12345 -o json`,
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

	if v.Format == view.FormatJSON {
		return v.RenderArtifact(jtkartifact.ProjectAutomationRule(rule, opts.ArtifactMode()))
	}

	model := jtkpresent.AutomationPresenter{}.PresentDetail(rule, showComponents)
	out := present.Render(model, opts.RenderStyle())
	fmt.Fprint(opts.Stdout, out.Stdout)
	fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}
