package configcmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	sharedconfig "github.com/open-cli-collective/atlassian-go/config"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	"github.com/open-cli-collective/confluence-cli/internal/config"
)

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
		Long: `Remove the stored cfl configuration file.

This will delete ~/.config/cfl/config.yml. Environment variables (CFL_*, ATLASSIAN_*)
will continue to work even after clearing the config file.`,
		Example: `  # Clear configuration (with confirmation)
  cfl config clear

  # Clear without confirmation
  cfl config clear --force`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runClear(clearOpts)
		},
	}

	cmd.Flags().BoolVarP(&clearOpts.force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runClear(opts *clearOptions) error {
	configPath := config.DefaultConfigPath()

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		_, _ = fmt.Fprintf(opts.Stderr, "No configuration file found at %s\n", configPath)
		return nil
	}

	// Confirm unless --force
	if !opts.force {
		_, _ = fmt.Fprintf(opts.Stderr, "This will remove: %s\n", configPath)
		_, _ = fmt.Fprint(opts.Stderr, "Are you sure? [y/N]: ")

		var response string
		_, err := fmt.Fscanln(opts.stdin, &response)
		if err != nil && err.Error() != "unexpected newline" {
			return err
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			_, _ = fmt.Fprintln(opts.Stderr, "Cancelled.")
			return nil
		}
	}

	// Remove the file
	if err := os.Remove(configPath); err != nil {
		return fmt.Errorf("removing config file: %w", err)
	}

	_, _ = fmt.Fprintf(opts.Stderr, "Configuration file removed: %s\n", configPath)

	// Check for active environment variables
	var envVars []string
	if sharedconfig.GetEnvWithFallback("CFL_URL", "ATLASSIAN_URL") != "" {
		envVars = append(envVars, "URL")
	}
	if sharedconfig.GetEnvWithFallback("CFL_EMAIL", "ATLASSIAN_EMAIL") != "" {
		envVars = append(envVars, "Email")
	}
	if sharedconfig.GetEnvWithFallback("CFL_API_TOKEN", "ATLASSIAN_API_TOKEN") != "" {
		envVars = append(envVars, "API Token")
	}

	if len(envVars) > 0 {
		_, _ = fmt.Fprintln(opts.Stderr)
		_, _ = fmt.Fprintf(opts.Stderr, "Note: The following are still configured via environment variables: %s\n",
			strings.Join(envVars, ", "))
		_, _ = fmt.Fprintln(opts.Stderr, "These will continue to be used. Unset them if you want to fully clear configuration.")
	}

	return nil
}
