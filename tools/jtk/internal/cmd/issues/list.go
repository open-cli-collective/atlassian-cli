package issues

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newListCmd(opts *root.Options) *cobra.Command {
	var project string
	var sprint string
	var maxResults int
	var nextPageToken string
	var full bool
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List issues",
		Long:  "List issues, optionally filtered by project and/or sprint.",
		Example: `  # List issues in a project
  jtk issues list --project MYPROJECT

  # List issues in the current sprint
  jtk issues list --project MYPROJECT --sprint current

  # Get up to 200 results (auto-paginates)
  jtk issues list --project MYPROJECT --max 200

  # Resume from a previous page token
  jtk issues list --project MYPROJECT --next-page-token <token>

  # List with full details (includes description)
  jtk issues list --project MYPROJECT --full

  # List with specific fields (e.g. custom fields)
  jtk issues list --project MYPROJECT --fields summary,status,customfield_10005 -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts, project, sprint, maxResults, nextPageToken, full, fieldsFlag)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Filter by project key")
	cmd.Flags().StringVarP(&sprint, "sprint", "s", "", "Filter by sprint (use 'current' for active sprint)")
	cmd.Flags().IntVarP(&maxResults, "max", "m", 25, "Maximum number of results to return")
	cmd.Flags().StringVar(&nextPageToken, "next-page-token", "", "Token for next page of results")
	cmd.Flags().BoolVar(&full, "full", false, "Include all fields (e.g. description)")
	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated list of fields to return (e.g. summary,customfield_10005)")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, project, sprint string, maxResults int, nextPageToken string, full bool, fieldsFlag string) error {
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

	fields := resolveFields(fieldsFlag, opts.Output, full)

	result, err := client.SearchPage(ctx, api.SearchPageOptions{
		JQL:           jql,
		MaxResults:    maxResults,
		Fields:        fields,
		NextPageToken: nextPageToken,
	})
	if err != nil {
		return err
	}

	if len(result.Issues) == 0 {
		v.Info("No issues found")
		return nil
	}

	// For JSON output, return the paginated wrapper
	if opts.Output == "json" {
		return v.JSON(result)
	}

	headers := []string{"KEY", "SUMMARY", "STATUS", "ASSIGNEE", "TYPE"}
	rows := make([][]string, 0, len(result.Issues))

	for _, issue := range result.Issues {
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

	if err := v.Table(headers, rows); err != nil {
		return err
	}

	// Print pagination footer on stderr when there are more results
	if !result.Pagination.IsLast {
		v.Info("More results available (use --next-page-token to fetch next page)")
	}

	return nil
}
