package space

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
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
  cfl space view DEV

  # View as JSON
  cfl space view DEV -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runView(args[0], opts)
		},
	}

	return cmd
}

func runView(spaceKey string, opts *viewOptions) error {
	if err := view.ValidateFormat(opts.Output); err != nil {
		return err
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	space, err := client.GetSpaceByKey(context.Background(), spaceKey)
	if err != nil {
		return fmt.Errorf("failed to get space: %w", err)
	}

	v := opts.View()

	if opts.Output == "json" {
		return v.JSON(space)
	}

	v.RenderKeyValue("Key", space.Key)
	v.RenderKeyValue("Name", space.Name)
	v.RenderKeyValue("ID", space.ID)
	v.RenderKeyValue("Type", space.Type)
	if space.Status != "" {
		v.RenderKeyValue("Status", space.Status)
	}
	if space.Description != nil && space.Description.Plain != nil && space.Description.Plain.Value != "" {
		v.RenderKeyValue("Description", space.Description.Plain.Value)
	}

	return nil
}
