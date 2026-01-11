package page

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rianjs/confluence-cli/api"
	"github.com/rianjs/confluence-cli/internal/config"
	"github.com/rianjs/confluence-cli/internal/view"
)

type listOptions struct {
	space   string
	limit   int
	status  string
	output  string
	noColor bool
}

// NewCmdList creates the page list command.
func NewCmdList() *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "search"},
		Short:   "List pages in a space",
		Long:    `List pages in a Confluence space.`,
		Example: `  # List pages in a space
  cfl page list --space DEV

  # List with limit
  cfl page list -s DEV -l 50

  # Output as JSON
  cfl page list -s DEV -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.output, _ = cmd.Flags().GetString("output")
			opts.noColor, _ = cmd.Flags().GetBool("no-color")
			return runList(opts, nil)
		},
	}

	cmd.Flags().StringVarP(&opts.space, "space", "s", "", "Space key or ID (required)")
	cmd.Flags().IntVarP(&opts.limit, "limit", "l", 25, "Maximum number of pages to return")
	cmd.Flags().StringVar(&opts.status, "status", "current", "Page status (current, archived, draft)")

	return cmd
}

func runList(opts *listOptions, client *api.Client) error {
	// Validate output format
	if err := view.ValidateFormat(opts.output); err != nil {
		return err
	}

	// Validate limit
	if opts.limit < 0 {
		return fmt.Errorf("invalid limit: %d (must be >= 0)", opts.limit)
	}

	// Render output
	renderer := view.NewRenderer(view.Format(opts.output), opts.noColor)

	// Handle limit 0 - return empty list
	if opts.limit == 0 {
		if opts.output == "json" {
			return renderer.RenderJSON([]interface{}{})
		}
		renderer.RenderText("No pages found.")
		return nil
	}

	// Determine space - for testing, opts.space can be provided directly
	spaceKey := opts.space

	// Create API client if not provided (allows injection for testing)
	if client == nil {
		cfg, err := config.LoadWithEnv(config.DefaultConfigPath())
		if err != nil {
			return fmt.Errorf("failed to load config: %w (run 'cfl init' to configure)", err)
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid config: %w (run 'cfl init' to configure)", err)
		}

		// Use default space from config if not specified
		if spaceKey == "" {
			spaceKey = cfg.DefaultSpace
		}

		client = api.NewClient(cfg.URL, cfg.Email, cfg.APIToken)
	}

	if spaceKey == "" {
		return fmt.Errorf("space is required: use --space flag or set default_space in config")
	}

	// Get space ID from key
	space, err := client.GetSpaceByKey(context.Background(), spaceKey)
	if err != nil {
		return fmt.Errorf("failed to find space '%s': %w", spaceKey, err)
	}

	// List pages
	apiOpts := &api.ListPagesOptions{
		Limit:  opts.limit,
		Status: opts.status,
	}

	result, err := client.ListPages(context.Background(), space.ID, apiOpts)
	if err != nil {
		return fmt.Errorf("failed to list pages: %w", err)
	}

	if len(result.Results) == 0 {
		renderer.RenderText(fmt.Sprintf("No pages found in space %s.", spaceKey))
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

	renderer.RenderTable(headers, rows)

	if result.HasMore() {
		fmt.Fprintf(os.Stderr, "\n(showing first %d results, use --limit to see more)\n", len(result.Results))
	}

	return nil
}
