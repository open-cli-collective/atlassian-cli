package page

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	cflpresent "github.com/open-cli-collective/confluence-cli/internal/present"
	"github.com/open-cli-collective/confluence-cli/pkg/md"
)

type editOptions struct {
	*root.Options
	pageID   string
	title    string
	file     string
	editor   bool
	markdown *bool // nil = auto-detect, true = force markdown, false = force storage format
	legacy   bool  // Use legacy editor (storage format) instead of cloud editor (ADF)
	storage  bool  // Use storage representation directly (implies --no-markdown)
	parent   string
}

func newEditCmd(rootOpts *root.Options) *cobra.Command {
	opts := &editOptions{Options: rootOpts}

	cmd := &cobra.Command{
		Use:   "edit <page-id>",
		Short: "Edit an existing page",
		Long: `Edit an existing Confluence page.

By default, pages are updated using the cloud editor format (ADF).
Use --legacy to update pages in the legacy editor format.

Content can be provided via:
- --file flag to read from a file (use --file - to read from stdin)
- Standard input (pipe content)
- Interactive editor with --editor

Content format:
- Markdown is the default for stdin, editor, and .md files
- Use --no-markdown to provide raw Confluence format (XHTML for legacy, ADF JSON for cloud)
- Use --storage to provide raw Confluence storage format (XHTML) and send it directly
  via the storage representation API, regardless of the page's editor type
- Files with .html/.xhtml extensions are treated as storage format`,
		Example: `  # Edit a page in the editor with current content
  cfl page edit 12345 --editor

  # Update page content from file
  cfl page edit 12345 --file content.md

  # Update page in legacy format
  cfl page edit 12345 --file content.md --legacy

  # Update page content from stdin
  echo "# Updated Content" | cfl page edit 12345

  # Update from stdin via explicit --file - (e.g. piping HTML as storage)
  echo "<p>Updated</p>" | cfl page edit 12345 --file - --storage

  # Update page title only
  cfl page edit 12345 --title "New Title"

  # Move page to a new parent
  cfl page edit 12345 --parent 67890

  # Move page and update title
  cfl page edit 12345 --parent 67890 --title "New Title"

  # Pipe raw Confluence storage format (XHTML) directly
  echo "<p>Updated</p>" | cfl page edit 12345 --storage

  # Extract, transform, and re-upload storage-format content
  cfl page view 12345 --raw --content-only | \
    sed 's/old/new/g' | cfl page edit 12345 --storage`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.pageID = args[0]
			opts.storage, _ = cmd.Flags().GetBool("storage")
			if opts.storage {
				// --storage implies --no-markdown (input is raw XHTML)
				useMd := false
				opts.markdown = &useMd
			}
			if cmd.Flags().Changed("no-markdown") {
				noMd, _ := cmd.Flags().GetBool("no-markdown")
				useMd := !noMd
				opts.markdown = &useMd
			}
			opts.legacy, _ = cmd.Flags().GetBool("legacy")
			return runEdit(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.title, "title", "t", "", "New page title")
	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "Read content from file")
	cmd.Flags().StringVarP(&opts.parent, "parent", "p", "", "Move page to new parent page ID")
	cmd.Flags().BoolVar(&opts.editor, "editor", false, "Open editor for content")
	cmd.Flags().Bool("no-markdown", false, "Disable markdown conversion (use raw XHTML)")
	cmd.Flags().Bool("storage", false, "Input is Confluence storage format (XHTML); sends via storage representation API")
	cmd.Flags().Bool("legacy", false, "Edit page in legacy editor format (default: cloud editor)")

	return cmd
}

func runEdit(ctx context.Context, opts *editOptions) error {
	// Validate file exists before making any network calls so we fail
	// fast on bad input without needing config or API access. "-" means
	// stdin, which has no path to stat.
	if opts.file != "" && opts.file != "-" {
		if _, err := os.Stat(opts.file); err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
	}

	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	if opts.title == "" && opts.parent == "" &&
		!hasContentSource(opts.Options, opts.file, opts.editor) {
		return errMissingContentSource()
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	existingPage, err := getPageWithBodyFallback(ctx, client, opts.pageID)
	if err != nil {
		return err
	}

	newTitle := opts.title
	if newTitle == "" {
		newTitle = existingPage.Title
	}

	var newContent string
	hasNewContent := false

	hasStdinData := opts.Stdin != nil && opts.Stdin != os.Stdin
	if !hasStdinData {
		stat, _ := os.Stdin.Stat()
		hasStdinData = (stat.Mode() & os.ModeCharDevice) == 0
	}

	if opts.file != "" || opts.editor || hasStdinData {
		content, isMarkdown, err := getEditContent(opts, existingPage)
		if err != nil {
			return err
		}

		if strings.TrimSpace(content) == "" {
			return fmt.Errorf("page content cannot be empty")
		}

		newContent, err = convertEditContent(content, isMarkdown, opts.storage || opts.legacy)
		if err != nil {
			return err
		}
		hasNewContent = true
	}

	req := &api.UpdatePageRequest{
		ID:     opts.pageID,
		Status: "current",
		Title:  newTitle,
		Version: &api.Version{
			Number:  existingPage.Version.Number + 1,
			Message: "Updated via cfl",
		},
	}

	if hasNewContent {
		if opts.storage || opts.legacy {
			req.Body = &api.Body{
				Storage: &api.BodyRepresentation{
					Representation: "storage",
					Value:          newContent,
				},
			}
		} else {
			req.Body = &api.Body{
				AtlasDocFormat: &api.BodyRepresentation{
					Representation: "atlas_doc_format",
					Value:          newContent,
				},
			}
		}
	} else {
		req.Body = existingPage.Body
	}

	page, err := client.UpdatePage(ctx, opts.pageID, req)
	if err != nil {
		return err
	}

	if opts.parent != "" {
		if err := client.MovePage(ctx, opts.pageID, opts.parent); err != nil {
			return fmt.Errorf("moving page to new parent: %w", err)
		}
	}

	return cflpresent.Emit(opts.Options, cflpresent.PagePresenter{}.PresentEdit(page, cfg.URL, opts.legacy && hasNewContent))
}

// convertEditContent converts content based on markdown flag and legacy mode.
func convertEditContent(content string, isMarkdown, legacy bool) (string, error) {
	if legacy {
		if isMarkdown {
			converted, err := md.ToConfluenceStorage([]byte(content))
			if err != nil {
				return "", fmt.Errorf("converting markdown: %w", err)
			}
			return converted, nil
		}
		return content, nil
	}

	if isMarkdown {
		adfContent, err := md.ToADF([]byte(content))
		if err != nil {
			return "", fmt.Errorf("converting markdown to ADF: %w", err)
		}
		return adfContent, nil
	}
	return content, nil
}

// getEditContent reads content for editing and returns (content, isMarkdown, error).
func getEditContent(opts *editOptions, existingPage *api.Page) (string, bool, error) {
	useMarkdown := func(filename string) bool {
		if opts.markdown != nil {
			return *opts.markdown
		}
		if filename != "" {
			ext := strings.ToLower(filepath.Ext(filename))
			switch ext {
			case ".html", ".xhtml", ".htm":
				return false
			case ".md", ".markdown":
				return true
			}
		}
		return true
	}

	if opts.file == "-" {
		data, err := io.ReadAll(stdinReader(opts.Options))
		if err != nil {
			return "", false, fmt.Errorf("reading stdin: %w", err)
		}
		return string(data), useMarkdown(""), nil
	}

	if opts.file != "" {
		data, err := os.ReadFile(opts.file)
		if err != nil {
			return "", false, fmt.Errorf("reading file: %w", err)
		}
		return string(data), useMarkdown(opts.file), nil
	}

	if opts.Stdin != nil && opts.Stdin != os.Stdin {
		data, err := io.ReadAll(opts.Stdin)
		if err != nil {
			return "", false, fmt.Errorf("reading stdin: %w", err)
		}
		return string(data), useMarkdown(""), nil
	}

	if hasPipedOSStdin(opts.Options) {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", false, fmt.Errorf("reading stdin: %w", err)
		}
		return string(data), useMarkdown(""), nil
	}

	if !opts.editor {
		return "", false, errMissingContentSource()
	}

	isMarkdown := useMarkdown("")
	content, err := openEditorForEdit(existingPage, isMarkdown)
	return content, isMarkdown, err
}

func openEditorForEdit(existingPage *api.Page, isMarkdown bool) (string, error) {
	ext := ".html"
	if isMarkdown {
		ext = ".md"
	}

	existingContent := ""
	if existingPage.Body != nil && existingPage.Body.Storage != nil {
		existingContent = existingPage.Body.Storage.Value
	} else if existingPage.Body != nil && existingPage.Body.AtlasDocFormat != nil {
		// ADF-native page: convert to markdown for the editor.
		markdown, err := md.FromADF(existingPage.Body.AtlasDocFormat.Value)
		if err == nil {
			existingContent = markdown
		}
	}

	editContent := existingContent
	if isMarkdown && existingContent != "" {
		editContent = "<!-- Edit your content below. This is Confluence storage format. -->\n<!-- Use --no-markdown flag to edit raw storage format -->\n\n" + existingContent
	}

	tmpfile, err := os.CreateTemp("", "cfl-edit-*"+ext)
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	if _, err := tmpfile.WriteString(editContent); err != nil {
		return "", err
	}
	_ = tmpfile.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, tmpfile.Name()) //nolint:gosec // launching user's editor is intentional CLI behavior
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor failed: %w", err)
	}

	data, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return "", fmt.Errorf("reading edited content: %w", err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return "", fmt.Errorf("no content provided")
	}

	return content, nil
}
