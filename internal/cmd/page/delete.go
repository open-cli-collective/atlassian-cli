package page

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rianjs/confluence-cli/api"
	"github.com/rianjs/confluence-cli/internal/config"
	"github.com/rianjs/confluence-cli/internal/view"
)

type deleteOptions struct {
	force   bool
	output  string
	noColor bool
}

// NewCmdDelete creates the page delete command.
func NewCmdDelete() *cobra.Command {
	opts := &deleteOptions{}

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
			opts.output, _ = cmd.Flags().GetString("output")
			opts.noColor, _ = cmd.Flags().GetBool("no-color")
			return runDelete(args[0], opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(pageID string, opts *deleteOptions) error {
	// Load config
	cfg, err := config.LoadWithEnv(config.DefaultConfigPath())
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'cfl init' to configure)", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w (run 'cfl init' to configure)", err)
	}

	// Create API client
	client := api.NewClient(cfg.URL, cfg.Email, cfg.APIToken)

	// Get page info first to show what we're deleting
	page, err := client.GetPage(context.Background(), pageID, nil)
	if err != nil {
		return fmt.Errorf("failed to get page: %w", err)
	}

	renderer := view.NewRenderer(view.Format(opts.output), opts.noColor)

	// Confirm deletion unless --force is used
	if !opts.force {
		fmt.Printf("About to delete page: %s (ID: %s)\n", page.Title, page.ID)
		fmt.Print("Are you sure? [y/N]: ")

		var confirm string
		fmt.Scanln(&confirm)

		if confirm != "y" && confirm != "Y" {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	// Delete the page
	if err := client.DeletePage(context.Background(), pageID); err != nil {
		return fmt.Errorf("failed to delete page: %w", err)
	}

	if opts.output == "json" {
		return renderer.RenderJSON(map[string]string{
			"status":  "deleted",
			"page_id": pageID,
			"title":   page.Title,
		})
	}

	renderer.Success(fmt.Sprintf("Deleted page: %s (ID: %s)", page.Title, pageID))

	return nil
}
