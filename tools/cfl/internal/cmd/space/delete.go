package space

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/prompt"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

type deleteOptions struct {
	*root.Options
	force bool
}

func newDeleteCmd(rootOpts *root.Options) *cobra.Command {
	opts := &deleteOptions{Options: rootOpts}

	cmd := &cobra.Command{
		Use:   "delete <space-key>",
		Short: "Delete a space",
		Long:  `Delete a Confluence space by its key.`,
		Example: `  # Delete a space
  cfl space delete TEST

  # Delete without confirmation
  cfl space delete TEST --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runDelete(args[0], opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(spaceKey string, opts *deleteOptions) error {
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

	if !opts.force {
		fmt.Printf("About to delete space: %s (%s)\n", space.Name, space.Key)
		fmt.Print("Are you sure? [y/N]: ")

		confirmed, err := prompt.Confirm(opts.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		if !confirmed {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	if err := client.DeleteSpace(context.Background(), spaceKey); err != nil {
		return fmt.Errorf("failed to delete space: %w", err)
	}

	if opts.Output == "json" {
		return v.JSON(map[string]string{
			"status":    "deleted",
			"space_key": spaceKey,
			"name":      space.Name,
		})
	}

	v.Success("Deleted space: %s (%s)", space.Name, spaceKey)

	return nil
}
