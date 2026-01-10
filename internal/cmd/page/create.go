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

	"github.com/rianjs/confluence-cli/api"
	"github.com/rianjs/confluence-cli/internal/config"
	"github.com/rianjs/confluence-cli/internal/view"
	"github.com/rianjs/confluence-cli/pkg/md"
)

type createOptions struct {
	space    string
	title    string
	parent   string
	file     string
	editor   bool
	markdown *bool // nil = auto-detect, true = force markdown, false = force storage format
	output   string
	noColor  bool
}

// NewCmdCreate creates the page create command.
func NewCmdCreate() *cobra.Command {
	opts := &createOptions{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new page",
		Long: `Create a new Confluence page.

Content can be provided via:
- --file flag to read from a file
- Standard input (pipe content)
- Interactive editor (default, or with --editor flag)

Content format:
- Markdown is the default for stdin, editor, and .md files
- Use --no-markdown to provide Confluence storage format (XHTML)
- Files with .html/.xhtml extensions are treated as storage format`,
		Example: `  # Create a page with title (opens markdown editor)
  cfl page create --space DEV --title "My Page"

  # Create from markdown file
  cfl page create -s DEV -t "My Page" --file content.md

  # Create from XHTML file
  cfl page create -s DEV -t "My Page" --file content.html

  # Create from stdin (markdown)
  echo "# Hello World" | cfl page create -s DEV -t "My Page"

  # Create from stdin (XHTML)
  echo "<p>Hello</p>" | cfl page create -s DEV -t "My Page" --no-markdown

  # Create as child of another page
  cfl page create -s DEV -t "Child Page" --parent 12345`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.output, _ = cmd.Flags().GetString("output")
			opts.noColor, _ = cmd.Flags().GetBool("no-color")

			// Handle markdown flag
			if cmd.Flags().Changed("no-markdown") {
				noMd, _ := cmd.Flags().GetBool("no-markdown")
				useMd := !noMd
				opts.markdown = &useMd
			}

			return runCreate(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.space, "space", "s", "", "Space key (required)")
	cmd.Flags().StringVarP(&opts.title, "title", "t", "", "Page title (required)")
	cmd.Flags().StringVarP(&opts.parent, "parent", "p", "", "Parent page ID")
	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "Read content from file")
	cmd.Flags().BoolVar(&opts.editor, "editor", false, "Open editor for content")
	cmd.Flags().Bool("no-markdown", false, "Disable markdown conversion (use raw XHTML)")

	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func runCreate(opts *createOptions) error {
	// Load config
	cfg, err := config.LoadWithEnv(config.DefaultConfigPath())
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'cfl init' to configure)", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w (run 'cfl init' to configure)", err)
	}

	// Determine space
	spaceKey := opts.space
	if spaceKey == "" {
		spaceKey = cfg.DefaultSpace
	}
	if spaceKey == "" {
		return fmt.Errorf("space is required: use --space flag or set default_space in config")
	}

	// Create API client
	client := api.NewClient(cfg.URL, cfg.Email, cfg.APIToken)

	// Get space ID
	space, err := client.GetSpaceByKey(context.Background(), spaceKey)
	if err != nil {
		return fmt.Errorf("failed to find space '%s': %w", spaceKey, err)
	}

	// Get content and determine if markdown conversion is needed
	content, isMarkdown, err := getContent(opts)
	if err != nil {
		return err
	}

	// Convert markdown to storage format if needed
	if isMarkdown {
		converted, err := md.ToConfluenceStorage([]byte(content))
		if err != nil {
			return fmt.Errorf("failed to convert markdown: %w", err)
		}
		content = converted
	}

	// Create page
	req := &api.CreatePageRequest{
		SpaceID: space.ID,
		Title:   opts.title,
		Status:  "current",
		Body: &api.Body{
			Storage: &api.BodyRepresentation{
				Representation: "storage",
				Value:          content,
			},
		},
	}

	if opts.parent != "" {
		req.ParentID = opts.parent
	}

	page, err := client.CreatePage(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}

	// Render output
	renderer := view.NewRenderer(view.Format(opts.output), opts.noColor)

	if opts.output == "json" {
		return renderer.RenderJSON(page)
	}

	renderer.Success(fmt.Sprintf("Created page: %s", page.Title))
	renderer.RenderKeyValue("ID", page.ID)
	renderer.RenderKeyValue("URL", cfg.URL+page.Links.WebUI)

	return nil
}

// getContent reads content and returns (content, isMarkdown, error).
// isMarkdown indicates whether the content should be converted from markdown.
func getContent(opts *createOptions) (string, bool, error) {
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
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", false, fmt.Errorf("failed to read stdin: %w", err)
		}
		return string(data), useMarkdown(""), nil
	}

	// Open editor (markdown mode)
	isMarkdown := useMarkdown("")
	content, err := openEditor(isMarkdown)
	return content, isMarkdown, err
}

func openEditor(isMarkdown bool) (string, error) {
	// Determine file extension and template based on format
	ext := ".html"
	template := `<p>Enter your page content here.</p>
`
	if isMarkdown {
		ext = ".md"
		template = `# Page Title

Enter your content here using markdown.

## Section

- List item 1
- List item 2
`
	}

	// Create temp file
	tmpfile, err := os.CreateTemp("", "cfl-*"+ext)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	// Write initial content
	if _, err := tmpfile.WriteString(template); err != nil {
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
	if content == "" || content == strings.TrimSpace(template) {
		return "", fmt.Errorf("no content provided (or content unchanged)")
	}

	return content, nil
}
