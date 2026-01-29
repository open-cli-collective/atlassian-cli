package issues

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
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

  # List options using field ID
  jtk issues field-options customfield_10001 --issue PROJ-123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFieldOptions(opts, args[0], issueKey)
		},
	}

	cmd.Flags().StringVar(&issueKey, "issue", "", "Issue key for context-specific options (recommended)")

	return cmd
}

func runFieldOptions(opts *root.Options, fieldNameOrID, issueKey string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// Get all fields to resolve name to ID
	fields, err := client.GetFields()
	if err != nil {
		return err
	}

	// Resolve field name/ID
	fieldID, err := api.ResolveFieldID(fields, fieldNameOrID)
	if err != nil {
		return err
	}

	// Get field info for display
	field := api.FindFieldByID(fields, fieldID)
	fieldName := fieldID
	if field != nil {
		fieldName = field.Name
	}

	// Get options
	var options []api.FieldOptionValue

	if issueKey != "" {
		// Use edit metadata for issue-specific context
		options, err = client.GetFieldOptionsFromEditMeta(issueKey, fieldID)
		if err != nil {
			return fmt.Errorf("failed to get options for field %s: %w", fieldName, err)
		}
	} else {
		// Try to get options without issue context
		options, err = client.GetFieldOptions(fieldID)
		if err != nil {
			v.Warning("Could not get field options without issue context. Use --issue flag for better results.")
			return fmt.Errorf("failed to get options for field %s: %w", fieldName, err)
		}
	}

	if len(options) == 0 {
		v.Info("No options found for field '%s'", fieldName)
		return nil
	}

	if opts.Output == "json" {
		return v.JSON(options)
	}

	// Display options table
	v.Info("Allowed values for field '%s':", fieldName)

	headers := []string{"VALUE", "ID"}
	var rows [][]string

	for _, opt := range options {
		value := opt.Value
		if value == "" {
			value = opt.Name
		}
		id := opt.ID
		if opt.Disabled {
			value = value + " (disabled)"
		}
		rows = append(rows, []string{value, id})
	}

	return v.Table(headers, rows)
}
