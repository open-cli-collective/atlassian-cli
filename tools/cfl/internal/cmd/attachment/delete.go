package attachment

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/open-cli-collective/atlassian-go/view"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/config"
)

type deleteOptions struct {
	force   bool
	output  string
	noColor bool
	stdin   io.Reader // For testing; defaults to os.Stdin
}

// NewCmdDelete creates the attachment delete command.
func NewCmdDelete() *cobra.Command {
	opts := &deleteOptions{
		stdin: os.Stdin,
	}

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
			return runDeleteAttachment(args[0], opts, nil)
		},
	}

	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

// runDeleteAttachment executes the delete. If client is nil, it creates one from config.
func runDeleteAttachment(attachmentID string, opts *deleteOptions, client *api.Client) error {
	// Create client from config if not provided (for testing)
	if client == nil {
		cfg, err := config.LoadWithEnv(config.DefaultConfigPath())
		if err != nil {
			return fmt.Errorf("failed to load config: %w (run 'cfl init' to configure)", err)
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid config: %w (run 'cfl init' to configure)", err)
		}

		client = api.NewClient(cfg.URL, cfg.Email, cfg.APIToken)
	}

	// Get attachment info first to show what we're deleting
	attachment, err := client.GetAttachment(context.Background(), attachmentID)
	if err != nil {
		return fmt.Errorf("failed to get attachment: %w", err)
	}

	v := view.New(view.Format(opts.output), opts.noColor)

	// Confirm deletion unless --force is used
	if !opts.force {
		fmt.Printf("About to delete attachment: %s (ID: %s)\n", attachment.Title, attachment.ID)
		fmt.Print("Are you sure? [y/N]: ")

		scanner := bufio.NewScanner(opts.stdin)
		var confirm string
		if scanner.Scan() {
			confirm = scanner.Text()
		}

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
		return v.JSON(map[string]string{
			"status":        "deleted",
			"attachment_id": attachmentID,
			"title":         attachment.Title,
		})
	}

	v.Success("Deleted attachment: %s (ID: %s)", attachment.Title, attachmentID)

	return nil
}
