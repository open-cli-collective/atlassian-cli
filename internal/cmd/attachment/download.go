package attachment

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/rianjs/confluence-cli/api"
	"github.com/rianjs/confluence-cli/internal/config"
	"github.com/rianjs/confluence-cli/internal/view"
)

type downloadOptions struct {
	output     string
	outputFile string
	noColor    bool
}

// NewCmdDownload creates the attachment download command.
func NewCmdDownload() *cobra.Command {
	opts := &downloadOptions{}

	cmd := &cobra.Command{
		Use:   "download <attachment-id>",
		Short: "Download an attachment",
		Long:  `Download an attachment by its ID.`,
		Example: `  # Download an attachment
  cfl attachment download abc123

  # Download to a specific file
  cfl attachment download abc123 --output document.pdf`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.output, _ = cmd.Flags().GetString("output")
			opts.noColor, _ = cmd.Flags().GetBool("no-color")
			return runDownload(args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.outputFile, "output-file", "o", "", "Output file path (default: original filename)")

	return cmd
}

func runDownload(attachmentID string, opts *downloadOptions) error {
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

	// Get attachment info first to get the filename
	attachment, err := client.GetAttachment(context.Background(), attachmentID)
	if err != nil {
		return fmt.Errorf("failed to get attachment info: %w", err)
	}

	// Determine output filename
	outputPath := opts.outputFile
	if outputPath == "" {
		outputPath = attachment.Title
	}

	// Download the attachment
	reader, err := client.DownloadAttachment(context.Background(), attachmentID)
	if err != nil {
		return fmt.Errorf("failed to download attachment: %w", err)
	}
	defer reader.Close()

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Copy content
	bytesWritten, err := io.Copy(outFile, reader)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Render output
	renderer := view.NewRenderer(view.Format(opts.output), opts.noColor)

	renderer.Success(fmt.Sprintf("Downloaded: %s", outputPath))
	renderer.RenderKeyValue("Size", formatFileSize(bytesWritten))

	return nil
}
