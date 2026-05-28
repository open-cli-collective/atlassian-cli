// Package setcredential provides the `cfl set-credential` command — a
// thin cobra wrapper over shared keyring.RunSetCredential. All read /
// validate / write logic lives in shared/keyring (which never imports
// cobra) so cfl and jtk get identical §1.5.2 behavior.
package setcredential

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/keyring"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

// Register adds the set-credential command to the root command.
func Register(rootCmd *cobra.Command, opts *root.Options) {
	var (
		ref, key, fromEnv      string
		useStdin, overwrite, j bool
	)

	cmd := &cobra.Command{
		Use:   "set-credential",
		Short: "Store the Atlassian API token in the OS keyring (§1.5.2 control-plane ingress)",
		Long: `Store the shared Atlassian API token in the OS keyring (non-interactive).

Exactly one of --stdin or --from-env VAR supplies the token value. The
value is never echoed.

--key is always required. --ref is required when no shared config exists;
when a shared config file is present, --ref defaults to the active
canonical ref. Today there is one shared token (atlassian-cli/default/
api_token) used by both jtk and cfl, so the only currently valid
explicit values are --ref atlassian-cli/default --key api_token; both
flags are forward-compat reservations for future multi-ref support.

Re-running set-credential against an existing entry requires --overwrite.

With --json, emits a control-plane envelope on stdout suitable for
installer-script parsing per cli-common §1.5.2.`,
		Example: `  # From a secrets manager
  op read 'op://Vault/Atlassian/token' | cfl set-credential \
    --ref atlassian-cli/default --key api_token --stdin

  # From an environment variable
  cfl set-credential --ref atlassian-cli/default --key api_token \
    --from-env CFL_API_TOKEN

  # Replace an existing entry
  op read 'op://Vault/Atlassian/token' | cfl set-credential \
    --ref atlassian-cli/default --key api_token --stdin --overwrite

  # Control-plane envelope for installer scripts
  cfl set-credential --ref atlassian-cli/default --key api_token \
    --from-env CFL_API_TOKEN --json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if j {
				cmd.SilenceErrors = true
				cmd.SilenceUsage = true
			}
			return keyring.RunSetCredential(keyring.SetCredentialOpts{
				Stdin:     opts.Stdin,
				Ref:       ref,
				Key:       key,
				FromEnv:   fromEnv,
				UseStdin:  useStdin,
				Overwrite: overwrite,
			}, opts.Stdout, opts.Stderr, j)
		},
	}

	cmd.Flags().StringVar(&ref, "ref", "", "Credential ref (defaults to atlassian-cli/default when a shared config exists; required otherwise)")
	cmd.Flags().StringVar(&key, "key", "", "Credential key (required; today only api_token is supported)")
	cmd.Flags().StringVar(&fromEnv, "from-env", "", "Read the token from this env var (xor with --stdin)")
	cmd.Flags().BoolVar(&useStdin, "stdin", false, "Read the token from stdin (xor with --from-env)")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Replace an existing entry (default: fail if present)")
	cmd.Flags().BoolVar(&j, "json", false, "Emit a §1.5.2 control-plane envelope to stdout")

	rootCmd.AddCommand(cmd)
}
