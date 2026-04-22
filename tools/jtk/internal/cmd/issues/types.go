package issues

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

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

  # Emit only type IDs
  jtk issues types --project MYPROJ --id`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTypes(cmd.Context(), opts, project)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Project key (required)")
	_ = cmd.MarkFlagRequired("project")

	return cmd
}

func runTypes(ctx context.Context, opts *root.Options, project string) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	resolvedProject, err := resolve.New(client).Project(ctx, project)
	if err != nil {
		return err
	}
	projectKey := resolvedProject.Key

	projectDetail, err := client.GetProject(ctx, projectKey, "issueTypes")
	if err != nil {
		return err
	}

	if len(projectDetail.IssueTypes) == 0 {
		return jtkpresent.Emit(opts, jtkpresent.IssuePresenter{}.PresentNoTypes(projectKey))
	}

	if opts.EmitIDOnly() {
		ids := make([]string, len(projectDetail.IssueTypes))
		for i, t := range projectDetail.IssueTypes {
			ids[i] = t.ID
		}
		return jtkpresent.EmitIDs(opts, ids)
	}

	v := opts.View()
	if v.Format == view.FormatJSON {
		return v.JSON(projectDetail.IssueTypes)
	}

	model := jtkpresent.IssuePresenter{}.PresentTypes(projectDetail.IssueTypes)
	return jtkpresent.Emit(opts, model)
}
