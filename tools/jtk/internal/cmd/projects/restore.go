package projects

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/mutation"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

func newRestoreCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:     "restore <project-key>",
		Short:   "Restore a deleted project",
		Long:    "Restore a project from the trash.",
		Example: `  jtk projects restore MYPROJ`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestore(cmd.Context(), opts, args[0])
		},
	}
}

func runRestore(ctx context.Context, opts *root.Options, keyOrID string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	project, err := client.RestoreProject(ctx, keyOrID)
	if err != nil {
		return err
	}

	_ = cache.Touch(cache.ProjectDependents()...)

	if opts.EmitIDOnly() {
		return jtkpresent.EmitIDs(opts, []string{project.Key})
	}

	if v.Format == view.FormatJSON {
		return v.JSON(project)
	}

	restoredName := project.Name
	return mutation.WriteAndPresent(ctx, opts, mutation.Config{
		Write: func(_ context.Context) (string, error) {
			return project.Key, nil
		},
		Fetch: func(ctx context.Context, id string) (*present.OutputModel, error) {
			fetched, err := client.GetProject(ctx, id, api.ProjectGetExpand)
			if err != nil {
				return nil, err
			}
			return jtkpresent.ProjectPresenter{}.PresentProjectDetail(fetched, opts.IsExtended()), nil
		},
		Fallback: func(id string) *present.OutputModel {
			return jtkpresent.ProjectPresenter{}.PresentRestored(id, restoredName)
		},
	})
}
