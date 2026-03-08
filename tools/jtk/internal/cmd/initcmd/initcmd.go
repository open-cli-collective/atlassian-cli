// Package initcmd provides the interactive setup wizard for the jtk CLI.
package initcmd

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/auth"
	sharedurl "github.com/open-cli-collective/atlassian-go/url"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
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
	configPath := config.Path()

	// Load existing config for pre-population
	existingCfg, _ := config.Load()

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

	// Initialize config with pre-filled values
	// Priority: CLI flag > existing config value
	cfg := &config.Config{}

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

	if prefillToken != "" {
		cfg.APIToken = prefillToken
	} else if existingCfg.APIToken != "" {
		cfg.APIToken = existingCfg.APIToken
	}

	if existingCfg.DefaultProject != "" {
		cfg.DefaultProject = existingCfg.DefaultProject
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

		user, err := client.GetCurrentUser(ctx)
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

	// Save configuration
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving configuration: %w", err)
	}

	v.Success("Configuration saved to %s", configPath)
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
