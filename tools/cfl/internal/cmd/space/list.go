package space

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
	limit     int
	spaceType string
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
  cfl space list -o json`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runList(opts)
		},
	}

	cmd.Flags().IntVarP(&opts.limit, "limit", "l", 25, "Maximum number of spaces to return")
	cmd.Flags().StringVarP(&opts.spaceType, "type", "t", "", "Filter by space type (global, personal)")

	return cmd
}

func runList(opts *listOptions) error {
	if err := view.ValidateFormat(opts.Output); err != nil {
		return err
	}

	if opts.limit < 0 {
		return fmt.Errorf("invalid limit: %d (must be >= 0)", opts.limit)
	}

	v := opts.View()

	if opts.limit == 0 {
		if opts.Output == "json" {
			return v.JSON([]interface{}{})
		}
		v.RenderText("No spaces found.")
		return nil
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	apiOpts := &api.ListSpacesOptions{
		Limit: opts.limit,
		Type:  opts.spaceType,
	}

	result, err := client.ListSpaces(context.Background(), apiOpts)
	if err != nil {
		return fmt.Errorf("failed to list spaces: %w", err)
	}

	if len(result.Results) == 0 {
		v.RenderText("No spaces found.")
		return nil
	}

	headers := []string{"KEY", "NAME", "TYPE", "DESCRIPTION"}
	var rows [][]string

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

	_ = v.RenderList(headers, rows, result.HasMore())

	if result.HasMore() && opts.Output != "json" {
		fmt.Fprintf(os.Stderr, "\n(showing first %d results, use --limit to see more)\n", len(result.Results))
	}

	return nil
}
