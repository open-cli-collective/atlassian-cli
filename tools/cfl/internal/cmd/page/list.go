package page

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
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
  cfl page list -s DEV -l 50

  # Output as JSON
  cfl page list -s DEV -o json`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runList(opts)
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

func runList(opts *listOptions) error {
	if err := view.ValidateFormat(opts.Output); err != nil {
		return err
	}

	if !validStatuses[opts.status] {
		return fmt.Errorf("invalid status %q: must be one of current, archived, trashed", opts.status)
	}

	if opts.limit < 0 {
		return fmt.Errorf("invalid limit: %d (must be >= 0)", opts.limit)
	}

	v := opts.View()

	if opts.limit == 0 {
		if opts.Output == "json" {
			return v.JSON([]interface{}{})
		}
		v.RenderText("No pages found.")
		return nil
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

	space, err := client.GetSpaceByKey(context.Background(), spaceKey)
	if err != nil {
		return fmt.Errorf("failed to find space '%s': %w", spaceKey, err)
	}

	apiOpts := &api.ListPagesOptions{
		Limit:  opts.limit,
		Status: opts.status,
	}

	result, err := client.ListPages(context.Background(), space.ID, apiOpts)
	if err != nil {
		return fmt.Errorf("failed to list pages: %w", err)
	}

	if len(result.Results) == 0 {
		v.RenderText(fmt.Sprintf("No pages found in space %s.", spaceKey))
		return nil
	}

	headers := []string{"ID", "TITLE", "STATUS", "VERSION"}
	var rows [][]string

	for _, page := range result.Results {
		version := ""
		if page.Version != nil {
			version = fmt.Sprintf("v%d", page.Version.Number)
		}
		rows = append(rows, []string{
			page.ID,
			view.Truncate(page.Title, 60),
			page.Status,
			version,
		})
	}

	_ = v.RenderList(headers, rows, result.HasMore())

	if result.HasMore() && opts.Output != "json" {
		fmt.Fprintf(os.Stderr, "\n(showing first %d results, use --limit to see more)\n", len(result.Results))
	}

	return nil
}
