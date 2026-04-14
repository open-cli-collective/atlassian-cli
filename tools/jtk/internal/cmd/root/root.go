// Package root provides the root command and shared options for the jtk CLI.
package root

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/version"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
)

// Options contains global options for commands
type Options struct {
	Output  string
	NoColor bool
	Full    bool
	Verbose bool
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer

	// testClient is used for testing; if set, APIClient() returns this instead
	testClient *api.Client

	// cachedClient caches the API client after first construction
	cachedClient *api.Client
}

// View returns a configured View instance, deriving policy from RenderMode.
func (o *Options) View() *view.View {
	v := view.NewWithFormat(o.Output, o.NoColor)
	// Derive legacy policy from RenderMode - single source of truth
	if o.RenderMode() == present.RenderModeAgent {
		v.SetPolicy(view.PolicyAgent)
	}
	v.Out = o.Stdout
	v.Err = o.Stderr
	return v
}

// ArtifactMode returns the artifact type based on the --full flag.
func (o *Options) ArtifactMode() artifact.Type {
	return artifact.Mode(o.Full)
}

// RenderMode returns the authoritative rendering mode.
// This is the single source of truth that both legacy View() and new render paths use.
// jtk always uses agent mode for token efficiency.
func (o *Options) RenderMode() present.RenderMode {
	return present.RenderModeAgent
}

// RenderStyle returns the presentation rendering style, derived from RenderMode.
func (o *Options) RenderStyle() present.Style {
	return present.StyleFromMode(o.RenderMode())
}

// APIClient returns the API client, creating it on first call.
// The client is cached so that PersistentPreRunE guards and
// subcommand Run functions share the same instance.
func (o *Options) APIClient() (*api.Client, error) {
	if o.testClient != nil {
		return o.testClient, nil
	}
	if o.cachedClient != nil {
		return o.cachedClient, nil
	}
	c, err := api.New(api.ClientConfig{
		URL:        config.GetURL(),
		Email:      config.GetEmail(),
		APIToken:   config.GetAPIToken(),
		Verbose:    o.Verbose,
		AuthMethod: config.GetAuthMethod(),
		CloudID:    config.GetCloudID(),
	})
	if err != nil {
		return nil, err
	}
	o.cachedClient = c
	return c, nil
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
		Use:     "jtk",
		Short:   "A CLI for managing Jira tickets",
		Long:    "jtk is a command-line interface for managing Jira Cloud tickets.",
		Version: version.Info(),
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			// Setup is done in flag binding
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetVersionTemplate("{{.Version}}\n") // Bare version output for token efficiency

	// Global flags - bound to opts struct
	cmd.PersistentFlags().StringVarP(&opts.Output, "output", "o", "table", "Output format: table, json, plain")
	cmd.PersistentFlags().BoolVar(&opts.NoColor, "no-color", false, "Disable colored output")
	cmd.PersistentFlags().BoolVar(&opts.Full, "full", false, "Show full inspection-oriented output (default: agent)")
	cmd.PersistentFlags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")

	return cmd, opts
}

// RegisterCommands registers subcommands with the root command
func RegisterCommands(root *cobra.Command, opts *Options, registrars ...func(*cobra.Command, *Options)) {
	for _, register := range registrars {
		register(root, opts)
	}
}

// GetOptions extracts Options from a root command
func GetOptions(cmd *cobra.Command) *Options {
	output, _ := cmd.Root().PersistentFlags().GetString("output")
	noColor, _ := cmd.Root().PersistentFlags().GetBool("no-color")
	full, _ := cmd.Root().PersistentFlags().GetBool("full")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")

	return &Options{
		Output:  output,
		NoColor: noColor,
		Full:    full,
		Verbose: verbose,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}
}
