// Package setcredential provides the `jtk set-credential` command — a
// thin cobra wrapper over shared keyring.SetCredential. All read/
// validate/write logic lives in shared/ (which never imports cobra).
package setcredential

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/keyring"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// Register adds the set-credential command to the root command.
func Register(rootCmd *cobra.Command, opts *root.Options) {
	var fromEnv string

	cmd := &cobra.Command{
		Use:   "set-credential",
		Short: "Store the Atlassian API token in the OS keyring",
		Long: `Store the Atlassian API token in the OS keyring (non-interactive).

The token is read from stdin, or from an environment variable with
--from-env. It is never echoed. There is one shared token (api_token)
used by both jtk and cfl.`,
		Example: `  # From a secrets manager, into the shared bundle
  op read op://vault/atlassian/token | jtk set-credential

  # From an environment variable
  jtk set-credential --from-env JIRA_API_TOKEN`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := keyring.SetCredential(opts.Stdin, fromEnv); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(opts.Stderr, "API token stored in the OS keyring.")
			return nil
		},
	}

	cmd.Flags().StringVar(&fromEnv, "from-env", "", "Read the token from this environment variable instead of stdin")

	rootCmd.AddCommand(cmd)
}
