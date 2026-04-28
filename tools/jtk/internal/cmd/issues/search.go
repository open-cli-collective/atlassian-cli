package issues

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
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

  # Project display columns — headers, Jira field IDs, or human names
  jtk issues search --jql "project = MYPROJECT" --fields SUMMARY,STATUS
  jtk issues search --jql "project = MYPROJECT" --fields "Issue Type"`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSearch(cmd.Context(), opts, jql, maxResults, nextPageToken, allFields, fieldsFlag)
		},
	}

	cmd.Flags().StringVar(&jql, "jql", "", "JQL query string (required)")
	cmd.Flags().IntVarP(&maxResults, "max", "m", 25, "Maximum number of results to return")
	cmd.Flags().StringVar(&nextPageToken, "next-page-token", "", "Token for next page of results")
	cmd.Flags().BoolVar(&allFields, "all-fields", false, "Include all fields (e.g. description)")
	_ = cmd.Flags().MarkDeprecated("all-fields", "use --fields description instead")
	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display columns (headers, Jira field IDs, or human names)")
	_ = cmd.MarkFlagRequired("jql")

	return cmd
}

func runSearch(ctx context.Context, opts *root.Options, jql string, maxResults int, nextPageToken string, allFields bool, fieldsFlag string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// --id wins over --fields: skip projection entirely when --id is set.
	// See list.go for rationale.
	idOnly := opts.EmitIDOnly()

	if !idOnly && fieldsFlag != "" && v.Format == view.FormatJSON {
		return jtkpresent.ErrFieldsWithJSON
	}

	var selected []projection.ColumnSpec
	var projected bool
	if !idOnly {
		var err error
		selected, projected, err = projection.Resolve(
			ctx,
			jtkpresent.IssueListSpec,
			opts.IsExtended(),
			fieldsFlag,
			fieldsFetcher(client),
			"issues search",
		)
		if err != nil {
			return err
		}
	}

	fields := deriveFetchFields(selected, projected, opts.IsExtended(), allFields, opts.Output)

	result, err := client.SearchPage(ctx, api.SearchPageOptions{
		JQL:           jql,
		MaxResults:    maxResults,
		Fields:        fields,
		NextPageToken: nextPageToken,
	})
	if err != nil {
		return err
	}

	hasMore := !result.Pagination.IsLast
	nextToken := result.Pagination.NextPageToken

	if idOnly {
		ids := make([]string, len(result.Issues))
		for i, issue := range result.Issues {
			ids[i] = issue.Key
		}
		return jtkpresent.EmitIDsWithPaginationToken(opts, ids, hasMore, nextToken)
	}

	if len(result.Issues) == 0 {
		if hasMore {
			return jtkpresent.Emit(opts, jtkpresent.PaginationOnlyModel(nextToken))
		}
		return jtkpresent.Emit(opts, jtkpresent.IssuePresenter{}.PresentEmpty())
	}

	if v.Format == view.FormatJSON {
		arts := jtkartifact.ProjectIssues(result.Issues, opts.ArtifactMode())
		return v.RenderArtifactList(artifact.NewListResult(arts, hasMore))
	}

	model := jtkpresent.IssuePresenter{}.PresentListWithPagination(result.Issues, opts.IsExtended(), hasMore, nextToken)
	if projected {
		jtkpresent.AppendDynamicTableColumns(model, result.Issues, projection.DynamicSpecs(selected))
		projection.ApplyToTableInModel(model, selected)
	}
	return jtkpresent.Emit(opts, model)
}
