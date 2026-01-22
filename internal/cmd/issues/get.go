package issues

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newGetCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "get <issue-key>",
		Short: "Get issue details",
		Long:  "Retrieve and display details for a specific issue.",
		Example: `  jira-ticket-cli issues get PROJ-123
  jira-ticket-cli issues get PROJ-123 -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(opts, args[0])
		},
	}
}

func runGet(opts *root.Options, issueKey string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	issue, err := client.GetIssue(issueKey)
	if err != nil {
		return err
	}

	// For JSON output, return the full issue
	if opts.Output == "json" {
		return v.JSON(issue)
	}

	// For table/plain output, display key details
	status := ""
	if issue.Fields.Status != nil {
		status = issue.Fields.Status.Name
	}

	issueType := ""
	if issue.Fields.IssueType != nil {
		issueType = issue.Fields.IssueType.Name
	}

	assignee := "Unassigned"
	if issue.Fields.Assignee != nil {
		assignee = issue.Fields.Assignee.DisplayName
	}

	priority := ""
	if issue.Fields.Priority != nil {
		priority = issue.Fields.Priority.Name
	}

	project := ""
	if issue.Fields.Project != nil {
		project = issue.Fields.Project.Key
	}

	description := ""
	if issue.Fields.Description != nil {
		description = issue.Fields.Description.ToPlainText()
		if len(description) > 200 {
			description = description[:200] + "..."
		}
	}

	v.Println("Key:         %s", issue.Key)
	v.Println("Summary:     %s", issue.Fields.Summary)
	v.Println("Status:      %s", status)
	v.Println("Type:        %s", issueType)
	v.Println("Priority:    %s", priority)
	v.Println("Assignee:    %s", assignee)
	v.Println("Project:     %s", project)
	if description != "" {
		v.Println("Description: %s", description)
	}
	v.Println("URL:         %s", client.IssueURL(issue.Key))

	return nil
}

func formatAssignee(name string) string {
	if name == "" {
		return "Unassigned"
	}
	return name
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func formatIssueRow(key, summary, status, assignee, issueType string) []string {
	return []string{
		key,
		truncate(summary, 50),
		orDash(status),
		formatAssignee(assignee),
		orDash(issueType),
	}
}

// Helper to safely extract string fields
func safeString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
