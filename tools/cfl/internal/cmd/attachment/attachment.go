// Package attachment provides attachment-related commands.
package attachment

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

// Register adds attachment commands to the root command.
func Register(rootCmd *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "attachment",
		Aliases: []string{"attachments", "att"},
		Short:   "Manage Confluence attachments",
		Long:    `Commands for listing, uploading, and downloading Confluence page attachments.`,
	}

	cmd.AddCommand(newListCmd(opts))
	cmd.AddCommand(newUploadCmd(opts))
	cmd.AddCommand(newDownloadCmd(opts))
	cmd.AddCommand(newDeleteCmd(opts))

	rootCmd.AddCommand(cmd)
}
