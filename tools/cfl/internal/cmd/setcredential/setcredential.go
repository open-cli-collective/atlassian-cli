// Package setcredential provides the `cfl set-credential` command — a
// thin cobra wrapper over shared keyring.SetCredential. All read/
// validate/write logic lives in shared/ (which never imports cobra).
package setcredential

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/keyring"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

// Register adds the set-credential command to the root command.
func Register(rootCmd *cobra.Command, opts *root.Options) {
	var fromEnv, key string

	cmd := &cobra.Command{
		Use:   "set-credential",
		Short: "Store the Atlassian API token in the OS keyring",
		Long: `Store the Atlassian API token in the OS keyring (non-interactive).

The token is read from stdin, or from an environment variable with
--from-env. It is never echoed. The default key (api_token) is the
shared token used by both cfl and jtk; use --key cfl_api_token to set a
cfl-only override. This command cannot write jtk's override key (cfl
would never read it) — use jtk set-credential for that.`,
		Example: `  # From a secrets manager, into the shared bundle
  op read op://vault/atlassian/token | cfl set-credential

  # From an environment variable, cfl-only override
  cfl set-credential --from-env CFL_API_TOKEN --key cfl_api_token`,
		RunE: func(_ *cobra.Command, _ []string) error {
			// Restrict to keys cfl actually resolves — writing the
			// sibling's override key would store a token cfl never reads.
			if key != keyring.KeyAPIToken && key != keyring.KeyFor(credstore.ToolCFL) {
				return fmt.Errorf("invalid --key %q for cfl (allowed: %s, %s)",
					key, keyring.KeyAPIToken, keyring.KeyFor(credstore.ToolCFL))
			}
			if err := keyring.SetCredential(opts.Stdin, key, fromEnv); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(opts.Stderr, "API token stored in the OS keyring.")
			return nil
		},
	}

	cmd.Flags().StringVar(&fromEnv, "from-env", "", "Read the token from this environment variable instead of stdin")
	cmd.Flags().StringVar(&key, "key", keyring.KeyAPIToken, "Bundle key: api_token (shared) or cfl_api_token (cfl-only override)")

	rootCmd.AddCommand(cmd)
}
