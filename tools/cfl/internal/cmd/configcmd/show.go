package configcmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	sharedconfig "github.com/open-cli-collective/atlassian-go/config"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	"github.com/open-cli-collective/confluence-cli/internal/config"
)

func newShowCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long: `Display the current cfl configuration with masked credentials.

Shows the source of each value (environment variable, config file, or not set).`,
		Example: `  # Show current configuration
  cfl config show`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runShow(opts)
		},
	}
}

func runShow(opts *root.Options) error {
	configPath := config.DefaultConfigPath()
	v := opts.View()

	// Load config file (if exists)
	fileCfg, fileErr := config.Load(configPath)
	if fileErr != nil {
		fileCfg = &config.Config{}
	}

	// Check environment variables
	envURL := sharedconfig.GetEnvWithFallback("CFL_URL", "ATLASSIAN_URL")
	envEmail := sharedconfig.GetEnvWithFallback("CFL_EMAIL", "ATLASSIAN_EMAIL")
	envToken := sharedconfig.GetEnvWithFallback("CFL_API_TOKEN", "ATLASSIAN_API_TOKEN")
	envSpace := os.Getenv("CFL_DEFAULT_SPACE")

	// Determine effective values and sources
	url, urlSource := getValueAndSource(envURL, fileCfg.URL, getEnvVarName("CFL_URL", "ATLASSIAN_URL"))
	email, emailSource := getValueAndSource(envEmail, fileCfg.Email, getEnvVarName("CFL_EMAIL", "ATLASSIAN_EMAIL"))
	token, tokenSource := getValueAndSource(envToken, fileCfg.APIToken, getEnvVarName("CFL_API_TOKEN", "ATLASSIAN_API_TOKEN"))
	space, spaceSource := getValueAndSource(envSpace, fileCfg.DefaultSpace, "CFL_DEFAULT_SPACE")

	// Display
	v.RenderKeyValue("URL", formatValueWithSource(url, urlSource))
	v.RenderKeyValue("Email", formatValueWithSource(email, emailSource))
	v.RenderKeyValue("API Token", formatValueWithSource(maskToken(token), tokenSource))
	v.RenderKeyValue("Default Space", formatValueWithSource(space, spaceSource))

	fmt.Println()
	fmt.Printf("Config file: %s\n", configPath)
	if fileErr != nil {
		fmt.Printf("  (file not found or unreadable)\n")
	}

	return nil
}

// getValueAndSource returns the effective value and its source.
func getValueAndSource(envValue, fileValue, envVarName string) (string, string) {
	if envValue != "" {
		return envValue, envVarName
	}
	if fileValue != "" {
		return fileValue, "config"
	}
	return "", "not set"
}

// getEnvVarName returns the name of the environment variable that is set.
func getEnvVarName(primary, fallback string) string {
	if os.Getenv(primary) != "" {
		return primary
	}
	if os.Getenv(fallback) != "" {
		return fallback
	}
	return primary // Default to primary if neither is set
}

// formatValueWithSource formats a value with its source indicator.
func formatValueWithSource(value, source string) string {
	if value == "" {
		return fmt.Sprintf("(source: %s)", source)
	}
	return fmt.Sprintf("%s  (source: %s)", value, source)
}

// maskToken masks the API token for display, showing first 4 and last 4 chars.
func maskToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return "********"
	}
	return token[:4] + "********" + token[len(token)-4:]
}
