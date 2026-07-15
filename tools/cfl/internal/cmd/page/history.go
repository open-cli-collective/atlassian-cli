package page

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	cflpresent "github.com/open-cli-collective/confluence-cli/internal/present"
)

type historyListOptions struct {
	*root.Options
	limit  int
	cursor string
	idOnly bool
}

func newHistoryCmd(rootOpts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Inspect Confluence page history",
		Long:  `Inspect Confluence page version history.`,
	}

	cmd.AddCommand(newHistoryListCmd(rootOpts))
	return cmd
}

func newHistoryListCmd(rootOpts *root.Options) *cobra.Command {
	opts := &historyListOptions{Options: rootOpts}

	cmd := &cobra.Command{
		Use:     "list <page-id>",
		Aliases: []string{"ls"},
		Short:   "List page versions",
		Long:    `List Confluence page versions in newest-first order.`,
		Example: `  # List recent page versions
  cfl page history list 12345

  # Print version numbers only
  cfl page history list 12345 --id

  # Paginate through versions
  cfl page history list 12345 --cursor "eyJpZCI6MTIzfQ=="`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistoryList(cmd.Context(), args[0], opts)
		},
	}

	cmd.Flags().IntVarP(&opts.limit, "limit", "l", 25, "Maximum number of versions to return")
	cmd.Flags().StringVar(&opts.cursor, "cursor", "", "Pagination cursor for next page")
	cmd.Flags().BoolVar(&opts.idOnly, "id", false, "Print only version numbers")

	return cmd
}

func runHistoryList(ctx context.Context, pageID string, opts *historyListOptions) error {
	if opts.limit < 0 {
		return fmt.Errorf("invalid limit: %d (must be >= 0)", opts.limit)
	}
	if opts.limit == 0 {
		return cflpresent.Emit(opts.Options, cflpresent.PageHistoryPresenter{}.PresentEmpty())
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	result, err := client.ListPageVersions(ctx, pageID, &api.ListPageVersionsOptions{
		Limit:  opts.limit,
		Cursor: opts.cursor,
		Sort:   "-modified-date",
	})
	if err != nil {
		return err
	}

	if len(result.Results) == 0 {
		return cflpresent.Emit(opts.Options, cflpresent.PageHistoryPresenter{}.PresentEmpty())
	}
	nextCursor := cflpresent.ExtractCursor(result.Links.Next)
	if opts.idOnly {
		return cflpresent.Emit(opts.Options, cflpresent.PageHistoryPresenter{}.PresentIDs(result.Results, nextCursor, pageID, result.HasMore()))
	}
	return cflpresent.Emit(opts.Options, cflpresent.PageHistoryPresenter{}.PresentList(result.Results, nextCursor, pageID, result.HasMore()))
}
