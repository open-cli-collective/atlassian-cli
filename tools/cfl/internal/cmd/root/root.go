// Package root provides the root command for the cfl CLI.
package root

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	cccredstore "github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/auth"
	"github.com/open-cli-collective/atlassian-go/keyring"
	"github.com/open-cli-collective/atlassian-go/version"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/config"
)

// Options contains global options for commands
type Options struct {
	Output         string
	NoColor        bool
	Full           bool
	NonInteractive bool // --non-interactive (§3.4): never prompt; fail loud on missing required values.
	Stdin          io.Reader
	Stdout         io.Writer
	Stderr         io.Writer

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

// ArtifactMode returns the artifact type based on the --full flag.
func (o *Options) ArtifactMode() artifact.Type {
	return artifact.Mode(o.Full)
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
		return nil, fmt.Errorf("loading config: %w (run 'cfl init' to configure)", err)
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
	if cfg.AuthMethod == auth.AuthMethodBearer {
		return api.NewBearerClient(cfg.APIToken, cfg.CloudID)
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
		// PersistentPreRunE wires --backend and keyring.backend (config)
		// into shared/keyring before any subcommand runs. Must NOT read
		// ATLASSIAN_CLI_KEYRING_BACKEND directly — credstore reads it.
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return wireBackendSelection(cmd)
		},
	}

	// Global flags - bound to opts struct
	cmd.PersistentFlags().StringP("config", "c", "", "config file (default: ~/.config/cfl/config.yml)")
	cmd.PersistentFlags().StringVarP(&opts.Output, "output", "o", "table", "output format: table, json, plain")
	cmd.PersistentFlags().BoolVar(&opts.NoColor, "no-color", false, "disable colored output")
	cmd.PersistentFlags().BoolVar(&opts.Full, "full", false, "show full inspection-oriented output (default: agent)")
	cmd.PersistentFlags().BoolVar(&opts.NonInteractive, "non-interactive", false, "Never prompt; fail loud naming any required value missing from flags/env/stdin (§3.4)")
	cmd.PersistentFlags().String(cccredstore.BackendFlagName, "", cccredstore.BackendFlagUsage())

	// Set version template
	cmd.SetVersionTemplate("cfl version {{.Version}} (commit: " + version.Commit + ", built: " + version.BuildDate + ")\n")

	return cmd, opts
}

// wireBackendSelection reads --backend (via cmd.Flag so the lookup
// works on any subcommand path that inherits the root persistent flag)
// and the keyring.backend config key, validates them via
// credstore.BindBackendFlag, and pushes the result into shared/keyring.
//
// Best-effort config load: commands that don't need credentials (e.g.,
// `cfl completion`) must not fail just because config is missing or
// malformed; commands that do need them handle their own load errors.
func wireBackendSelection(cmd *cobra.Command) error {
	var flagValue string
	var flagSet bool
	if bf := cmd.Flag(cccredstore.BackendFlagName); bf != nil {
		flagValue = bf.Value.String()
		flagSet = bf.Changed
	}

	var configBackend string
	cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath()
	}
	if cfg, err := config.Load(cfgPath); err == nil && cfg != nil {
		configBackend = cfg.Keyring.Backend
	}

	opts := &cccredstore.Options{}
	if err := cccredstore.BindBackendFlag(opts, flagValue, flagSet, configBackend); err != nil {
		return fmt.Errorf("--%s: %w", cccredstore.BackendFlagName, err)
	}
	keyring.SetBackendSelection(opts.Backend, opts.ConfigBackend)

	// §3.4: thread --non-interactive into the keyring package state so
	// the file-backend passphrase callback fails loud under
	// --non-interactive regardless of TTY. Folded into this single
	// chokepoint so any future shadowing PersistentPreRunE that calls
	// wireBackendSelection gets both wires.
	var nonInteractive bool
	if nif := cmd.Flag("non-interactive"); nif != nil {
		nonInteractive = nif.Value.String() == "true"
	}
	keyring.SetNonInteractive(nonInteractive)

	return nil
}

// RegisterCommands registers subcommands with the root command
func RegisterCommands(root *cobra.Command, opts *Options, registrars ...func(*cobra.Command, *Options)) {
	for _, register := range registrars {
		register(root, opts)
	}
}
