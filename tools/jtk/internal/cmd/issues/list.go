package issues

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
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
	var allFields bool
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List issues",
		Long:  "List issues, optionally filtered by project and/or sprint.",
		Example: `  # --project accepts a key or name; --sprint accepts a name, numeric ID, or "current"
  jtk issues list --project MYPROJECT
  jtk issues list --project "Platform Development" --sprint "MON Sprint 70"
  jtk issues list --project MYPROJECT --sprint current

  # Get up to 200 results (auto-paginates)
  jtk issues list --project MYPROJECT --max 200

  # Resume from a previous page token
  jtk issues list --project MYPROJECT --next-page-token <token>

  # List with all fields (includes description)
  jtk issues list --project MYPROJECT --all-fields

  # Project display columns — headers, Jira field IDs, or human names
  jtk issues list --project MYPROJECT --fields SUMMARY,STATUS
  jtk issues list --project MYPROJECT --fields "Issue Type"`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts, project, sprint, maxResults, nextPageToken, allFields, fieldsFlag)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Filter by project key or name")
	cmd.Flags().StringVarP(&sprint, "sprint", "s", "", "Filter by sprint name, numeric ID, or 'current'")
	cmd.Flags().IntVarP(&maxResults, "max", "m", 25, "Maximum number of results to return")
	cmd.Flags().StringVar(&nextPageToken, "next-page-token", "", "Token for next page of results")
	cmd.Flags().BoolVar(&allFields, "all-fields", false, "Include all fields (e.g. description)")
	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display columns (headers, Jira field IDs, or human names)")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, project, sprint string, maxResults int, nextPageToken string, allFields bool, fieldsFlag string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// --id wins over --fields: skip projection entirely when --id is set so
	// we don't waste a GetFields() call for a --fields token whose display
	// result would be thrown away. --id also overrides the JSON + --fields
	// error since we're not producing JSON.
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
			client.GetFields,
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
		sprintClause, err := buildSprintClause(ctx, resolver, sprint, opts.Stderr)
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

	if idOnly {
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
	if projected {
		projection.ApplyToTableInModel(model, selected)
	}
	return jtkpresent.Emit(opts, model)
}

// buildSprintClause builds the JQL `sprint` clause. Rules:
//
//   - "current" → sprint in openSprints()
//   - numeric input → sprint = <N> (passed straight through, no cache hit
//     needed to validate; Jira rejects bad IDs)
//   - name input → try the resolver for a canonical ID; on ambiguity or
//     not-found, fall through to a quoted name clause so Jira's own JQL
//     engine can resolve it (the pre-resolver behavior). The resolver's
//     global unique-match requirement is too strict for JQL — names that
//     repeat across boards are legal JQL targets and Jira handles them
//     natively in the project/board context.
//
// When the fallback fires because of genuine ambiguity, a warning is
// written to stderr so the user knows Jira's engine may return more than
// they expected. `warn` is the caller's stderr (opts.Stderr) so command
// tests can capture it without touching the process stderr.
func buildSprintClause(ctx context.Context, resolver *resolve.Resolver, sprint string, warn io.Writer) (string, error) {
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
	if err == nil && resolved.ID != 0 {
		return fmt.Sprintf("sprint = %d", resolved.ID), nil
	}
	if warn != nil {
		var amb *resolve.AmbiguousMatchError
		var nf *resolve.NotFoundError
		switch {
		case errors.As(err, &amb):
			fmt.Fprintf(warn,
				"warning: sprint name %q matched multiple cached boards; falling back to JQL name resolution — "+
					"results may span sprints on different boards.\n", sprint)
		case errors.As(err, &nf):
			fmt.Fprintf(warn,
				"warning: sprint %q not found in cache; falling back to JQL name resolution — "+
					"Jira will resolve the name or return an empty result set. Run `jtk refresh sprints` to update the cache.\n", sprint)
		case err != nil:
			// Network failure, auth error, or other non-cache error during
			// resolution. Surface it rather than silently falling through.
			fmt.Fprintf(warn,
				"warning: sprint resolver failed for %q (%v); falling back to JQL name resolution.\n", sprint, err)
		case resolved.ID == 0:
			// Resolver returned a synthetic (shouldn't happen for sprints today, but
			// future-proofs the warning if sprint pass-through is ever added).
			fmt.Fprintf(warn,
				"warning: sprint %q not resolved to a cached ID; falling back to JQL name resolution.\n", sprint)
		}
	}
	// Cache miss, ambiguity, or synthetic-without-ID → let Jira's JQL
	// engine resolve the name. Quote to handle spaces and escape any
	// JQL metacharacters.
	return fmt.Sprintf(`sprint = "%s"`, jqlEscape(sprint)), nil
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
