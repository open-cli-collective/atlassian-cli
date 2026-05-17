package configcmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	sharedconfig "github.com/open-cli-collective/atlassian-go/config"
	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/keyring"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	"github.com/open-cli-collective/confluence-cli/internal/config"
)

func newShowCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long: `Display the current cfl configuration.

The API token value is never displayed — only whether one is configured,
where it resolves from, and the OS keyring backend in use. Token/keyring
reporting is authoritative.

Note: the non-secret rows (URL, email, etc.) reflect environment
variables and the legacy per-tool file ONLY. A value set solely in the
shared store at ~/.config/atlassian-cli/config.yml is shown as "not set"
here even though the tool resolves and uses it at runtime — run a real
command to confirm effective configuration.`,
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
	envSpace := os.Getenv("CFL_DEFAULT_SPACE")
	envAuthMethod := sharedconfig.GetEnvWithFallback("CFL_AUTH_METHOD", "ATLASSIAN_AUTH_METHOD")
	envCloudID := sharedconfig.GetEnvWithFallback("CFL_CLOUD_ID", "ATLASSIAN_CLOUD_ID")

	// Determine effective values and sources
	url, urlSource := getValueAndSource(envURL, fileCfg.URL, getEnvVarName("CFL_URL", "ATLASSIAN_URL"))
	email, emailSource := getValueAndSource(envEmail, fileCfg.Email, getEnvVarName("CFL_EMAIL", "ATLASSIAN_EMAIL"))
	space, spaceSource := getValueAndSource(envSpace, fileCfg.DefaultSpace, "CFL_DEFAULT_SPACE")
	authMethod, authMethodSource := getValueAndSource(envAuthMethod, fileCfg.AuthMethod, getEnvVarName("CFL_AUTH_METHOD", "ATLASSIAN_AUTH_METHOD"))
	cloudID, cloudIDSource := getValueAndSource(envCloudID, fileCfg.CloudID, getEnvVarName("CFL_CLOUD_ID", "ATLASSIAN_CLOUD_ID"))

	// Default auth method display
	if authMethod == "" {
		authMethod = "basic"
		authMethodSource = "default"
	}

	// Non-secret keyring description (non-migrating: show stays usable
	// during an unresolved §1.8 conflict). The token VALUE is never
	// shown — presence + source only (§1.12).
	kr, krErr := keyring.InspectForTool(credstore.ToolCFL)
	tokenStatus := "not set"
	if kr.TokenConfigured {
		tokenStatus = "configured"
	}
	tokenSource := kr.TokenSource
	if krErr != nil {
		tokenSource = "keyring error: " + krErr.Error()
	}

	// Display
	v.RenderKeyValue("URL", formatValueWithSource(url, urlSource))
	v.RenderKeyValue("Email", formatValueWithSource(email, emailSource))
	v.RenderKeyValue("API Token", formatValueWithSource(tokenStatus, tokenSource))
	v.RenderKeyValue("Default Space", formatValueWithSource(space, spaceSource))
	v.RenderKeyValue("Auth Method", formatValueWithSource(authMethod, authMethodSource))
	v.RenderKeyValue("Cloud ID", formatValueWithSource(cloudID, cloudIDSource))
	v.RenderKeyValue("Keyring Ref", formatValueWithSource(kr.Ref, "fixed"))
	if kr.Backend != "" {
		backend := kr.Backend
		if kr.BackendSource != "" {
			backend += " (" + kr.BackendSource + ")"
		}
		v.RenderKeyValue("Keyring Backend", formatValueWithSource(backend, "-"))
	}
	if kr.PassphraseSource != "" {
		v.RenderKeyValue("Keyring Passphrase", formatValueWithSource(kr.PassphraseSource, "-"))
	}

	_, _ = fmt.Fprintln(opts.Stderr)
	_, _ = fmt.Fprintf(opts.Stderr, "Config file: %s\n", configPath)
	if fileErr != nil {
		_, _ = fmt.Fprintf(opts.Stderr, "  (file not found or unreadable)\n")
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
