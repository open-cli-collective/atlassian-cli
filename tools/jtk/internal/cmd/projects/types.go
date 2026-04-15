package projects

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

func newTypesCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:     "types",
		Short:   "List project types",
		Long:    "List available project types for creating new projects.",
		Example: `  jtk projects types`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTypes(cmd.Context(), opts)
		},
	}
}

func runTypes(ctx context.Context, opts *root.Options) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	types, err := client.ListProjectTypes(ctx)
	if err != nil {
		return err
	}

	if len(types) == 0 {
		model := jtkpresent.ProjectPresenter{}.PresentNoTypes()
		out := present.Render(model, opts.RenderStyle())
		_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
		return nil
	}

	if v.Format == view.FormatJSON {
		return v.JSON(types)
	}

	model := jtkpresent.ProjectPresenter{}.PresentTypes(types)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}
