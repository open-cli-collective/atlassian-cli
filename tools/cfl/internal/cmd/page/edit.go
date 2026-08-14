package page

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	cflpresent "github.com/open-cli-collective/confluence-cli/internal/present"
	"github.com/open-cli-collective/confluence-cli/pkg/md"
)

type editOptions struct {
	*root.Options
	pageID             string
	title              string
	file               string
	editor             bool
	bodyFormat         string
	bodyFormatExplicit bool
	legacy             bool
	parent             string
	noVerify           bool
	allowLossy         bool
}

func newEditCmd(rootOpts *root.Options) *cobra.Command {
	opts := &editOptions{Options: rootOpts}

	cmd := &cobra.Command{
		Use:   "edit <page-id>",
		Short: "Edit an existing page",
		Long: `Edit an existing Confluence page.

Markdown input is converted to cloud editor format (ADF) by default.
Use --legacy with Markdown to update pages in storage XHTML instead.

Content can be provided via:
- --file flag to read from a file (use --file - to read from stdin)
- Standard input (pipe content)
- Interactive editor with --editor

Content format is selected with --body-format markdown|adf|xhtml.
Omitting --body-format means Markdown. ADF and XHTML are validated or sent
without conversion.

ADF IS LOSSY. Confluence's ADF representation does not carry everything its
storage representation does: emphasis wrapping an inline code span is
dropped, and internal links become plain expanded URLs. Reading a page as
ADF and writing it back therefore destroys content the caller never touched,
and the caller cannot see it because their copy is already missing it. A
write that would remove such content is refused; use --body-format xhtml to
edit those pages, or --allow-lossy to proceed anyway.

For --body-format adf or xhtml the page is read back after writing and the
stored body compared against what was sent, because Confluence can store
something other than what it was given. Losing text is an error; attributes
the server normalizes away are reported and tolerated. Markdown is converted
before sending, so there is nothing to compare it against and the check is
skipped. Use --no-verify to skip the read.`,
		Example: `  # Edit a page in the editor with current content
  cfl page edit 12345 --editor

  # Update page content from file
  cfl page edit 12345 --file content.md

  # Update page in legacy format
  cfl page edit 12345 --file content.md --legacy

  # Update page content from stdin
  echo "# Updated Content" | cfl page edit 12345

  # Update from exact ADF JSON
  cfl page edit 12345 --file content.json --body-format adf

  # Update from exact storage XHTML
  echo "<p>Updated</p>" | cfl page edit 12345 --body-format xhtml

  # Update page title only
  cfl page edit 12345 --title "New Title"

  # Move page to a new parent
  cfl page edit 12345 --parent 67890

  # Move page and update title
  cfl page edit 12345 --parent 67890 --title "New Title"

  # Extract, transform, and re-upload storage-format content
  cfl page view 12345 --body-format xhtml --content-only | \
    sed 's/old/new/g' | cfl page edit 12345 --body-format xhtml`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return err
			}
			format, err := resolveBodyFormat(opts.bodyFormat, cmd.Flags().Changed("body-format"))
			if err != nil {
				return err
			}
			if opts.legacy && format != bodyFormatMarkdown {
				return fmt.Errorf("--legacy is only supported with --body-format markdown")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.pageID = args[0]
			opts.bodyFormatExplicit = cmd.Flags().Changed("body-format")
			return runEdit(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.title, "title", "t", "", "New page title")
	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "Read content from file")
	cmd.Flags().StringVarP(&opts.parent, "parent", "p", "", "Move page to new parent page ID")
	cmd.Flags().BoolVar(&opts.editor, "editor", false, "Open editor for content")
	cmd.Flags().StringVar(&opts.bodyFormat, "body-format", bodyFormatMarkdown, "Input format: markdown, adf, or xhtml")
	cmd.Flags().BoolVar(&opts.legacy, "legacy", false, "Edit page in legacy editor format (Markdown input only)")
	cmd.Flags().BoolVar(&opts.noVerify, "no-verify", false, "Skip reading the page back to confirm what Confluence stored (adf/xhtml input)")
	cmd.Flags().BoolVar(&opts.allowLossy, "allow-lossy", false, "Write in a body format that cannot carry content the page currently has")

	return cmd
}

func runEdit(ctx context.Context, opts *editOptions) error {
	bodyFormat, err := resolveBodyFormat(opts.bodyFormat, opts.bodyFormatExplicit)
	if err != nil {
		return err
	}
	if opts.legacy && bodyFormat != bodyFormatMarkdown {
		return fmt.Errorf("--legacy is only supported with --body-format markdown")
	}

	// Validate file exists before making any network calls so we fail
	// fast on bad input without needing config or API access. "-" means
	// stdin, which has no path to stat.
	if opts.file != "" && opts.file != "-" {
		if _, err := os.Stat(opts.file); err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
	}

	if opts.title == "" && opts.parent == "" &&
		!hasContentSource(opts.Options, opts.file, opts.editor) {
		return errMissingContentSource()
	}
	hasStdinData := (opts.Stdin != nil && opts.Stdin != os.Stdin) || hasPipedOSStdin(opts.Options)
	hasNewContent := opts.file != "" || opts.editor || hasStdinData
	var newBody *api.Body
	var sentContent string
	if hasNewContent && !opts.editor {
		content, err := getEditContent(opts, nil, bodyFormat)
		if err != nil {
			return err
		}
		if strings.TrimSpace(content) == "" {
			return fmt.Errorf("page content cannot be empty")
		}
		sentContent = content
		newBody, err = bodyForInput(content, bodyFormat, opts.legacy)
		if err != nil {
			return err
		}
	}

	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	var existingPage *api.Page
	if opts.editor {
		existingPage, err = getPageWithBodyFormat(ctx, client, opts.pageID, bodyFormat)
	} else {
		existingPage, err = getPageWithBodyFallback(ctx, client, opts.pageID)
	}
	if err != nil {
		return err
	}

	newTitle := opts.title
	if newTitle == "" {
		newTitle = existingPage.Title
	}

	if opts.editor {
		content, err := getEditContent(opts, existingPage, bodyFormat)
		if err != nil {
			return err
		}

		if strings.TrimSpace(content) == "" {
			return fmt.Errorf("page content cannot be empty")
		}

		sentContent = content
		newBody, err = bodyForInput(content, bodyFormat, opts.legacy)
		if err != nil {
			return err
		}
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
		req.Body = newBody
	} else {
		req.Body = existingPage.Body
	}

	// Captured before the write: the storage body is what reveals loss the
	// caller inherited from a lossy read of their own.
	// The page was already fetched above, and the non-editor path asks for
	// the storage representation, so the baseline is usually in hand.
	// Refuse before writing, not after: once the ADF is written the content
	// is gone, and the caller's own copy never had it to restore from.
	if hasNewContent && bodyFormat == bodyFormatADF && !opts.allowLossy {
		current := bodyValue(existingPage, bodyFormatXHTML)
		if current == "" {
			if body, err := readStorageBody(ctx, client, opts.pageID); err == nil {
				current = body
			}
		}
		if findings := findLossyConstructs(current); len(findings) > 0 {
			return lossyFormatError(findings, bodyFormat)
		}
	}

	var storageBefore, storageBeforeErr string
	if verificationApplies(bodyFormat, hasNewContent, opts.noVerify) {
		storageBefore = bodyValue(existingPage, bodyFormatXHTML)
		if storageBefore == "" {
			body, err := readStorageBody(ctx, client, opts.pageID)
			switch {
			case err != nil:
				storageBeforeErr = err.Error()
			case body == "":
				storageBeforeErr = "the page returned no storage body"
			default:
				storageBefore = body
			}
		}
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

	if err := cflpresent.Emit(opts.Options, cflpresent.PagePresenter{}.PresentEdit(page, cfg.URL, opts.legacy && hasNewContent)); err != nil {
		return err
	}

	return verifyStoredBody(ctx, verifyRequest{
		opts:              opts.Options,
		client:            client,
		pageID:            opts.pageID,
		bodyFormat:        bodyFormat,
		sentContent:       sentContent,
		storageBefore:     storageBefore,
		storageBeforeErr:  storageBeforeErr,
		comparePriorState: true,
		enabled:           verificationApplies(bodyFormat, hasNewContent, opts.noVerify),
	})
}

func getEditContent(opts *editOptions, existingPage *api.Page, bodyFormat string) (string, error) {
	if opts.file == "-" {
		data, err := io.ReadAll(stdinReader(opts.Options))
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return string(data), nil
	}

	if opts.file != "" {
		data, err := os.ReadFile(opts.file)
		if err != nil {
			return "", fmt.Errorf("reading file: %w", err)
		}
		return string(data), nil
	}

	if opts.Stdin != nil && opts.Stdin != os.Stdin {
		data, err := io.ReadAll(opts.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return string(data), nil
	}

	if hasPipedOSStdin(opts.Options) {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return string(data), nil
	}

	if !opts.editor {
		return "", errMissingContentSource()
	}

	return openEditorForEdit(existingPage, bodyFormat)
}

func openEditorForEdit(existingPage *api.Page, bodyFormat string) (string, error) {
	ext, editContent, err := editorContent(existingPage, bodyFormat)
	if err != nil {
		return "", err
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

	content := string(data)
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("no content provided")
	}
	if bodyFormat == bodyFormatMarkdown {
		return strings.TrimSpace(content), nil
	}
	return content, nil
}

func editorContent(page *api.Page, bodyFormat string) (string, string, error) {
	switch bodyFormat {
	case bodyFormatMarkdown:
		if page.Body != nil && page.Body.Storage != nil {
			content, err := md.FromConfluenceStorage(page.Body.Storage.Value)
			if err != nil {
				return "", "", fmt.Errorf("cannot produce markdown editor content: %w", err)
			}
			return ".md", content, nil
		}
		if page.Body != nil && page.Body.AtlasDocFormat != nil {
			content, err := md.FromADF(page.Body.AtlasDocFormat.Value)
			if err != nil {
				return "", "", fmt.Errorf("cannot produce markdown editor content: %w", err)
			}
			return ".md", content, nil
		}
	case bodyFormatADF:
		if page.Body != nil && page.Body.AtlasDocFormat != nil {
			return ".json", page.Body.AtlasDocFormat.Value, nil
		}
	case bodyFormatXHTML:
		if page.Body != nil && page.Body.Storage != nil {
			return ".xhtml", page.Body.Storage.Value, nil
		}
	}
	return "", "", fmt.Errorf("cannot produce %s editor content for this page", bodyFormat)
}
