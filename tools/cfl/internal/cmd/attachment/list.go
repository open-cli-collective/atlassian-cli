package attachment

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	"github.com/open-cli-collective/confluence-cli/internal/pageview"
	cflpresent "github.com/open-cli-collective/confluence-cli/internal/present"
)

type listOptions struct {
	*root.Options
	pageID string
	limit  int
	unused bool
}

func newListCmd(rootOpts *root.Options) *cobra.Command {
	opts := &listOptions{Options: rootOpts}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List attachments on a page",
		Long:    `List all attachments on a Confluence page.`,
		Example: `  # List attachments on a page
  cfl attachment list --page 12345

  # List with custom limit
  cfl attachment list --page 12345 --limit 50

  # List unused (orphaned) attachments not referenced in page content
  cfl attachment list --page 12345 --unused`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return err
			}
			return validateListOptions(opts)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.pageID, "page", "p", "", "Page ID (required)")
	cmd.Flags().IntVarP(&opts.limit, "limit", "l", 25, "Maximum number of attachments to return (must be greater than 0)")
	cmd.Flags().BoolVar(&opts.unused, "unused", false, "Show only attachments not referenced in page content")

	_ = cmd.MarkFlagRequired("page")

	return cmd
}

func runList(ctx context.Context, opts *listOptions) error {
	if err := validateListOptions(opts); err != nil {
		return err
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if opts.unused {
		return runListUnused(ctx, client, opts)
	}

	result, err := client.ListAttachments(ctx, opts.pageID, &api.ListAttachmentsOptions{Limit: opts.limit})
	if err != nil {
		return fmt.Errorf("listing attachments: %w", err)
	}
	return emitAttachments(opts, result.Results, result.HasMore())
}

// runListUnused pages through every attachment until it has found one more
// unused attachment than --limit, so the "more results" hint is exact.
func runListUnused(ctx context.Context, client *api.Client, opts *listOptions) error {
	page, err := pageview.GetPageWithBodyFallback(ctx, client, opts.pageID)
	if err != nil {
		return fmt.Errorf("getting page content: %w", err)
	}
	refs, err := pageAttachmentRefs(page)
	if err != nil {
		return err
	}

	var unused []api.Attachment
	cursor := ""
	for len(unused) <= opts.limit {
		result, err := client.ListAttachments(ctx, opts.pageID, &api.ListAttachmentsOptions{
			Limit:  opts.limit,
			Cursor: cursor,
		})
		if err != nil {
			return fmt.Errorf("listing attachments: %w", err)
		}
		for _, att := range result.Results {
			if !refs.references(att) {
				unused = append(unused, att)
			}
		}
		if !result.HasMore() {
			break
		}
		cursor = cflpresent.ExtractCursor(result.Links.Next)
		if cursor == "" {
			return fmt.Errorf("listing attachments: next page link has no cursor: %q", result.Links.Next)
		}
	}

	hasMore := len(unused) > opts.limit
	if hasMore {
		unused = unused[:opts.limit]
	}
	return emitAttachments(opts, unused, hasMore)
}

func emitAttachments(opts *listOptions, attachments []api.Attachment, hasMore bool) error {
	if len(attachments) == 0 {
		return cflpresent.Emit(opts.Options, cflpresent.AttachmentPresenter{}.PresentEmpty(opts.unused))
	}
	return cflpresent.Emit(opts.Options, cflpresent.AttachmentPresenter{}.PresentList(attachments, opts.Full, hasMore))
}

func validateListOptions(opts *listOptions) error {
	if opts.limit <= 0 {
		return fmt.Errorf("invalid limit: %d (must be greater than 0)", opts.limit)
	}
	return nil
}
