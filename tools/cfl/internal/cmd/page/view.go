package page

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	"github.com/open-cli-collective/confluence-cli/pkg/md"
)

type viewOptions struct {
	*root.Options
	raw         bool
	web         bool
	showMacros  bool
	contentOnly bool
}

func newViewCmd(rootOpts *root.Options) *cobra.Command {
	opts := &viewOptions{Options: rootOpts}

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
		RunE: func(_ *cobra.Command, args []string) error {
			return runView(args[0], opts)
		},
	}

	cmd.Flags().BoolVar(&opts.raw, "raw", false, "Show raw Confluence storage format")
	cmd.Flags().BoolVarP(&opts.web, "web", "w", false, "Open in browser instead of displaying")
	cmd.Flags().BoolVar(&opts.showMacros, "show-macros", false, "Show Confluence macro placeholders (e.g., [TOC]) instead of stripping them")
	cmd.Flags().BoolVar(&opts.contentOnly, "content-only", false, "Output only page content (no metadata headers)")

	return cmd
}

func runView(pageID string, opts *viewOptions) error {
	if err := view.ValidateFormat(opts.Output); err != nil {
		return err
	}

	if opts.contentOnly {
		if opts.Output == "json" {
			return fmt.Errorf("--content-only is incompatible with --output json")
		}
		if opts.web {
			return fmt.Errorf("--content-only is incompatible with --web")
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

	page, err := client.GetPage(context.Background(), pageID, nil)
	if err != nil {
		return fmt.Errorf("failed to get page: %w", err)
	}

	if opts.web {
		url := cfg.URL + page.Links.WebUI
		return openBrowser(url)
	}

	v := opts.View()

	if opts.Output == "json" {
		return v.JSON(page)
	}

	if !opts.contentOnly {
		v.RenderKeyValue("Title", page.Title)
		v.RenderKeyValue("ID", page.ID)
		if page.Version != nil {
			v.RenderKeyValue("Version", fmt.Sprintf("%d", page.Version.Number))
		}
		fmt.Println()
	}

	if page.Body != nil && page.Body.Storage != nil {
		content := page.Body.Storage.Value
		if opts.raw {
			fmt.Println(content)
		} else {
			convertOpts := md.ConvertOptions{
				ShowMacros: opts.showMacros,
			}
			markdown, err := md.FromConfluenceStorageWithOptions(content, convertOpts)
			if err != nil {
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
