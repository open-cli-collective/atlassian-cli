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
	"github.com/open-cli-collective/confluence-cli/pkg/md"
)

type createOptions struct {
	*root.Options
	space    string
	title    string
	parent   string
	file     string
	editor   bool
	markdown *bool // nil = auto-detect, true = force markdown, false = force storage format
	legacy   bool  // Use legacy editor (storage format) instead of cloud editor (ADF)
}

func newCreateCmd(rootOpts *root.Options) *cobra.Command {
	opts := &createOptions{Options: rootOpts}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new page",
		Long: `Create a new Confluence page.

By default, pages are created using the cloud editor format (ADF).
Use --legacy to create pages in the legacy editor format.

Content can be provided via:
- --file flag to read from a file
- Standard input (pipe content)
- Interactive editor (default, or with --editor flag)

Content format:
- Markdown is the default for stdin, editor, and .md files
- Use --no-markdown to provide raw Confluence format (XHTML for legacy, ADF JSON for cloud)
- Files with .html/.xhtml extensions are treated as storage format`,
		Example: `  # Create a page with title (opens markdown editor, cloud editor format)
  cfl page create --space DEV --title "My Page"

  # Create from markdown file
  cfl page create -s DEV -t "My Page" --file content.md

  # Create in legacy editor format
  cfl page create -s DEV -t "My Page" --file content.md --legacy

  # Create from XHTML file (legacy mode)
  cfl page create -s DEV -t "My Page" --file content.html --legacy

  # Create from stdin (markdown)
  echo "# Hello World" | cfl page create -s DEV -t "My Page"

  # Create from stdin with legacy format (XHTML)
  echo "<p>Hello</p>" | cfl page create -s DEV -t "My Page" --no-markdown --legacy

  # Create as child of another page
  cfl page create -s DEV -t "Child Page" --parent 12345`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Flags().Changed("no-markdown") {
				noMd, _ := cmd.Flags().GetBool("no-markdown")
				useMd := !noMd
				opts.markdown = &useMd
			}
			opts.legacy, _ = cmd.Flags().GetBool("legacy")
			return runCreate(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.space, "space", "s", "", "Space key (required)")
	cmd.Flags().StringVarP(&opts.title, "title", "t", "", "Page title (required)")
	cmd.Flags().StringVarP(&opts.parent, "parent", "p", "", "Parent page ID")
	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "Read content from file")
	cmd.Flags().BoolVar(&opts.editor, "editor", false, "Open editor for content")
	cmd.Flags().Bool("no-markdown", false, "Disable markdown conversion (use raw XHTML)")
	cmd.Flags().Bool("legacy", false, "Create page in legacy editor format (default: cloud editor)")

	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func runCreate(opts *createOptions) error {
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

	spaceKey := opts.space
	if spaceKey == "" {
		spaceKey = cfg.DefaultSpace
	}

	if spaceKey == "" {
		return fmt.Errorf("space is required: use --space flag or set default_space in config")
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	space, err := client.GetSpaceByKey(context.Background(), spaceKey)
	if err != nil {
		return fmt.Errorf("failed to find space '%s': %w", spaceKey, err)
	}

	content, isMarkdown, err := getContent(opts)
	if err != nil {
		return err
	}

	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("page content cannot be empty")
	}

	var body *api.Body

	if opts.legacy {
		if isMarkdown {
			converted, err := md.ToConfluenceStorage([]byte(content))
			if err != nil {
				return fmt.Errorf("failed to convert markdown: %w", err)
			}
			content = converted
		}
		body = &api.Body{
			Storage: &api.BodyRepresentation{
				Representation: "storage",
				Value:          content,
			},
		}
	} else {
		if isMarkdown {
			adfContent, err := md.ToADF([]byte(content))
			if err != nil {
				return fmt.Errorf("failed to convert markdown to ADF: %w", err)
			}
			content = adfContent
		}
		body = &api.Body{
			AtlasDocFormat: &api.BodyRepresentation{
				Representation: "atlas_doc_format",
				Value:          content,
			},
		}
	}

	req := &api.CreatePageRequest{
		SpaceID: space.ID,
		Title:   opts.title,
		Status:  "current",
		Body:    body,
	}

	if opts.parent != "" {
		req.ParentID = opts.parent
	}

	page, err := client.CreatePage(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}

	v := opts.View()

	if opts.Output == "json" {
		return v.JSON(page)
	}

	v.Success("Created page: %s", page.Title)
	v.RenderKeyValue("ID", page.ID)
	v.RenderKeyValue("URL", cfg.URL+page.Links.WebUI)

	return nil
}

// getContent reads content and returns (content, isMarkdown, error).
// isMarkdown indicates whether the content should be converted from markdown.
func getContent(opts *createOptions) (string, bool, error) {
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
	content, err := openEditor(isMarkdown)
	return content, isMarkdown, err
}

func openEditor(isMarkdown bool) (string, error) {
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

	tmpfile, err := os.CreateTemp("", "cfl-*"+ext)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	if _, err := tmpfile.WriteString(template); err != nil {
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
	if content == "" || content == strings.TrimSpace(template) {
		return "", fmt.Errorf("no content provided (or content unchanged)")
	}

	return content, nil
}
