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

	"github.com/rianjs/confluence-cli/api"
	"github.com/rianjs/confluence-cli/internal/config"
	"github.com/rianjs/confluence-cli/internal/view"
	"github.com/rianjs/confluence-cli/pkg/md"
)

type editOptions struct {
	pageID   string
	title    string
	file     string
	editor   bool
	markdown *bool // nil = auto-detect, true = force markdown, false = force storage format
	output   string
	noColor  bool
}

// NewCmdEdit creates the page edit command.
func NewCmdEdit() *cobra.Command {
	opts := &editOptions{}

	cmd := &cobra.Command{
		Use:   "edit <page-id>",
		Short: "Edit an existing page",
		Long: `Edit an existing Confluence page.

Content can be provided via:
- --file flag to read from a file
- Standard input (pipe content)
- Interactive editor (default, or with --editor flag)

Content format:
- Markdown is the default for stdin, editor, and .md files
- Use --no-markdown to provide Confluence storage format (XHTML)
- Files with .html/.xhtml extensions are treated as storage format`,
		Example: `  # Edit a page (opens editor with current content)
  cfl page edit 12345

  # Update page content from file
  cfl page edit 12345 --file content.md

  # Update page content from stdin
  echo "# Updated Content" | cfl page edit 12345

  # Update page title only
  cfl page edit 12345 --title "New Title"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.pageID = args[0]
			opts.output, _ = cmd.Flags().GetString("output")
			opts.noColor, _ = cmd.Flags().GetBool("no-color")

			// Handle markdown flag
			if cmd.Flags().Changed("no-markdown") {
				noMd, _ := cmd.Flags().GetBool("no-markdown")
				useMd := !noMd
				opts.markdown = &useMd
			}

			return runEdit(opts, nil)
		},
	}

	cmd.Flags().StringVarP(&opts.title, "title", "t", "", "New page title")
	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "Read content from file")
	cmd.Flags().BoolVar(&opts.editor, "editor", false, "Open editor for content")
	cmd.Flags().Bool("no-markdown", false, "Disable markdown conversion (use raw XHTML)")

	return cmd
}

func runEdit(opts *editOptions, client *api.Client) error {
	// Track base URL for output (only available when loading config)
	var baseURL string

	// Create API client if not provided (allows injection for testing)
	if client == nil {
		cfg, err := config.LoadWithEnv(config.DefaultConfigPath())
		if err != nil {
			return fmt.Errorf("failed to load config: %w (run 'cfl init' to configure)", err)
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid config: %w (run 'cfl init' to configure)", err)
		}

		baseURL = cfg.URL
		client = api.NewClient(cfg.URL, cfg.Email, cfg.APIToken)
	}

	// Get existing page
	existingPage, err := client.GetPage(context.Background(), opts.pageID, &api.GetPageOptions{
		BodyFormat: "storage",
	})
	if err != nil {
		return fmt.Errorf("failed to get page: %w", err)
	}

	// Determine new title (use existing if not specified)
	newTitle := opts.title
	if newTitle == "" {
		newTitle = existingPage.Title
	}

	// Get new content
	var newContent string
	hasNewContent := false

	// Check if content is provided via file or stdin
	if opts.file != "" || opts.editor || !isTerminal() {
		content, isMarkdown, err := getEditContent(opts, existingPage)
		if err != nil {
			return err
		}

		// Convert markdown to storage format if needed
		if isMarkdown {
			converted, err := md.ToConfluenceStorage([]byte(content))
			if err != nil {
				return fmt.Errorf("failed to convert markdown: %w", err)
			}
			newContent = converted
		} else {
			newContent = content
		}
		hasNewContent = true
	}

	// If no new content and no new title, open editor by default
	if !hasNewContent && opts.title == "" {
		content, isMarkdown, err := getEditContent(&editOptions{editor: true, markdown: opts.markdown}, existingPage)
		if err != nil {
			return err
		}

		if isMarkdown {
			converted, err := md.ToConfluenceStorage([]byte(content))
			if err != nil {
				return fmt.Errorf("failed to convert markdown: %w", err)
			}
			newContent = converted
		} else {
			newContent = content
		}
		hasNewContent = true
	}

	// Build update request
	req := &api.UpdatePageRequest{
		ID:     opts.pageID,
		Status: "current",
		Title:  newTitle,
		Version: &api.Version{
			Number:  existingPage.Version.Number + 1,
			Message: "Updated via cfl",
		},
	}

	// Only update body if we have new content
	if hasNewContent {
		req.Body = &api.Body{
			Storage: &api.BodyRepresentation{
				Representation: "storage",
				Value:          newContent,
			},
		}
	} else {
		// Keep existing body when only updating title
		req.Body = existingPage.Body
	}

	// Update page
	page, err := client.UpdatePage(context.Background(), opts.pageID, req)
	if err != nil {
		return fmt.Errorf("failed to update page: %w", err)
	}

	// Render output
	renderer := view.NewRenderer(view.Format(opts.output), opts.noColor)

	if opts.output == "json" {
		return renderer.RenderJSON(page)
	}

	renderer.Success(fmt.Sprintf("Updated page: %s", page.Title))
	renderer.RenderKeyValue("ID", page.ID)
	renderer.RenderKeyValue("Version", strconv.Itoa(page.Version.Number))
	renderer.RenderKeyValue("URL", baseURL+page.Links.WebUI)

	return nil
}

// isTerminal checks if stdin is a terminal
func isTerminal() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// getEditContent reads content for editing and returns (content, isMarkdown, error).
func getEditContent(opts *editOptions, existingPage *api.Page) (string, bool, error) {
	// Determine if we should use markdown based on explicit flag or file extension
	useMarkdown := func(filename string) bool {
		// If explicitly set via flag, use that
		if opts.markdown != nil {
			return *opts.markdown
		}
		// Auto-detect based on file extension
		if filename != "" {
			ext := strings.ToLower(filepath.Ext(filename))
			switch ext {
			case ".html", ".xhtml", ".htm":
				return false
			case ".md", ".markdown":
				return true
			}
		}
		// Default to markdown for stdin and editor
		return true
	}

	// Read from file
	if opts.file != "" {
		data, err := os.ReadFile(opts.file)
		if err != nil {
			return "", false, fmt.Errorf("failed to read file: %w", err)
		}
		return string(data), useMarkdown(opts.file), nil
	}

	// Check if stdin has data
	if !isTerminal() {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", false, fmt.Errorf("failed to read stdin: %w", err)
		}
		return string(data), useMarkdown(""), nil
	}

	// Open editor with existing content
	isMarkdown := useMarkdown("")
	content, err := openEditorForEdit(existingPage, isMarkdown)
	return content, isMarkdown, err
}

func openEditorForEdit(existingPage *api.Page, isMarkdown bool) (string, error) {
	// Determine file extension
	ext := ".html"
	if isMarkdown {
		ext = ".md"
	}

	// Get existing content
	existingContent := ""
	if existingPage.Body != nil && existingPage.Body.Storage != nil {
		existingContent = existingPage.Body.Storage.Value
	}

	// For markdown mode, convert storage format to markdown for editing
	// Note: This is a best-effort conversion - complex formatting may not survive round-trip
	editContent := existingContent
	if isMarkdown && existingContent != "" {
		// For now, just use the storage format - a proper implementation would convert to markdown
		// Users can use --no-markdown flag if they want to edit raw storage format
		editContent = "<!-- Edit your content below. This is Confluence storage format. -->\n<!-- Use --no-markdown flag to edit raw storage format -->\n\n" + existingContent
	}

	// Create temp file
	tmpfile, err := os.CreateTemp("", "cfl-edit-*"+ext)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	// Write existing content
	if _, err := tmpfile.WriteString(editContent); err != nil {
		return "", err
	}
	_ = tmpfile.Close()

	// Get editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	// Open editor
	cmd := exec.Command(editor, tmpfile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor failed: %w", err)
	}

	// Read content
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
