package issues

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newFieldsCmd(opts *root.Options) *cobra.Command {
	var customOnly bool

	cmd := &cobra.Command{
		Use:   "fields [issue-key]",
		Short: "List available fields",
		Long:  "List fields that can be used when creating or updating issues. If an issue key is provided, shows the editable fields for that specific issue.",
		Example: `  # List all fields
  jira-ticket-cli issues fields

  # List only custom fields
  jira-ticket-cli issues fields --custom

  # List editable fields for a specific issue
  jira-ticket-cli issues fields PROJ-123`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueKey := ""
			if len(args) > 0 {
				issueKey = args[0]
			}
			return runFields(opts, issueKey, customOnly)
		},
	}

	cmd.Flags().BoolVar(&customOnly, "custom", false, "Show only custom fields")

	return cmd
}

func runFields(opts *root.Options, issueKey string, customOnly bool) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if issueKey != "" {
		// Get editable fields for a specific issue
		meta, err := client.GetIssueEditMeta(issueKey)
		if err != nil {
			return err
		}

		if opts.Output == "json" {
			return v.JSON(meta)
		}

		// Extract field information from metadata
		fieldsData, ok := meta["fields"].(map[string]interface{})
		if !ok {
			v.Info("No editable fields found for %s", issueKey)
			return nil
		}

		headers := []string{"ID", "NAME", "TYPE", "REQUIRED"}
		var rows [][]string

		for id, data := range fieldsData {
			fieldData, ok := data.(map[string]interface{})
			if !ok {
				continue
			}

			name := safeString(fieldData["name"])
			required := "no"
			if req, ok := fieldData["required"].(bool); ok && req {
				required = "yes"
			}

			// Get schema type
			fieldType := ""
			if schema, ok := fieldData["schema"].(map[string]interface{}); ok {
				fieldType = safeString(schema["type"])
			}

			rows = append(rows, []string{id, name, fieldType, required})
		}

		return v.Table(headers, rows)
	}

	// List all fields
	var fields []api.Field
	if customOnly {
		fields, err = client.GetCustomFields()
	} else {
		fields, err = client.GetFields()
	}

	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(fields)
	}

	headers := []string{"ID", "NAME", "TYPE", "CUSTOM"}
	var rows [][]string

	for _, f := range fields {
		custom := "no"
		if f.Custom {
			custom = "yes"
		}
		rows = append(rows, []string{f.ID, f.Name, f.Schema.Type, custom})
	}

	return v.Table(headers, rows)
}
