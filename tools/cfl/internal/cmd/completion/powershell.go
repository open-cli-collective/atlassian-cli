package completion

import (
	"github.com/spf13/cobra"
)

// newPowerShellCmd creates the PowerShell completion command.
func newPowerShellCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "powershell",
		Short: "Generate PowerShell completion script",
		Long: `Generate PowerShell completion script for cfl.

To load completions in your current shell session:

  cfl completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output to your
PowerShell profile ($PROFILE).`,
		Example: `  # Load in current session
  cfl completion powershell | Out-String | Invoke-Expression

  # Install permanently (add to $PROFILE)
  cfl completion powershell >> $PROFILE`,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
		},
	}
}
