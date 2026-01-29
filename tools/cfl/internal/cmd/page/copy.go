package page

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/config"
)

type copyOptions struct {
	title         string
	space         string
	noAttachments bool
	noLabels      bool
	output        string
	noColor       bool
}

// NewCmdCopy creates the page copy command.
func NewCmdCopy() *cobra.Command {
	opts := &copyOptions{}

	cmd := &cobra.Command{
		Use:   "copy <page-id>",
		Short: "Copy a page",
		Long:  `Create a copy of a Confluence page with a new title.`,
		Example: `  # Copy a page with a new title
  cfl page copy 12345 --title "Copy of My Page"

  # Copy to a different space
  cfl page copy 12345 --title "My Page" --space OTHERSPACE

  # Copy without attachments
  cfl page copy 12345 --title "Lightweight Copy" --no-attachments

  # Copy without labels
  cfl page copy 12345 --title "Fresh Copy" --no-labels`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.output, _ = cmd.Flags().GetString("output")
			opts.noColor, _ = cmd.Flags().GetBool("no-color")
			return runCopy(args[0], opts, nil)
		},
	}

	cmd.Flags().StringVarP(&opts.title, "title", "t", "", "Title for the copied page (required)")
	cmd.Flags().StringVarP(&opts.space, "space", "s", "", "Destination space key (default: same space)")
	cmd.Flags().BoolVar(&opts.noAttachments, "no-attachments", false, "Don't copy attachments")
	cmd.Flags().BoolVar(&opts.noLabels, "no-labels", false, "Don't copy labels")

	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func runCopy(pageID string, opts *copyOptions, client *api.Client) error {
	// Validate output format
	if err := view.ValidateFormat(opts.output); err != nil {
		return err
	}

	// Create API client if not provided (allows testing with mock client)
	if client == nil {
		cfg, err := config.LoadWithEnv(config.DefaultConfigPath())
		if err != nil {
			return fmt.Errorf("failed to load config: %w (run 'cfl init' to configure)", err)
		}
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid config: %w (run 'cfl init' to configure)", err)
		}
		client = api.NewClient(cfg.URL, cfg.Email, cfg.APIToken)
	}

	// If no destination space specified, get source page's space key
	destSpace := opts.space
	if destSpace == "" {
		sourcePage, err := client.GetPage(context.Background(), pageID, nil)
		if err != nil {
			return fmt.Errorf("failed to get source page: %w", err)
		}
		// SpaceID is numeric (e.g., "3367829530"), but copy API needs space key
		space, err := client.GetSpace(context.Background(), sourcePage.SpaceID)
		if err != nil {
			return fmt.Errorf("failed to get space: %w", err)
		}
		destSpace = space.Key
	}

	// Copy the page
	// Default all copy flags to true, override with --no-* flags
	copyOpts := &api.CopyPageOptions{
		Title:              opts.title,
		DestinationSpace:   destSpace,
		CopyAttachments:    !opts.noAttachments,
		CopyPermissions:    true,
		CopyProperties:     true,
		CopyLabels:         !opts.noLabels,
		CopyCustomContents: true,
	}

	newPage, err := client.CopyPage(context.Background(), pageID, copyOpts)
	if err != nil {
		return fmt.Errorf("failed to copy page: %w", err)
	}

	// Render output
	v := view.New(view.Format(opts.output), opts.noColor)

	if opts.output == "json" {
		return v.JSON(newPage)
	}

	v.Success("Copied page to: %s", newPage.Title)
	v.RenderKeyValue("ID", newPage.ID)
	v.RenderKeyValue("Title", newPage.Title)
	v.RenderKeyValue("Space", newPage.SpaceID)
	if newPage.Version != nil {
		v.RenderKeyValue("Version", fmt.Sprintf("%d", newPage.Version.Number))
	}

	return nil
}
