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
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd.Context(), args[0], opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(ctx context.Context, spaceKey string, opts *deleteOptions) error {
	if err := view.ValidateFormat(opts.Output); err != nil {
		return err
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	space, err := client.GetSpaceByKey(ctx, spaceKey)
	if err != nil {
		return fmt.Errorf("getting space: %w", err)
	}

	v := opts.View()

	if !opts.force {
		_, _ = fmt.Fprintf(opts.Stderr, "About to delete space: %s (%s)\n", space.Name, space.Key)
		_, _ = fmt.Fprint(opts.Stderr, "Are you sure? [y/N]: ")

		confirmed, err := prompt.Confirm(opts.Stdin)
		if err != nil {
			return fmt.Errorf("reading confirmation: %w", err)
		}
		if !confirmed {
			_, _ = fmt.Fprintln(opts.Stderr, "Deletion cancelled.")
			return nil
		}
	}

	if err := client.DeleteSpace(ctx, spaceKey); err != nil {
		return fmt.Errorf("deleting space: %w", err)
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
