package attachment

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/config"
	"github.com/open-cli-collective/confluence-cli/internal/view"
)

type downloadOptions struct {
	output     string
	outputFile string
	noColor    bool
	force      bool
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
  cfl attachment download abc123 -O document.pdf`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.output, _ = cmd.Flags().GetString("output")
			opts.noColor, _ = cmd.Flags().GetBool("no-color")
			return runDownload(args[0], opts, nil)
		},
	}

	cmd.Flags().StringVarP(&opts.outputFile, "output-file", "O", "", "Output file path (default: original filename)")
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Overwrite existing file without warning")

	return cmd
}

func runDownload(attachmentID string, opts *downloadOptions, client *api.Client) error {
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

	// Get attachment info first to get the filename
	attachment, err := client.GetAttachment(context.Background(), attachmentID)
	if err != nil {
		return fmt.Errorf("failed to get attachment info: %w", err)
	}

	// Determine output filename
	outputPath := opts.outputFile
	if outputPath == "" {
		// Sanitize filename to prevent path traversal attacks
		outputPath = filepath.Base(attachment.Title)
		if outputPath == "" || outputPath == "." || outputPath == ".." {
			return fmt.Errorf("invalid attachment filename: %q", attachment.Title)
		}
	}

	// Check if file already exists (unless --force is used)
	if !opts.force {
		if _, err := os.Stat(outputPath); err == nil {
			return fmt.Errorf("file already exists: %s (use --force to overwrite)", outputPath)
		}
	}

	// Download the attachment
	reader, err := client.DownloadAttachment(context.Background(), attachmentID)
	if err != nil {
		return fmt.Errorf("failed to download attachment: %w", err)
	}
	defer func() { _ = reader.Close() }()

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = outFile.Close() }()

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
