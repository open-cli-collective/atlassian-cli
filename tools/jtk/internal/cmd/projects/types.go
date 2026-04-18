package projects

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

func newTypesCmd(opts *root.Options) *cobra.Command {
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "types",
		Short: "List project types",
		Long:  "List available project types for creating new projects.",
		Example: `  jtk projects types

  # Include DESCRIPTION_KEY column
  jtk projects types --extended

  # Emit just the project-type keys
  jtk projects types --id

  # Project output to selected columns
  jtk projects types --fields KEY`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTypes(cmd.Context(), opts, fieldsFlag)
		},
	}

	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display columns (ProjectTypeSpec headers)")

	return cmd
}

func runTypes(ctx context.Context, opts *root.Options, fieldsFlag string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	idOnly := opts.EmitIDOnly()

	if !idOnly && fieldsFlag != "" && v.Format == view.FormatJSON {
		return jtkpresent.ErrFieldsWithJSON
	}

	var selected []projection.ColumnSpec
	var projected bool
	if !idOnly {
		selected, projected, err = projection.Resolve(
			ctx,
			jtkpresent.ProjectTypeSpec,
			opts.IsExtended(),
			fieldsFlag,
			noFieldFetch,
			"projects types",
		)
		if err != nil {
			return err
		}
	}

	types, err := client.ListProjectTypes(ctx)
	if err != nil {
		return err
	}

	if idOnly {
		ids := make([]string, len(types))
		for i, t := range types {
			ids[i] = t.Key
		}
		return jtkpresent.EmitIDs(opts, ids)
	}

	if len(types) == 0 {
		return jtkpresent.Emit(opts, jtkpresent.ProjectPresenter{}.PresentNoTypes())
	}

	if v.Format == view.FormatJSON {
		return v.JSON(types)
	}

	presenter := jtkpresent.ProjectPresenter{}
	model := presenter.PresentProjectTypes(types, opts.IsExtended())
	if projected {
		projection.ApplyToTableInModel(model, selected)
	}
	return jtkpresent.Emit(opts, model)
}
