package page

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	cflartifact "github.com/open-cli-collective/confluence-cli/internal/artifact"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	"github.com/open-cli-collective/confluence-cli/pkg/md"
)

// maxViewChars is the default character limit for page body output.
// Content beyond this limit is truncated with an indicator.
// Use --full to show complete content without truncation.
const maxViewChars = 5000

type viewOptions struct {
	*root.Options
	raw         bool
	web         bool
	noTruncate  bool
	showMacros  bool
	contentOnly bool
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
The --content-only flag implies --no-truncate since it is intended for piping.
JSON output (--output json) always includes the full body.`,
		Example: `  # View a page (markdown, truncated if large)
  cfl page view 12345

  # View full content without truncation
  cfl page view 12345 --no-truncate

  # View raw storage format (XHTML)
  cfl page view 12345 --raw

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

	return cmd
}

func runView(ctx context.Context, pageID string, opts *viewOptions) error {
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

	// --web only needs page links, not body content
	if opts.web {
		page, err := client.GetPage(ctx, pageID, nil)
		if err != nil {
			return fmt.Errorf("getting page: %w", err)
		}
		url := cfg.URL + page.Links.WebUI
		return openBrowser(url)
	}

	page, err := getPageWithBodyFallback(ctx, client, pageID)
	if err != nil {
		return fmt.Errorf("getting page: %w", err)
	}

	v := opts.View()

	// Look up space key for display
	spaceKey := ""
	if page.SpaceID != "" {
		space, err := client.GetSpace(ctx, page.SpaceID)
		if err == nil && space != nil {
			spaceKey = space.Key
		}
		// Graceful fallback: if GetSpace fails, we just won't show the key
	}

	if opts.Output == "json" {
		// Extract content (XHTML storage format by default, until #196 validates markdown)
		content := ""
		if hasStorageContent(page) {
			content = page.Body.Storage.Value
		} else if hasADFContent(page) {
			content = page.Body.AtlasDocFormat.Value
		}

		art := cflartifact.ProjectPage(page, spaceKey, content, opts.ArtifactMode())
		return v.RenderArtifact(art)
	}

	if !opts.contentOnly {
		v.RenderKeyValue("Title", page.Title)
		v.RenderKeyValue("ID", page.ID)
		if spaceKey != "" {
			v.RenderKeyValue("Space", fmt.Sprintf("%s (ID: %s)", spaceKey, page.SpaceID))
		} else if page.SpaceID != "" {
			v.RenderKeyValue("Space ID", page.SpaceID)
		}
		if page.Version != nil {
			v.RenderKeyValue("Version", fmt.Sprintf("%d", page.Version.Number))
		}
		_, _ = fmt.Fprintln(v.Out)
	}

	if hasStorageContent(page) {
		content := page.Body.Storage.Value
		if opts.raw {
			_, _ = fmt.Fprintln(v.Out, truncateContent(content, opts))
		} else {
			convertOpts := md.ConvertOptions{
				ShowMacros: opts.showMacros,
			}
			markdown, err := md.FromConfluenceStorageWithOptions(content, convertOpts)
			if err != nil {
				_, _ = fmt.Fprintln(v.Out, "(Failed to convert to markdown, showing raw HTML)")
				_, _ = fmt.Fprintln(v.Out)
				_, _ = fmt.Fprintln(v.Out, truncateContent(content, opts))
			} else {
				_, _ = fmt.Fprintln(v.Out, truncateContent(markdown, opts))
			}
		}
	} else if hasADFContent(page) {
		content := page.Body.AtlasDocFormat.Value
		if opts.raw {
			_, _ = fmt.Fprintln(v.Out, truncateContent(content, opts))
		} else {
			markdown, err := md.FromADF(content)
			if err != nil {
				_, _ = fmt.Fprintln(v.Out, "(Failed to convert ADF to markdown, showing raw ADF)")
				_, _ = fmt.Fprintln(v.Out)
				_, _ = fmt.Fprintln(v.Out, truncateContent(content, opts))
			} else {
				_, _ = fmt.Fprintln(v.Out, truncateContent(markdown, opts))
			}
		}
	} else {
		_, _ = fmt.Fprintln(v.Out, "(No content)")
	}

	return nil
}

// truncateContent truncates content if it exceeds the character limit.
// Uses rune count to avoid splitting multi-byte UTF-8 characters.
// --content-only implies --no-truncate since it is intended for piping.
func truncateContent(content string, opts *viewOptions) string {
	if opts.noTruncate || opts.contentOnly {
		return content
	}
	runes := []rune(content)
	if len(runes) > maxViewChars {
		return string(runes[:maxViewChars]) + fmt.Sprintf("\n\n... [truncated at %d chars, use --no-truncate for complete text]", maxViewChars)
	}
	return content
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
