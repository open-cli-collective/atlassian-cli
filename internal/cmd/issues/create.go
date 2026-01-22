package issues

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newCreateCmd(opts *root.Options) *cobra.Command {
	var project string
	var issueType string
	var summary string
	var description string
	var fields []string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new issue",
		Long:  "Create a new Jira issue with the specified fields.",
		Example: `  # Create a basic task
  jira-ticket-cli issues create --project MYPROJECT --type Task --summary "Fix login bug"

  # Create with description
  jira-ticket-cli issues create --project MYPROJECT --type Bug --summary "Login fails" --description "Users cannot log in with SSO"

  # Create with custom fields
  jira-ticket-cli issues create --project MYPROJECT --type Story --summary "New feature" --field priority=High`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(opts, project, issueType, summary, description, fields)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Project key (required)")
	cmd.Flags().StringVarP(&issueType, "type", "t", "Task", "Issue type (Task, Bug, Story, etc.)")
	cmd.Flags().StringVarP(&summary, "summary", "s", "", "Issue summary (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Issue description")
	cmd.Flags().StringArrayVarP(&fields, "field", "f", nil, "Additional fields (key=value)")

	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("summary")

	return cmd
}

func runCreate(opts *root.Options, project, issueType, summary, description string, fieldArgs []string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// Parse additional fields
	extraFields := make(map[string]interface{})
	if len(fieldArgs) > 0 {
		// Get field metadata to resolve names to IDs
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

			// Try to resolve field name to ID
			fieldID, err := api.ResolveFieldID(allFields, key)
			if err != nil {
				// Use the key as-is if not found
				fieldID = key
			}

			extraFields[fieldID] = value
		}
	}

	req := api.BuildCreateRequest(project, issueType, summary, description, extraFields)

	issue, err := client.CreateIssue(req)
	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(issue)
	}

	v.Success("Created issue %s", issue.Key)
	v.Info("URL: %s", client.IssueURL(issue.Key))

	return nil
}
