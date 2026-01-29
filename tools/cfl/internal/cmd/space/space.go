// Package space provides space-related commands.
package space

import (
	"github.com/spf13/cobra"
)

// NewCmdSpace creates the space command.
func NewCmdSpace() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "space",
		Aliases: []string{"spaces"},
		Short:   "Manage Confluence spaces",
		Long:    `Commands for listing and viewing Confluence spaces.`,
	}

	cmd.AddCommand(NewCmdList())

	return cmd
}
