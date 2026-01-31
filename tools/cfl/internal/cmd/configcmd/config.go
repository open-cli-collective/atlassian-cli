// Package configcmd provides the config command for cfl.
package configcmd

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

// Register adds the config command to the root command.
func Register(rootCmd *cobra.Command, opts *root.Options) {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage cfl configuration",
		Long:  `Commands for viewing, testing, and managing cfl configuration.`,
	}

	configCmd.AddCommand(newShowCmd(opts))
	configCmd.AddCommand(newTestCmd(opts))
	configCmd.AddCommand(newClearCmd(opts))

	rootCmd.AddCommand(configCmd)
}
