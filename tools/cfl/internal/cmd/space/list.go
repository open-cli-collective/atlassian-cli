package space

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	cflpresent "github.com/open-cli-collective/confluence-cli/internal/present"
)

type listOptions struct {
	*root.Options
	limit     int
	spaceType string
	cursor    string
}

func newListCmd(rootOpts *root.Options) *cobra.Command {
	opts := &listOptions{Options: rootOpts}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List Confluence spaces",
		Long:    `List all Confluence spaces you have access to.`,
		Example: `  # List all spaces
  cfl space list

  # List only global spaces
  cfl space list --type global

  # Paginate through results
  cfl space list --cursor "eyJpZCI6MTIzfQ=="`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts)
		},
	}

	cmd.Flags().IntVarP(&opts.limit, "limit", "l", 25, "Maximum number of spaces to return")
	cmd.Flags().StringVarP(&opts.spaceType, "type", "t", "", "Filter by space type (global, personal)")
	cmd.Flags().StringVar(&opts.cursor, "cursor", "", "Pagination cursor for next page")

	return cmd
}

func runList(ctx context.Context, opts *listOptions) error {
	if opts.limit < 0 {
		return fmt.Errorf("invalid limit: %d (must be >= 0)", opts.limit)
	}

	if opts.limit == 0 {
		return cflpresent.Emit(opts.Options, cflpresent.SpacePresenter{}.PresentEmpty())
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	apiOpts := &api.ListSpacesOptions{
		Limit:  opts.limit,
		Type:   opts.spaceType,
		Cursor: opts.cursor,
	}

	result, err := client.ListSpaces(ctx, apiOpts)
	if err != nil {
		return fmt.Errorf("listing spaces: %w", err)
	}

	if len(result.Results) == 0 {
		return cflpresent.Emit(opts.Options, cflpresent.SpacePresenter{}.PresentEmpty())
	}
	return cflpresent.Emit(opts.Options, cflpresent.SpacePresenter{}.PresentList(result.Results, opts.Full, cflpresent.ExtractCursor(result.Links.Next), result.HasMore()))
}
