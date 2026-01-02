// Package page provides page-related commands.
package page

import (
	"github.com/spf13/cobra"
)

// NewCmdPage creates the page command.
func NewCmdPage() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "page",
		Aliases: []string{"pages"},
		Short:   "Manage Confluence pages",
		Long:    `Commands for creating, viewing, editing, and listing Confluence pages.`,
	}

	cmd.AddCommand(NewCmdList())
	cmd.AddCommand(NewCmdView())
	cmd.AddCommand(NewCmdCreate())
	cmd.AddCommand(NewCmdDelete())

	return cmd
}
