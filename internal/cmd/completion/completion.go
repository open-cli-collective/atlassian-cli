package completion

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// Register registers the completion command
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for jira-ticket-cli.

To load completions:

Bash:
  $ source <(jira-ticket-cli completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ jira-ticket-cli completion bash > /etc/bash_completion.d/jira-ticket-cli
  # macOS:
  $ jira-ticket-cli completion bash > $(brew --prefix)/etc/bash_completion.d/jira-ticket-cli

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  # To load completions for each session, execute once:
  $ jira-ticket-cli completion zsh > "${fpath[1]}/_jira-ticket-cli"
  # You will need to start a new shell for this setup to take effect.

Fish:
  $ jira-ticket-cli completion fish | source
  # To load completions for each session, execute once:
  $ jira-ticket-cli completion fish > ~/.config/fish/completions/jira-ticket-cli.fish

PowerShell:
  PS> jira-ticket-cli completion powershell | Out-String | Invoke-Expression
  # To load completions for every new session, run:
  PS> jira-ticket-cli completion powershell > jira-ticket-cli.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}

	parent.AddCommand(cmd)
}
