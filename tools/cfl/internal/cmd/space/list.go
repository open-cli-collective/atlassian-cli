package space

import (
	"context"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/confluence-cli/api"
	cflartifact "github.com/open-cli-collective/confluence-cli/internal/artifact"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
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

  # Output as JSON
  cfl space list -o json

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
	if err := view.ValidateFormat(opts.Output); err != nil {
		return err
	}

	if opts.limit < 0 {
		return fmt.Errorf("invalid limit: %d (must be >= 0)", opts.limit)
	}

	v := opts.View()

	if opts.limit == 0 {
		if opts.Output == "json" {
			arts := cflartifact.ProjectSpaces(nil, opts.ArtifactMode())
			return v.RenderArtifactList(artifact.NewListResult(arts, false))
		}
		v.RenderText("No spaces found.")
		return nil
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
		if opts.Output == "json" {
			arts := cflartifact.ProjectSpaces(nil, opts.ArtifactMode())
			return v.RenderArtifactList(artifact.NewListResult(arts, false))
		}
		v.RenderText("No spaces found.")
		return nil
	}

	// JSON output uses artifact projection
	if opts.Output == "json" {
		arts := cflartifact.ProjectSpaces(result.Results, opts.ArtifactMode())
		return v.RenderArtifactList(artifact.NewListResult(arts, result.HasMore()))
	}

	// Table output
	headers := []string{"KEY", "NAME", "TYPE", "DESCRIPTION"}
	rows := make([][]string, 0, len(result.Results))

	for _, space := range result.Results {
		desc := ""
		if space.Description != nil && space.Description.Plain != nil {
			desc = view.Truncate(space.Description.Plain.Value, 50)
		}
		rows = append(rows, []string{
			space.Key,
			space.Name,
			space.Type,
			desc,
		})
	}

	if err := v.Table(headers, rows); err != nil {
		return err
	}

	if result.HasMore() {
		nextCursor := extractCursor(result.Links.Next)
		if nextCursor != "" {
			_, _ = fmt.Fprintf(opts.Stderr, "\nNext page: cfl space list --cursor %q\n", nextCursor)
		} else {
			_, _ = fmt.Fprintf(opts.Stderr, "\n(showing first %d results, use --limit to see more)\n", len(result.Results))
		}
	}

	return nil
}

// extractCursor parses the cursor query parameter from a next link URL.
func extractCursor(nextLink string) string {
	if nextLink == "" {
		return ""
	}
	parsed, err := url.Parse(nextLink)
	if err != nil {
		return ""
	}
	return parsed.Query().Get("cursor")
}
