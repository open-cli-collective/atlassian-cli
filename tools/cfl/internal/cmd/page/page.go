// Package page provides page-related commands.
package page

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

// Register adds page commands to the root command.
func Register(rootCmd *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "page",
		Aliases: []string{"pages"},
		Short:   "Manage Confluence pages",
		Long:    `Commands for creating, viewing, editing, and listing Confluence pages.`,
	}

	cmd.AddCommand(newListCmd(opts))
	cmd.AddCommand(newViewCmd(opts))
	cmd.AddCommand(newCreateCmd(opts))
	cmd.AddCommand(newEditCmd(opts))
	cmd.AddCommand(newDeleteCmd(opts))
	cmd.AddCommand(newCopyCmd(opts))

	rootCmd.AddCommand(cmd)
}
