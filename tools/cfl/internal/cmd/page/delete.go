package page

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/prompt"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

type deleteOptions struct {
	*root.Options
	force bool
}

func newDeleteCmd(rootOpts *root.Options) *cobra.Command {
	opts := &deleteOptions{Options: rootOpts}

	cmd := &cobra.Command{
		Use:   "delete <page-id>",
		Short: "Delete a page",
		Long:  `Delete a Confluence page by its ID.`,
		Example: `  # Delete a page
  cfl page delete 12345

  # Delete without confirmation
  cfl page delete 12345 --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd.Context(), args[0], opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(ctx context.Context, pageID string, opts *deleteOptions) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// nil opts: body content is not needed, only title for the confirmation prompt
	page, err := client.GetPage(ctx, pageID, nil)
	if err != nil {
		return fmt.Errorf("getting page: %w", err)
	}

	v := opts.View()

	if !opts.force {
		_, _ = fmt.Fprintf(opts.Stderr, "About to delete page: %s (ID: %s)\n", page.Title, page.ID)
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

	if err := client.DeletePage(ctx, pageID); err != nil {
		return fmt.Errorf("deleting page: %w", err)
	}

	if opts.Output == "json" {
		return v.JSON(map[string]string{
			"status":  "deleted",
			"page_id": pageID,
			"title":   page.Title,
		})
	}

	v.Success("Deleted page: %s (ID: %s)", page.Title, pageID)

	return nil
}
