package page

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	"github.com/open-cli-collective/confluence-cli/internal/pageview"
	cflpresent "github.com/open-cli-collective/confluence-cli/internal/present"
)

// maxViewChars is the default character limit for page body output.
// Content beyond this limit is truncated with an indicator.
// Use --no-truncate to show complete content without truncation.
const maxViewChars = pageview.MaxChars

type viewOptions struct {
	*root.Options
	raw         bool
	web         bool
	noTruncate  bool
	showMacros  bool
	contentOnly bool
	version     int
}

func newViewCmd(rootOpts *root.Options) *cobra.Command {
	opts := &viewOptions{Options: rootOpts}

	cmd := &cobra.Command{
		Use:   "view <page-id>",
		Short: "View a page",
		Long: `View a Confluence page content.

The page body is fetched in storage format (XHTML) and converted to
markdown for display. Use --raw to see the original storage format.

By default, output is truncated to 5000 characters for concise display.
Use --no-truncate to show the complete page content without truncation.
The --content-only flag implies --no-truncate since it is intended for piping.`,
		Example: `  # View a page (markdown, truncated if large)
  cfl page view 12345

  # View full content without truncation
  cfl page view 12345 --no-truncate

  # View raw storage format (XHTML)
  cfl page view 12345 --raw

  # View a specific historical version
  cfl page view 12345 --version 7

  # Open in browser
  cfl page view 12345 --web

  # Pipe raw content to edit (lossless roundtrip)
  cfl page view 12345 --raw --content-only | cfl page edit 12345 --no-markdown --legacy

  # Pipe markdown content to edit
  cfl page view 12345 --content-only | cfl page edit 12345 --legacy`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runView(cmd.Context(), args[0], opts)
		},
	}

	cmd.Flags().BoolVar(&opts.raw, "raw", false, "Show raw Confluence storage format (XHTML) instead of markdown")
	cmd.Flags().BoolVarP(&opts.web, "web", "w", false, "Open in browser instead of displaying")
	cmd.Flags().BoolVar(&opts.noTruncate, "no-truncate", false, "Show full content without truncation")
	cmd.Flags().BoolVar(&opts.showMacros, "show-macros", false, "Show Confluence macro placeholders (e.g., [TOC]) instead of stripping them")
	cmd.Flags().BoolVar(&opts.contentOnly, "content-only", false, "Output only page content (no metadata headers); implies --no-truncate")
	cmd.Flags().IntVar(&opts.version, "version", 0, "View a specific page version")

	return cmd
}

func runView(ctx context.Context, pageID string, opts *viewOptions) error {
	if opts.contentOnly {
		if opts.web {
			return fmt.Errorf("--content-only is incompatible with --web")
		}
	}
	if opts.version < 0 {
		return fmt.Errorf("invalid version: %d (must be >= 0)", opts.version)
	}
	if opts.version > 0 && opts.web {
		return fmt.Errorf("--version is incompatible with --web")
	}

	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// --web only needs page links, not body content
	if opts.web {
		page, err := client.GetPage(ctx, pageID, nil)
		if err != nil {
			return err
		}
		url := cfg.URL + page.Links.WebUI
		return openBrowser(url)
	}

	var page *api.Page
	if opts.version > 0 {
		page, err = getPageVersionWithBodyFallback(ctx, client, pageID, opts.version)
	} else {
		page, err = getPageWithBodyFallback(ctx, client, pageID)
	}
	if err != nil {
		return err
	}

	// Look up space key for display
	spaceKey := ""
	if page.SpaceID != "" {
		space, err := client.GetSpace(ctx, page.SpaceID)
		if err == nil && space != nil {
			spaceKey = space.Key
		}
		// Graceful fallback: if GetSpace fails, we just won't show the key
	}

	proj := pageview.Project(page, spaceKey, pageview.Options{
		Raw:         opts.raw,
		NoTruncate:  opts.noTruncate,
		ShowMacros:  opts.showMacros,
		ContentOnly: opts.contentOnly,
	})

	return cflpresent.Emit(opts.Options, cflpresent.PagePresenter{}.PresentView(proj))
}
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url) //nolint:gosec // url is constructed internally from Confluence API links
	case "linux":
		cmd = exec.Command("xdg-open", url) //nolint:gosec // url is constructed internally from Confluence API links
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url) //nolint:gosec // url is constructed internally from Confluence API links
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}
