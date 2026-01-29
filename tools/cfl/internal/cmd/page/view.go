package page

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/config"
	"github.com/open-cli-collective/confluence-cli/pkg/md"
)

type viewOptions struct {
	raw         bool
	web         bool
	showMacros  bool
	contentOnly bool
	output      string
	noColor     bool
}

// NewCmdView creates the page view command.
func NewCmdView() *cobra.Command {
	opts := &viewOptions{}

	cmd := &cobra.Command{
		Use:   "view <page-id>",
		Short: "View a page",
		Long:  `View a Confluence page content.`,
		Example: `  # View a page
  cfl page view 12345

  # View raw storage format
  cfl page view 12345 --raw

  # Open in browser
  cfl page view 12345 --web

  # Output content only (for piping to edit)
  cfl page view 12345 --show-macros --content-only | cfl page edit 12345 --legacy`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.output, _ = cmd.Flags().GetString("output")
			opts.noColor, _ = cmd.Flags().GetBool("no-color")
			return runView(args[0], opts, nil)
		},
	}

	cmd.Flags().BoolVar(&opts.raw, "raw", false, "Show raw Confluence storage format")
	cmd.Flags().BoolVarP(&opts.web, "web", "w", false, "Open in browser instead of displaying")
	cmd.Flags().BoolVar(&opts.showMacros, "show-macros", false, "Show Confluence macro placeholders (e.g., [TOC]) instead of stripping them")
	cmd.Flags().BoolVar(&opts.contentOnly, "content-only", false, "Output only page content (no metadata headers)")

	return cmd
}

func runView(pageID string, opts *viewOptions, client *api.Client) error {
	// Track base URL for --web flag
	var baseURL string

	// Validate output format
	if err := view.ValidateFormat(opts.output); err != nil {
		return err
	}

	// Validate flag combinations
	if opts.contentOnly {
		if opts.output == "json" {
			return fmt.Errorf("--content-only is incompatible with --output json")
		}
		if opts.web {
			return fmt.Errorf("--content-only is incompatible with --web")
		}
	}

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

	// Get page with body
	apiOpts := &api.GetPageOptions{
		BodyFormat: "storage",
	}

	page, err := client.GetPage(context.Background(), pageID, apiOpts)
	if err != nil {
		return fmt.Errorf("failed to get page: %w", err)
	}

	// Open in browser if requested
	if opts.web {
		url := baseURL + page.Links.WebUI
		return openBrowser(url)
	}

	// Render output
	v := view.New(view.Format(opts.output), opts.noColor)

	if opts.output == "json" {
		return v.JSON(page)
	}

	// Show page info (unless content-only mode)
	if !opts.contentOnly {
		v.RenderKeyValue("Title", page.Title)
		v.RenderKeyValue("ID", page.ID)
		if page.Version != nil {
			v.RenderKeyValue("Version", fmt.Sprintf("%d", page.Version.Number))
		}
		fmt.Println()
	}

	// Show content
	if page.Body != nil && page.Body.Storage != nil {
		content := page.Body.Storage.Value
		if opts.raw {
			fmt.Println(content)
		} else {
			// Convert storage format (HTML) to markdown
			convertOpts := md.ConvertOptions{
				ShowMacros: opts.showMacros,
			}
			markdown, err := md.FromConfluenceStorageWithOptions(content, convertOpts)
			if err != nil {
				// Fall back to raw content if conversion fails
				fmt.Println("(Failed to convert to markdown, showing raw HTML)")
				fmt.Println()
				fmt.Println(content)
			} else {
				fmt.Println(markdown)
			}
		}
	} else {
		fmt.Println("(No content)")
	}

	return nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}
