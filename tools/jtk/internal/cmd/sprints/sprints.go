// Package sprints provides CLI commands for managing Jira sprints.
package sprints

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
	"github.com/open-cli-collective/jira-ticket-cli/internal/resolve"
)

func noFieldFetch(_ context.Context) ([]api.Field, error) { return nil, nil }

// validateBoardRef rejects inputs that would parse as numeric but produce a
// synthetic Board{ID: n} with n <= 0, which the downstream Agile endpoints
// return confusing 404s for. Non-numeric names pass through unchanged —
// board-name resolution is handled by the resolver.
func validateBoardRef(board string) error {
	if board == "" {
		return fmt.Errorf("--board is required")
	}
	if n, err := strconv.Atoi(board); err == nil && n <= 0 {
		return fmt.Errorf("--board numeric ID must be positive (got %s)", board)
	}
	return nil
}

// Register registers the sprints commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "sprints",
		Aliases: []string{"sprint", "sp"},
		Short:   "Manage sprints",
		Long:    "Commands for viewing sprints and sprint issues.",
		// SupportsAgile checks AgileURL — the correct guard for Agile API commands.
		// Non-Agile scope-restricted commands (automation, dashboards) use IsBearerAuth() instead.
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			client, err := opts.APIClient()
			if err != nil {
				return err
			}
			if !client.SupportsAgile() {
				return api.ErrAgileUnavailable
			}
			return nil
		},
	}

	cmd.AddCommand(newListCmd(opts))
	cmd.AddCommand(newCurrentCmd(opts))
	cmd.AddCommand(newIssuesCmd(opts))
	cmd.AddCommand(newAddCmd(opts))

	parent.AddCommand(cmd)
}

func newListCmd(opts *root.Options) *cobra.Command {
	var board string
	var state string
	var maxResults int
	var nextPageToken string
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sprints for a board",
		Long:  "List sprints for a specific board. --board accepts a board ID or name.",
		Example: `  # List all sprints
  jtk sprints list --board 123
  jtk sprints list --board "MON board"

  # List only active sprints
  jtk sprints list --board 123 --state active

  # Extended output with completion dates, board, goal
  jtk sprints list --board 123 --extended

  # Emit only sprint IDs
  jtk sprints list --board 123 --id`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validateBoardRef(board); err != nil {
				return err
			}
			client, err := opts.APIClient()
			if err != nil {
				return err
			}
			resolvedBoard, err := resolve.New(client).Board(cmd.Context(), board)
			if err != nil {
				return err
			}
			return runList(cmd.Context(), opts, client, resolvedBoard.ID, state, maxResults, nextPageToken, fieldsFlag)
		},
	}

	cmd.Flags().StringVarP(&board, "board", "b", "", "Board ID or name (required)")
	cmd.Flags().StringVarP(&state, "state", "s", "", "Filter by state (active, closed, future)")
	cmd.Flags().IntVarP(&maxResults, "max", "m", 50, "Maximum number of results")
	cmd.Flags().StringVar(&nextPageToken, "next-page-token", "", "Decimal startAt for the next page")
	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display columns")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, client *api.Client, boardID int, state string, maxResults int, nextPageToken, fieldsFlag string) error {
	v := opts.View()

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
			jtkpresent.SprintListSpec,
			opts.IsExtended(),
			fieldsFlag,
			noFieldFetch,
			"sprints list",
		)
		if err != nil {
			return err
		}
	}

	result, err := client.ListSprints(ctx, boardID, state, startAt, maxResults)
	if err != nil {
		return err
	}

	hasMore := !result.IsLast
	if hasMore && len(result.Values) == 0 {
		return fmt.Errorf("unexpected paginated response: IsLast=false with empty values (startAt=%d)", startAt)
	}
	nextToken := ""
	if hasMore {
		nextToken = strconv.Itoa(startAt + len(result.Values))
	}

	if idOnly {
		ids := make([]string, len(result.Values))
		for i, s := range result.Values {
			ids[i] = strconv.Itoa(s.ID)
		}
		return jtkpresent.EmitIDsWithPaginationToken(opts, ids, hasMore, nextToken)
	}

	if len(result.Values) == 0 {
		return jtkpresent.Emit(opts, jtkpresent.SprintPresenter{}.PresentEmpty())
	}

	if v.Format == view.FormatJSON {
		arts := jtkartifact.ProjectSprints(result.Values, opts.ArtifactMode())
		return v.RenderArtifactList(artifact.NewListResult(arts, hasMore))
	}

	presenter := jtkpresent.SprintPresenter{}
	model := presenter.PresentList(result.Values, opts.IsExtended())
	if projected {
		projection.ApplyToTableInModel(model, selected)
	}
	model.Sections = jtkpresent.AppendPaginationHintWithToken(model.Sections, hasMore, nextToken)
	return jtkpresent.Emit(opts, model)
}

func newCurrentCmd(opts *root.Options) *cobra.Command {
	var board string
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "current",
		Short: "Show current sprint",
		Long:  "Show the current active sprint for a board. --board accepts a board ID or name.",
		Example: `  jtk sprints current --board 123
  jtk sprints current --board "MON board"
  jtk sprints current --board 123 --extended`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validateBoardRef(board); err != nil {
				return err
			}
			client, err := opts.APIClient()
			if err != nil {
				return err
			}
			resolvedBoard, err := resolve.New(client).Board(cmd.Context(), board)
			if err != nil {
				return err
			}
			return runCurrent(cmd.Context(), opts, client, &resolvedBoard, fieldsFlag)
		},
	}

	cmd.Flags().StringVarP(&board, "board", "b", "", "Board ID or name (required)")
	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display fields")

	return cmd
}

func runCurrent(ctx context.Context, opts *root.Options, client *api.Client, board *api.Board, fieldsFlag string) error {
	v := opts.View()

	if !opts.EmitIDOnly() && fieldsFlag != "" && v.Format == view.FormatJSON {
		return jtkpresent.ErrFieldsWithJSON
	}

	var selected []projection.ColumnSpec
	var projected bool
	if !opts.EmitIDOnly() {
		var err error
		selected, projected, err = projection.Resolve(
			ctx,
			jtkpresent.SprintDetailSpec,
			opts.IsExtended(),
			fieldsFlag,
			noFieldFetch,
			"sprints current",
		)
		if err != nil {
			return err
		}
	}

	sprint, err := client.GetCurrentSprint(ctx, board.ID)
	if err != nil {
		return err
	}

	if opts.EmitIDOnly() {
		return jtkpresent.EmitIDs(opts, []string{strconv.Itoa(sprint.ID)})
	}

	if v.Format == view.FormatJSON {
		return v.RenderArtifact(jtkartifact.ProjectSprint(sprint, opts.ArtifactMode()))
	}

	presenter := jtkpresent.SprintPresenter{}
	if projected {
		model := presenter.PresentDetailProjection(sprint, board)
		projection.ApplyToDetailInModel(model, selected)
		return jtkpresent.Emit(opts, model)
	}

	model := presenter.PresentDetail(sprint, board, opts.IsExtended())
	return jtkpresent.Emit(opts, model)
}

func newIssuesCmd(opts *root.Options) *cobra.Command {
	var maxResults int
	var nextPageToken string

	cmd := &cobra.Command{
		Use:   "issues <sprint>",
		Short: "List issues in a sprint",
		Long:  "List all issues in a specific sprint. Accepts a sprint ID or name (resolved via cache).",
		Example: `  jtk sprints issues 456
  jtk sprints issues "MON Sprint 70"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.APIClient()
			if err != nil {
				return err
			}
			resolvedSprint, err := resolve.New(client).Sprint(cmd.Context(), args[0], 0)
			if err != nil {
				return err
			}
			return runIssues(cmd.Context(), opts, resolvedSprint.ID, maxResults, nextPageToken)
		},
	}

	cmd.Flags().IntVarP(&maxResults, "max", "m", 50, "Maximum number of results")
	cmd.Flags().StringVar(&nextPageToken, "next-page-token", "", "Decimal startAt for the next page")

	return cmd
}

func runIssues(ctx context.Context, opts *root.Options, sprintID int, maxResults int, nextPageToken string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	startAt, err := jtkpresent.ParseStartAtToken(nextPageToken)
	if err != nil {
		return err
	}

	result, err := client.GetSprintIssues(ctx, sprintID, startAt, maxResults)
	if err != nil {
		return err
	}

	var hasMore bool
	if result.Total < 0 {
		hasMore = len(result.Issues) == maxResults
	} else {
		hasMore = result.StartAt+len(result.Issues) < result.Total
	}
	nextToken := ""
	if hasMore {
		nextToken = strconv.Itoa(startAt + len(result.Issues))
	}

	if opts.EmitIDOnly() {
		ids := make([]string, len(result.Issues))
		for i, issue := range result.Issues {
			ids[i] = issue.Key
		}
		return jtkpresent.EmitIDsWithPaginationToken(opts, ids, hasMore, nextToken)
	}

	if len(result.Issues) == 0 {
		return jtkpresent.Emit(opts, jtkpresent.SprintPresenter{}.PresentNoIssues())
	}

	if v.Format == view.FormatJSON {
		arts := jtkartifact.ProjectIssues(result.Issues, opts.ArtifactMode())
		return v.RenderArtifactList(artifact.NewListResult(arts, hasMore))
	}

	model := jtkpresent.IssuePresenter{}.PresentList(result.Issues)
	model.Sections = jtkpresent.AppendPaginationHintWithToken(model.Sections, hasMore, nextToken)
	return jtkpresent.Emit(opts, model)
}

func newAddCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <sprint> <issue-key>...",
		Short: "Move issues to a sprint",
		Long:  "Move one or more issues to a specific sprint. <sprint> accepts a sprint ID or name.",
		Example: `  # Move a single issue by sprint ID
  jtk sprints add 123 PROJ-456

  # Move by sprint name (resolved via cache)
  jtk sprints add "MON Sprint 70" PROJ-456

  # Move multiple issues
  jtk sprints add 123 PROJ-456 PROJ-789 PROJ-101`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.APIClient()
			if err != nil {
				return err
			}
			resolvedSprint, err := resolve.New(client).Sprint(cmd.Context(), args[0], 0)
			if err != nil {
				return err
			}
			return runAdd(cmd.Context(), opts, client, resolvedSprint.ID, args[1:])
		},
	}

	return cmd
}

func runAdd(ctx context.Context, opts *root.Options, client *api.Client, sprintID int, issueKeys []string) error {
	if err := client.MoveIssuesToSprint(ctx, sprintID, issueKeys); err != nil {
		return err
	}

	model := jtkpresent.SprintPresenter{}.PresentMoved(issueKeys, sprintID)
	return jtkpresent.Emit(opts, model)
}
