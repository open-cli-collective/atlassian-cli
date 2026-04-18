package projects

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

func newGetCmd(opts *root.Options) *cobra.Command {
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "get <project-key>",
		Short: "Get project details",
		Long:  "Get details for a specific project by key or ID.",
		Example: `  # Spec-shaped default output
  jtk projects get MYPROJECT

  # Admin/audit detail (components enumerated, Simplified/Private flags)
  jtk projects get MYPROJECT --extended

  # Just the project key
  jtk projects get MYPROJECT --id

  # Project detail rendered as labeled fields
  jtk projects get MYPROJECT --fields NAME,LEAD`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(cmd.Context(), opts, args[0], fieldsFlag)
		},
	}

	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display fields (ProjectDetailSpec headers)")

	return cmd
}

func runGet(ctx context.Context, opts *root.Options, keyOrID, fieldsFlag string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if opts.EmitIDOnly() {
		// Skip expand in --id mode: the canonical key is in every /project
		// response by default, and anything else (description, components,
		// versions) is payload we'd immediately discard. The fetch itself is
		// still necessary because numeric-ID inputs need to be canonicalized
		// to a key.
		project, err := client.GetProject(ctx, keyOrID, "")
		if err != nil {
			return err
		}
		return jtkpresent.EmitIDs(opts, []string{project.Key})
	}

	if fieldsFlag != "" && v.Format == view.FormatJSON {
		return jtkpresent.ErrFieldsWithJSON
	}

	selected, projected, err := projection.Resolve(
		ctx,
		jtkpresent.ProjectDetailSpec,
		opts.IsExtended(),
		fieldsFlag,
		noFieldFetch,
		"projects get",
	)
	if err != nil {
		return err
	}

	project, err := client.GetProject(ctx, keyOrID, api.ProjectGetExpand)
	if err != nil {
		return err
	}

	if v.Format == view.FormatJSON {
		return v.JSON(project)
	}

	presenter := jtkpresent.ProjectPresenter{}
	if projected {
		model := presenter.PresentProjectDetailProjection(project)
		projection.ApplyToDetailInModel(model, selected)
		return jtkpresent.Emit(opts, model)
	}
	return jtkpresent.Emit(opts, presenter.PresentProjectDetail(project, opts.IsExtended()))
}
