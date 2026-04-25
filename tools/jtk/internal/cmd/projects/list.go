package projects

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// noFieldFetch is the projection.Resolve fetcher for project commands — the
// same no-op rationale as comments / users: projection tokens here resolve
// against ProjectListSpec / ProjectDetailSpec / ProjectTypeSpec, not Jira
// issue field metadata. Package-local by intent; the function is too
// trivial to deserve a shared home and it reads more clearly next to the
// specs it supports.
func noFieldFetch(_ context.Context) ([]api.Field, error) { return nil, nil }

func newListCmd(opts *root.Options) *cobra.Command {
	var query string
	var maxResults int
	var nextPageToken string
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects",
		Long:  "List Jira projects, optionally filtered by a search query.",
		Example: `  # List all projects
  jtk projects list

  # Include STYLE / ISSUE_TYPES / COMPONENTS columns
  jtk projects list --extended

  # Emit just the project keys
  jtk projects list --id

  # Project output to selected columns
  jtk projects list --fields KEY,NAME

  # Fetch the next page
  jtk projects list --max 5 --next-page-token 5`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts, query, maxResults, nextPageToken, fieldsFlag)
		},
	}

	cmd.Flags().StringVarP(&query, "query", "q", "", "Filter projects by name")
	cmd.Flags().IntVarP(&maxResults, "max", "m", 50, "Maximum number of results")
	cmd.Flags().StringVar(&nextPageToken, "next-page-token", "", "Decimal startAt for the next page")
	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display columns (ProjectListSpec headers)")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, query string, maxResults int, nextPageToken, fieldsFlag string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	idOnly := opts.EmitIDOnly()

	startAt, err := jtkpresent.ParseStartAtToken(nextPageToken)
	if err != nil {
		return err
	}

	if !idOnly && fieldsFlag != "" && v.Format == view.FormatJSON {
		return jtkpresent.ErrFieldsWithJSON
	}

	var selected []projection.ColumnSpec
	var projected bool
	if !idOnly {
		selected, projected, err = projection.Resolve(
			ctx,
			jtkpresent.ProjectListSpec,
			opts.IsExtended(),
			fieldsFlag,
			noFieldFetch,
			"projects list",
		)
		if err != nil {
			return err
		}
	}

	// --id mode emits just keys, which are on every /project/search response
	// by default; skip expansion entirely. Default-mode list also only needs
	// LEAD. Extended adds STYLE|ISSUE_TYPES|COMPONENTS and requires the full
	// set.
	expand := ""
	if !idOnly {
		expand = "lead"
		if opts.IsExtended() {
			expand = api.ProjectListExpand
		}
	}
	result, err := client.SearchProjects(ctx, query, startAt, maxResults, expand)
	if err != nil {
		return err
	}

	hasMore := !result.IsLast
	if hasMore && len(result.Values) == 0 {
		// Jira's /project/search has always paired IsLast=false with a
		// non-empty page in practice, but guarding against the degenerate
		// "hasMore + empty Values" response matters: a client that blindly
		// follows nextToken would loop forever on the same page. Fail loudly
		// instead of emitting a self-referential continuation token.
		return fmt.Errorf("unexpected paginated response: IsLast=false with empty values (startAt=%d)", startAt)
	}
	nextToken := ""
	if hasMore {
		nextToken = strconv.Itoa(startAt + len(result.Values))
	}

	if idOnly {
		ids := make([]string, len(result.Values))
		for i, p := range result.Values {
			ids[i] = p.Key
		}
		return jtkpresent.EmitIDsWithPaginationToken(opts, ids, hasMore, nextToken)
	}

	if len(result.Values) == 0 {
		return jtkpresent.Emit(opts, jtkpresent.ProjectPresenter{}.PresentEmpty())
	}

	if v.Format == view.FormatJSON {
		return v.JSON(result.Values)
	}

	model := jtkpresent.ProjectPresenter{}.PresentProjectListWithPagination(result.Values, opts.IsExtended(), hasMore, nextToken)
	if projected {
		projection.ApplyToTableInModel(model, selected)
	}
	return jtkpresent.Emit(opts, model)
}
