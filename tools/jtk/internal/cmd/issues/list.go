package issues

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
	"github.com/open-cli-collective/jira-ticket-cli/internal/resolve"
)

func newListCmd(opts *root.Options) *cobra.Command {
	var project string
	var sprint string
	var maxResults int
	var nextPageToken string
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List issues",
		Long:  "List issues, optionally filtered by project and/or sprint.",
		Example: `  # --project accepts a key or name; --sprint accepts a name, numeric ID, or "current"
  jtk issues list --project MYPROJECT
  jtk issues list --project "Platform Development" --sprint "MON Sprint 70"
  jtk issues list --project MYPROJECT --sprint current

  # Request one page of up to 100 results
  jtk issues list --project MYPROJECT --max 100

  # Resume from a previous page token
  jtk issues list --project MYPROJECT --next-page-token <token>
  # Project display columns — headers, Jira field IDs, or human names
  jtk issues list --project MYPROJECT --fields SUMMARY,STATUS
  jtk issues list --project MYPROJECT --fields "Issue Type"`,
		Args: func(_ *cobra.Command, _ []string) error {
			return jtkpresent.ValidateMaxAtMost(maxResults, 100)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts, project, sprint, maxResults, nextPageToken, fieldsFlag)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Filter by project key or name")
	cmd.Flags().StringVarP(&sprint, "sprint", "s", "", "Filter by unique sprint name, numeric ID, or 'current'")
	cmd.Flags().IntVarP(&maxResults, "max", "m", 50, "Page size (maximum 100)")
	cmd.Flags().StringVar(&nextPageToken, "next-page-token", "", "Token for next page of results")
	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display columns (headers, Jira field IDs, or human names)")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, project, sprint string, maxResults int, nextPageToken, fieldsFlag string) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// --id wins over --fields: skip projection entirely when --id is set so
	// we don't waste a GetFields() call for a --fields token whose display
	// result would be thrown away. --id also overrides the JSON + --fields
	// error since we're not producing JSON.
	idOnly := opts.EmitIDOnly()

	var selected []projection.ColumnSpec
	var projected bool
	if !idOnly {
		var err error
		selected, projected, err = projection.Resolve(
			ctx,
			jtkpresent.IssueListSpec,
			fieldsFlag,
			fieldsFetcher(client),
			"issues list",
		)
		if err != nil {
			return err
		}
	}

	// Build JQL query
	resolver := resolve.New(client)

	var jql string
	if project != "" {
		resolvedProject, err := resolver.Project(ctx, project)
		if err != nil {
			return err
		}
		// Quote the key so any shape-pass-through value that happens to
		// include JQL metacharacters can't produce malformed queries.
		jql = fmt.Sprintf(`project = "%s"`, jqlEscape(resolvedProject.Key))
	}

	if sprint != "" {
		sprintClause, err := buildSprintClause(ctx, resolver, sprint)
		if err != nil {
			return err
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

	fields := deriveFetchFields(selected, projected)

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

	model := jtkpresent.IssuePresenter{}.PresentListWithPagination(result.Issues, projection.HasOptionalFields(selected, jtkpresent.IssueListSpec), hasMore, nextToken)
	if projected {
		jtkpresent.AppendDynamicTableColumns(model, result.Issues, projection.DynamicSpecs(selected))
		projection.ApplyToTableInModel(model, selected)
	}
	return jtkpresent.Emit(opts, model)
}

// buildSprintClause builds the JQL `sprint` clause. Rules:
//
//   - "current" → sprint in openSprints()
//   - numeric input → sprint = <N> (passed straight through, no cache hit
//     needed to validate; Jira rejects bad IDs)
//   - name input → resolve one canonical ID or fail closed.
func buildSprintClause(ctx context.Context, resolver *resolve.Resolver, sprint string) (string, error) {
	if sprint == "current" {
		return "sprint in openSprints()", nil
	}
	if n, err := strconv.Atoi(sprint); err == nil {
		if n <= 0 {
			return "", fmt.Errorf("--sprint numeric ID must be positive (got %s)", sprint)
		}
		return fmt.Sprintf("sprint = %d", n), nil
	}
	resolved, err := resolver.Sprint(ctx, sprint, 0)
	if err != nil {
		return "", fmt.Errorf("resolve sprint %q: %w; refresh the sprint cache or pass a numeric sprint ID", sprint, err)
	}
	if resolved.ID == 0 {
		return "", fmt.Errorf("resolve sprint %q: resolver returned no ID; refresh the sprint cache or pass a numeric sprint ID", sprint)
	}
	return fmt.Sprintf("sprint = %d", resolved.ID), nil
}

// jqlEscape makes a string safe to embed between JQL double quotes. JQL
// parses backslash as an escape character inside quoted strings, so we
// must escape backslashes before the double-quote pass to avoid producing
// malformed queries for names like `Sprint\Eng` or keys smuggled in via
// shape pass-through. Ordering matters: backslash first, then quote.
func jqlEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
