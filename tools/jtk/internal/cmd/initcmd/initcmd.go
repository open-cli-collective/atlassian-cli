// Package initcmd provides the interactive setup wizard for the jtk CLI.
//
// This package intentionally uses direct view output (v.Println, v.Success, etc.)
// rather than the presenter pattern used elsewhere. The presenter model is designed
// for structured results (tables, detail views, messages) that get rendered once.
// Interactive wizards have a different flow: prompts, stdin reads, progressive
// feedback, and conversational back-and-forth that doesn't fit that model cleanly.
package initcmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/auth"
	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/keyring"
	"github.com/open-cli-collective/atlassian-go/prompt"
	sharedurl "github.com/open-cli-collective/atlassian-go/url"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
)

// Register registers the init command
func Register(parent *cobra.Command, opts *root.Options) {
	var url, email, token, tokenFromEnv, authMethod, cloudID string
	var tokenStdin, noVerify bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize jtk with guided setup",
		Long: `Interactive setup wizard for configuring jtk.

Prompts for your Jira URL, email, and API token, then verifies
the connection before saving the configuration.

For classic API tokens (basic auth):
  Get your token from: https://id.atlassian.com/manage-profile/security/api-tokens

For service account scoped tokens (bearer auth):
  Use --auth-method bearer with your scoped API token and Cloud ID.
  Find your Cloud ID at: https://your-site.atlassian.net/_edge/tenant_info

Scripted ingress (§1.5.1): use --token-stdin or --token-from-env VAR for
the API token. The legacy --token <value> flag is deprecated — it leaks
the secret into shell history and process listings — and will be removed
in a future release.`,
		Example: `  # Interactive setup (basic auth)
  jtk init

  # Non-interactive basic auth setup via stdin pipe (§1.10 idiom)
  op read 'op://Vault/Atlassian/token' | jtk init --non-interactive \
    --url https://mycompany.atlassian.net --email user@example.com --token-stdin

  # Non-interactive setup via env var
  jtk init --non-interactive \
    --url https://mycompany.atlassian.net --email user@example.com --token-from-env JIRA_API_TOKEN

  # Service account (bearer auth) setup
  jtk init --auth-method bearer --url https://mycompany.atlassian.net \
    --token-from-env JIRA_API_TOKEN --cloud-id YOUR_CLOUD_ID

  # Skip connection verification
  jtk init --no-verify`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(cmd.Context(), opts, url, email, token, tokenStdin, tokenFromEnv, authMethod, cloudID, noVerify)
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "Jira URL (e.g., https://mycompany.atlassian.net)")
	cmd.Flags().StringVar(&email, "email", "", "Email address for authentication")
	cmd.Flags().StringVar(&token, "token", "", "DEPRECATED: API token literal (use --token-stdin or --token-from-env; §1.5.1)")
	cmd.Flags().BoolVar(&tokenStdin, "token-stdin", false, "Read the API token from stdin (xor with --token-from-env)")
	cmd.Flags().StringVar(&tokenFromEnv, "token-from-env", "", "Read the API token from this env var (xor with --token-stdin)")
	cmd.Flags().StringVar(&authMethod, "auth-method", "", "Authentication method: basic (default) or bearer")
	cmd.Flags().StringVar(&cloudID, "cloud-id", "", "Atlassian Cloud ID (required for bearer auth)")
	cmd.Flags().BoolVar(&noVerify, "no-verify", false, "Skip connection verification")

	parent.AddCommand(cmd)
}

func runInit(ctx context.Context, opts *root.Options, prefillURL, prefillEmail, prefillToken string, tokenStdin bool, tokenFromEnv, prefillAuthMethod, prefillCloudID string, noVerify bool) error {
	// Validate --auth-method flag early, before any interactive prompts
	if prefillAuthMethod != "" {
		if err := auth.ValidateAuthMethod(prefillAuthMethod); err != nil {
			return err
		}
	}

	v := opts.View()

	// §1.5.1 token-ingress resolution. Explicit ingress flags win over
	// the keyring backfill below (token-rotation contract: a user must
	// be able to re-run init --token-stdin to replace a stale value).
	// Mutual exclusion + empty-value validation happen here. Mutual-
	// exclusion checks MUST precede the TTY guard so the more specific
	// "pick one" error wins over the more general TTY conflict.
	switch {
	case tokenStdin && tokenFromEnv != "":
		return errors.New("--token-stdin and --token-from-env are mutually exclusive; pick one")
	case tokenStdin && prefillToken != "":
		return errors.New("--token and --token-stdin are mutually exclusive; pick one (and prefer --token-stdin — §1.5.1)")
	case tokenFromEnv != "" && prefillToken != "":
		return errors.New("--token and --token-from-env are mutually exclusive; pick one (and prefer --token-from-env — §1.5.1)")
	}
	// --token-stdin drains stdin before the interactive form would read
	// from it. This only matters when the form would actually run —
	// i.e., a real TTY with no --non-interactive. Piped stdin is already
	// non-TTY (WantPrompt=false), so canonical CI usage
	// `op read | jtk init --token-stdin ...` passes through here without
	// requiring --non-interactive. Only reject the TTY + --token-stdin
	// + interactive combo, where the form's first read would EOF.
	if tokenStdin && prompt.WantPrompt(opts.NonInteractive, opts.Stdin) {
		return errors.New("--token-stdin from a TTY conflicts with the interactive form; pipe stdin or pass --non-interactive")
	}
	if scripted, err := prompt.ReadSecretFromIngress(opts.Stdin, tokenStdin, tokenFromEnv); err != nil {
		return err
	} else if scripted != "" {
		prefillToken = scripted
	} else if prefillToken != "" {
		// User passed the deprecated --token <value> flag. Print the
		// §1.5.1 deprecation warning to stderr before proceeding so the
		// signal lands even when the run later errors for an unrelated
		// reason.
		fmt.Fprintln(opts.Stderr,
			"warning: --token is deprecated and will be removed in a future release; "+
				"use --token-stdin or --token-from-env VAR to avoid leaking the secret "+
				"via shell history / process listings (cli-common §1.5.1).")
	}

	sharedPath, err := credstore.DefaultPath()
	if err != nil {
		v.Error("Cannot resolve the shared credential store path: %v", err)
		v.Error("Set XDG_CONFIG_HOME to an absolute path (or unset it), then re-run jtk init.")
		return err
	}
	jtkLegacyPath := credstore.LegacyJTKPath()
	cflLegacyPath := credstore.LegacyCFLPath()

	// §2.2 ordering (MON-5328): detect connection divergence FIRST,
	// before any mutation — detectAndReconcile fails loud (mutating
	// nothing) on divergent per-tool/legacy connections. Only then run
	// the §1.8 token migration, so a connection conflict can never be
	// preempted by a token migration/scrub and a divergent file is never
	// mutated.
	result, err := detectAndReconcile(v, jtkLegacyPath, cflLegacyPath, sharedPath,
		prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)
	if err != nil {
		return err
	}
	cfg := result.prefill

	// Now the one-time §1.8 token migration (token-only,
	// connection-preserving scrub).
	if err := keyring.EnsureMigrated(); err != nil {
		v.Error("Could not prepare secure credential storage: %v", err)
		return err
	}

	// EnsureMigrated relocated any legacy plaintext token into the
	// keyring and scrubbed it from disk, so prefill.APIToken is empty
	// even though the token still exists. Backfill from the keyring so a
	// returning user isn't forced to re-enter a just-migrated token.
	// NoMigrate: migration already ran. Value stays password-masked.
	if cfg.APIToken == "" {
		if tok, _, terr := keyring.ResolveTokenNoMigrate(credstore.ToolJTK); terr == nil {
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
				Title("Jira URL").
				Description("Your Jira instance URL (used for browse links)").
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
				Title("Default Project (optional)").
				Description("Default project key for commands").
				Placeholder("MYPROJ").
				Value(&cfg.DefaultProject),
		))
	} else {
		// Basic auth: URL + email + token
		formGroups = append(formGroups, huh.NewGroup(
			huh.NewInput().
				Title("Jira URL").
				Description("Your Jira instance URL").
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
				Title("Default Project (optional)").
				Description("Default project key for commands").
				Placeholder("MYPROJ").
				Value(&cfg.DefaultProject),
		))
	}

	// §3.4: under --non-interactive (or a non-TTY stdin), the huh form
	// can't run — every required value must already be in cfg from the
	// flag prefills. Fail loud naming the first missing field.
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

	// Normalize URL
	cfg.URL = sharedurl.NormalizeURL(cfg.URL)

	// Verify connection unless --no-verify
	if !noVerify {
		v.Println("Testing connection...")

		client, err := api.New(api.ClientConfig{
			URL:        cfg.URL,
			Email:      cfg.Email,
			APIToken:   cfg.APIToken,
			AuthMethod: cfg.AuthMethod,
			CloudID:    cfg.CloudID,
		})
		if err != nil {
			return fmt.Errorf("creating client: %w", err)
		}

		user, err := client.GetCurrentUser(ctx, "")
		if err != nil {
			v.Error("Connection failed: %v", err)
			v.Println("")
			v.Info("Check your credentials and try again")
			return fmt.Errorf("authentication failed")
		}

		v.Success("Connected to %s", cfg.URL)
		v.Success("Authenticated as %s (%s)", user.DisplayName, user.EmailAddress)
		v.Println("")
	}

	if result.affectsSibling {
		if !prompt.WantPrompt(opts.NonInteractive, opts.Stdin) {
			// §3.4: scripted ingress opted in to shared-store mutation by
			// passing --non-interactive; surface the sibling impact on
			// stderr for the audit trail but proceed with the save.
			v.Info("Saving credentials affects cfl (shared default section); proceeding under --non-interactive.")
		} else {
			var confirm bool
			if err := huh.NewConfirm().
				Title("Save will affect cfl").
				Description("These credentials are stored in shared `default` and used by both jtk and cfl. Continue?").
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

	// Save to shared credential store. Per-tool defaults always live in
	// the jtk section; credential edits go to the section detectAndReconcile
	// chose (default vs jtk override).
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
		v.Error("Recover by storing just the token (no need to re-run init): `jtk set-credential --ref atlassian-cli/default --key api_token --stdin --overwrite` (reads stdin; use --from-env VAR for env-driven setup).")
		return err
	}
	v.Success("Configuration saved to %s (token stored in the OS keyring)", sharedPath)

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

	v.Println("")
	v.Println("Try it out:")
	v.Println("  jtk me")
	v.Println("  jtk issues list --project <PROJECT>")

	if isBearer {
		v.Println("")
		v.Info("To switch back to basic auth later, run: jtk init --auth-method basic")
	}

	return nil
}

// requireNonInteractiveFields enforces the §3.4 fail-loud contract for
// scripted/CI runs of `jtk init`: any required value missing from the
// flag prefills (which already populated cfg) produces an error naming
// the first missing field, with the auth-mode shape baked into the
// message so the operator knows which flag set is required.
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
		return fmt.Errorf("--non-interactive: missing required value for --token-stdin or --token-from-env VAR (or pre-stage with `jtk set-credential --ref atlassian-cli/default --key api_token --stdin`)")
	}
	return nil
}
