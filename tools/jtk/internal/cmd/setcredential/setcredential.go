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
	var fromEnv, key, ref string

	cmd := &cobra.Command{
		Use:   "set-credential",
		Short: "Store the Atlassian API token in the OS keyring",
		Long: `Store the Atlassian API token in the OS keyring (non-interactive).

The token is read from stdin, or from an environment variable with
--from-env. It is never echoed. The default key (api_token) is the
shared token used by both jtk and cfl; use --key jtk_api_token to set a
jtk-only override.`,
		Example: `  # From a secrets manager, into the shared bundle
  op read op://vault/atlassian/token | jtk set-credential

  # From an environment variable, jtk-only override
  jtk set-credential --from-env JIRA_API_TOKEN --key jtk_api_token`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := keyring.SetCredential(opts.Stdin, key, fromEnv, ref); err != nil {
				return err
			}
			fmt.Fprintln(opts.Stderr, "API token stored in the OS keyring.")
			return nil
		},
	}

	cmd.Flags().StringVar(&fromEnv, "from-env", "", "Read the token from this environment variable instead of stdin")
	cmd.Flags().StringVar(&key, "key", keyring.KeyAPIToken, "Bundle key: api_token (shared), cfl_api_token, or jtk_api_token")
	cmd.Flags().StringVar(&ref, "ref", "", "Keyring ref (default: the shared atlassian-cli/default bundle)")

	rootCmd.AddCommand(cmd)
}
