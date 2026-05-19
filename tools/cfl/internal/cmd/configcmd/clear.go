package configcmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/keyring"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

type clearOptions struct {
	*root.Options
	force bool
	all   bool
	stdin io.Reader // For testing
}

func newClearCmd(opts *root.Options) *cobra.Command {
	clearOpts := &clearOptions{
		Options: opts,
		stdin:   os.Stdin,
	}

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear the stored Atlassian API token from the OS keyring",
		Long: `Remove the stored API token from the OS keyring.

By default this deletes the single shared api_token (cfl and jtk
resolve the same key, so jtk also loses access — you will be warned).
The exact ref and key are previewed before deletion.

Use --all to remove the ENTIRE shared bundle plus the shared non-secret
config file and scrub any surviving legacy plaintext files.

Note: CFL_API_TOKEN / ATLASSIAN_API_TOKEN environment variables still
override at runtime and cannot be cleared by this command.`,
		Example: `  # Clear cfl's resolved token key (with confirmation + preview)
  cfl config clear

  # Clear without confirmation
  cfl config clear --force

  # Remove the entire shared bundle and config file
  cfl config clear --all`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runClear(clearOpts)
		},
	}

	cmd.Flags().BoolVarP(&clearOpts.force, "force", "f", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&clearOpts.all, "all", false, "Remove the entire shared bundle + config file (destructive)")

	return cmd
}

func runClear(opts *clearOptions) error {
	// One keyring open for the whole flow: PlanClear hands back the open
	// store the delete/clear step reuses (no second passphrase prompt).
	// The env + plaintext-file fields are populated even when the keyring
	// cannot be opened, so `--all` can still clean plaintext artifacts.
	plan, store, err := keyring.PlanClear(credstore.ToolCFL, opts.all)
	if store != nil {
		defer func() { _ = store.Close() }()
	}
	if err != nil && !opts.all {
		return fmt.Errorf("inspecting keyring: %w", err)
	}

	confirm := func(prompt string) (bool, error) {
		if opts.force {
			return true, nil
		}
		_, _ = fmt.Fprint(opts.Stderr, prompt+" [y/N]: ")
		var response string
		_, ferr := fmt.Fscanln(opts.stdin, &response)
		if ferr != nil && ferr.Error() != "unexpected newline" {
			return false, ferr
		}
		response = strings.TrimSpace(strings.ToLower(response))
		return response == "y" || response == "yes", nil
	}

	envNote := func() {
		if len(plan.EnvActive) > 0 {
			_, _ = fmt.Fprintf(opts.Stderr,
				"Note: %s still set in the environment and will continue to override at runtime (not cleared).\n",
				strings.Join(plan.EnvActive, ", "))
		}
	}

	if opts.all {
		_, _ = fmt.Fprintf(opts.Stderr, "This will remove the ENTIRE shared keyring bundle %s", plan.Ref)
		if len(plan.ExistingKeys) > 0 {
			_, _ = fmt.Fprintf(opts.Stderr, " (keys: %s)", strings.Join(plan.ExistingKeys, ", "))
		}
		_, _ = fmt.Fprintln(opts.Stderr, ".")
		if plan.SharedConfigPath != "" {
			_, _ = fmt.Fprintf(opts.Stderr, "It will also delete the shared config file: %s\n", plan.SharedConfigPath)
		}
		if plan.OldSharedConfigPath != "" {
			_, _ = fmt.Fprintf(opts.Stderr, "It will also delete the prior shared config file: %s\n", plan.OldSharedConfigPath)
		}
		for _, lp := range plan.LegacyPaths {
			_, _ = fmt.Fprintf(opts.Stderr, "It will scrub the legacy plaintext file: %s\n", lp)
		}
		if err != nil {
			_, _ = fmt.Fprintf(opts.Stderr,
				"Note: the keyring could not be opened (%v); plaintext artifacts will still be cleaned, but the keyring bundle will be left intact.\n", err)
		}
		ok, cerr := confirm("Proceed?")
		if cerr != nil {
			return cerr
		}
		if !ok {
			_, _ = fmt.Fprintln(opts.Stderr, "Cancelled. Nothing was cleared.")
			return nil
		}
		cleared, aerr := keyring.ClearAll(store)
		if aerr != nil {
			return aerr
		}
		if !cleared {
			return fmt.Errorf(
				"plaintext artifacts were cleaned, but the keyring bundle %s was NOT cleared because the keyring is unavailable (%w); fix the keyring and re-run `cfl config clear --all`",
				plan.Ref, err)
		}
		_, _ = fmt.Fprintln(opts.Stderr, "Removed the shared keyring bundle and config file.")
		envNote()
		return nil
	}

	if plan.ToolKey == "" {
		_, _ = fmt.Fprintf(opts.Stderr, "No stored API token in keyring %s for cfl; nothing to clear.\n", plan.Ref)
		envNote()
		return nil
	}

	_, _ = fmt.Fprintf(opts.Stderr, "This will delete key %q from keyring %s.\n", plan.ToolKey, plan.Ref)
	// One key per logical credential (§1.11.10): the only deletable key
	// is the shared api_token, so clearing it always deauths the sibling.
	_, _ = fmt.Fprintln(opts.Stderr,
		"Warning: this is the SHARED token (api_token). jtk will also lose access (cfl and jtk resolve the same key).")
	ok, cerr := confirm("Proceed?")
	if cerr != nil {
		return cerr
	}
	if !ok {
		_, _ = fmt.Fprintln(opts.Stderr, "Cancelled. Nothing was cleared.")
		return nil
	}
	if err := store.DeleteToken(plan.ToolKey); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(opts.Stderr, "Removed key %q from keyring %s.\n", plan.ToolKey, plan.Ref)
	envNote()
	return nil
}
