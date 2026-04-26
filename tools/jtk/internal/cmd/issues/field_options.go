package issues

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

func newFieldOptionsCmd(opts *root.Options) *cobra.Command {
	var issueFlag string

	cmd := &cobra.Command{
		Use:   "field-options <issue-key> <field-name-or-id>",
		Short: "List allowed values for a field",
		Long: `List the allowed values for an option/select field.

Provide an issue key and a field name or ID to see allowed values in that
issue's project context. For read-only fields that don't appear in edit
metadata, the default field context is used.`,
		Example: `  # List options using issue context (recommended)
  jtk issues field-options PROJ-123 "Priority"

  # List options for a custom field
  jtk issues field-options PROJ-123 customfield_10050

  # List options without issue context (global)
  jtk issues field-options "Priority"

  # Emit only option IDs
  jtk issues field-options PROJ-123 "Priority" --id`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var issueKey, fieldNameOrID string
			if len(args) == 2 {
				issueKey = args[0]
				fieldNameOrID = args[1]
			} else {
				fieldNameOrID = args[0]
				if issueFlag != "" {
					issueKey = issueFlag
				}
			}
			return runFieldOptions(cmd.Context(), opts, fieldNameOrID, issueKey)
		},
	}

	cmd.Flags().StringVar(&issueFlag, "issue", "", "Issue key for context-specific options")
	_ = cmd.Flags().MarkDeprecated("issue", "use positional arg: jtk issues field-options <issue-key> <field>")

	return cmd
}

func runFieldOptions(ctx context.Context, opts *root.Options, fieldNameOrID, issueKey string) error {
	fp := jtkpresent.FieldPresenter{}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	fields, err := cache.GetFieldsCacheFirst(ctx, client)
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
		options, err = api.ResolveFieldOptions(ctx, client, issueKey, fieldID)
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
