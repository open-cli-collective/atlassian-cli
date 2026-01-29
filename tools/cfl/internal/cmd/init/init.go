// Package init provides the init command for cfl.
package init

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/internal/config"
)

// NewCmdInit creates the init command.
func NewCmdInit() *cobra.Command {
	var (
		url      string
		email    string
		noVerify bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize cfl configuration",
		Long: `Initialize cfl with your Confluence Cloud credentials.

This command will guide you through setting up your Confluence URL,
email, and API token. The configuration will be saved to ~/.config/cfl/config.yml.

To generate an API token:
  1. Go to https://id.atlassian.com/manage-profile/security/api-tokens
  2. Click "Create API token"
  3. Copy the token (it won't be shown again)`,
		Example: `  # Interactive setup
  cfl init

  # Pre-populate URL
  cfl init --url https://mycompany.atlassian.net`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runInit(url, email, noVerify)
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "Confluence URL (e.g., https://mycompany.atlassian.net)")
	cmd.Flags().StringVar(&email, "email", "", "Your Atlassian account email")
	cmd.Flags().BoolVar(&noVerify, "no-verify", false, "Skip connection verification")

	return cmd
}

func runInit(prefillURL, prefillEmail string, noVerify bool) error {
	configPath := config.DefaultConfigPath()

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
			fmt.Println("Initialization cancelled.")
			return nil
		}
	}

	cfg := &config.Config{}

	// Use prefilled values or prompt
	if prefillURL != "" {
		cfg.URL = prefillURL
	}
	if prefillEmail != "" {
		cfg.Email = prefillEmail
	}

	// Build the form
	form := huh.NewForm(
		huh.NewGroup(
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
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	// Normalize URL
	cfg.NormalizeURL()

	// Validate
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Verify connection unless skipped
	if !noVerify {
		fmt.Print("Verifying connection... ")
		if err := verifyConnection(cfg); err != nil {
			fmt.Println("failed!")
			return fmt.Errorf("connection verification failed: %w", err)
		}
		fmt.Println("success!")
	}

	// Save configuration
	if err := cfg.Save(configPath); err != nil {
		return err
	}

	fmt.Printf("\nConfiguration saved to %s\n", configPath)
	fmt.Println("\nYou're all set! Try running:")
	fmt.Println("  cfl space list")
	fmt.Println("  cfl page list --space <SPACE_KEY>")

	return nil
}

func verifyConnection(cfg *config.Config) error {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", cfg.URL+"/api/v2/spaces?limit=1", nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(cfg.Email, cfg.APIToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 401 {
		return fmt.Errorf("authentication failed - check your email and API token")
	}
	if resp.StatusCode == 403 {
		return fmt.Errorf("access denied - check your permissions")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
