package completion

import (
	"github.com/spf13/cobra"
)

// NewCmdBash creates the bash completion command.
func NewCmdBash() *cobra.Command {
	return &cobra.Command{
		Use:   "bash",
		Short: "Generate bash completion script",
		Long: `Generate bash completion script for cfl.

To load completions in your current shell session:

  source <(cfl completion bash)

To load completions for every new session:

  # Linux
  cfl completion bash > /etc/bash_completion.d/cfl

  # macOS (requires bash-completion)
  cfl completion bash > $(brew --prefix)/etc/bash_completion.d/cfl`,
		Example: `  # Load in current session
  source <(cfl completion bash)

  # Install permanently (Linux)
  cfl completion bash | sudo tee /etc/bash_completion.d/cfl > /dev/null

  # Install permanently (macOS with Homebrew)
  cfl completion bash > $(brew --prefix)/etc/bash_completion.d/cfl`,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
		},
	}
}
