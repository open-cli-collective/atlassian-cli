// Package configcmd provides CLI commands for managing jtk configuration.
package configcmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
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
		RunE: func(_ *cobra.Command, _ []string) error {
			url := config.GetURL()
			email := config.GetEmail()
			token := config.GetAPIToken()
			defaultProject := config.GetDefaultProject()
			authMethod := config.GetAuthMethod()
			cloudID := config.GetCloudID()

			maskedToken := maskToken(token)

			// JSON output
			if opts.Output == "json" {
				data := map[string]string{
					"url":             url,
					"email":           email,
					"api_token":       maskedToken,
					"default_project": defaultProject,
					"auth_method":     authMethod,
					"cloud_id":        cloudID,
					"path":            config.Path(),
				}
				return opts.View().JSON(data)
			}

			// Text output
			_, authMethodSource := config.GetAuthMethodWithSource()
			_, cloudIDSource := config.GetCloudIDWithSource()

			entries := []jtkpresent.ConfigEntry{
				{Key: "url", Value: url, Source: getURLSource()},
				{Key: "email", Value: email, Source: getEmailSource()},
				{Key: "api_token", Value: maskedToken, Source: getAPITokenSource()},
				{Key: "default_project", Value: defaultProject, Source: getDefaultProjectSource()},
				{Key: "auth_method", Value: authMethod, Source: authMethodSource},
				{Key: "cloud_id", Value: cloudID, Source: cloudIDSource},
			}

			model := jtkpresent.ConfigPresenter{}.PresentConfigWithPath(entries, config.Path())
			out := present.Render(model, opts.RenderStyle())
			fmt.Fprint(opts.Stdout, out.Stdout)
			fmt.Fprint(opts.Stderr, out.Stderr)
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runClear(cmd.Context(), clearOpts)
		},
	}

	cmd.Flags().BoolVarP(&clearOpts.force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runClear(ctx context.Context, opts *clearOptions) error {
	_ = ctx
	configPath := config.Path()

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		model := jtkpresent.ConfigPresenter{}.PresentNoConfig(configPath)
		out := present.Render(model, opts.RenderStyle())
		fmt.Fprint(opts.Stdout, out.Stdout)
		fmt.Fprint(opts.Stderr, out.Stderr)
		return nil
	}

	// Confirm unless --force
	if !opts.force {
		fmt.Fprintf(opts.Stderr, "This will remove: %s\n", configPath)
		fmt.Fprint(opts.Stderr, "Are you sure? [y/N]: ")

		var response string
		_, err := fmt.Fscanln(opts.stdin, &response)
		if err != nil && err.Error() != "unexpected newline" {
			return err
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			cancelModel := jtkpresent.ConfigPresenter{}.PresentClearCancelled()
			cancelOut := present.Render(cancelModel, opts.RenderStyle())
			fmt.Fprint(opts.Stdout, cancelOut.Stdout)
			fmt.Fprint(opts.Stderr, cancelOut.Stderr)
			return nil
		}
	}

	if err := config.Clear(); err != nil {
		return err
	}

	// Check for active environment variables
	var envVars []string
	if os.Getenv("JIRA_URL") != "" || os.Getenv("ATLASSIAN_URL") != "" {
		envVars = append(envVars, "URL")
	}
	if os.Getenv("JIRA_EMAIL") != "" || os.Getenv("ATLASSIAN_EMAIL") != "" {
		envVars = append(envVars, "Email")
	}
	if os.Getenv("JIRA_API_TOKEN") != "" || os.Getenv("ATLASSIAN_API_TOKEN") != "" {
		envVars = append(envVars, "API Token")
	}

	model := jtkpresent.ConfigPresenter{}.PresentClearedWithEnvVars(configPath, envVars)
	out := present.Render(model, opts.RenderStyle())
	fmt.Fprint(opts.Stdout, out.Stdout)
	fmt.Fprint(opts.Stderr, out.Stderr)

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
		RunE: func(cmd *cobra.Command, _ []string) error {
			result := jtkpresent.TestResult{URL: config.GetURL()}

			if result.URL != "" {
				client, err := opts.APIClient()
				if err != nil {
					result.ClientError = err
				} else {
					user, err := client.GetCurrentUser(cmd.Context())
					if err != nil {
						result.AuthError = err
					} else {
						result.User = user
					}
				}
			}

			model := jtkpresent.ConfigPresenter{}.PresentTestResult(result)
			out := present.Render(model, opts.RenderStyle())
			fmt.Fprint(opts.Stdout, out.Stdout)
			fmt.Fprint(opts.Stderr, out.Stderr)
			return nil
		},
	}
}
