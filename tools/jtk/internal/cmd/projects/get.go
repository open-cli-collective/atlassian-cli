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

func newGetCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "get <project-key>",
		Short: "Get project details",
		Long:  "Get details for a specific project by key or ID.",
		Example: `  jtk projects get MYPROJECT
  jtk projects get 10001`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(cmd.Context(), opts, args[0])
		},
	}
}

func runGet(ctx context.Context, opts *root.Options, keyOrID string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	project, err := client.GetProject(ctx, keyOrID)
	if err != nil {
		return err
	}

	if v.Format == view.FormatJSON {
		return v.JSON(project)
	}

	model := jtkpresent.ProjectPresenter{}.Present(project)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}
