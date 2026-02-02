package configcmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

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

	cmd.AddCommand(newShowCmd(opts))
	cmd.AddCommand(newClearCmd(opts))
	cmd.AddCommand(newTestCmd(opts))

	parent.AddCommand(cmd)
}

func maskToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return "********"
	}
	return token[:4] + "********" + token[len(token)-4:]
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
			defaultProject := config.GetDefaultProject()

			maskedToken := maskToken(token)

			headers := []string{"KEY", "VALUE", "SOURCE"}
			rows := [][]string{
				{"url", url, getURLSource()},
				{"email", email, getEmailSource()},
				{"api_token", maskedToken, getAPITokenSource()},
				{"default_project", defaultProject, getDefaultProjectSource()},
			}

			data := map[string]string{
				"url":             url,
				"email":           email,
				"api_token":       maskedToken,
				"default_project": defaultProject,
				"path":            config.Path(),
			}

			if err := v.Render(headers, rows, data); err != nil {
				return err
			}

			v.Info("\nConfig file: %s", config.Path())
			return nil
		},
	}
}

type clearOptions struct {
	*root.Options
	force bool
	stdin io.Reader // For testing
}

func newClearCmd(opts *root.Options) *cobra.Command {
	clearOpts := &clearOptions{
		Options: opts,
		stdin:   os.Stdin,
	}

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear stored configuration",
		Long: `Remove the stored configuration file.

Note: Environment variables (JIRA_*, ATLASSIAN_*) will still be used if set.`,
		Example: `  # Clear configuration (with confirmation)
  jtk config clear

  # Clear without confirmation
  jtk config clear --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClear(clearOpts)
		},
	}

	cmd.Flags().BoolVarP(&clearOpts.force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runClear(opts *clearOptions) error {
	v := opts.View()
	configPath := config.Path()

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		v.Info("No configuration file found at %s", configPath)
		return nil
	}

	// Confirm unless --force
	if !opts.force {
		fmt.Printf("This will remove: %s\n", configPath)
		fmt.Print("Are you sure? [y/N]: ")

		var response string
		_, err := fmt.Fscanln(opts.stdin, &response)
		if err != nil && err.Error() != "unexpected newline" {
			return err
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			v.Info("Cancelled.")
			return nil
		}
	}

	if err := config.Clear(); err != nil {
		return err
	}

	v.Success("Configuration file removed: %s", configPath)

	// Check for active environment variables
	envVars := []string{}
	if os.Getenv("JIRA_URL") != "" || os.Getenv("ATLASSIAN_URL") != "" {
		envVars = append(envVars, "URL")
	}
	if os.Getenv("JIRA_EMAIL") != "" || os.Getenv("ATLASSIAN_EMAIL") != "" {
		envVars = append(envVars, "Email")
	}
	if os.Getenv("JIRA_API_TOKEN") != "" || os.Getenv("ATLASSIAN_API_TOKEN") != "" {
		envVars = append(envVars, "API Token")
	}

	if len(envVars) > 0 {
		fmt.Println()
		fmt.Printf("Note: The following are still configured via environment variables: %s\n",
			strings.Join(envVars, ", "))
		fmt.Println("These will continue to be used. Unset them if you want to fully clear configuration.")
	}

	return nil
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

func getDefaultProjectSource() string {
	if os.Getenv("JIRA_DEFAULT_PROJECT") != "" {
		return "env (JIRA_DEFAULT_PROJECT)"
	}
	cfg, err := config.Load()
	if err != nil {
		return "-"
	}
	if cfg.DefaultProject != "" {
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
