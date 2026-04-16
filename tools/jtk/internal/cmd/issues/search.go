package issues

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
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
  jtk issues search --jql "project = MYPROJECT" --fields summary,status,customfield_10005`,
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
		model := jtkpresent.IssuePresenter{}.PresentEmpty()
		out := present.Render(model, opts.RenderStyle())
		_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
		return nil
	}

	if v.Format == view.FormatJSON {
		arts := jtkartifact.ProjectIssues(result.Issues, opts.ArtifactMode())
		hasMore := !result.Pagination.IsLast
		return v.RenderArtifactList(artifact.NewListResult(arts, hasMore))
	}

	// Text path: presenter → render → write
	hasMore := !result.Pagination.IsLast
	model := jtkpresent.IssuePresenter{}.PresentListWithPagination(result.Issues, hasMore)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)

	return nil
}
