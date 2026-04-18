package issues

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/resolve"
)

func newTypesCmd(opts *root.Options) *cobra.Command {
	var project string

	cmd := &cobra.Command{
		Use:   "types",
		Short: "List valid issue types for a project",
		Long:  "List all valid issue types that can be used when creating issues in a specific project.",
		Example: `  # List issue types for a project
  jtk issues types --project MYPROJ

  # Using short flag
  jtk issues types -p MYPROJ`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTypes(cmd.Context(), opts, project)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Project key (required)")
	_ = cmd.MarkFlagRequired("project")

	return cmd
}

func runTypes(ctx context.Context, opts *root.Options, project string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	resolvedProject, err := resolve.New(client).Project(ctx, project)
	if err != nil {
		return err
	}
	projectKey := resolvedProject.Key

	projectDetail, err := client.GetProject(ctx, projectKey)
	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(projectDetail.IssueTypes)
	}

	if len(projectDetail.IssueTypes) == 0 {
		model := jtkpresent.IssuePresenter{}.PresentNoTypes(projectKey)
		out := present.Render(model, opts.RenderStyle())
		_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
		return nil
	}

	// Text path: presenter → render → write
	model := jtkpresent.IssuePresenter{}.PresentTypes(projectDetail.IssueTypes)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}
