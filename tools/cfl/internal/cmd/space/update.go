package space

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

type updateOptions struct {
	*root.Options
	name        string
	description string
}

func newUpdateCmd(rootOpts *root.Options) *cobra.Command {
	opts := &updateOptions{Options: rootOpts}

	cmd := &cobra.Command{
		Use:   "update <space-key>",
		Short: "Update a space",
		Long:  `Update the name or description of a Confluence space.`,
		Example: `  # Update space name
  cfl space update DEV --name "Development Team"

  # Update space description
  cfl space update DEV --description "Updated description"

  # Update both
  cfl space update DEV --name "Development Team" --description "Updated description"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd.Context(), args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.name, "name", "n", "", "New space name")
	cmd.Flags().StringVarP(&opts.description, "description", "d", "", "New space description")

	return cmd
}

func runUpdate(ctx context.Context, spaceKey string, opts *updateOptions) error {
	if opts.name == "" && opts.description == "" {
		return fmt.Errorf("at least one of --name or --description is required")
	}

	if err := view.ValidateFormat(opts.Output); err != nil {
		return err
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	req := &api.UpdateSpaceRequest{
		Key: spaceKey,
	}

	if opts.name != "" {
		req.Name = opts.name
	}

	if opts.description != "" {
		req.Description = &api.V1SpaceDescription{
			Plain: &api.V1DescriptionValue{
				Value:          opts.description,
				Representation: "plain",
			},
		}
	}

	space, err := client.UpdateSpace(ctx, spaceKey, req)
	if err != nil {
		return fmt.Errorf("updating space: %w", err)
	}

	v := opts.View()

	if opts.Output == "json" {
		return v.JSON(space)
	}

	v.Success("Updated space: %s (%s)", space.Name, space.Key)

	return nil
}
