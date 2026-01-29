// Package completion provides shell completion generation commands.
package completion

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

// Register adds the completion command to the root command.
func Register(rootCmd *cobra.Command, _ *root.Options) {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for cfl.

These scripts enable tab-completion for commands, flags, and arguments.
See each sub-command's help for installation instructions.`,
	}

	cmd.AddCommand(newBashCmd())
	cmd.AddCommand(newZshCmd())
	cmd.AddCommand(newFishCmd())
	cmd.AddCommand(newPowerShellCmd())

	rootCmd.AddCommand(cmd)
}
