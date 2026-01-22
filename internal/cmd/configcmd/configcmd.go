package configcmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
)

// Register registers the config commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
		Long:  "Commands for managing jira-ticket-cli configuration and credentials.",
	}

	cmd.AddCommand(newSetCmd(opts))
	cmd.AddCommand(newShowCmd(opts))
	cmd.AddCommand(newClearCmd(opts))

	parent.AddCommand(cmd)
}

func newSetCmd(opts *root.Options) *cobra.Command {
	var domain, email, token string

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set configuration values",
		Long:  "Set Jira credentials. All values are required.",
		Example: `  # Set all credentials
  jira-ticket-cli config set --domain mycompany --email user@example.com --token YOUR_API_TOKEN

  # Using environment variables instead
  export JIRA_DOMAIN=mycompany
  export JIRA_EMAIL=user@example.com
  export JIRA_API_TOKEN=YOUR_API_TOKEN`,
		RunE: func(cmd *cobra.Command, args []string) error {
			v := opts.View()

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if domain != "" {
				cfg.Domain = domain
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

	cmd.Flags().StringVar(&domain, "domain", "", "Jira domain (e.g., 'mycompany' for mycompany.atlassian.net)")
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

			domain := config.GetDomain()
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
				{"domain", domain, getSource("JIRA_DOMAIN", domain)},
				{"email", email, getSource("JIRA_EMAIL", email)},
				{"api_token", maskedToken, getSource("JIRA_API_TOKEN", token)},
			}

			data := map[string]string{
				"domain":    domain,
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

func getSource(envVar, value string) string {
	if value == "" {
		return "-"
	}
	if os.Getenv(envVar) != "" {
		return "env"
	}
	return "config"
}
