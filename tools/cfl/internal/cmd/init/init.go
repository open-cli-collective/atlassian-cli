// Package init provides the init command for cfl.
package init

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/auth"

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

	configPath := config.DefaultConfigPath()

	// Load existing config for pre-population
	existingCfg, _ := config.Load(configPath)
	if existingCfg == nil {
		existingCfg = &config.Config{}
	}

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		var overwrite bool
		err := huh.NewConfirm().
			Title("Configuration already exists").
			Description(fmt.Sprintf("Overwrite %s?", configPath)).
			Value(&overwrite).
			Run()
		if err != nil {
			return err
		}
		if !overwrite {
			v.Info("Initialization cancelled.")
			return nil
		}
	}

	cfg := &config.Config{}

	// Pre-fill from existing config, then override with CLI flags
	// Priority: CLI flag > existing config value
	if prefillURL != "" {
		cfg.URL = prefillURL
	} else if existingCfg.URL != "" {
		cfg.URL = existingCfg.URL
	}

	if prefillEmail != "" {
		cfg.Email = prefillEmail
	} else if existingCfg.Email != "" {
		cfg.Email = existingCfg.Email
	}

	if existingCfg.APIToken != "" {
		cfg.APIToken = existingCfg.APIToken
	}

	if existingCfg.DefaultSpace != "" {
		cfg.DefaultSpace = existingCfg.DefaultSpace
	}

	if prefillAuthMethod != "" {
		cfg.AuthMethod = prefillAuthMethod
	} else if existingCfg.AuthMethod != "" {
		cfg.AuthMethod = existingCfg.AuthMethod
	}

	if prefillCloudID != "" {
		cfg.CloudID = prefillCloudID
	} else if existingCfg.CloudID != "" {
		cfg.CloudID = existingCfg.CloudID
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

	form := huh.NewForm(formGroups...)

	if err := form.Run(); err != nil {
		return err
	}

	cfg.NormalizeURL()

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return finalizeInit(ctx, opts, cfg, configPath, noVerify, defaultClientBuilder)
}

// finalizeInit runs the verify/save/render pipeline after the form has produced
// a normalized + validated config. Extracted as a non-interactive seam so tests
// can supply an httptest-backed clientBuilder and a temp configPath.
func finalizeInit(
	ctx context.Context,
	opts *root.Options,
	cfg *config.Config,
	configPath string,
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

	if err := cfg.Save(configPath); err != nil {
		return err
	}

	v.Success("Configuration saved to %s", configPath)

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
