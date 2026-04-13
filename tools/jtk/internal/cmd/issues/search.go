package issues

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newSearchCmd(opts *root.Options) *cobra.Command {
	var jql string
	var maxResults int
	var nextPageToken string
	var allFields bool
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search issues using JQL",
		Long:  "Search for issues using Jira Query Language (JQL).",
		Example: `  # Search by JQL
  jtk issues search --jql "project = MYPROJECT AND status = 'In Progress'"

  # Search for recent issues
  jtk issues search --jql "project = MYPROJECT AND updated >= -7d"

  # Get up to 200 results (auto-paginates)
  jtk issues search --jql "project = MYPROJECT" --max 200

  # Resume from a previous page token
  jtk issues search --jql "project = MYPROJECT" --next-page-token <token>

  # Search with all fields (includes description)
  jtk issues search --jql "project = MYPROJECT" --all-fields

  # Search with specific fields (e.g. custom fields)
  jtk issues search --jql "project = MYPROJECT" --fields summary,status,customfield_10005 -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSearch(cmd.Context(), opts, jql, maxResults, nextPageToken, allFields, fieldsFlag)
		},
	}

	cmd.Flags().StringVar(&jql, "jql", "", "JQL query string (required)")
	cmd.Flags().IntVarP(&maxResults, "max", "m", 25, "Maximum number of results to return")
	cmd.Flags().StringVar(&nextPageToken, "next-page-token", "", "Token for next page of results")
	cmd.Flags().BoolVar(&allFields, "all-fields", false, "Include all fields (e.g. description)")
	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated list of fields to return (e.g. summary,customfield_10005)")
	_ = cmd.MarkFlagRequired("jql")

	return cmd
}

func runSearch(ctx context.Context, opts *root.Options, jql string, maxResults int, nextPageToken string, allFields bool, fieldsFlag string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	fields := resolveFields(fieldsFlag, opts.Output, allFields)

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

	if !result.Pagination.IsLast {
		v.Info("More results available (use --next-page-token to fetch next page)")
	}

	return nil
}
