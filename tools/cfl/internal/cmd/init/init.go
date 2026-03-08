// Package init provides the init command for cfl.
package init

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/auth"
	"github.com/open-cli-collective/atlassian-go/client"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	"github.com/open-cli-collective/confluence-cli/internal/config"
)

// Register adds the init command to the root command.
func Register(rootCmd *cobra.Command, _ *root.Options) {
	rootCmd.AddCommand(newInitCmd())
}

// newInitCmd creates the init command.
func newInitCmd() *cobra.Command {
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
			return runInit(cmd.Context(), url, email, authMethod, cloudID, noVerify)
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "Confluence URL (e.g., https://mycompany.atlassian.net)")
	cmd.Flags().StringVar(&email, "email", "", "Your Atlassian account email")
	cmd.Flags().StringVar(&authMethod, "auth-method", "", "Authentication method: basic (default) or bearer")
	cmd.Flags().StringVar(&cloudID, "cloud-id", "", "Atlassian Cloud ID (required for bearer auth)")
	cmd.Flags().BoolVar(&noVerify, "no-verify", false, "Skip connection verification")

	return cmd
}

func runInit(ctx context.Context, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID string, noVerify bool) error {
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
			_, _ = fmt.Fprintln(os.Stderr, "Initialization cancelled.")
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

	// Normalize URL
	cfg.NormalizeURL()

	// Validate
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Verify connection unless skipped
	if !noVerify {
		_, _ = fmt.Fprint(os.Stderr, "Verifying connection... ")
		if err := verifyConnection(ctx, cfg); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "failed!")
			return fmt.Errorf("verifying connection: %w", err)
		}
		_, _ = fmt.Fprintln(os.Stderr, "success!")
	}

	// Save configuration
	if err := cfg.Save(configPath); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(os.Stderr, "\nConfiguration saved to %s\n", configPath)
	_, _ = fmt.Fprintln(os.Stderr, "\nYou're all set! Try running:")
	_, _ = fmt.Fprintln(os.Stderr, "  cfl space list")
	_, _ = fmt.Fprintln(os.Stderr, "  cfl page list --space <SPACE_KEY>")

	if isBearer {
		_, _ = fmt.Fprintln(os.Stderr, "")
		_, _ = fmt.Fprintln(os.Stderr, "To switch back to basic auth later, run: cfl init --auth-method basic")
	}

	return nil
}

func verifyConnection(ctx context.Context, cfg *config.Config) error {
	httpClient := &http.Client{Timeout: 10 * time.Second}

	var verifyURL string

	if cfg.AuthMethod == auth.AuthMethodBearer {
		if cfg.CloudID == "" {
			return fmt.Errorf("cloud ID is required for bearer auth connection verification")
		}
		// Bearer auth: use API gateway
		verifyURL = fmt.Sprintf("%s/ex/confluence/%s/wiki/api/v2/spaces?limit=1", client.GatewayBaseURL, cfg.CloudID)
	} else {
		// Basic auth: use instance URL
		verifyURL = cfg.URL + "/api/v2/spaces?limit=1"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", verifyURL, nil)
	if err != nil {
		return err
	}

	if cfg.AuthMethod == auth.AuthMethodBearer {
		req.Header.Set("Authorization", auth.BearerAuthHeader(cfg.APIToken))
	} else {
		req.SetBasicAuth(cfg.Email, cfg.APIToken)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 401 {
		if cfg.AuthMethod == auth.AuthMethodBearer {
			return fmt.Errorf("authentication failed - check your API token and cloud ID")
		}
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
