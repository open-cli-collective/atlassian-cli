package attachment

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

type downloadOptions struct {
	*root.Options
	outputFile string
	force      bool
}

func newDownloadCmd(rootOpts *root.Options) *cobra.Command {
	opts := &downloadOptions{Options: rootOpts}

	cmd := &cobra.Command{
		Use:   "download <attachment-id>",
		Short: "Download an attachment",
		Long:  `Download an attachment by its ID.`,
		Example: `  # Download an attachment
  cfl attachment download abc123

  # Download to a specific file
  cfl attachment download abc123 -O document.pdf`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runDownload(args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.outputFile, "output-file", "O", "", "Output file path (default: original filename)")
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Overwrite existing file without warning")

	return cmd
}

func runDownload(attachmentID string, opts *downloadOptions) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	attachment, err := client.GetAttachment(context.Background(), attachmentID)
	if err != nil {
		return fmt.Errorf("failed to get attachment info: %w", err)
	}

	outputPath := opts.outputFile
	if outputPath == "" {
		outputPath = filepath.Base(attachment.Title)
		if outputPath == "" || outputPath == "." || outputPath == ".." {
			return fmt.Errorf("invalid attachment filename: %q", attachment.Title)
		}
	}

	if !opts.force {
		if _, err := os.Stat(outputPath); err == nil {
			return fmt.Errorf("file already exists: %s (use --force to overwrite)", outputPath)
		}
	}

	reader, err := client.DownloadAttachment(context.Background(), attachmentID)
	if err != nil {
		return fmt.Errorf("failed to download attachment: %w", err)
	}
	defer func() { _ = reader.Close() }()

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = outFile.Close() }()

	bytesWritten, err := io.Copy(outFile, reader)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	v := opts.View()

	v.Success("Downloaded: %s", outputPath)
	v.RenderKeyValue("Size", formatFileSize(bytesWritten))

	return nil
}
