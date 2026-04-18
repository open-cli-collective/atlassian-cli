package projects

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// errFieldsWithJSON is returned when --fields is combined with --output json.
var errFieldsWithJSON = errors.New("--fields is not supported with --output json")

// noFieldFetch is the projection.Resolve fetcher for project commands — the
// same no-op rationale as comments / users: projection tokens here resolve
// against ProjectListSpec / ProjectDetailSpec / ProjectTypeSpec, not Jira
// issue field metadata.
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

	startAt, err := parseStartAtToken(nextPageToken)
	if err != nil {
		return err
	}

	if !idOnly && fieldsFlag != "" && v.Format == view.FormatJSON {
		return errFieldsWithJSON
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

	result, err := client.SearchProjects(ctx, query, startAt, maxResults)
	if err != nil {
		return err
	}

	hasMore := !result.IsLast
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

	presenter := jtkpresent.ProjectPresenter{}
	model := presenter.PresentProjectList(result.Values, opts.IsExtended())
	if projected {
		projectTableSectionInModel(model, selected)
	}
	model.Sections = jtkpresent.AppendPaginationHintWithToken(model.Sections, hasMore, nextToken)
	return jtkpresent.Emit(opts, model)
}

// parseStartAtToken converts a `--next-page-token` value to a 0-based offset.
// Non-numeric or negative input is rejected up-front so callers get a clean
// error instead of a silently rewound page.
func parseStartAtToken(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(token)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid --next-page-token %q: expected a non-negative decimal", token)
	}
	return n, nil
}

// projectDetailSectionInModel rewrites the first DetailSection of model to the
// selected fields. No-op when there is no DetailSection.
func projectDetailSectionInModel(model *present.OutputModel, selected []projection.ColumnSpec) {
	for i, s := range model.Sections {
		if ds, ok := s.(*present.DetailSection); ok {
			model.Sections[i] = projection.ProjectDetail(ds, selected)
			return
		}
	}
}

// projectTableSectionInModel rewrites the first TableSection of model to the
// selected columns. No-op when there is no TableSection.
func projectTableSectionInModel(model *present.OutputModel, selected []projection.ColumnSpec) {
	for i, s := range model.Sections {
		if ts, ok := s.(*present.TableSection); ok {
			model.Sections[i] = projection.ProjectTable(ts, selected)
			return
		}
	}
}
