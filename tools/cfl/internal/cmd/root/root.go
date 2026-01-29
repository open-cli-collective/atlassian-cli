// Package root provides the root command for the cfl CLI.
package root

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/version"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/config"
)

// Options contains global options for commands
type Options struct {
	Output  string
	NoColor bool
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer

	// testClient is used for testing; if set, APIClient() returns this instead
	testClient *api.Client

	// cachedConfig stores loaded config for reuse
	cachedConfig *config.Config
}

// View returns a configured View instance
func (o *Options) View() *view.View {
	v := view.NewWithFormat(o.Output, o.NoColor)
	v.Out = o.Stdout
	v.Err = o.Stderr
	return v
}

// Config loads and returns the config, caching it for reuse.
// If a test client is set and no config is cached, returns an empty config
// (since tests inject their own client and typically don't need real config).
func (o *Options) Config() (*config.Config, error) {
	if o.cachedConfig != nil {
		return o.cachedConfig, nil
	}
	// If test client is set, return empty config since tests inject their own client
	if o.testClient != nil {
		o.cachedConfig = &config.Config{}
		return o.cachedConfig, nil
	}
	cfg, err := config.LoadWithEnv(config.DefaultConfigPath())
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w (run 'cfl init' to configure)", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w (run 'cfl init' to configure)", err)
	}
	o.cachedConfig = cfg
	return cfg, nil
}

// SetConfig sets a test config (for testing only)
func (o *Options) SetConfig(cfg *config.Config) {
	o.cachedConfig = cfg
}

// APIClient creates a new API client from config
func (o *Options) APIClient() (*api.Client, error) {
	if o.testClient != nil {
		return o.testClient, nil
	}
	cfg, err := o.Config()
	if err != nil {
		return nil, err
	}
	return api.NewClient(cfg.URL, cfg.Email, cfg.APIToken), nil
}

// SetAPIClient sets a test client (for testing only)
func (o *Options) SetAPIClient(client *api.Client) {
	o.testClient = client
}

// NewCmd creates the root command and returns the options struct
func NewCmd() (*cobra.Command, *Options) {
	opts := &Options{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	cmd := &cobra.Command{
		Use:   "cfl",
		Short: "A command-line interface for Atlassian Confluence",
		Long: `cfl is a CLI tool for interacting with Atlassian Confluence Cloud.

It provides commands for managing pages, spaces, and attachments
with a markdown-first approach for content editing.

Get started by running: cfl init`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.Version,
	}

	// Global flags - bound to opts struct
	cmd.PersistentFlags().StringP("config", "c", "", "config file (default: ~/.config/cfl/config.yml)")
	cmd.PersistentFlags().StringVarP(&opts.Output, "output", "o", "table", "output format: table, json, plain")
	cmd.PersistentFlags().BoolVar(&opts.NoColor, "no-color", false, "disable colored output")

	// Set version template
	cmd.SetVersionTemplate("cfl version {{.Version}} (commit: " + version.Commit + ", built: " + version.BuildDate + ")\n")

	return cmd, opts
}

// RegisterCommands registers subcommands with the root command
func RegisterCommands(root *cobra.Command, opts *Options, registrars ...func(*cobra.Command, *Options)) {
	for _, register := range registrars {
		register(root, opts)
	}
}
