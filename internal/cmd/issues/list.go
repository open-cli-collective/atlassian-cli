package issues

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newListCmd(opts *root.Options) *cobra.Command {
	var project string
	var sprint string
	var maxResults int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List issues",
		Long:  "List issues, optionally filtered by project and/or sprint.",
		Example: `  # List issues in a project
  jira-ticket-cli issues list --project MYPROJECT

  # List issues in the current sprint
  jira-ticket-cli issues list --project MYPROJECT --sprint current

  # List issues with custom limit
  jira-ticket-cli issues list --project MYPROJECT --max 100`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opts, project, sprint, maxResults)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Filter by project key")
	cmd.Flags().StringVarP(&sprint, "sprint", "s", "", "Filter by sprint (use 'current' for active sprint)")
	cmd.Flags().IntVarP(&maxResults, "max", "m", 50, "Maximum number of results")

	return cmd
}

func runList(opts *root.Options, project, sprint string, maxResults int) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// Build JQL query
	var jql string
	if project != "" {
		jql = fmt.Sprintf("project = %s", project)
	}

	if sprint != "" {
		sprintClause := ""
		if sprint == "current" {
			sprintClause = "sprint in openSprints()"
		} else {
			sprintClause = fmt.Sprintf("sprint = \"%s\"", sprint)
		}

		if jql != "" {
			jql += " AND " + sprintClause
		} else {
			jql = sprintClause
		}
	}

	if jql == "" {
		jql = "ORDER BY updated DESC"
	} else {
		jql += " ORDER BY updated DESC"
	}

	issues, err := client.SearchAll(jql, maxResults)
	if err != nil {
		return err
	}

	if len(issues) == 0 {
		v.Info("No issues found")
		return nil
	}

	// For JSON output
	if opts.Output == "json" {
		return v.JSON(issues)
	}

	headers := []string{"KEY", "SUMMARY", "STATUS", "ASSIGNEE", "TYPE"}
	var rows [][]string

	for _, issue := range issues {
		status := ""
		if issue.Fields.Status != nil {
			status = issue.Fields.Status.Name
		}

		assignee := ""
		if issue.Fields.Assignee != nil {
			assignee = issue.Fields.Assignee.DisplayName
		}

		issueType := ""
		if issue.Fields.IssueType != nil {
			issueType = issue.Fields.IssueType.Name
		}

		rows = append(rows, formatIssueRow(issue.Key, issue.Fields.Summary, status, assignee, issueType))
	}

	return v.Table(headers, rows)
}
