package issues

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"

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
  jtk issues fields --custom

  # List editable fields for a specific issue
  jtk issues fields PROJ-123`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueKey := ""
			if len(args) > 0 {
				issueKey = args[0]
			}
			return runFields(cmd.Context(), opts, issueKey, customOnly)
		},
	}

	cmd.Flags().BoolVar(&customOnly, "custom", false, "Show only custom fields")

	return cmd
}

func runFields(ctx context.Context, opts *root.Options, issueKey string, customOnly bool) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if issueKey != "" {
		// Get editable fields for a specific issue
		meta, err := client.GetIssueEditMeta(ctx, issueKey)
		if err != nil {
			return err
		}

		if opts.Output == "json" {
			return v.JSON(meta)
		}

		// Extract field information from metadata
		fieldsData, ok := meta["fields"].(map[string]any)
		if !ok {
			model := jtkpresent.IssuePresenter{}.PresentNoEditableFields(issueKey)
			out := present.Render(model, opts.RenderStyle())
			_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
			return nil
		}

		// Build editable fields list
		editableFields := make([]jtkpresent.EditableField, 0, len(fieldsData))

		for id, data := range fieldsData {
			fieldData, ok := data.(map[string]any)
			if !ok {
				continue
			}

			name := safeString(fieldData["name"])
			required := false
			if req, ok := fieldData["required"].(bool); ok && req {
				required = true
			}

			// Get schema type
			fieldType := ""
			if schema, ok := fieldData["schema"].(map[string]any); ok {
				fieldType = safeString(schema["type"])
			}

			editableFields = append(editableFields, jtkpresent.EditableField{
				ID:       id,
				Name:     name,
				Type:     fieldType,
				Required: required,
			})
		}

		model := jtkpresent.FieldPresenter{}.PresentEditableFields(editableFields)
		out := present.Render(model, opts.RenderStyle())
		_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
		_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
		return nil
	}

	// List all fields
	var fields []api.Field
	if customOnly {
		fields, err = client.GetCustomFields(ctx)
	} else {
		fields, err = client.GetFields(ctx)
	}

	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(fields)
	}

	model := jtkpresent.FieldPresenter{}.PresentList(fields)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}
