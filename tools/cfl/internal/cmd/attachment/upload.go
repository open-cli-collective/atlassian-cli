package attachment

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-cli-collective/atlassian-go/view"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/config"
)

type uploadOptions struct {
	pageID  string
	file    string
	comment string
	output  string
	noColor bool
}

// NewCmdUpload creates the attachment upload command.
func NewCmdUpload() *cobra.Command {
	opts := &uploadOptions{}

	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload an attachment to a page",
		Long:  `Upload a file as an attachment to a Confluence page.`,
		Example: `  # Upload a file
  cfl attachment upload --page 12345 --file document.pdf

  # Upload with a comment (-m for message/comment)
  cfl attachment upload --page 12345 --file image.png -m "Screenshot"`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.output, _ = cmd.Flags().GetString("output")
			opts.noColor, _ = cmd.Flags().GetBool("no-color")
			return runUpload(opts, nil)
		},
	}

	cmd.Flags().StringVarP(&opts.pageID, "page", "p", "", "Page ID (required)")
	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "File to upload (required)")
	cmd.Flags().StringVarP(&opts.comment, "comment", "m", "", "Comment for the attachment")

	_ = cmd.MarkFlagRequired("page")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func runUpload(opts *uploadOptions, client *api.Client) error {
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

	// Open file
	file, err := os.Open(opts.file)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Get filename from path
	filename := filepath.Base(opts.file)

	// Upload attachment
	attachment, err := client.UploadAttachment(context.Background(), opts.pageID, filename, file, opts.comment)
	if err != nil {
		return fmt.Errorf("failed to upload attachment: %w", err)
	}

	// Render output
	v := view.New(view.Format(opts.output), opts.noColor)

	if opts.output == "json" {
		return v.JSON(attachment)
	}

	v.Success("Uploaded: %s", filename)
	v.RenderKeyValue("ID", attachment.ID)
	v.RenderKeyValue("Title", attachment.Title)
	v.RenderKeyValue("Size", formatFileSize(attachment.FileSize))

	return nil
}
