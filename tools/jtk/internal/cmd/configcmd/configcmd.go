// Package configcmd provides CLI commands for managing jtk configuration.
package configcmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/keyring"
	"github.com/open-cli-collective/atlassian-go/present"
	promptpkg "github.com/open-cli-collective/atlassian-go/prompt"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

// Register registers the config commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
		Long:  "Commands for managing jtk configuration and credentials.",
	}

	cmd.AddCommand(newShowCmd(opts))
	cmd.AddCommand(newClearCmd(opts))
	cmd.AddCommand(newTestCmd(opts))

	parent.AddCommand(cmd)
}

func newShowCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long: `Display the current configuration.

The API token is shown as a presence status only (its value lives in the
OS keyring and is never displayed); token/keyring reporting is
authoritative. The non-secret rows reflect environment variables and the
legacy per-tool file ONLY — a value set solely in the shared
~/.config/atlassian-cli/config.yml is shown as "-" here even though jtk
resolves and uses it at runtime.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg := config.GetValuesWithSources()

			model := jtkpresent.ConfigPresenter{}.PresentConfigShow(
				cfg.URL, cfg.URLSource,
				cfg.Email, cfg.EmailSource,
				cfg.TokenConfigured, cfg.TokenSource,
				cfg.KeyringRef, cfg.KeyringBackend, cfg.KeyringPassphrase,
				cfg.DefaultProject, cfg.ProjectSource,
				cfg.AuthMethod, cfg.AuthMethodSrc,
				cfg.CloudID, cfg.CloudIDSrc,
				cfg.Path,
			)
			out := present.Render(model, opts.RenderStyle())
			fmt.Fprint(opts.Stdout, out.Stdout)
			fmt.Fprint(opts.Stderr, out.Stderr)
			return nil
		},
	}
}

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

By default this deletes the single shared api_token (jtk and cfl
resolve the same key, so cfl also loses access — you will be warned).
The exact ref and key are previewed before deletion.

Use --all to remove the ENTIRE shared bundle plus the shared non-secret
config file and scrub any surviving legacy plaintext files.

Note: JIRA_API_TOKEN / ATLASSIAN_API_TOKEN environment variables still
override at runtime and cannot be cleared by this command.`,
		Example: `  # Clear jtk's resolved token key (with confirmation + preview)
  jtk config clear

  # Clear without confirmation
  jtk config clear --force

  # Remove the entire shared bundle and config file
  jtk config clear --all`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runClear(cmd.Context(), clearOpts)
		},
	}

	cmd.Flags().BoolVarP(&clearOpts.force, "force", "f", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&clearOpts.all, "all", false, "Remove the entire shared bundle + config file (destructive)")

	return cmd
}

func runClear(ctx context.Context, opts *clearOptions) error {
	_ = ctx

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
	plan, store, err := keyring.PlanClear(credstore.ToolJTK, opts.all)
	if store != nil {
		defer func() { _ = store.Close() }()
	}
	if err != nil && !opts.all {
		return fmt.Errorf("inspecting keyring: %w", err)
	}

	confirm := func(promptText string) (bool, error) {
		if !opts.force && !opts.NonInteractive {
			fmt.Fprint(opts.Stderr, promptText+" [y/N]: ")
		}
		return promptpkg.ConfirmOrFail(opts.force, opts.NonInteractive, opts.stdin)
	}

	envNote := func() {
		if len(plan.EnvActive) > 0 {
			fmt.Fprintf(opts.Stderr,
				"Note: %s still set in the environment and will continue to override at runtime (not cleared).\n",
				strings.Join(plan.EnvActive, ", "))
		}
	}

	if opts.all {
		fmt.Fprintf(opts.Stderr, "This will remove the ENTIRE shared keyring bundle %s", plan.Ref)
		if len(plan.ExistingKeys) > 0 {
			fmt.Fprintf(opts.Stderr, " (keys: %s)", strings.Join(plan.ExistingKeys, ", "))
		}
		fmt.Fprintln(opts.Stderr, ".")
		if plan.SharedConfigPath != "" {
			fmt.Fprintf(opts.Stderr, "It will also delete the shared config file: %s\n", plan.SharedConfigPath)
		}
		if plan.OldSharedConfigPath != "" {
			fmt.Fprintf(opts.Stderr, "It will also delete the prior shared config file: %s\n", plan.OldSharedConfigPath)
		}
		for _, lp := range plan.LegacyPaths {
			fmt.Fprintf(opts.Stderr, "It will scrub the legacy plaintext file: %s\n", lp)
		}
		if err != nil {
			fmt.Fprintf(opts.Stderr,
				"Note: the keyring could not be opened (%v); plaintext artifacts will still be cleaned, but the keyring bundle will be left intact.\n", err)
		}
		ok, cerr := confirm("Proceed?")
		if cerr != nil {
			return cerr
		}
		if !ok {
			fmt.Fprintln(opts.Stdout, "Cancelled. Nothing was cleared.")
			return nil
		}
		cleared, aerr := keyring.ClearAll(store)
		if aerr != nil {
			return aerr
		}
		if !cleared {
			return fmt.Errorf(
				"plaintext artifacts were cleaned, but the keyring bundle %s was NOT cleared because the keyring is unavailable (%w); fix the keyring and re-run `jtk config clear --all`",
				plan.Ref, err)
		}
		fmt.Fprintln(opts.Stdout, "Removed the shared keyring bundle and config file.")
		envNote()
		return nil
	}

	if plan.ToolKey == "" {
		fmt.Fprintf(opts.Stdout, "No stored API token in keyring %s for jtk; nothing to clear.\n", plan.Ref)
		envNote()
		return nil
	}

	fmt.Fprintf(opts.Stderr, "This will delete key %q from keyring %s.\n", plan.ToolKey, plan.Ref)
	// One key per logical credential (§1.11.10): the only deletable key
	// is the shared api_token, so clearing it always deauths the sibling.
	fmt.Fprintln(opts.Stderr,
		"Warning: this is the SHARED token (api_token). cfl will also lose access (jtk and cfl resolve the same key).")
	ok, cerr := confirm("Proceed?")
	if cerr != nil {
		return cerr
	}
	if !ok {
		fmt.Fprintln(opts.Stdout, "Cancelled. Nothing was cleared.")
		return nil
	}
	if err := store.DeleteToken(plan.ToolKey); err != nil {
		return err
	}
	fmt.Fprintf(opts.Stdout, "Removed key %q from keyring %s.\n", plan.ToolKey, plan.Ref)
	envNote()
	return nil
}

func newTestCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Test connection to Jira",
		Long: `Verify that jtk can connect to Jira with the current configuration.

This command tests authentication and API access, providing clear
pass/fail status and troubleshooting suggestions on failure.`,
		Example: `  # Test connection
  jtk config test`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			url := config.GetURL()
			var user *api.User
			var clientErr, authErr error

			if url != "" {
				client, err := opts.APIClient()
				if err != nil {
					clientErr = err
				} else {
					user, authErr = client.GetCurrentUser(cmd.Context(), "")
				}
			}

			model := jtkpresent.ConfigPresenter{}.PresentTestResult(url, user, clientErr, authErr)
			out := present.Render(model, opts.RenderStyle())
			fmt.Fprint(opts.Stdout, out.Stdout)
			fmt.Fprint(opts.Stderr, out.Stderr)
			return nil
		},
	}
}
