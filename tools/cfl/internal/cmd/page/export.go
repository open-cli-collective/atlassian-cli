package page

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	cflpresent "github.com/open-cli-collective/confluence-cli/internal/present"
)

// exportFormatPDF is the only representation Confluence renders for a
// single page through its export task.
const exportFormatPDF = "pdf"

// pollInterval paces the progress reads while Confluence renders.
const pollInterval = 2 * time.Second

type exportOptions struct {
	*root.Options
	format     string
	outputFile string
	force      bool
	timeout    time.Duration
}

func newExportCmd(rootOpts *root.Options) *cobra.Command {
	opts := &exportOptions{Options: rootOpts}

	cmd := &cobra.Command{
		Use:   "export <page-id>",
		Short: "Export a page to a file",
		Long: `Export a Confluence page as a PDF.

Confluence renders the document server-side, so the export runs as a task
that is polled until it finishes and the result is then downloaded.`,
		Example: `  # Export to a file named after the page
  cfl page export 123456

  # Export to a specific file
  cfl page export 123456 -O handoff.pdf

  # Overwrite an existing file
  cfl page export 123456 -O handoff.pdf --force

  # Allow longer for a large page
  cfl page export 123456 --timeout 10m`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(cmd.Context(), args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.format, "format", exportFormatPDF, "Export format: pdf")
	cmd.Flags().StringVarP(&opts.outputFile, "output-file", "O", "", "Output file path (default: derived from the page title)")
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Overwrite existing file without warning")
	cmd.Flags().DurationVar(&opts.timeout, "timeout", 5*time.Minute, "How long to wait for Confluence to render the export")

	return cmd
}

func runExport(ctx context.Context, pageID string, opts *exportOptions) error {
	if err := validateExportFormat(opts.format); err != nil {
		return err
	}
	if opts.timeout <= 0 {
		return fmt.Errorf("invalid --timeout: %s (must be greater than zero)", opts.timeout)
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	outputPath, err := exportOutputPath(ctx, client, pageID, opts)
	if err != nil {
		return err
	}

	// Check before starting so a refusal costs no server-side render.
	if !opts.force {
		if _, err := os.Stat(outputPath); err == nil {
			return fmt.Errorf("file already exists: %s (use --force to overwrite)", outputPath)
		}
	}

	export, err := client.StartPDFExport(ctx, pageID)
	if err != nil {
		return err
	}

	progress, err := awaitExport(ctx, client, export, opts)
	if err != nil {
		return err
	}

	reader, err := client.OpenPDFExport(ctx, export, progress)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()

	bytesWritten, err := writeExport(outputPath, reader)
	if err != nil {
		return err
	}

	return cflpresent.Emit(opts.Options, cflpresent.PagePresenter{}.PresentExport(outputPath, bytesWritten))
}

// validateExportFormat holds --format to the formats the command can
// actually produce, so an unsupported value fails before any work starts.
func validateExportFormat(format string) error {
	if format == exportFormatPDF {
		return nil
	}
	return fmt.Errorf("invalid export format: %q (valid formats: %s)", format, exportFormatPDF)
}

// exportOutputPath resolves where the document is written. An explicit path
// is used as given; otherwise the page title names the file, which costs a
// read of the page.
func exportOutputPath(ctx context.Context, client *api.Client, pageID string, opts *exportOptions) (string, error) {
	if opts.outputFile != "" {
		return opts.outputFile, nil
	}

	page, err := client.GetPage(ctx, pageID, nil)
	if err != nil {
		return "", fmt.Errorf("getting page: %w", err)
	}

	return exportFilename(page.Title, pageID), nil
}

// exportFilename turns a page title into a filename. Titles are free text
// and carry separators, so the result is reduced to a single path element
// and falls back to the page ID when a title leaves nothing usable.
func exportFilename(title, pageID string) string {
	cleaned := strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		}
		if r < 32 {
			return ' '
		}
		return r
	}, title)

	cleaned = strings.TrimSpace(filepath.Base(strings.TrimSpace(cleaned)))
	if cleaned == "" || cleaned == "." || cleaned == ".." {
		cleaned = pageID
	}

	return cleaned + ".pdf"
}

// awaitExport polls until Confluence finishes rendering, reporting progress
// on stderr so a wait is distinguishable from a stall.
func awaitExport(ctx context.Context, client *api.Client, export *api.PDFExport, opts *exportOptions) (*api.PDFExportProgress, error) {
	ctx, cancel := context.WithTimeout(ctx, opts.timeout)
	defer cancel()

	lastReported := -1
	for {
		progress, err := client.GetPDFExportProgress(ctx, export)
		if err != nil {
			return nil, fmt.Errorf("waiting for export: %w", err)
		}
		if progress.Failed() {
			return nil, api.ErrExportFailed
		}
		if progress.Done() {
			return progress, nil
		}

		if progress.Progress != lastReported {
			lastReported = progress.Progress
			_ = cflpresent.Emit(opts.Options, cflpresent.PagePresenter{}.PresentExportProgress(progress.Progress))
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("waiting for export: %w (use --timeout to wait longer)", ctx.Err())
		case <-time.After(pollInterval):
		}
	}
}

// writeExport streams the document to disk and reports its size.
func writeExport(outputPath string, reader io.Reader) (int64, error) {
	outFile, err := os.Create(outputPath) //nolint:gosec // CLI tool creates user-specified output file
	if err != nil {
		return 0, fmt.Errorf("creating output file: %w", err)
	}
	defer func() { _ = outFile.Close() }()

	bytesWritten, err := io.Copy(outFile, reader)
	if err != nil {
		return 0, fmt.Errorf("writing file: %w", err)
	}

	return bytesWritten, nil
}
