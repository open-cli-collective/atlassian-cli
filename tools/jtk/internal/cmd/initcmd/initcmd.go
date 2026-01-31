package initcmd

import (
	"bufio"
	"fmt"
	"strings"

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

func runInit(opts *root.Options, url, email, token string, noVerify bool) error {
	v := opts.View()
	reader := bufio.NewReader(opts.Stdin)

	v.Println("Jira CLI Setup")
	v.Println("")

	// Check for existing config
	existingCfg, _ := config.Load()
	if existingCfg.URL != "" || existingCfg.Email != "" || existingCfg.APIToken != "" {
		v.Warning("Existing configuration found at %s", config.Path())
		v.Println("")

		overwrite, err := promptYesNo(reader, "Overwrite existing configuration?", false)
		if err != nil {
			return err
		}
		if !overwrite {
			v.Info("Setup cancelled")
			return nil
		}
		v.Println("")
	}

	// Prompt for URL if not provided
	if url == "" {
		v.Println("Enter your Jira URL")
		v.Println("  Examples: https://mycompany.atlassian.net")
		v.Println("            https://jira.internal.corp.com")
		v.Println("")

		var err error
		url, err = promptRequired(reader, "URL")
		if err != nil {
			return err
		}
	}
	url = sharedurl.NormalizeURL(url)

	// Prompt for email if not provided
	if email == "" {
		v.Println("")
		var err error
		email, err = promptRequired(reader, "Email")
		if err != nil {
			return err
		}
	}

	// Prompt for token if not provided
	if token == "" {
		v.Println("")
		v.Println("Get your API token from:")
		v.Println("  https://id.atlassian.com/manage-profile/security/api-tokens")
		v.Println("")

		var err error
		token, err = promptRequired(reader, "API Token")
		if err != nil {
			return err
		}
	}

	v.Println("")

	// Verify connection unless --no-verify
	if !noVerify {
		v.Println("Testing connection...")

		client, err := api.New(api.ClientConfig{
			URL:      url,
			Email:    email,
			APIToken: token,
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

		v.Success("Connected to %s", url)
		v.Success("Authenticated as %s (%s)", user.DisplayName, user.EmailAddress)
		v.Println("")
	}

	// Save configuration
	cfg := &config.Config{
		URL:      url,
		Email:    email,
		APIToken: token,
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	v.Success("Configuration saved to %s", config.Path())
	v.Println("")
	v.Println("Try it out:")
	v.Println("  jtk me")
	v.Println("  jtk issues list --project <PROJECT>")

	return nil
}

func promptRequired(reader *bufio.Reader, label string) (string, error) {
	for {
		fmt.Printf("%s: ", label)
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		input = strings.TrimSpace(input)
		if input != "" {
			return input, nil
		}
		fmt.Printf("  %s is required\n", label)
	}
}

func promptYesNo(reader *bufio.Reader, question string, defaultYes bool) (bool, error) {
	suffix := " [y/N]: "
	if defaultYes {
		suffix = " [Y/n]: "
	}

	fmt.Print(question + suffix)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		return defaultYes, nil
	}
	return input == "y" || input == "yes", nil
}
