package attachment

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/config"
	"github.com/open-cli-collective/confluence-cli/internal/view"
)

type listOptions struct {
	pageID  string
	limit   int
	unused  bool
	output  string
	noColor bool
}

// NewCmdList creates the attachment list command.
func NewCmdList() *cobra.Command {
	opts := &listOptions{}

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
			opts.output, _ = cmd.Flags().GetString("output")
			opts.noColor, _ = cmd.Flags().GetBool("no-color")
			return runList(opts, nil)
		},
	}

	cmd.Flags().StringVarP(&opts.pageID, "page", "p", "", "Page ID (required)")
	cmd.Flags().IntVarP(&opts.limit, "limit", "l", 25, "Maximum number of attachments to return")
	cmd.Flags().BoolVar(&opts.unused, "unused", false, "Show only attachments not referenced in page content")

	_ = cmd.MarkFlagRequired("page")

	return cmd
}

func runList(opts *listOptions, client *api.Client) error {
	// Validate output format
	if err := view.ValidateFormat(opts.output); err != nil {
		return err
	}

	// Create API client if not provided (allows injection for testing)
	if client == nil {
		cfg, err := config.LoadWithEnv(config.DefaultConfigPath())
		if err != nil {
			return fmt.Errorf("failed to load config: %w (run 'cfl init' to configure)", err)
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid config: %w (run 'cfl init' to configure)", err)
		}

		client = api.NewClient(cfg.URL, cfg.Email, cfg.APIToken)
	}

	// List attachments
	apiOpts := &api.ListAttachmentsOptions{
		Limit: opts.limit,
	}

	result, err := client.ListAttachments(context.Background(), opts.pageID, apiOpts)
	if err != nil {
		return fmt.Errorf("failed to list attachments: %w", err)
	}

	attachments := result.Results

	// Filter to unused attachments if requested
	if opts.unused {
		// Fetch page content in storage format
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

	// Render output
	renderer := view.NewRenderer(view.Format(opts.output), opts.noColor)

	// Build table rows
	headers := []string{"ID", "Title", "Media Type", "File Size"}
	var rows [][]string
	for _, att := range attachments {
		size := formatFileSize(att.FileSize)
		rows = append(rows, []string{att.ID, att.Title, att.MediaType, size})
	}

	// Handle empty result for non-JSON output
	if len(attachments) == 0 && opts.output != "json" {
		if opts.unused {
			fmt.Println("No unused attachments found.")
		} else {
			fmt.Println("No attachments found.")
		}
		return nil
	}

	renderer.RenderList(headers, rows, result.HasMore())

	if result.HasMore() && opts.output != "json" {
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
	// Check for ri:filename="attachment.ext" pattern (most common)
	if strings.Contains(content, fmt.Sprintf(`ri:filename="%s"`, filename)) {
		return true
	}

	// Check for URL-encoded filename in href (e.g., spaces become %20)
	encodedFilename := strings.ReplaceAll(filename, " ", "%20")
	if strings.Contains(content, encodedFilename) {
		return true
	}

	// Check for plain filename reference (fallback)
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
