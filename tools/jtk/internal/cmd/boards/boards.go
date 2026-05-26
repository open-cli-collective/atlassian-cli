// Package boards provides CLI commands for managing Jira agile boards.
package boards

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
	"github.com/open-cli-collective/jira-ticket-cli/internal/resolve"
)

func noFieldFetch(_ context.Context) ([]api.Field, error) { return nil, nil }

// Register registers the boards commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "boards",
		Aliases: []string{"board", "b"},
		Short:   "Manage agile boards",
		Long:    "Commands for viewing agile boards.",
		// SupportsAgile checks AgileURL — the correct guard for Agile API commands.
		// Non-Agile scope-restricted commands (automation, dashboards) use IsBearerAuth() instead.
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Cobra does not chain PersistentPreRunE — this hook shadows
			// the root's, so we must invoke the backend-selection wiring
			// explicitly. Without this, --backend / keyring.backend silently
			// stop applying on the `boards` command path.
			if err := root.WireBackendSelection(cmd); err != nil {
				return err
			}
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
	cmd.AddCommand(newGetCmd(opts))

	parent.AddCommand(cmd)
}

func newListCmd(opts *root.Options) *cobra.Command {
	var project string
	var maxResults int
	var nextPageToken string
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List boards",
		Long:  "List agile boards, optionally filtered by project.",
		Example: `  # List all boards
  jtk boards list

  # List boards for a project (accepts key or name)
  jtk boards list --project MYPROJECT
  jtk boards list --project "Platform Development"

  # Extended output with project names
  jtk boards list --extended

  # Emit only board IDs
  jtk boards list --id`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts, project, maxResults, nextPageToken, fieldsFlag)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Filter by project key or name")
	cmd.Flags().IntVarP(&maxResults, "max", "m", 50, "Maximum number of results")
	cmd.Flags().StringVar(&nextPageToken, "next-page-token", "", "Decimal startAt for the next page")
	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display columns")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, project string, maxResults int, nextPageToken, fieldsFlag string) error {
	idOnly := opts.EmitIDOnly()

	startAt, err := jtkpresent.ParseStartAtToken(nextPageToken)
	if err != nil {
		return err
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	var selected []projection.ColumnSpec
	var projected bool
	if !idOnly {
		selected, projected, err = projection.Resolve(
			ctx,
			jtkpresent.BoardListSpec,
			opts.IsExtended(),
			fieldsFlag,
			noFieldFetch,
			"boards list",
		)
		if err != nil {
			return err
		}
	}

	projectFilter := project
	if project != "" {
		resolvedProject, resolveErr := resolve.New(client).Project(ctx, project)
		if resolveErr != nil {
			return resolveErr
		}
		projectFilter = resolvedProject.Key
	}

	result, err := cache.GetBoardsCacheFirst(ctx, client, projectFilter, startAt, maxResults)
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
		for i, b := range result.Values {
			ids[i] = strconv.Itoa(b.ID)
		}
		return jtkpresent.EmitIDsWithPaginationToken(opts, ids, hasMore, nextToken)
	}

	if len(result.Values) == 0 {
		return jtkpresent.Emit(opts, jtkpresent.BoardPresenter{}.PresentEmpty())
	}

	model := jtkpresent.BoardPresenter{}.PresentListWithPagination(result.Values, opts.IsExtended(), hasMore, nextToken)
	if projected {
		projection.ApplyToTableInModel(model, selected)
	}
	return jtkpresent.Emit(opts, model)
}

func newGetCmd(opts *root.Options) *cobra.Command {
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "get <board>",
		Short: "Get board details",
		Long:  "Get details for a specific board. Accepts a board ID or name (resolved via cache).",
		Example: `  jtk boards get 123
  jtk boards get "MON board"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.APIClient()
			if err != nil {
				return err
			}
			resolvedBoard, err := resolve.New(client).Board(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return runGet(cmd.Context(), opts, client, &resolvedBoard, fieldsFlag)
		},
	}

	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display fields")

	return cmd
}

func runGet(ctx context.Context, opts *root.Options, client *api.Client, resolvedBoard *api.Board, fieldsFlag string) error {
	if opts.EmitIDOnly() {
		return jtkpresent.EmitIDs(opts, []string{strconv.Itoa(resolvedBoard.ID)})
	}

	selected, projected, err := projection.Resolve(
		ctx,
		jtkpresent.BoardDetailSpec,
		opts.IsExtended(),
		fieldsFlag,
		noFieldFetch,
		"boards get",
	)
	if err != nil {
		return err
	}

	board, err := client.GetBoard(ctx, resolvedBoard.ID)
	if err != nil {
		return err
	}

	// Preserve the resolved name if the API response lacks it
	if board.Name == "" && resolvedBoard.Name != "" {
		board.Name = resolvedBoard.Name
	}

	var config *api.BoardConfiguration
	needsConfig := opts.IsExtended() || projection.HasExtendedFields(selected, jtkpresent.BoardDetailSpec)
	if needsConfig {
		var configErr error
		config, configErr = client.GetBoardConfiguration(ctx, board.ID)
		if configErr != nil {
			_ = jtkpresent.Emit(opts, jtkpresent.BoardPresenter{}.PresentConfigFetchWarning(configErr))
		}
		if config != nil && config.Filter.ID != "" && config.Filter.Name == "" {
			if f, err := client.GetFilter(ctx, config.Filter.ID); err == nil && f.Name != "" {
				config.Filter.Name = f.Name
			}
		}
	}

	presenter := jtkpresent.BoardPresenter{}
	if projected {
		model := presenter.PresentDetailProjection(board, config)
		projection.ApplyToDetailInModel(model, selected)
		return jtkpresent.Emit(opts, model)
	}

	model := presenter.PresentDetail(board, config, opts.IsExtended())
	return jtkpresent.Emit(opts, model)
}
