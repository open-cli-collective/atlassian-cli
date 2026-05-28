// Package init provides the init command for cfl.
package init

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/auth"
	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/keyring"
	"github.com/open-cli-collective/atlassian-go/prompt"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/me"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	"github.com/open-cli-collective/confluence-cli/internal/config"
)

// clientBuilder constructs an *api.Client from a config.
// Pulled out as a parameter so tests can inject an httptest-pointed client
// without depending on api.NewBearerClient's hardcoded gateway URL.
type clientBuilder func(cfg *config.Config) (*api.Client, error)

func defaultClientBuilder(cfg *config.Config) (*api.Client, error) {
	if cfg.AuthMethod == auth.AuthMethodBearer {
		return api.NewBearerClient(cfg.APIToken, cfg.CloudID)
	}
	return api.NewClient(cfg.URL, cfg.Email, cfg.APIToken), nil
}

// Register adds the init command to the root command.
func Register(rootCmd *cobra.Command, opts *root.Options) {
	rootCmd.AddCommand(newInitCmd(opts))
}

// newInitCmd creates the init command.
func newInitCmd(opts *root.Options) *cobra.Command {
	var (
		url        string
		email      string
		authMethod string
		cloudID    string
		noVerify   bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize cfl configuration",
		Long: `Initialize cfl with your Confluence Cloud credentials.

This command will guide you through setting up your Confluence URL,
email, and API token. The configuration will be saved to ~/.config/cfl/config.yml.

For classic API tokens (basic auth):
  1. Go to https://id.atlassian.com/manage-profile/security/api-tokens
  2. Click "Create API token"
  3. Copy the token (it won't be shown again)

For service account scoped tokens (bearer auth):
  Use --auth-method bearer with your scoped API token and Cloud ID.
  Find your Cloud ID at: https://your-site.atlassian.net/_edge/tenant_info`,
		Example: `  # Interactive setup (basic auth)
  cfl init

  # Pre-populate URL
  cfl init --url https://mycompany.atlassian.net

  # Service account (bearer auth) setup
  cfl init --auth-method bearer --url https://mycompany.atlassian.net --cloud-id YOUR_CLOUD_ID`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(cmd.Context(), opts, url, email, authMethod, cloudID, noVerify)
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "Confluence URL (e.g., https://mycompany.atlassian.net)")
	cmd.Flags().StringVar(&email, "email", "", "Your Atlassian account email")
	cmd.Flags().StringVar(&authMethod, "auth-method", "", "Authentication method: basic (default) or bearer")
	cmd.Flags().StringVar(&cloudID, "cloud-id", "", "Atlassian Cloud ID (required for bearer auth)")
	cmd.Flags().BoolVar(&noVerify, "no-verify", false, "Skip connection verification")

	return cmd
}

func runInit(ctx context.Context, opts *root.Options, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID string, noVerify bool) error {
	v := opts.View()

	// Validate --auth-method flag early, before any interactive prompts
	if prefillAuthMethod != "" {
		if err := auth.ValidateAuthMethod(prefillAuthMethod); err != nil {
			return err
		}
	}

	legacyPath := config.DefaultConfigPath()
	sharedPath, err := credstore.DefaultPath()
	if err != nil {
		v.Error("Cannot resolve the shared credential store path: %v", err)
		v.Error("Set XDG_CONFIG_HOME to an absolute path (or unset it), then re-run cfl init.")
		return err
	}
	jtkLegacyPath := credstore.LegacyJTKPath()

	// §2.2 ordering (MON-5328): detect connection divergence FIRST,
	// before any mutation. detectAndReconcile reads the pre-migration
	// projection and fails loud (mutating nothing) if per-tool / legacy
	// connections diverge. Only once that passes do we run the §1.8
	// token migration — so a connection conflict can never be preempted
	// by a token migration/scrub, and a divergent file is never mutated.
	result, err := detectAndReconcile(v, legacyPath, jtkLegacyPath, sharedPath,
		prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID)
	if err != nil {
		return err
	}
	cfg := result.prefill

	// Now the one-time §1.8 token migration: relocate any pre-existing
	// legacy plaintext token into the single shared keyring api_token
	// (token-only, connection-preserving scrub) before the user sets a
	// new one.
	if err := keyring.EnsureMigrated(); err != nil {
		v.Error("Could not prepare secure credential storage: %v", err)
		return err
	}

	// EnsureMigrated relocated any legacy plaintext token into the
	// keyring and scrubbed it from disk, so prefill.APIToken is empty
	// even though the token still exists. Backfill it from the keyring
	// so a returning user isn't forced to re-enter a just-migrated
	// token. NoMigrate: migration already ran. Value stays
	// password-masked in the form; never displayed.
	if cfg.APIToken == "" {
		if tok, _, terr := keyring.ResolveTokenNoMigrate(credstore.ToolCFL); terr == nil {
			cfg.APIToken = tok
		}
	}

	// Determine auth method for form building
	isBearer := cfg.AuthMethod == auth.AuthMethodBearer

	// Build the form based on auth method
	var formGroups []*huh.Group

	if isBearer {
		// Bearer auth: URL + token + cloud ID (no email)
		formGroups = append(formGroups, huh.NewGroup(
			huh.NewInput().
				Title("Confluence URL").
				Description("Instance URL for display purposes only (API calls go through the gateway)").
				Placeholder("https://mycompany.atlassian.net").
				Value(&cfg.URL).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("URL is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("API Token").
				Description("Scoped API token for your service account").
				EchoMode(huh.EchoModePassword).
				Value(&cfg.APIToken).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("API token is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Cloud ID").
				Description("Find at: https://your-site.atlassian.net/_edge/tenant_info").
				Value(&cfg.CloudID).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("cloud ID is required for bearer auth")
					}
					return nil
				}),

			huh.NewInput().
				Title("Default Space (optional)").
				Description("Default space key for page operations").
				Placeholder("MYSPACE").
				Value(&cfg.DefaultSpace),
		))
	} else {
		// Basic auth: URL + email + token
		formGroups = append(formGroups, huh.NewGroup(
			huh.NewInput().
				Title("Confluence URL").
				Description("Your Confluence Cloud instance URL").
				Placeholder("https://mycompany.atlassian.net").
				Value(&cfg.URL).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("URL is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Email").
				Description("Your Atlassian account email").
				Placeholder("you@example.com").
				Value(&cfg.Email).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("email is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("API Token").
				Description("Generate at: id.atlassian.com/manage-profile/security/api-tokens").
				EchoMode(huh.EchoModePassword).
				Value(&cfg.APIToken).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("API token is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Default Space (optional)").
				Description("Default space key for page operations").
				Placeholder("MYSPACE").
				Value(&cfg.DefaultSpace),
		))
	}

	// §3.4: under --non-interactive (or a non-TTY stdin), the huh form
	// can't run — every required value must already be in cfg from the
	// flag prefills and the keyring backfill. cfl init has no --token
	// flag, so the token MUST come from a pre-staged keyring entry
	// (via `cfl set-credential`). Fail loud naming the first missing
	// field.
	if !prompt.WantPrompt(opts.NonInteractive, opts.Stdin) {
		if err := requireNonInteractiveFields(cfg, isBearer); err != nil {
			return err
		}
	} else {
		form := huh.NewForm(formGroups...)
		if err := form.Run(); err != nil {
			return err
		}
	}

	cfg.NormalizeURL()

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return finalizeInit(ctx, opts, cfg, result, sharedPath, noVerify, defaultClientBuilder)
}

// finalizeInit runs the verify/save/render pipeline after the form has produced
// a normalized + validated config. Extracted as a non-interactive seam so tests
// can supply an httptest-backed clientBuilder and temp paths.
func finalizeInit(
	ctx context.Context,
	opts *root.Options,
	cfg *config.Config,
	result *reconcileResult,
	sharedPath string,
	noVerify bool,
	build clientBuilder,
) error {
	v := opts.View()

	var verifiedUser *api.User

	if !noVerify {
		client, err := build(cfg)
		if err != nil {
			v.Error("Could not construct API client: %v", err)
			return fmt.Errorf("creating client: %w", err)
		}

		user, err := client.GetCurrentUser(ctx)
		if err != nil {
			// Both lines go to stderr (via v.Error) so a script capturing
			// only stderr sees the failure AND the remediation hint.
			v.Error("Connection failed: %v", err)
			v.Error("Check your credentials and try again")
			return fmt.Errorf("authentication failed: %w", err)
		}

		v.Success("Connected to %s", cfg.URL)
		verifiedUser = user
	}

	if result.affectsSibling {
		if !prompt.WantPrompt(opts.NonInteractive, opts.Stdin) {
			// §3.4: scripted ingress opted in to shared-store mutation by
			// passing --non-interactive; surface the sibling impact on
			// stderr for the audit trail but proceed with the save.
			v.Info("Saving credentials affects jtk (shared default section); proceeding under --non-interactive.")
		} else {
			var confirm bool
			if err := huh.NewConfirm().
				Title("Save will affect jtk").
				Description("These credentials are stored in shared `default` and used by both cfl and jtk. Continue?").
				Affirmative("Save").
				Negative("Cancel").
				Value(&confirm).
				Run(); err != nil {
				return err
			}
			if !confirm {
				v.Info("Initialization cancelled. No changes saved.")
				return nil
			}
		}
	}

	applyResultToStore(result.store, cfg)
	if err := result.store.Save(sharedPath); err != nil {
		return fmt.Errorf("saving shared store: %w", err)
	}

	// The token never lands in the plaintext store (Save strips it) — it
	// goes to the OS keyring under the single shared api_token (§1.11.10:
	// one key for both jtk and cfl; the reconcile write-target governs
	// only NON-secret placement, untouched here).
	if err := keyring.PersistToken(cfg.APIToken); err != nil {
		v.Error("Saved the non-secret config to %s, but could not store the API token in the keyring: %v", sharedPath, err)
		v.Error("Recover by storing just the token (no need to re-run init): `cfl set-credential --ref atlassian-cli/default --key api_token --stdin --overwrite` (reads stdin; use --from-env VAR for env-driven setup).")
		return err
	}
	v.Success("Configuration saved to %s (token stored in the OS keyring)", sharedPath)

	// Optional: clean up legacy files we just migrated.
	for _, lp := range result.consumedLegacies {
		if !prompt.WantPrompt(opts.NonInteractive, opts.Stdin) {
			// §3.4 non-destructive default: under --non-interactive we
			// neither prompt nor delete. The migration already moved the
			// data; leaving the legacy file in place is safe and reversible.
			v.Info("Skipping cleanup of %s under --non-interactive; remove manually if desired.", lp)
			continue
		}
		var deleteIt bool
		if err := huh.NewConfirm().
			Title(fmt.Sprintf("Delete legacy config at %s?", lp)).
			Description("Migrated to the shared store; this file is no longer used.").
			Affirmative("Delete").
			Negative("Keep").
			Value(&deleteIt).
			Run(); err != nil {
			return err
		}
		if deleteIt {
			if err := os.Remove(lp); err != nil {
				v.Error("Could not remove %s: %v", lp, err)
			} else {
				v.Info("Removed %s", lp)
			}
		}
	}

	// Render the equivalent of `cfl me` using the user we already fetched
	// during verify. No second API call, no opts state mutation.
	if verifiedUser != nil {
		v.Println("")
		me.RenderUserOneLiner(v, verifiedUser)
	}

	v.Println("")
	v.Println("You're all set! Try running:")
	v.Println("  cfl space list")
	v.Println("  cfl page list --space <SPACE_KEY>")

	if cfg.AuthMethod == auth.AuthMethodBearer {
		v.Println("")
		v.Info("To switch back to basic auth later, run: cfl init --auth-method basic")
	}

	return nil
}

// requireNonInteractiveFields enforces the §3.4 fail-loud contract for
// scripted/CI runs of `cfl init`. cfl init has no --token flag, so the
// token MUST come from a pre-staged keyring entry via cfl set-credential;
// the error names that path explicitly.
func requireNonInteractiveFields(cfg *config.Config, isBearer bool) error {
	if cfg.URL == "" {
		return fmt.Errorf("--non-interactive: missing required value for --url")
	}
	if isBearer {
		if cfg.CloudID == "" {
			return fmt.Errorf("--non-interactive: missing required value for --cloud-id (bearer auth)")
		}
	} else {
		if cfg.Email == "" {
			return fmt.Errorf("--non-interactive: missing required value for --email (basic auth)")
		}
	}
	if cfg.APIToken == "" {
		return fmt.Errorf("--non-interactive: no API token (cfl init has no --token flag; pre-stage with `cfl set-credential --ref atlassian-cli/default --key api_token --stdin`)")
	}
	return nil
}
