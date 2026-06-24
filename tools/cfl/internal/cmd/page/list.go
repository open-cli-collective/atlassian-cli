package page

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
	space  string
	limit  int
	status string
}

func newListCmd(rootOpts *root.Options) *cobra.Command {
	opts := &listOptions{Options: rootOpts}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List pages in a space",
		Long: `List pages in a Confluence space.

Shows page metadata (ID, title, status, version). Page body content
is not included in list output. Use 'cfl page view <id>' to see
page content.`,
		Example: `  # List pages in a space
  cfl page list --space DEV

  # List with limit
  cfl page list -s DEV -l 50`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.space, "space", "s", "", "Space key or ID (required)")
	cmd.Flags().IntVarP(&opts.limit, "limit", "l", 25, "Maximum number of pages to return")
	cmd.Flags().StringVar(&opts.status, "status", "current", "Page status: current, archived, trashed")

	return cmd
}

// validStatuses are the page statuses accepted by the Confluence API.
var validStatuses = map[string]bool{
	"current":  true,
	"archived": true,
	"trashed":  true,
	"deleted":  true,
}

func runList(ctx context.Context, opts *listOptions) error {
	if !validStatuses[opts.status] {
		return fmt.Errorf("invalid status %q: must be one of current, archived, trashed", opts.status)
	}

	if opts.limit < 0 {
		return fmt.Errorf("invalid limit: %d (must be >= 0)", opts.limit)
	}

	if opts.limit == 0 {
		return cflpresent.Emit(opts.Options, cflpresent.PagePresenter{}.PresentEmpty(""))
	}

	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	spaceKey := opts.space
	if spaceKey == "" {
		spaceKey = cfg.DefaultSpace
	}

	if spaceKey == "" {
		return fmt.Errorf("space is required: use --space flag or set default_space in config")
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	space, err := client.GetSpaceByKey(ctx, spaceKey)
	if err != nil {
		return fmt.Errorf("finding space '%s': %w", spaceKey, err)
	}

	apiOpts := &api.ListPagesOptions{
		Limit:  opts.limit,
		Status: opts.status,
	}

	result, err := client.ListPages(ctx, space.ID, apiOpts)
	if err != nil {
		return fmt.Errorf("listing pages: %w", err)
	}

	if len(result.Results) == 0 {
		return cflpresent.Emit(opts.Options, cflpresent.PagePresenter{}.PresentEmpty(spaceKey))
	}
	return cflpresent.Emit(opts.Options, cflpresent.PagePresenter{}.PresentList(result.Results, opts.Full, result.HasMore()))
}
