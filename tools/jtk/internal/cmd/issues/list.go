package issues

import (
	"context"
	"errors"
	"fmt"
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
		sprintClause, warning, err := buildSprintClause(ctx, resolver, sprint)
		if err != nil {
			return err
		}
		if warning != nil {
			_ = jtkpresent.Emit(opts, emitSprintWarning(warning))
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
		projection.ApplyToTableInModel(model, selected)
	}
	return jtkpresent.Emit(opts, model)
}

type sprintWarningKind int

const (
	sprintWarningAmbiguity sprintWarningKind = iota
	sprintWarningCacheMiss
	sprintWarningResolverError
	sprintWarningSynthetic
)

type sprintWarning struct {
	Kind       sprintWarningKind
	SprintName string
	Err        error
}

// emitSprintWarning maps a sprintWarning to the appropriate presenter method.
func emitSprintWarning(w *sprintWarning) *present.OutputModel {
	p := jtkpresent.SprintPresenter{}
	switch w.Kind {
	case sprintWarningAmbiguity:
		return p.PresentResolutionAmbiguity(w.SprintName)
	case sprintWarningCacheMiss:
		return p.PresentResolutionCacheMiss(w.SprintName)
	case sprintWarningResolverError:
		return p.PresentResolutionError(w.SprintName, w.Err)
	case sprintWarningSynthetic:
		return p.PresentResolutionSynthetic(w.SprintName)
	default:
		panic(fmt.Sprintf("unhandled sprintWarningKind %d", w.Kind))
	}
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
// Returns structured warning metadata when a fallback fires; the caller
// maps it to the appropriate presenter method.
func buildSprintClause(ctx context.Context, resolver *resolve.Resolver, sprint string) (string, *sprintWarning, error) {
	if sprint == "current" {
		return "sprint in openSprints()", nil, nil
	}
	if n, err := strconv.Atoi(sprint); err == nil {
		if n <= 0 {
			return "", nil, fmt.Errorf("--sprint numeric ID must be positive (got %s)", sprint)
		}
		return fmt.Sprintf("sprint = %d", n), nil, nil
	}
	resolved, err := resolver.Sprint(ctx, sprint, 0)
	if err == nil && resolved.ID != 0 {
		return fmt.Sprintf("sprint = %d", resolved.ID), nil, nil
	}
	var w *sprintWarning
	var amb *resolve.AmbiguousMatchError
	var nf *resolve.NotFoundError
	switch {
	case errors.As(err, &amb):
		w = &sprintWarning{Kind: sprintWarningAmbiguity, SprintName: sprint}
	case errors.As(err, &nf):
		w = &sprintWarning{Kind: sprintWarningCacheMiss, SprintName: sprint}
	case err != nil:
		w = &sprintWarning{Kind: sprintWarningResolverError, SprintName: sprint, Err: err}
	case resolved.ID == 0:
		w = &sprintWarning{Kind: sprintWarningSynthetic, SprintName: sprint}
	}
	return fmt.Sprintf(`sprint = "%s"`, jqlEscape(sprint)), w, nil
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
