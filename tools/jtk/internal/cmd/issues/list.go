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

func newListCmd(opts *root.Options) *cobra.Command {
	var project string
	var sprint string
	var maxResults int
	var nextPageToken string
	var allFields bool
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

  # List with all fields (includes description)
  jtk issues list --project MYPROJECT --all-fields

  # List with specific fields (e.g. custom fields)
  jtk issues list --project MYPROJECT --fields summary,status,customfield_10005`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts, project, sprint, maxResults, nextPageToken, allFields, fieldsFlag)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Filter by project key")
	cmd.Flags().StringVarP(&sprint, "sprint", "s", "", "Filter by sprint (use 'current' for active sprint)")
	cmd.Flags().IntVarP(&maxResults, "max", "m", 25, "Maximum number of results to return")
	cmd.Flags().StringVar(&nextPageToken, "next-page-token", "", "Token for next page of results")
	cmd.Flags().BoolVar(&allFields, "all-fields", false, "Include all fields (e.g. description)")
	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated list of fields to return (e.g. summary,customfield_10005)")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, project, sprint string, maxResults int, nextPageToken string, allFields bool, fieldsFlag string) error {
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

	hasMore := !result.Pagination.IsLast

	if opts.EmitIDOnly() {
		ids := make([]string, len(result.Issues))
		for i, issue := range result.Issues {
			ids[i] = issue.Key
		}
		return jtkpresent.EmitIDsWithPagination(opts, ids, hasMore)
	}

	if len(result.Issues) == 0 {
		// Two cases, each with a single unambiguous message:
		//   hasMore=false → "No issues found" (the query's result set is empty)
		//   hasMore=true  → pagination hint only (this page is empty but more
		//                    pages exist; the agent should keep paging)
		// Emitting both together would contradict itself; pick one.
		if hasMore {
			return jtkpresent.Emit(opts, &present.OutputModel{
				Sections: jtkpresent.AppendPaginationHint(nil, true),
			})
		}
		return jtkpresent.Emit(opts, jtkpresent.IssuePresenter{}.PresentEmpty())
	}

	// For JSON output, return the projected artifacts
	if v.Format == view.FormatJSON {
		arts := jtkartifact.ProjectIssues(result.Issues, opts.ArtifactMode())
		return v.RenderArtifactList(artifact.NewListResult(arts, hasMore))
	}

	model := jtkpresent.IssuePresenter{}.PresentListWithPagination(result.Issues, hasMore)
	return jtkpresent.Emit(opts, model)
}
