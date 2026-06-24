package page

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/atime"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

const historyMessageMaxChars = 80

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
	if err := view.ValidateFormat(opts.Output); err != nil {
		return err
	}
	if opts.limit < 0 {
		return fmt.Errorf("invalid limit: %d (must be >= 0)", opts.limit)
	}

	v := opts.View()
	if opts.limit == 0 {
		v.RenderText("No page versions found.")
		return nil
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
		v.RenderText("No page versions found.")
		return nil
	}

	if opts.idOnly {
		for _, version := range result.Results {
			_, _ = fmt.Fprintln(v.Out, version.Number)
		}
	} else {
		headers := []string{"VERSION", "CREATED", "AUTHOR", "MINOR", "MESSAGE"}
		rows := make([][]string, 0, len(result.Results))

		for _, version := range result.Results {
			rows = append(rows, []string{
				strconv.Itoa(version.Number),
				formatHistoryTime(version.CreatedAt),
				emptyDash(version.AuthorID),
				formatHistoryBool(version.MinorEdit),
				formatHistoryMessage(version.Message),
			})
		}

		if err := v.Table(headers, rows); err != nil {
			return err
		}
	}

	if result.HasMore() {
		nextCursor := extractHistoryCursor(result.Links.Next)
		if nextCursor != "" {
			_, _ = fmt.Fprintf(opts.Stderr, "\nNext page: cfl page history list %s --cursor %q\n", pageID, nextCursor)
		} else {
			_, _ = fmt.Fprintf(opts.Stderr, "\n(showing first %d results, use --limit to see more)\n", len(result.Results))
		}
	}

	return nil
}

func formatHistoryTime(t *atime.AtlassianTime) string {
	if t == nil || t.IsZero() {
		return "-"
	}
	return t.UTC().Format("2006-01-02T15:04:05Z07:00")
}

func formatHistoryBool(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func formatHistoryMessage(message string) string {
	if message == "" {
		return "-"
	}
	return view.Truncate(message, historyMessageMaxChars)
}

func emptyDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func extractHistoryCursor(nextLink string) string {
	if nextLink == "" {
		return ""
	}
	parsed, err := url.Parse(nextLink)
	if err != nil {
		return ""
	}
	return parsed.Query().Get("cursor")
}
