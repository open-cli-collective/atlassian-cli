package issues

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

func newFieldOptionsCmd(opts *root.Options) *cobra.Command {
	var issueKey string

	cmd := &cobra.Command{
		Use:   "field-options <field-name-or-id>",
		Short: "List allowed values for a field",
		Long: `List the allowed values for an option/select field.

When used with --issue, shows the allowed values in the context of that specific issue.
Without --issue, attempts to show all possible values for the field.`,
		Example: `  # List options for a field using issue context
  jtk issues field-options "Priority" --issue PROJ-123

  # Emit only option IDs
  jtk issues field-options "Priority" --issue PROJ-123 --id`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFieldOptions(cmd.Context(), opts, args[0], issueKey)
		},
	}

	cmd.Flags().StringVar(&issueKey, "issue", "", "Issue key for context-specific options (recommended)")

	return cmd
}

func runFieldOptions(ctx context.Context, opts *root.Options, fieldNameOrID, issueKey string) error {
	fp := jtkpresent.FieldPresenter{}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	fields, err := client.GetFields(ctx)
	if err != nil {
		return err
	}

	fieldID, err := api.ResolveFieldID(fields, fieldNameOrID)
	if err != nil {
		return err
	}

	field := api.FindFieldByID(fields, fieldID)
	fieldName := fieldID
	if field != nil {
		fieldName = field.Name
	}

	var options []api.FieldOptionValue

	if issueKey != "" {
		options, err = client.GetFieldOptionsFromEditMeta(ctx, issueKey, fieldID)
		if err != nil {
			return fmt.Errorf("getting options for field %s: %w", fieldName, err)
		}
	} else {
		options, err = client.GetFieldOptions(ctx, fieldID)
		if err != nil {
			warnModel := fp.PresentOptionsNoContext()
			_ = jtkpresent.Emit(opts, warnModel)
			return fmt.Errorf("getting options for field %s: %w", fieldName, err)
		}
	}

	if len(options) == 0 {
		return jtkpresent.Emit(opts, fp.PresentNoOptions(fieldID))
	}

	if opts.EmitIDOnly() {
		ids := make([]string, len(options))
		for i, opt := range options {
			ids[i] = opt.ID
		}
		return jtkpresent.EmitIDs(opts, ids)
	}

	v := opts.View()
	if v.Format == view.FormatJSON {
		return v.JSON(options)
	}

	model := fp.PresentFieldOptionsWithHeader(fieldName, options)
	return jtkpresent.Emit(opts, model)
}
