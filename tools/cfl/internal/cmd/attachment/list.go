package attachment

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.pageID, "page", "p", "", "Page ID (required)")
	cmd.Flags().IntVarP(&opts.limit, "limit", "l", 25, "Maximum number of attachments to return")
	cmd.Flags().BoolVar(&opts.unused, "unused", false, "Show only attachments not referenced in page content")

	_ = cmd.MarkFlagRequired("page")

	return cmd
}

func runList(ctx context.Context, opts *listOptions) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	apiOpts := &api.ListAttachmentsOptions{
		Limit: opts.limit,
	}

	result, err := client.ListAttachments(ctx, opts.pageID, apiOpts)
	if err != nil {
		return fmt.Errorf("listing attachments: %w", err)
	}

	attachments := result.Results

	if opts.unused {
		page, err := client.GetPage(ctx, opts.pageID, &api.GetPageOptions{
			BodyFormat: "storage",
		})
		if err != nil {
			return fmt.Errorf("getting page content: %w", err)
		}

		pageContent := ""
		if page.Body != nil && page.Body.Storage != nil {
			pageContent = page.Body.Storage.Value
		}

		attachments = filterUnusedAttachments(attachments, pageContent)
	}

	if len(attachments) == 0 {
		return cflpresent.Emit(opts.Options, cflpresent.AttachmentPresenter{}.PresentEmpty(opts.unused))
	}

	return cflpresent.Emit(opts.Options, cflpresent.AttachmentPresenter{}.PresentList(attachments, opts.Full, result.HasMore()))
}

// filterUnusedAttachments returns attachments that are not referenced in the page content.
// Confluence references attachments in storage format as:
//   - <ri:attachment ri:filename="example.png"/>
//   - Attachment filename may also appear in href attributes
func filterUnusedAttachments(attachments []api.Attachment, pageContent string) []api.Attachment {
	var unused []api.Attachment
	for _, att := range attachments {
		if !isAttachmentReferenced(att.Title, pageContent) {
			unused = append(unused, att)
		}
	}
	return unused
}

// isAttachmentReferenced checks if an attachment filename appears in page content.
func isAttachmentReferenced(filename, content string) bool {
	if strings.Contains(content, fmt.Sprintf(`ri:filename="%s"`, filename)) {
		return true
	}

	encodedFilename := strings.ReplaceAll(filename, " ", "%20")
	if strings.Contains(content, encodedFilename) {
		return true
	}

	if strings.Contains(content, filename) {
		return true
	}

	return false
}
