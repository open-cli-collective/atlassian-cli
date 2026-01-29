package attachment

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
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
		RunE: func(_ *cobra.Command, _ []string) error {
			return runList(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.pageID, "page", "p", "", "Page ID (required)")
	cmd.Flags().IntVarP(&opts.limit, "limit", "l", 25, "Maximum number of attachments to return")
	cmd.Flags().BoolVar(&opts.unused, "unused", false, "Show only attachments not referenced in page content")

	_ = cmd.MarkFlagRequired("page")

	return cmd
}

func runList(opts *listOptions) error {
	if err := view.ValidateFormat(opts.Output); err != nil {
		return err
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	apiOpts := &api.ListAttachmentsOptions{
		Limit: opts.limit,
	}

	result, err := client.ListAttachments(context.Background(), opts.pageID, apiOpts)
	if err != nil {
		return fmt.Errorf("failed to list attachments: %w", err)
	}

	attachments := result.Results

	if opts.unused {
		page, err := client.GetPage(context.Background(), opts.pageID, &api.GetPageOptions{
			BodyFormat: "storage",
		})
		if err != nil {
			return fmt.Errorf("failed to get page content: %w", err)
		}

		pageContent := ""
		if page.Body != nil && page.Body.Storage != nil {
			pageContent = page.Body.Storage.Value
		}

		attachments = filterUnusedAttachments(attachments, pageContent)
	}

	v := opts.View()

	headers := []string{"ID", "Title", "Media Type", "File Size"}
	var rows [][]string
	for _, att := range attachments {
		size := formatFileSize(att.FileSize)
		rows = append(rows, []string{att.ID, att.Title, att.MediaType, size})
	}

	if len(attachments) == 0 && opts.Output != "json" {
		if opts.unused {
			fmt.Println("No unused attachments found.")
		} else {
			fmt.Println("No attachments found.")
		}
		return nil
	}

	_ = v.RenderList(headers, rows, result.HasMore())

	if result.HasMore() && opts.Output != "json" {
		fmt.Fprintf(os.Stderr, "\n(showing first %d results, use --limit to see more)\n", len(attachments))
	}

	return nil
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

func formatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
