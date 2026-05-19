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
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/auth"
	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/keyring"
	sharedurl "github.com/open-cli-collective/atlassian-go/url"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// Register registers the init command
func Register(parent *cobra.Command, opts *root.Options) {
	var url, email, token, authMethod, cloudID string
	var noVerify bool

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
  Find your Cloud ID at: https://your-site.atlassian.net/_edge/tenant_info`,
		Example: `  # Interactive setup (basic auth)
  jtk init

  # Non-interactive basic auth setup
  jtk init --url https://mycompany.atlassian.net --email user@example.com --token YOUR_TOKEN

  # Service account (bearer auth) setup
  jtk init --auth-method bearer --url https://mycompany.atlassian.net --token SCOPED_TOKEN --cloud-id YOUR_CLOUD_ID

  # Skip connection verification
  jtk init --no-verify`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(cmd.Context(), opts, url, email, token, authMethod, cloudID, noVerify)
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "Jira URL (e.g., https://mycompany.atlassian.net)")
	cmd.Flags().StringVar(&email, "email", "", "Email address for authentication")
	cmd.Flags().StringVar(&token, "token", "", "API token")
	cmd.Flags().StringVar(&authMethod, "auth-method", "", "Authentication method: basic (default) or bearer")
	cmd.Flags().StringVar(&cloudID, "cloud-id", "", "Atlassian Cloud ID (required for bearer auth)")
	cmd.Flags().BoolVar(&noVerify, "no-verify", false, "Skip connection verification")

	parent.AddCommand(cmd)
}

func runInit(ctx context.Context, opts *root.Options, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID string, noVerify bool) error {
	// Validate --auth-method flag early, before any interactive prompts
	if prefillAuthMethod != "" {
		if err := auth.ValidateAuthMethod(prefillAuthMethod); err != nil {
			return err
		}
	}

	v := opts.View()

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

	form := huh.NewForm(formGroups...)

	if err := form.Run(); err != nil {
		return err
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
		v.Error("Recover by storing just the token (no need to re-run init): `jtk set-credential` (reads stdin or --from-env VAR).")
		return err
	}
	v.Success("Configuration saved to %s (token stored in the OS keyring)", sharedPath)

	for _, lp := range result.consumedLegacies {
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
