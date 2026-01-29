// Package completion provides shell completion generation commands.
package completion

import (
	"github.com/spf13/cobra"
)

// NewCmdCompletion creates the completion command.
func NewCmdCompletion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for cfl.

These scripts enable tab-completion for commands, flags, and arguments.
See each sub-command's help for installation instructions.`,
	}

	cmd.AddCommand(NewCmdBash())
	cmd.AddCommand(NewCmdZsh())
	cmd.AddCommand(NewCmdFish())
	cmd.AddCommand(NewCmdPowerShell())

	return cmd
}
