package issues

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newGetCmd(opts *root.Options) *cobra.Command {
	var noTruncate bool

	cmd := &cobra.Command{
		Use:   "get <issue-key>",
		Short: "Get issue details",
		Long:  "Retrieve and display details for a specific issue.",
		Example: `  jtk issues get PROJ-123
  jtk issues get PROJ-123 --no-truncate
  jtk issues get PROJ-123 -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(cmd.Context(), opts, args[0], noTruncate)
		},
	}

	cmd.Flags().BoolVar(&noTruncate, "no-truncate", false, "Show full description without truncation")

	return cmd
}

func runGet(ctx context.Context, opts *root.Options, issueKey string, noTruncate bool) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	issue, err := client.GetIssue(ctx, issueKey)
	if err != nil {
		return err
	}

	// For JSON output, return the projected artifact
	if v.Format == view.FormatJSON {
		return v.RenderArtifact(jtkartifact.ProjectIssue(issue, opts.ArtifactMode()))
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
		if !noTruncate && len(description) > 200 {
			description = description[:200] + "... [truncated, use --no-truncate for complete text]"
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

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func formatAssignee(name string) string {
	if name == "" {
		return "Unassigned"
	}
	return name
}

func formatIssueRow(key, summary, status, assignee, issueType string) []string {
	return []string{
		key,
		view.Truncate(summary, 50),
		orDash(status),
		formatAssignee(assignee),
		orDash(issueType),
	}
}

// safeString extracts string from an interface value
func safeString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
