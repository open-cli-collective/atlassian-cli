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
	bodyFormat         string
	bodyFormatExplicit bool
	web                bool
	noTruncate         bool
	showMacros         bool
	contentOnly        bool
	version            int
}

func newViewCmd(rootOpts *root.Options) *cobra.Command {
	opts := &viewOptions{Options: rootOpts}

	cmd := &cobra.Command{
		Use:   "view <page-id>",
		Short: "View a page",
		Long: `View a Confluence page content.

The page body is displayed as Markdown by default. Use --body-format adf
for exact Atlassian Document Format JSON or --body-format xhtml for exact
Confluence storage XHTML.

By default, output is truncated to 5000 characters for concise display.
Use --no-truncate to show the complete page content without truncation.
The --content-only flag implies --no-truncate since it is intended for piping.`,
		Example: `  # View a page (markdown, truncated if large)
  cfl page view 12345

  # View full content without truncation
  cfl page view 12345 --no-truncate

  # View exact storage format (XHTML)
  cfl page view 12345 --body-format xhtml

  # View exact ADF JSON
  cfl page view 12345 --body-format adf

  # View a specific historical version
  cfl page view 12345 --version 7

  # Open in browser
  cfl page view 12345 --web

  # Pipe XHTML content to edit (lossless roundtrip)
  cfl page view 12345 --body-format xhtml --content-only | cfl page edit 12345 --body-format xhtml

  # Pipe markdown content to edit
  cfl page view 12345 --content-only | cfl page edit 12345 --legacy`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return err
			}
			format, err := resolveBodyFormat(opts.bodyFormat, cmd.Flags().Changed("body-format"))
			if err != nil {
				return err
			}
			if opts.showMacros && format != bodyFormatMarkdown {
				return fmt.Errorf("--show-macros is only supported with --body-format markdown")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.bodyFormatExplicit = cmd.Flags().Changed("body-format")
			return runView(cmd.Context(), args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.bodyFormat, "body-format", bodyFormatMarkdown, "Body format: markdown, adf, or xhtml")
	cmd.Flags().BoolVarP(&opts.web, "web", "w", false, "Open in browser instead of displaying")
	cmd.Flags().BoolVar(&opts.noTruncate, "no-truncate", false, "Show full content without truncation")
	cmd.Flags().BoolVar(&opts.showMacros, "show-macros", false, "Show Confluence macro placeholders (e.g., [TOC]) instead of stripping them")
	cmd.Flags().BoolVar(&opts.contentOnly, "content-only", false, "Output only page content (no metadata headers); implies --no-truncate")
	cmd.Flags().IntVar(&opts.version, "version", 0, "View a specific page version")

	return cmd
}

func runView(ctx context.Context, pageID string, opts *viewOptions) error {
	bodyFormat, err := resolveBodyFormat(opts.bodyFormat, opts.bodyFormatExplicit)
	if err != nil {
		return err
	}
	if opts.showMacros && bodyFormat != bodyFormatMarkdown {
		return fmt.Errorf("--show-macros is only supported with --body-format markdown")
	}
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
		page, err = getPageVersionWithBodyFormat(ctx, client, pageID, opts.version, bodyFormat)
	} else {
		page, err = getPageWithBodyFormat(ctx, client, pageID, bodyFormat)
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

	proj, err := pageview.Project(page, spaceKey, pageview.Options{
		BodyFormat:  bodyFormat,
		NoTruncate:  opts.noTruncate,
		ShowMacros:  opts.showMacros,
		ContentOnly: opts.contentOnly,
	})
	if err != nil {
		return cflpresent.EmitError(opts.Options, err)
	}

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
