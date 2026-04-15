package issues

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/text"
)

func newCreateCmd(opts *root.Options) *cobra.Command {
	var project string
	var issueType string
	var summary string
	var description string
	var parent string
	var assignee string
	var fields []string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new issue",
		Long:  "Create a new Jira issue with the specified fields.",
		Example: `  # Create a basic task
  jtk issues create --project MYPROJECT --type Task --summary "Fix login bug"

  # Create with description
  jtk issues create --project MYPROJECT --type Bug --summary "Login fails" --description "Users cannot log in with SSO"

  # Create as child of an epic
  jtk issues create --project MYPROJECT --type Task --summary "Subtask" --parent MYPROJECT-100

  # Assign to yourself
  jtk issues create --project MYPROJECT --type Task --summary "My task" --assignee me

  # Assign by email
  jtk issues create --project MYPROJECT --type Task --summary "Their task" --assignee user@example.com

  # Create with custom fields
  jtk issues create --project MYPROJECT --type Story --summary "New feature" --field priority=High`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCreate(cmd.Context(), opts, project, issueType, summary, description, parent, assignee, fields)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Project key (required)")
	cmd.Flags().StringVarP(&issueType, "type", "t", "Task", "Issue type (Task, Bug, Story, etc.)")
	cmd.Flags().StringVarP(&summary, "summary", "s", "", "Issue summary (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Issue description")
	cmd.Flags().StringVar(&parent, "parent", "", "Parent issue key (epic or parent issue)")
	cmd.Flags().StringVarP(&assignee, "assignee", "a", "", "Assignee (account ID, email, or \"me\")")
	cmd.Flags().StringArrayVarP(&fields, "field", "f", nil, "Additional fields (key=value)")

	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("summary")

	return cmd
}

func runCreate(ctx context.Context, opts *root.Options, project, issueType, summary, description, parent, assignee string, fieldArgs []string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// Parse additional fields
	extraFields := make(map[string]any)
	if len(fieldArgs) > 0 {
		// Get field metadata to resolve names to IDs
		allFields, err := client.GetFields(ctx)
		if err != nil {
			return fmt.Errorf("getting field metadata: %w", err)
		}

		for _, f := range fieldArgs {
			parts := strings.SplitN(f, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid field format: %s (expected key=value)", f)
			}

			key, value := parts[0], parts[1]

			// Try to resolve field name to ID and get field info
			var fieldID string
			var field *api.Field
			if resolved := api.FindFieldByName(allFields, key); resolved != nil {
				fieldID = resolved.ID
				field = resolved
			} else if resolved := api.FindFieldByID(allFields, key); resolved != nil {
				fieldID = resolved.ID
				field = resolved
			} else {
				fieldID = key
			}

			// Format value based on field type, merging with existing if same key repeated
			formatted := api.FormatFieldValue(field, value)
			if existing, ok := extraFields[fieldID]; ok {
				extraFields[fieldID] = api.MergeFieldValues(existing, formatted)
			} else {
				extraFields[fieldID] = formatted
			}
		}
	}

	if parent != "" {
		extraFields["parent"] = map[string]string{"key": parent}
	}

	if assignee != "" {
		accountID, err := resolveAssignee(ctx, client, assignee)
		if err != nil {
			return err
		}
		extraFields["assignee"] = map[string]string{"accountId": accountID}
	}

	req := api.BuildCreateRequest(project, issueType, summary, text.InterpretEscapes(description), extraFields)

	issue, err := client.CreateIssue(ctx, req)
	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(issue)
	}

	// Success message includes the URL for convenience
	model := jtkpresent.IssuePresenter{}.PresentCreated(issue.Key, client.IssueURL(issue.Key))
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}
