package attachment

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
		Use:   "delete <attachment-id>",
		Short: "Delete an attachment",
		Long:  `Delete an attachment by its ID.`,
		Example: `  # Delete an attachment
  cfl attachment delete att123

  # Delete without confirmation
  cfl attachment delete att123 --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runDeleteAttachment(args[0], opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runDeleteAttachment(attachmentID string, opts *deleteOptions) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	attachment, err := client.GetAttachment(context.Background(), attachmentID)
	if err != nil {
		return fmt.Errorf("failed to get attachment: %w", err)
	}

	v := opts.View()

	if !opts.force {
		fmt.Printf("About to delete attachment: %s (ID: %s)\n", attachment.Title, attachment.ID)
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

	if err := client.DeleteAttachment(context.Background(), attachmentID); err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	if opts.Output == "json" {
		return v.JSON(map[string]string{
			"status":        "deleted",
			"attachment_id": attachmentID,
			"title":         attachment.Title,
		})
	}

	v.Success("Deleted attachment: %s (ID: %s)", attachment.Title, attachmentID)

	return nil
}
