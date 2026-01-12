// Package search provides the search command for finding Confluence content.
package search

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rianjs/confluence-cli/api"
	"github.com/rianjs/confluence-cli/internal/config"
	"github.com/rianjs/confluence-cli/internal/view"
)

type searchOptions struct {
	// Query building
	query       string // Positional arg: free-text search
	cql         string // Raw CQL (power users)
	space       string // Filter by space key
	contentType string // page, blogpost, attachment, comment
	title       string // Title contains
	label       string // Label filter

	// Pagination
	limit int

	// Output
	output  string
	noColor bool
}

// validTypes are the content types accepted by Confluence search.
var validTypes = map[string]bool{
	"page":       true,
	"blogpost":   true,
	"attachment": true,
	"comment":    true,
}

// NewCmdSearch creates the search command.
func NewCmdSearch() *cobra.Command {
	opts := &searchOptions{}

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search Confluence content",
		Long: `Search for pages, blog posts, attachments, and comments in Confluence.

Uses Confluence Query Language (CQL) under the hood. You can use the
convenient flags for common filters, or provide raw CQL for advanced queries.`,
		Example: `  # Full-text search across all content
  cfl search "deployment guide"

  # Search within a specific space
  cfl search "api docs" --space DEV

  # Find pages only
  cfl search "meeting notes" --type page

  # Filter by label
  cfl search --label documentation --space TEAM

  # Search by title
  cfl search --title "Release Notes"

  # Combine filters
  cfl search "kubernetes" --space DEV --type page --label infrastructure

  # Power user: raw CQL query
  cfl search --cql "type=page AND space=DEV AND lastModified > now('-7d')"

  # Output as JSON for scripting
  cfl search "config" -o json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.query = args[0]
			}
			opts.output, _ = cmd.Flags().GetString("output")
			opts.noColor, _ = cmd.Flags().GetBool("no-color")
			return runSearch(opts, nil)
		},
	}

	// Query building flags
	cmd.Flags().StringVar(&opts.cql, "cql", "", "Raw CQL query (advanced)")
	cmd.Flags().StringVarP(&opts.space, "space", "s", "", "Filter by space key")
	cmd.Flags().StringVarP(&opts.contentType, "type", "t", "", "Content type: page, blogpost, attachment, comment")
	cmd.Flags().StringVar(&opts.title, "title", "", "Filter by title (contains)")
	cmd.Flags().StringVar(&opts.label, "label", "", "Filter by label")

	// Pagination
	cmd.Flags().IntVarP(&opts.limit, "limit", "l", 25, "Maximum number of results")

	return cmd
}

func runSearch(opts *searchOptions, client *api.Client) error {
	// Validate output format
	if err := view.ValidateFormat(opts.output); err != nil {
		return err
	}

	// Validate type if provided
	if opts.contentType != "" && !validTypes[opts.contentType] {
		validList := []string{"page", "blogpost", "attachment", "comment"}
		return fmt.Errorf("invalid type %q: must be one of %s", opts.contentType, strings.Join(validList, ", "))
	}

	// Validate that we have something to search for
	if opts.cql == "" && opts.query == "" && opts.space == "" && opts.title == "" && opts.label == "" {
		return fmt.Errorf("search requires a query, --cql, or at least one filter (--space, --title, --label)")
	}

	// Validate limit
	if opts.limit < 0 {
		return fmt.Errorf("invalid limit: %d (must be >= 0)", opts.limit)
	}

	renderer := view.NewRenderer(view.Format(opts.output), opts.noColor)

	// Handle limit 0 - return empty
	if opts.limit == 0 {
		if opts.output == "json" {
			return renderer.RenderJSON([]interface{}{})
		}
		renderer.RenderText("No results.")
		return nil
	}

	// Create API client if not provided
	if client == nil {
		cfg, err := config.LoadWithEnv(config.DefaultConfigPath())
		if err != nil {
			return fmt.Errorf("failed to load config: %w (run 'cfl init' to configure)", err)
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid config: %w (run 'cfl init' to configure)", err)
		}

		// Use default space from config if not specified and no cql override
		if opts.space == "" && opts.cql == "" {
			opts.space = cfg.DefaultSpace
		}

		client = api.NewClient(cfg.URL, cfg.Email, cfg.APIToken)
	}

	// Build API options
	apiOpts := &api.SearchOptions{
		CQL:   opts.cql,
		Text:  opts.query,
		Space: opts.space,
		Type:  opts.contentType,
		Title: opts.title,
		Label: opts.label,
		Limit: opts.limit,
	}

	result, err := client.Search(context.Background(), apiOpts)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(result.Results) == 0 {
		renderer.RenderText("No results found.")
		return nil
	}

	// Render results
	headers := []string{"ID", "TYPE", "SPACE", "TITLE"}
	var rows [][]string

	for _, r := range result.Results {
		space := r.ResultGlobalContainer.Title
		rows = append(rows, []string{
			r.Content.ID,
			r.Content.Type,
			view.Truncate(space, 15),
			view.Truncate(r.Content.Title, 50),
		})
	}

	renderer.RenderList(headers, rows, result.HasMore())

	if result.HasMore() && opts.output != "json" {
		fmt.Fprintf(os.Stderr, "\n(showing %d of %d results, use --limit to see more)\n",
			len(result.Results), result.TotalSize)
	}

	return nil
}
