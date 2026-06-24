package space

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	cflpresent "github.com/open-cli-collective/confluence-cli/internal/present"
)

type viewOptions struct {
	*root.Options
}

func newViewCmd(rootOpts *root.Options) *cobra.Command {
	opts := &viewOptions{Options: rootOpts}

	cmd := &cobra.Command{
		Use:     "view <space-key>",
		Aliases: []string{"get"},
		Short:   "View space details",
		Long:    `View details of a Confluence space by its key.`,
		Example: `  # View a space
  cfl space view DEV`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runView(cmd.Context(), args[0], opts)
		},
	}

	return cmd
}

func runView(ctx context.Context, spaceKey string, opts *viewOptions) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	space, err := client.GetSpaceByKey(ctx, spaceKey)
	if err != nil {
		return err
	}

	return cflpresent.Emit(opts.Options, cflpresent.SpacePresenter{}.PresentDetail(space, opts.Full))
}
