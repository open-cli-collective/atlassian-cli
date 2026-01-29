package attachment

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

type uploadOptions struct {
	*root.Options
	pageID  string
	file    string
	comment string
}

func newUploadCmd(rootOpts *root.Options) *cobra.Command {
	opts := &uploadOptions{Options: rootOpts}

	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload an attachment to a page",
		Long:  `Upload a file as an attachment to a Confluence page.`,
		Example: `  # Upload a file
  cfl attachment upload --page 12345 --file document.pdf

  # Upload with a comment (-m for message/comment)
  cfl attachment upload --page 12345 --file image.png -m "Screenshot"`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runUpload(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.pageID, "page", "p", "", "Page ID (required)")
	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "File to upload (required)")
	cmd.Flags().StringVarP(&opts.comment, "comment", "m", "", "Comment for the attachment")

	_ = cmd.MarkFlagRequired("page")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func runUpload(opts *uploadOptions) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	file, err := os.Open(opts.file)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	filename := filepath.Base(opts.file)

	attachment, err := client.UploadAttachment(context.Background(), opts.pageID, filename, file, opts.comment)
	if err != nil {
		return fmt.Errorf("failed to upload attachment: %w", err)
	}

	v := opts.View()

	if opts.Output == "json" {
		return v.JSON(attachment)
	}

	v.Success("Uploaded: %s", filename)
	v.RenderKeyValue("ID", attachment.ID)
	v.RenderKeyValue("Title", attachment.Title)
	v.RenderKeyValue("Size", formatFileSize(attachment.FileSize))

	return nil
}
