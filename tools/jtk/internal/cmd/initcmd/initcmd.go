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

	// Run the one-time §1.8 migration up front so a pre-existing legacy
	// plaintext token is relocated (and scrubbed) into the keyring before
	// the user sets a new one — otherwise it could collide later.
	if err := keyring.EnsureMigrated(); err != nil {
		v.Error("Could not prepare secure credential storage: %v", err)
		return err
	}

	sharedPath := credstore.DefaultPath()
	jtkLegacyPath := credstore.LegacyJTKPath()
	cflLegacyPath := credstore.LegacyCFLPath()

	result, err := detectAndReconcile(v, jtkLegacyPath, cflLegacyPath, sharedPath,
		prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)
	if err != nil {
		return err
	}
	cfg := result.prefill

	// The up-front EnsureMigrated relocates any legacy plaintext token into
	// the single shared keyring api_token and scrubs the file, so
	// detectAndReconcile (which reads the now-scrubbed legacy/config files)
	// leaves prefill.APIToken empty even though the token still exists.
	// Backfill it from the keyring so a returning user isn't forced to
	// re-enter a token that was just migrated. NoMigrate: migration already
	// ran above. Value stays password-masked in the form (same ingress as
	// before); never displayed.
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
	applyResultToStore(result.store, cfg, result.target)
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
