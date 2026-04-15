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

func newListCmd(opts *root.Options) *cobra.Command {
	var query string
	var maxResults int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects",
		Long:  "List Jira projects, optionally filtered by a search query.",
		Example: `  # List all projects
  jtk projects list

  # Search projects by name
  jtk projects list --query "my project"

  # Limit results
  jtk projects list --max 10`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts, query, maxResults)
		},
	}

	cmd.Flags().StringVarP(&query, "query", "q", "", "Filter projects by name")
	cmd.Flags().IntVarP(&maxResults, "max", "m", 50, "Maximum number of results")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, query string, maxResults int) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	result, err := client.SearchProjects(ctx, query, 0, maxResults)
	if err != nil {
		return err
	}

	if len(result.Values) == 0 {
		model := jtkpresent.ProjectPresenter{}.PresentEmpty()
		out := present.Render(model, opts.RenderStyle())
		_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
		return nil
	}

	if v.Format == view.FormatJSON {
		return v.JSON(result.Values)
	}

	model := jtkpresent.ProjectPresenter{}.PresentList(result.Values)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}
