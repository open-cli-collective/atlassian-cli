package page

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

type copyOptions struct {
	*root.Options
	title         string
	space         string
	noAttachments bool
	noLabels      bool
}

func newCopyCmd(rootOpts *root.Options) *cobra.Command {
	opts := &copyOptions{Options: rootOpts}

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
		RunE: func(_ *cobra.Command, args []string) error {
			return runCopy(args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.title, "title", "t", "", "Title for the copied page (required)")
	cmd.Flags().StringVarP(&opts.space, "space", "s", "", "Destination space key (default: same space)")
	cmd.Flags().BoolVar(&opts.noAttachments, "no-attachments", false, "Don't copy attachments")
	cmd.Flags().BoolVar(&opts.noLabels, "no-labels", false, "Don't copy labels")

	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func runCopy(pageID string, opts *copyOptions) error {
	if err := view.ValidateFormat(opts.Output); err != nil {
		return err
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	destSpace := opts.space
	if destSpace == "" {
		sourcePage, err := client.GetPage(context.Background(), pageID, nil)
		if err != nil {
			return fmt.Errorf("failed to get source page: %w", err)
		}
		space, err := client.GetSpace(context.Background(), sourcePage.SpaceID)
		if err != nil {
			return fmt.Errorf("failed to get space: %w", err)
		}
		destSpace = space.Key
	}

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

	v := opts.View()

	if opts.Output == "json" {
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
