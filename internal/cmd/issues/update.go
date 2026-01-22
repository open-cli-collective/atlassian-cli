package issues

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newUpdateCmd(opts *root.Options) *cobra.Command {
	var summary string
	var description string
	var fields []string

	cmd := &cobra.Command{
		Use:   "update <issue-key>",
		Short: "Update an issue",
		Long:  "Update fields on an existing Jira issue.",
		Example: `  # Update summary
  jira-ticket-cli issues update PROJ-123 --summary "New summary"

  # Update description
  jira-ticket-cli issues update PROJ-123 --description "Updated description"

  # Update custom fields
  jira-ticket-cli issues update PROJ-123 --field priority=High --field "Story Points"=5`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(opts, args[0], summary, description, fields)
		},
	}

	cmd.Flags().StringVarP(&summary, "summary", "s", "", "New summary")
	cmd.Flags().StringVarP(&description, "description", "d", "", "New description")
	cmd.Flags().StringArrayVarP(&fields, "field", "f", nil, "Fields to update (key=value)")

	return cmd
}

func runUpdate(opts *root.Options, issueKey, summary, description string, fieldArgs []string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	fields := make(map[string]interface{})

	if summary != "" {
		fields["summary"] = summary
	}

	if description != "" {
		fields["description"] = api.NewADFDocument(description)
	}

	// Parse additional fields
	if len(fieldArgs) > 0 {
		allFields, err := client.GetFields()
		if err != nil {
			return fmt.Errorf("failed to get field metadata: %w", err)
		}

		for _, f := range fieldArgs {
			parts := strings.SplitN(f, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid field format: %s (expected key=value)", f)
			}

			key, value := parts[0], parts[1]

			fieldID, err := api.ResolveFieldID(allFields, key)
			if err != nil {
				fieldID = key
			}

			fields[fieldID] = value
		}
	}

	if len(fields) == 0 {
		return fmt.Errorf("no fields specified to update")
	}

	req := api.BuildUpdateRequest(fields)

	if err := client.UpdateIssue(issueKey, req); err != nil {
		return err
	}

	v.Success("Updated issue %s", issueKey)
	return nil
}
