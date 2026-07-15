package configcmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/keyring"
	promptpkg "github.com/open-cli-collective/atlassian-go/prompt"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	cflpresent "github.com/open-cli-collective/confluence-cli/internal/present"
)

type clearOptions struct {
	*root.Options
	force bool
	all   bool
	stdin io.Reader // For testing
}

var (
	planClear = keyring.PlanClear
	clearAll  = keyring.ClearAll
)

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
	// §3.4: short-circuit BEFORE any keyring inspection so
	// --non-interactive without --force returns ErrConfirmationRequired
	// even if PlanClear would have failed first on a locked/unavailable
	// keyring or surface warning text that contaminates CI logs.
	if opts.NonInteractive && !opts.force {
		return promptpkg.ErrConfirmationRequired
	}

	// One keyring open for the whole flow: PlanClear hands back the open
	// store the delete/clear step reuses (no second passphrase prompt).
	// The env + plaintext-file fields are populated even when the keyring
	// cannot be opened, so `--all` can still clean plaintext artifacts.
	plan, store, err := planClear(credstore.ToolCFL, opts.all)
	if store != nil {
		defer func() { _ = store.Close() }()
	}
	if err != nil && !opts.all {
		return fmt.Errorf("inspecting keyring: %w", err)
	}

	confirm := func(promptText string) (bool, error) {
		if !opts.force && !opts.NonInteractive {
			_, _ = fmt.Fprint(opts.Stderr, promptText+" [y/N]: ")
		}
		return promptpkg.ConfirmOrFail(opts.force, opts.NonInteractive, opts.stdin)
	}

	if opts.all {
		_ = cflpresent.Emit(opts.Options, cflpresent.ConfigPresenter{}.PresentClearAllPlan(plan, err))
		ok, cerr := confirm("Proceed?")
		if cerr != nil {
			return cerr
		}
		if !ok {
			return cflpresent.Emit(opts.Options, cflpresent.ConfigPresenter{}.PresentClearCancelled())
		}
		cleared, aerr := clearAll(store)
		if aerr != nil {
			return aerr
		}
		if !cleared {
			return fmt.Errorf(
				"plaintext artifacts were cleaned, but the keyring bundle %s was NOT cleared because the keyring is unavailable (%w); fix the keyring and re-run `cfl config clear --all`",
				plan.Ref, err)
		}
		return cflpresent.Emit(opts.Options, cflpresent.ConfigPresenter{}.PresentClearAllSuccess(plan))
	}

	if plan.ToolKey == "" {
		return cflpresent.Emit(opts.Options, cflpresent.ConfigPresenter{}.PresentClearNoStoredToken(plan))
	}

	_ = cflpresent.Emit(opts.Options, cflpresent.ConfigPresenter{}.PresentClearDefaultPlan(plan))
	ok, cerr := confirm("Proceed?")
	if cerr != nil {
		return cerr
	}
	if !ok {
		return cflpresent.Emit(opts.Options, cflpresent.ConfigPresenter{}.PresentClearCancelled())
	}
	if err := store.DeleteToken(plan.ToolKey); err != nil {
		return err
	}
	return cflpresent.Emit(opts.Options, cflpresent.ConfigPresenter{}.PresentClearDefaultSuccess(plan))
}
