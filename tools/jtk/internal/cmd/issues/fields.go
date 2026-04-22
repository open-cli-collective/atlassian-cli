package issues

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

func newFieldsCmd(opts *root.Options) *cobra.Command {
	var customOnly bool

	cmd := &cobra.Command{
		Use:   "fields [issue-key]",
		Short: "List available fields",
		Long:  "List fields that can be used when creating or updating issues. If an issue key is provided, shows the editable fields for that specific issue.",
		Example: `  # List all fields
  jtk issues fields

  # List only custom fields
  jtk issues fields --custom-fields

  # List editable fields for a specific issue
  jtk issues fields PROJ-123

  # Extended output with searchable/navigable/orderable/clause names
  jtk issues fields --extended

  # Emit only field IDs
  jtk issues fields --id`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueKey := ""
			if len(args) > 0 {
				issueKey = args[0]
			}
			return runFields(cmd.Context(), opts, issueKey, customOnly)
		},
	}

	cmd.Flags().BoolVar(&customOnly, "custom-fields", false, "Show only custom fields")
	cmd.Flags().BoolVar(&customOnly, "custom", false, "Show only custom fields")
	_ = cmd.Flags().MarkHidden("custom")

	return cmd
}

func runFields(ctx context.Context, opts *root.Options, issueKey string, customOnly bool) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if issueKey != "" {
		return runEditableFields(ctx, opts, client, issueKey)
	}
	return runGlobalFields(ctx, opts, client, customOnly)
}

func runGlobalFields(ctx context.Context, opts *root.Options, client *api.Client, customOnly bool) error {
	var fields []api.Field
	var err error
	if customOnly {
		fields, err = client.GetCustomFields(ctx)
	} else {
		fields, err = client.GetFields(ctx)
	}
	if err != nil {
		return err
	}

	if opts.EmitIDOnly() {
		ids := make([]string, len(fields))
		for i, f := range fields {
			ids[i] = f.ID
		}
		return jtkpresent.EmitIDs(opts, ids)
	}

	if len(fields) == 0 {
		return jtkpresent.Emit(opts, jtkpresent.FieldPresenter{}.PresentEmpty())
	}

	v := opts.View()
	if v.Format == view.FormatJSON {
		return v.JSON(fields)
	}

	model := jtkpresent.FieldPresenter{}.PresentList(fields, opts.IsExtended())
	return jtkpresent.Emit(opts, model)
}

func runEditableFields(ctx context.Context, opts *root.Options, client *api.Client, issueKey string) error {
	meta, err := client.GetIssueEditMeta(ctx, issueKey)
	if err != nil {
		return err
	}

	v := opts.View()
	if v.Format == view.FormatJSON {
		return v.JSON(meta)
	}

	fieldsData, ok := meta["fields"].(map[string]any)
	if !ok {
		return jtkpresent.Emit(opts, jtkpresent.IssuePresenter{}.PresentNoEditableFields(issueKey))
	}

	editableFields := api.ParseEditMeta(fieldsData)

	if opts.EmitIDOnly() {
		ids := make([]string, len(editableFields))
		for i, f := range editableFields {
			ids[i] = f.ID
		}
		return jtkpresent.EmitIDs(opts, ids)
	}

	model := jtkpresent.FieldPresenter{}.PresentEditableFields(editableFields)
	return jtkpresent.Emit(opts, model)
}
