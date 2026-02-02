package initcmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	sharedurl "github.com/open-cli-collective/atlassian-go/url"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
)

// Register registers the init command
func Register(parent *cobra.Command, opts *root.Options) {
	var url, email, token string
	var noVerify bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize jtk with guided setup",
		Long: `Interactive setup wizard for configuring jtk.

Prompts for your Jira URL, email, and API token, then verifies
the connection before saving the configuration.

Get your API token from: https://id.atlassian.com/manage-profile/security/api-tokens`,
		Example: `  # Interactive setup
  jtk init

  # Non-interactive setup
  jtk init --url https://mycompany.atlassian.net --email user@example.com --token YOUR_TOKEN

  # Skip connection verification
  jtk init --no-verify`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(opts, url, email, token, noVerify)
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "Jira URL (e.g., https://mycompany.atlassian.net)")
	cmd.Flags().StringVar(&email, "email", "", "Email address for authentication")
	cmd.Flags().StringVar(&token, "token", "", "API token")
	cmd.Flags().BoolVar(&noVerify, "no-verify", false, "Skip connection verification")

	parent.AddCommand(cmd)
}

func runInit(opts *root.Options, prefillURL, prefillEmail, prefillToken string, noVerify bool) error {
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

	// Build the form
	form := huh.NewForm(
		huh.NewGroup(
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
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	// Normalize URL
	cfg.URL = sharedurl.NormalizeURL(cfg.URL)

	// Verify connection unless --no-verify
	if !noVerify {
		v.Println("Testing connection...")

		client, err := api.New(api.ClientConfig{
			URL:      cfg.URL,
			Email:    cfg.Email,
			APIToken: cfg.APIToken,
		})
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		user, err := client.GetCurrentUser()
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
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	v.Success("Configuration saved to %s", configPath)
	v.Println("")
	v.Println("Try it out:")
	v.Println("  jtk me")
	v.Println("  jtk issues list --project <PROJECT>")

	return nil
}
