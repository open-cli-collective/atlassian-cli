package space

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

type createOptions struct {
	*root.Options
	key         string
	name        string
	description string
	spaceType   string
}

func newCreateCmd(rootOpts *root.Options) *cobra.Command {
	opts := &createOptions{Options: rootOpts}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new space",
		Long:  `Create a new Confluence space.`,
		Example: `  # Create a global space
  cfl space create --key DEV --name "Development"

  # Create with description
  cfl space create --key DEV --name "Development" --description "Development team space"`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCreate(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.key, "key", "k", "", "Space key (required)")
	cmd.Flags().StringVarP(&opts.name, "name", "n", "", "Space name (required)")
	cmd.Flags().StringVarP(&opts.description, "description", "d", "", "Space description")
	cmd.Flags().StringVarP(&opts.spaceType, "type", "t", "global", "Space type (global, personal)")

	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func runCreate(ctx context.Context, opts *createOptions) error {
	if err := view.ValidateFormat(opts.Output); err != nil {
		return err
	}

	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	req := &api.CreateSpaceRequest{
		Key:  opts.key,
		Name: opts.name,
		Type: opts.spaceType,
	}

	if opts.description != "" {
		req.Description = &api.SpaceDescription{
			Plain: &api.DescriptionValue{Value: opts.description},
		}
	}

	space, err := client.CreateSpace(ctx, req)
	if err != nil {
		return fmt.Errorf("creating space: %w", err)
	}

	v := opts.View()

	if opts.Output == "json" {
		return v.JSON(space)
	}

	v.Success("Created space: %s", space.Name)
	v.RenderKeyValue("Key", space.Key)
	if space.Links.WebUI != "" {
		v.RenderKeyValue("URL", cfg.URL+space.Links.WebUI)
	}

	return nil
}
