package attachment

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rianjs/confluence-cli/api"
	"github.com/rianjs/confluence-cli/internal/config"
	"github.com/rianjs/confluence-cli/internal/view"
)

type listOptions struct {
	pageID  string
	limit   int
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
  cfl attachment list --page 12345 --limit 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.output, _ = cmd.Flags().GetString("output")
			opts.noColor, _ = cmd.Flags().GetBool("no-color")
			return runList(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.pageID, "page", "p", "", "Page ID (required)")
	cmd.Flags().IntVarP(&opts.limit, "limit", "l", 25, "Maximum number of attachments to return")

	_ = cmd.MarkFlagRequired("page")

	return cmd
}

func runList(opts *listOptions) error {
	// Load config
	cfg, err := config.LoadWithEnv(config.DefaultConfigPath())
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'cfl init' to configure)", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w (run 'cfl init' to configure)", err)
	}

	// Create API client
	client := api.NewClient(cfg.URL, cfg.Email, cfg.APIToken)

	// List attachments
	apiOpts := &api.ListAttachmentsOptions{
		Limit: opts.limit,
	}

	result, err := client.ListAttachments(context.Background(), opts.pageID, apiOpts)
	if err != nil {
		return fmt.Errorf("failed to list attachments: %w", err)
	}

	// Render output
	renderer := view.NewRenderer(view.Format(opts.output), opts.noColor)

	if opts.output == "json" {
		return renderer.RenderJSON(result.Results)
	}

	if len(result.Results) == 0 {
		fmt.Println("No attachments found.")
		return nil
	}

	// Render as table
	headers := []string{"ID", "Title", "Media Type", "File Size"}
	var rows [][]string
	for _, att := range result.Results {
		size := formatFileSize(att.FileSize)
		rows = append(rows, []string{att.ID, att.Title, att.MediaType, size})
	}

	renderer.RenderTable(headers, rows)

	return nil
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
