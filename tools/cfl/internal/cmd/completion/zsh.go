package completion

import (
	"github.com/spf13/cobra"
)

// newZshCmd creates the zsh completion command.
func newZshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "zsh",
		Short: "Generate zsh completion script",
		Long: `Generate zsh completion script for cfl.

To load completions in your current shell session:

  source <(cfl completion zsh)

To load completions for every new session, first ensure completion is enabled
(add to ~/.zshrc if not already present):

  autoload -Uz compinit && compinit

Then add the completion script to your fpath:

  cfl completion zsh > "${fpath[1]}/_cfl"

You may need to start a new shell for completions to take effect.`,
		Example: `  # Load in current session
  source <(cfl completion zsh)

  # Install permanently
  mkdir -p ~/.zsh/completions
  cfl completion zsh > ~/.zsh/completions/_cfl

  # Then add to ~/.zshrc:
  # fpath=(~/.zsh/completions $fpath)
  # autoload -Uz compinit && compinit`,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
		},
	}
}
