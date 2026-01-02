// Package attachment provides attachment-related commands.
package attachment

import (
	"github.com/spf13/cobra"
)

// NewCmdAttachment creates the attachment command.
func NewCmdAttachment() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "attachment",
		Aliases: []string{"attachments", "att"},
		Short:   "Manage Confluence attachments",
		Long:    `Commands for listing, uploading, and downloading Confluence page attachments.`,
	}

	cmd.AddCommand(NewCmdList())
	cmd.AddCommand(NewCmdUpload())
	cmd.AddCommand(NewCmdDownload())

	return cmd
}
