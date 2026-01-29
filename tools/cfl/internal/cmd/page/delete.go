package page

import (
	"bufio"
	"context"
	"fmt"

	"github.com/spf13/cobra"

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
		RunE: func(_ *cobra.Command, args []string) error {
			return runDelete(args[0], opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(pageID string, opts *deleteOptions) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	page, err := client.GetPage(context.Background(), pageID, nil)
	if err != nil {
		return fmt.Errorf("failed to get page: %w", err)
	}

	v := opts.View()

	if !opts.force {
		fmt.Printf("About to delete page: %s (ID: %s)\n", page.Title, page.ID)
		fmt.Print("Are you sure? [y/N]: ")

		scanner := bufio.NewScanner(opts.Stdin)
		var confirm string
		if scanner.Scan() {
			confirm = scanner.Text()
		}

		if confirm != "y" && confirm != "Y" {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	if err := client.DeletePage(context.Background(), pageID); err != nil {
		return fmt.Errorf("failed to delete page: %w", err)
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
