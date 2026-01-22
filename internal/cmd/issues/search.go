package issues

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newSearchCmd(opts *root.Options) *cobra.Command {
	var jql string
	var maxResults int

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search issues using JQL",
		Long:  "Search for issues using Jira Query Language (JQL).",
		Example: `  # Search by JQL
  jira-ticket-cli issues search --jql "project = MYPROJECT AND status = 'In Progress'"

  # Search for recent issues
  jira-ticket-cli issues search --jql "project = MYPROJECT AND updated >= -7d"

  # Search issues assigned to current user
  jira-ticket-cli issues search --jql "assignee = currentUser() AND resolution = Unresolved"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(opts, jql, maxResults)
		},
	}

	cmd.Flags().StringVar(&jql, "jql", "", "JQL query string (required)")
	cmd.Flags().IntVarP(&maxResults, "max", "m", 50, "Maximum number of results")
	_ = cmd.MarkFlagRequired("jql")

	return cmd
}

func runSearch(opts *root.Options, jql string, maxResults int) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	issues, err := client.SearchAll(jql, maxResults)
	if err != nil {
		return err
	}

	if len(issues) == 0 {
		v.Info("No issues found")
		return nil
	}

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
