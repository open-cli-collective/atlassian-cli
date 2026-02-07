package page

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
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
- --file flag to read from a file
- Standard input (pipe content)
- Interactive editor (default, or with --editor flag)

Content format:
- Markdown is the default for stdin, editor, and .md files
- Use --no-markdown to provide raw Confluence format (XHTML for legacy, ADF JSON for cloud)
- Files with .html/.xhtml extensions are treated as storage format`,
		Example: `  # Edit a page (opens editor with current content)
  cfl page edit 12345

  # Update page content from file
  cfl page edit 12345 --file content.md

  # Update page in legacy format
  cfl page edit 12345 --file content.md --legacy

  # Update page content from stdin
  echo "# Updated Content" | cfl page edit 12345

  # Update page title only
  cfl page edit 12345 --title "New Title"

  # Move page to a new parent
  cfl page edit 12345 --parent 67890

  # Move page and update title
  cfl page edit 12345 --parent 67890 --title "New Title"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.pageID = args[0]
			if cmd.Flags().Changed("no-markdown") {
				noMd, _ := cmd.Flags().GetBool("no-markdown")
				useMd := !noMd
				opts.markdown = &useMd
			}
			opts.legacy, _ = cmd.Flags().GetBool("legacy")
			return runEdit(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.title, "title", "t", "", "New page title")
	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "Read content from file")
	cmd.Flags().StringVarP(&opts.parent, "parent", "p", "", "Move page to new parent page ID")
	cmd.Flags().BoolVar(&opts.editor, "editor", false, "Open editor for content")
	cmd.Flags().Bool("no-markdown", false, "Disable markdown conversion (use raw XHTML)")
	cmd.Flags().Bool("legacy", false, "Edit page in legacy editor format (default: cloud editor)")

	return cmd
}

func runEdit(opts *editOptions) error {
	// Validate file exists before making any network calls so we fail
	// fast on bad input without needing config or API access.
	if opts.file != "" {
		if _, err := os.Stat(opts.file); err != nil {
			return fmt.Errorf("failed to read file: %w", err)
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

	existingPage, err := client.GetPage(context.Background(), opts.pageID, &api.GetPageOptions{
		BodyFormat: "storage",
	})
	if err != nil {
		return fmt.Errorf("failed to get page: %w", err)
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

		newContent, err = convertEditContent(content, isMarkdown, opts.legacy)
		if err != nil {
			return err
		}
		hasNewContent = true
	}

	if !hasNewContent && opts.title == "" && opts.parent == "" {
		content, isMarkdown, err := getEditContent(&editOptions{Options: opts.Options, editor: true, markdown: opts.markdown}, existingPage)
		if err != nil {
			return err
		}

		if strings.TrimSpace(content) == "" {
			return fmt.Errorf("page content cannot be empty")
		}

		newContent, err = convertEditContent(content, isMarkdown, opts.legacy)
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
		if opts.legacy {
			v := opts.View()
			v.Warning("Using --legacy flag. If this page uses the cloud editor, it may switch to the legacy editor.")

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

	page, err := client.UpdatePage(context.Background(), opts.pageID, req)
	if err != nil {
		return fmt.Errorf("failed to update page: %w", err)
	}

	if opts.parent != "" {
		if err := client.MovePage(context.Background(), opts.pageID, opts.parent); err != nil {
			return fmt.Errorf("failed to move page to new parent: %w", err)
		}
	}

	v := opts.View()

	if opts.Output == "json" {
		return v.JSON(page)
	}

	v.Success("Updated page: %s", page.Title)
	v.RenderKeyValue("ID", page.ID)
	v.RenderKeyValue("Version", strconv.Itoa(page.Version.Number))
	v.RenderKeyValue("URL", cfg.URL+page.Links.WebUI)

	return nil
}

// convertEditContent converts content based on markdown flag and legacy mode.
func convertEditContent(content string, isMarkdown, legacy bool) (string, error) {
	if legacy {
		if isMarkdown {
			converted, err := md.ToConfluenceStorage([]byte(content))
			if err != nil {
				return "", fmt.Errorf("failed to convert markdown: %w", err)
			}
			return converted, nil
		}
		return content, nil
	}

	if isMarkdown {
		adfContent, err := md.ToADF([]byte(content))
		if err != nil {
			return "", fmt.Errorf("failed to convert markdown to ADF: %w", err)
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

	if opts.file != "" {
		data, err := os.ReadFile(opts.file)
		if err != nil {
			return "", false, fmt.Errorf("failed to read file: %w", err)
		}
		return string(data), useMarkdown(opts.file), nil
	}

	if opts.Stdin != nil && opts.Stdin != os.Stdin {
		data, err := io.ReadAll(opts.Stdin)
		if err != nil {
			return "", false, fmt.Errorf("failed to read stdin: %w", err)
		}
		return string(data), useMarkdown(""), nil
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", false, fmt.Errorf("failed to read stdin: %w", err)
		}
		return string(data), useMarkdown(""), nil
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
	}

	editContent := existingContent
	if isMarkdown && existingContent != "" {
		editContent = "<!-- Edit your content below. This is Confluence storage format. -->\n<!-- Use --no-markdown flag to edit raw storage format -->\n\n" + existingContent
	}

	tmpfile, err := os.CreateTemp("", "cfl-edit-*"+ext)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
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

	cmd := exec.Command(editor, tmpfile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor failed: %w", err)
	}

	data, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read edited content: %w", err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return "", fmt.Errorf("no content provided")
	}

	return content, nil
}
