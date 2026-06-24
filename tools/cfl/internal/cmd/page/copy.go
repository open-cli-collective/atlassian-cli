package page

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	cflpresent "github.com/open-cli-collective/confluence-cli/internal/present"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCopy(cmd.Context(), args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.title, "title", "t", "", "Title for the copied page (required)")
	cmd.Flags().StringVarP(&opts.space, "space", "s", "", "Destination space key (default: same space)")
	cmd.Flags().BoolVar(&opts.noAttachments, "no-attachments", false, "Don't copy attachments")
	cmd.Flags().BoolVar(&opts.noLabels, "no-labels", false, "Don't copy labels")

	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func runCopy(ctx context.Context, pageID string, opts *copyOptions) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	destSpace := opts.space
	if destSpace == "" {
		// nil opts: body content is not needed, only SpaceID for determining destination
		sourcePage, err := client.GetPage(ctx, pageID, nil)
		if err != nil {
			return err
		}
		space, err := client.GetSpace(ctx, sourcePage.SpaceID)
		if err != nil {
			return err
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

	newPage, err := client.CopyPage(ctx, pageID, copyOpts)
	if err != nil {
		return err
	}

	return cflpresent.Emit(opts.Options, cflpresent.PagePresenter{}.PresentCopy(newPage))
}
