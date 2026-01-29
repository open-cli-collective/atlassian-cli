package completion

import (
	"github.com/spf13/cobra"
)

// newFishCmd creates the fish completion command.
func newFishCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fish",
		Short: "Generate fish completion script",
		Long: `Generate fish completion script for cfl.

To load completions in your current shell session:

  cfl completion fish | source

To load completions for every new session:

  cfl completion fish > ~/.config/fish/completions/cfl.fish`,
		Example: `  # Load in current session
  cfl completion fish | source

  # Install permanently
  cfl completion fish > ~/.config/fish/completions/cfl.fish`,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
		},
	}
}
