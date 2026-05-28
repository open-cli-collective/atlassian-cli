// Package root provides the root command and shared options for the jtk CLI.
package root

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	cccredstore "github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/keyring"
	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/version"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
)

// ErrAlreadyReported signals that the command has already rendered its failure
// output to stderr. main.go checks for this to avoid double-printing.
var ErrAlreadyReported = errors.New("already reported")

// Options contains global options for commands
type Options struct {
	NoColor        bool
	Extended       bool // --extended: include admin/schema/audit fields.
	FullText       bool // --fulltext: disable truncation of descriptions/comments.
	IDOnly         bool // --id: emit only the primary identifier; takes precedence over Extended/FullText.
	Verbose        bool
	NonInteractive bool // --non-interactive (§3.4): never prompt; fail loud on missing required values.
	Stdin          io.Reader
	Stdout         io.Writer
	Stderr         io.Writer

	// testClient is used for testing; if set, APIClient() returns this instead
	testClient *api.Client

	// cachedClient caches the API client after first construction
	cachedClient *api.Client
}

// EmitIDOnly reports whether output should collapse to the primary identifier.
func (o *Options) EmitIDOnly() bool { return o.IDOnly }

// IsExtended reports whether extended output is requested, honoring --id precedence (--id wins).
func (o *Options) IsExtended() bool { return !o.IDOnly && o.Extended }

// IsFullText reports whether body truncation is disabled, honoring --id precedence (--id wins).
func (o *Options) IsFullText() bool { return !o.IDOnly && o.FullText }

// View returns a configured View instance, deriving policy from RenderMode.
// Format is hardcoded to table; legacy format selection is removed from JTK.
func (o *Options) View() *view.View {
	v := view.NewWithFormat("table", o.NoColor)
	if o.RenderMode() == present.RenderModeAgent {
		v.SetPolicy(view.PolicyAgent)
	}
	v.Out = o.Stdout
	v.Err = o.Stderr
	return v
}

// ArtifactMode returns the artifact type based on the --extended flag,
// honoring --id precedence (--id collapses output, so Extended is ignored).
func (o *Options) ArtifactMode() artifact.Type {
	return artifact.Mode(o.IsExtended())
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
	token, err := config.ResolveAPIToken()
	if err != nil {
		return nil, err
	}
	c, err := api.New(api.ClientConfig{
		URL:        config.GetURL(),
		Email:      config.GetEmail(),
		APIToken:   token,
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
		// PersistentPreRunE runs before any subcommand RunE. It wires the
		// --backend flag and config keyring.backend into shared/keyring's
		// SetBackendSelection so every subsequent keyring.Open* uses the
		// caller-selected backend. Must NOT read ATLASSIAN_CLI_KEYRING_BACKEND
		// directly — credstore reads it inside selectBackend, and remapping
		// would corrupt SourceEnv attribution.
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return WireBackendSelection(cmd)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetVersionTemplate("{{.Version}}\n") // Bare version output for token efficiency

	// Global flags - bound to opts struct
	cmd.PersistentFlags().BoolVar(&opts.NoColor, "no-color", false, "Disable colored output")
	cmd.PersistentFlags().BoolVar(&opts.Extended, "extended", false, "Include admin/schema/audit fields in output")
	cmd.PersistentFlags().BoolVar(&opts.FullText, "fulltext", false, "Disable truncation of descriptions and comments")
	cmd.PersistentFlags().BoolVar(&opts.IDOnly, "id", false, "Emit only the primary identifier (takes precedence over --extended and --fulltext)")
	cmd.PersistentFlags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Log each request's method/URL, JSON body, and any 4xx/5xx response body (each capped at 4 KB)")
	cmd.PersistentFlags().BoolVar(&opts.NonInteractive, "non-interactive", false, "Never prompt; fail loud naming any required value missing from flags/env/stdin (§3.4)")
	cmd.PersistentFlags().String(cccredstore.BackendFlagName, "", cccredstore.BackendFlagUsage())

	return cmd, opts
}

// WireBackendSelection reads --backend (looked up via cmd.Flag so the
// lookup works on any subcommand path that inherits the root's
// persistent flag) and the keyring.backend config key, validates them
// via credstore.BindBackendFlag, and pushes the result into
// shared/keyring's package-level state for every subsequent Open*.
//
// It ALSO threads the --non-interactive root flag into shared/keyring's
// package state so the file-backend passphrase callback fails loud
// under --non-interactive even on a real TTY (§3.4). Folded into a
// single chokepoint because cobra does NOT chain PersistentPreRunE: a
// shadowing subcommand (jtk has four — dashboards, boards, automation,
// sprints) that calls WireBackendSelection gets BOTH wires; splitting
// them risks one wire missing on those paths.
//
// Exported because cobra does NOT chain PersistentPreRunE — a
// subcommand that defines its own PersistentPreRunE silently shadows
// the root's. Such subcommands (jtk has four: dashboards, boards,
// automation, sprints) must call WireBackendSelection(cmd) at the top
// of their own PersistentPreRunE so the backend-selection wiring still
// runs on those command paths. Subcommands without their own
// PersistentPreRunE inherit the root's and get wiring for free.
//
// Failure paths:
//   - --backend with an unrecognized value -> wrapping ErrBackendNotImplemented
//   - --backend= (empty) -> fails closed instead of silent flag-loss
//   - config keyring.backend invalid -> surfaces later at credstore.Open
//     (intentional pass-through; the helper layer doesn't validate config).
func WireBackendSelection(cmd *cobra.Command) error {
	var flagValue string
	var flagSet bool
	if bf := cmd.Flag(cccredstore.BackendFlagName); bf != nil {
		flagValue = bf.Value.String()
		flagSet = bf.Changed
	}

	// Best-effort config load. A missing/unreadable config must not block
	// commands that don't need credentials (e.g., `jtk completion`); the
	// commands that do need them already handle their own load errors.
	cfg, _ := config.Load()
	var configBackend string
	if cfg != nil {
		configBackend = cfg.Keyring.Backend
	}

	opts := &cccredstore.Options{}
	if err := cccredstore.BindBackendFlag(opts, flagValue, flagSet, configBackend); err != nil {
		return fmt.Errorf("--%s: %w", cccredstore.BackendFlagName, err)
	}
	keyring.SetBackendSelection(opts.Backend, opts.ConfigBackend)

	// §3.4: thread --non-interactive into the keyring package state so
	// the file-backend passphrase callback fails loud under
	// --non-interactive regardless of TTY.
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

// GetOptions extracts Options from a root command
func GetOptions(cmd *cobra.Command) *Options {
	noColor, _ := cmd.Root().PersistentFlags().GetBool("no-color")
	extended, _ := cmd.Root().PersistentFlags().GetBool("extended")
	fullText, _ := cmd.Root().PersistentFlags().GetBool("fulltext")
	idOnly, _ := cmd.Root().PersistentFlags().GetBool("id")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")

	return &Options{
		NoColor:  noColor,
		Extended: extended,
		FullText: fullText,
		IDOnly:   idOnly,
		Verbose:  verbose,
		Stdin:    os.Stdin,
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
	}
}
