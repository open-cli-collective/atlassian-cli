package configcmd

import (
	"os"

	"github.com/spf13/cobra"

	sharedurl "github.com/open-cli-collective/atlassian-go/url"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
)

// Register registers the config commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
		Long:  "Commands for managing jtk configuration and credentials.",
	}

	cmd.AddCommand(newSetCmd(opts))
	cmd.AddCommand(newShowCmd(opts))
	cmd.AddCommand(newClearCmd(opts))
	cmd.AddCommand(newTestCmd(opts))

	parent.AddCommand(cmd)
}

func newSetCmd(opts *root.Options) *cobra.Command {
	var url, email, token string

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set configuration values",
		Long:  "Set Jira credentials. All values are required.",
		Example: `  # Set all credentials (Jira Cloud)
  jtk config set --url https://mycompany.atlassian.net --email user@example.com --token YOUR_API_TOKEN

  # Self-hosted Jira
  jtk config set --url https://jira.internal.corp.com --email user@example.com --token YOUR_API_TOKEN

  # Using environment variables instead
  export JIRA_URL=https://mycompany.atlassian.net
  export JIRA_EMAIL=user@example.com
  export JIRA_API_TOKEN=YOUR_API_TOKEN`,
		RunE: func(cmd *cobra.Command, args []string) error {
			v := opts.View()

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if url != "" {
				cfg.URL = sharedurl.NormalizeURL(url)
				cfg.Domain = "" // Clear deprecated field when URL is set
			}
			if email != "" {
				cfg.Email = email
			}
			if token != "" {
				cfg.APIToken = token
			}

			if err := config.Save(cfg); err != nil {
				return err
			}

			v.Success("Configuration saved to %s", config.Path())
			return nil
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "Jira URL (e.g., 'https://mycompany.atlassian.net' or 'https://jira.internal.corp.com')")
	cmd.Flags().StringVar(&email, "email", "", "Email address for authentication")
	cmd.Flags().StringVar(&token, "token", "", "API token (create at https://id.atlassian.com/manage-profile/security/api-tokens)")

	return cmd
}

func newShowCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long:  "Display the current configuration values (token is masked).",
		RunE: func(cmd *cobra.Command, args []string) error {
			v := opts.View()

			url := config.GetURL()
			email := config.GetEmail()
			token := config.GetAPIToken()

			// Mask the token
			maskedToken := ""
			if token != "" {
				if len(token) > 8 {
					maskedToken = token[:4] + "..." + token[len(token)-4:]
				} else {
					maskedToken = "****"
				}
			}

			headers := []string{"KEY", "VALUE", "SOURCE"}
			rows := [][]string{
				{"url", url, getURLSource()},
				{"email", email, getEmailSource()},
				{"api_token", maskedToken, getAPITokenSource()},
			}

			data := map[string]string{
				"url":       url,
				"email":     email,
				"api_token": maskedToken,
				"path":      config.Path(),
			}

			if err := v.Render(headers, rows, data); err != nil {
				return err
			}

			v.Info("\nConfig file: %s", config.Path())
			return nil
		},
	}
}

func newClearCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Clear stored configuration",
		Long:  "Remove the stored configuration file. Environment variables will still work.",
		RunE: func(cmd *cobra.Command, args []string) error {
			v := opts.View()

			if err := config.Clear(); err != nil {
				return err
			}

			v.Success("Configuration cleared")
			return nil
		},
	}
}

func getURLSource() string {
	if os.Getenv("JIRA_URL") != "" {
		return "env (JIRA_URL)"
	}
	if os.Getenv("ATLASSIAN_URL") != "" {
		return "env (ATLASSIAN_URL)"
	}
	cfg, err := config.Load()
	if err != nil {
		return "-"
	}
	if cfg.URL != "" {
		return "config"
	}
	// Check legacy domain sources
	if os.Getenv("JIRA_DOMAIN") != "" {
		return "env (JIRA_DOMAIN, deprecated)"
	}
	if cfg.Domain != "" {
		return "config (domain, deprecated)"
	}
	return "-"
}

func getEmailSource() string {
	if os.Getenv("JIRA_EMAIL") != "" {
		return "env (JIRA_EMAIL)"
	}
	if os.Getenv("ATLASSIAN_EMAIL") != "" {
		return "env (ATLASSIAN_EMAIL)"
	}
	cfg, err := config.Load()
	if err != nil {
		return "-"
	}
	if cfg.Email != "" {
		return "config"
	}
	return "-"
}

func getAPITokenSource() string {
	if os.Getenv("JIRA_API_TOKEN") != "" {
		return "env (JIRA_API_TOKEN)"
	}
	if os.Getenv("ATLASSIAN_API_TOKEN") != "" {
		return "env (ATLASSIAN_API_TOKEN)"
	}
	cfg, err := config.Load()
	if err != nil {
		return "-"
	}
	if cfg.APIToken != "" {
		return "config"
	}
	return "-"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			v := opts.View()

			url := config.GetURL()
			if url == "" {
				v.Error("No Jira URL configured")
				v.Println("")
				v.Info("Configure with: jtk init")
				v.Info("Or set environment variable: JIRA_URL")
				return nil
			}

			v.Println("Testing connection to %s...", url)
			v.Println("")

			client, err := opts.APIClient()
			if err != nil {
				v.Error("Failed to create client: %v", err)
				v.Println("")
				v.Info("Check your configuration with: jtk config show")
				v.Info("Reconfigure with: jtk init")
				return nil
			}

			user, err := client.GetCurrentUser()
			if err != nil {
				v.Error("Authentication failed: %v", err)
				v.Println("")
				v.Info("Check your credentials with: jtk config show")
				v.Info("Reconfigure with: jtk init")
				return nil
			}

			v.Success("Authentication successful")
			v.Success("API access verified")
			v.Println("")
			v.Println("Authenticated as: %s (%s)", user.DisplayName, user.EmailAddress)
			v.Println("Account ID: %s", user.AccountID)

			return nil
		},
	}
}
