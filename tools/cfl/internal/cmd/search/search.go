// Package search provides the search command for finding Confluence content.
package search

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	cflpresent "github.com/open-cli-collective/confluence-cli/internal/present"
)

type searchOptions struct {
	*root.Options

	// Query building
	query       string // Positional arg: free-text search
	cql         string // Raw CQL (power users)
	space       string // Filter by space key
	contentType string // page, blogpost, attachment, comment
	title       string // Title contains
	label       string // Label filter

	// Pagination
	limit int
}

// validTypes are the content types accepted by Confluence search.
var validTypes = map[string]bool{
	"page":       true,
	"blogpost":   true,
	"attachment": true,
	"comment":    true,
}

// Register adds the search command to the root command.
func Register(rootCmd *cobra.Command, opts *root.Options) {
	rootCmd.AddCommand(newSearchCmd(opts))
}

// newSearchCmd creates the search command.
func newSearchCmd(rootOpts *root.Options) *cobra.Command {
	opts := &searchOptions{Options: rootOpts}

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
  cfl search --cql "type=page AND space=DEV AND lastModified > now('-7d')"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.query = args[0]
			}
			return runSearch(cmd.Context(), opts)
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

func runSearch(ctx context.Context, opts *searchOptions) error {
	// Validate type if provided
	if opts.contentType != "" && !validTypes[opts.contentType] {
		validList := []string{"page", "blogpost", "attachment", "comment"}
		return fmt.Errorf("invalid type %q: must be one of %s", opts.contentType, strings.Join(validList, ", "))
	}

	// Validate that we have something to search for
	if opts.cql == "" && opts.query == "" && opts.space == "" && opts.contentType == "" && opts.title == "" && opts.label == "" {
		return fmt.Errorf("search requires a query, --cql, or at least one filter (--space, --type, --title, --label)")
	}

	// Validate limit
	if opts.limit < 0 {
		return fmt.Errorf("invalid limit: %d (must be >= 0)", opts.limit)
	}

	if opts.limit == 0 {
		return cflpresent.Emit(opts.Options, cflpresent.SearchPresenter{}.PresentEmpty())
	}

	// Get config for default space
	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	// Use default space from config if not specified and no cql override
	if opts.space == "" && opts.cql == "" {
		opts.space = cfg.DefaultSpace
	}

	// Get API client
	client, err := opts.APIClient()
	if err != nil {
		return err
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

	result, err := client.Search(ctx, apiOpts)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(result.Results) == 0 {
		return cflpresent.Emit(opts.Options, cflpresent.SearchPresenter{}.PresentEmpty())
	}
	return cflpresent.Emit(opts.Options, cflpresent.SearchPresenter{}.PresentList(result.Results, opts.Full, result.TotalSize, result.HasMore()))
}
