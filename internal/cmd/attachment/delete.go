package attachment

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

// NewCmdDelete creates the attachment delete command.
func NewCmdDelete() *cobra.Command {
	opts := &deleteOptions{}

	cmd := &cobra.Command{
		Use:   "delete <attachment-id>",
		Short: "Delete an attachment",
		Long:  `Delete an attachment by its ID.`,
		Example: `  # Delete an attachment
  cfl attachment delete att123

  # Delete without confirmation
  cfl attachment delete att123 --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.output, _ = cmd.Flags().GetString("output")
			opts.noColor, _ = cmd.Flags().GetBool("no-color")
			return runDeleteAttachment(args[0], opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runDeleteAttachment(attachmentID string, opts *deleteOptions) error {
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

	// Get attachment info first to show what we're deleting
	attachment, err := client.GetAttachment(context.Background(), attachmentID)
	if err != nil {
		return fmt.Errorf("failed to get attachment: %w", err)
	}

	renderer := view.NewRenderer(view.Format(opts.output), opts.noColor)

	// Confirm deletion unless --force is used
	if !opts.force {
		fmt.Printf("About to delete attachment: %s (ID: %s)\n", attachment.Title, attachment.ID)
		fmt.Print("Are you sure? [y/N]: ")

		var confirm string
		_, _ = fmt.Scanln(&confirm)

		if confirm != "y" && confirm != "Y" {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	// Delete the attachment
	if err := client.DeleteAttachment(context.Background(), attachmentID); err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	if opts.output == "json" {
		return renderer.RenderJSON(map[string]string{
			"status":        "deleted",
			"attachment_id": attachmentID,
			"title":         attachment.Title,
		})
	}

	renderer.Success(fmt.Sprintf("Deleted attachment: %s (ID: %s)", attachment.Title, attachmentID))

	return nil
}
