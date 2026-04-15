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

	if v.Format == view.FormatJSON {
		return v.JSON(project)
	}

	model := jtkpresent.ProjectPresenter{}.PresentRestored(project.Key, project.Name)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	return nil
}
