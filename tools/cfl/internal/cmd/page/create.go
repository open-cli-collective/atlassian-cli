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
)

type createOptions struct {
	*root.Options
	space              string
	title              string
	parent             string
	file               string
	editor             bool
	bodyFormat         string
	bodyFormatExplicit bool
	legacy             bool
}

func newCreateCmd(rootOpts *root.Options) *cobra.Command {
	opts := &createOptions{Options: rootOpts}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new page",
		Long: `Create a new Confluence page.

Markdown input is converted to cloud editor format (ADF) by default.
Use --legacy with Markdown to create pages in storage XHTML instead.

Content can be provided via:
- --file flag to read from a file (use --file - to read from stdin)
- Standard input (pipe content)
- Interactive editor with --editor

Content format is selected with --body-format markdown|adf|xhtml.
Omitting --body-format means Markdown. ADF and XHTML are validated or sent
without conversion.`,
		Example: `  # Create a page with title in the editor (cloud editor format)
  cfl page create --space DEV --title "My Page" --editor

  # Create from markdown file
  cfl page create -s DEV -t "My Page" --file content.md

  # Create in legacy editor format
  cfl page create -s DEV -t "My Page" --file content.md --legacy

  # Create from exact ADF JSON
  cfl page create -s DEV -t "My Page" --file content.json --body-format adf

  # Create from exact storage XHTML
  cfl page create -s DEV -t "My Page" --file content.xhtml --body-format xhtml

  # Create from stdin (markdown)
  echo "# Hello World" | cfl page create -s DEV -t "My Page"

  # Create from stdin with exact storage XHTML
  echo "<p>Hello</p>" | cfl page create -s DEV -t "My Page" --body-format xhtml

  # Create as child of another page
  cfl page create -s DEV -t "Child Page" --parent 12345`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.bodyFormatExplicit = cmd.Flags().Changed("body-format")
			return runCreate(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.space, "space", "s", "", "Space key (required)")
	cmd.Flags().StringVarP(&opts.title, "title", "t", "", "Page title (required)")
	cmd.Flags().StringVarP(&opts.parent, "parent", "p", "", "Parent page ID")
	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "Read content from file")
	cmd.Flags().BoolVar(&opts.editor, "editor", false, "Open editor for content")
	cmd.Flags().StringVar(&opts.bodyFormat, "body-format", bodyFormatMarkdown, "Input format: markdown, adf, or xhtml")
	cmd.Flags().BoolVar(&opts.legacy, "legacy", false, "Create page in legacy editor format (Markdown input only)")

	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func runCreate(ctx context.Context, opts *createOptions) error {
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
	if !hasContentSource(opts.Options, opts.file, opts.editor) {
		return errMissingContentSource()
	}
	content, err := getContent(opts, bodyFormat)
	if err != nil {
		return err
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("page content cannot be empty")
	}
	body, err := bodyForInput(content, bodyFormat, opts.legacy)
	if err != nil {
		return err
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

	space, err := client.GetSpaceByKey(ctx, spaceKey)
	if err != nil {
		return fmt.Errorf("finding space '%s': %w", spaceKey, err)
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

	page, err := client.CreatePage(ctx, req)
	if err != nil {
		return err
	}

	return cflpresent.Emit(opts.Options, cflpresent.PagePresenter{}.PresentCreate(page, cfg.URL))
}

func getContent(opts *createOptions, bodyFormat string) (string, error) {
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

	return openEditor(bodyFormat)
}

func openEditor(bodyFormat string) (string, error) {
	ext := ".md"
	template := `# Page Title

Enter your content here using markdown.

## Section

- List item 1
- List item 2
`
	switch bodyFormat {
	case bodyFormatADF:
		ext = ".json"
		template = `{"type":"doc","version":1,"content":[]}
`
	case bodyFormatXHTML:
		ext = ".xhtml"
		template = `<p>Enter your page content here.</p>
`
	}

	tmpfile, err := os.CreateTemp("", "cfl-*"+ext)
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
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
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || trimmed == strings.TrimSpace(template) {
		return "", fmt.Errorf("no content provided (or content unchanged)")
	}
	if bodyFormat == bodyFormatMarkdown {
		return trimmed, nil
	}
	return content, nil
}
