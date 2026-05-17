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

By default this deletes only the key cfl resolves to: cfl_api_token if a
cfl-specific override exists, otherwise the shared api_token (which jtk
also uses — you will be warned). The exact ref and key are previewed
before deletion.

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
	plan, err := keyring.PlanClear(credstore.ToolCFL)
	if err != nil {
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
		for _, lp := range plan.LegacyPaths {
			_, _ = fmt.Fprintf(opts.Stderr, "It will scrub the legacy plaintext file: %s\n", lp)
		}
		ok, cerr := confirm("Proceed?")
		if cerr != nil {
			return cerr
		}
		if !ok {
			_, _ = fmt.Fprintln(opts.Stderr, "Cancelled. Nothing was cleared.")
			return nil
		}
		if err := keyring.ClearAll(); err != nil {
			return err
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
	if plan.SharedDefault {
		_, _ = fmt.Fprintln(opts.Stderr,
			"Warning: this is the SHARED token (api_token). jtk will also lose access. Use a cfl_api_token override if you want cfl-only credentials.")
	}
	ok, cerr := confirm("Proceed?")
	if cerr != nil {
		return cerr
	}
	if !ok {
		_, _ = fmt.Fprintln(opts.Stderr, "Cancelled. Nothing was cleared.")
		return nil
	}
	deleted, err := keyring.ClearToolKey(credstore.ToolCFL)
	if err != nil {
		return err
	}
	if deleted == "" {
		_, _ = fmt.Fprintln(opts.Stderr, "Nothing to clear.")
	} else {
		_, _ = fmt.Fprintf(opts.Stderr, "Removed key %q from keyring %s.\n", deleted, plan.Ref)
	}
	envNote()
	return nil
}
