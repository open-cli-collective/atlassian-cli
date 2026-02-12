// Package space provides space-related commands.
package space

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

// Register adds space commands to the root command.
func Register(rootCmd *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "space",
		Aliases: []string{"spaces"},
		Short:   "Manage Confluence spaces",
		Long:    `Commands for listing, viewing, creating, updating, and deleting Confluence spaces.`,
	}

	cmd.AddCommand(newListCmd(opts))
	cmd.AddCommand(newViewCmd(opts))
	cmd.AddCommand(newCreateCmd(opts))
	cmd.AddCommand(newUpdateCmd(opts))
	cmd.AddCommand(newDeleteCmd(opts))

	rootCmd.AddCommand(cmd)
}
