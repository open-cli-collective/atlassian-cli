package root

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/keyring"
	cccredstore "github.com/open-cli-collective/cli-common/credstore"
)

// newProbeCmd returns a no-op subcommand suitable as a leaf for
// command-tree wiring tests. Its RunE does nothing; the root command's
// PersistentPreRunE is what we're actually exercising.
func newProbeCmd(name string) *cobra.Command {
	return &cobra.Command{
		Use:  name,
		RunE: func(*cobra.Command, []string) error { return nil },
	}
}

// TestWireBackendSelection_FlagSet exercises the persistent-flag
// inheritance path: --backend registered on the root command must be
// readable from a subcommand via cmd.Flag(), and its Changed bit must
// be true when the user supplied a value.
func TestWireBackendSelection_FlagSet(t *testing.T) {
	keyring.SetBackendSelection("", "") // reset side-effects
	defer keyring.SetBackendSelection("", "")
	t.Setenv(cccredstore.BackendEnvVar(keyring.Service), "") // defeat env precedence

	rootCmd, _ := NewCmd()
	sub := newProbeCmd("probe")
	rootCmd.AddCommand(sub)
	rootCmd.SetArgs([]string{"probe", "--backend", "memory"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	gotBackend, _ := keyring.GetBackendSelection() // see helper below
	if gotBackend != cccredstore.BackendMemory {
		t.Errorf("Backend = %q, want %q (flag should have populated Options.Backend)", gotBackend, cccredstore.BackendMemory)
	}
}

// TestWireBackendSelection_NonInteractiveFlagThreaded — the
// --non-interactive root flag is folded into WireBackendSelection so
// any shadowing subcommand (dashboards/boards/automation/sprints) that
// calls WireBackendSelection at the top of its PersistentPreRunE also
// gets the non-interactive wire. Pins that coupling.
func TestWireBackendSelection_NonInteractiveFlagThreaded(t *testing.T) {
	keyring.SetBackendSelection("", "")
	keyring.SetNonInteractive(false)
	defer keyring.SetBackendSelection("", "")
	defer keyring.SetNonInteractive(false)

	rootCmd, _ := NewCmd()
	sub := newProbeCmd("probe")
	rootCmd.AddCommand(sub)
	rootCmd.SetArgs([]string{"probe", "--non-interactive"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !keyring.GetNonInteractive() {
		t.Fatal("WireBackendSelection must thread --non-interactive into the keyring package state")
	}
}

// TestWireBackendSelection_NonInteractiveDefaultsFalse — without the
// flag, the keyring package state must stay false (no accidental flip).
func TestWireBackendSelection_NonInteractiveDefaultsFalse(t *testing.T) {
	keyring.SetBackendSelection("", "")
	keyring.SetNonInteractive(false)
	defer keyring.SetBackendSelection("", "")
	defer keyring.SetNonInteractive(false)

	rootCmd, _ := NewCmd()
	sub := newProbeCmd("probe")
	rootCmd.AddCommand(sub)
	rootCmd.SetArgs([]string{"probe"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if keyring.GetNonInteractive() {
		t.Fatal("absence of --non-interactive must leave keyring state false")
	}
}

// TestWireBackendSelection_FlagInvalid asserts a bogus --backend value
// returns an error wrapping ErrBackendNotImplemented.
func TestWireBackendSelection_FlagInvalid(t *testing.T) {
	keyring.SetBackendSelection("", "")
	defer keyring.SetBackendSelection("", "")
	t.Setenv(cccredstore.BackendEnvVar(keyring.Service), "")

	rootCmd, _ := NewCmd()
	sub := newProbeCmd("probe")
	rootCmd.AddCommand(sub)
	rootCmd.SetArgs([]string{"probe", "--backend", "bogus"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, cccredstore.ErrBackendNotImplemented) {
		t.Errorf("errors.Is(_, ErrBackendNotImplemented) = false; err=%v", err)
	}
	if !strings.Contains(err.Error(), "backend") {
		t.Errorf("error should mention --backend: %v", err)
	}
}

// TestWireBackendSelection_FlagOmittedWithConfig asserts the config
// passthrough: when --backend is not supplied, Options.ConfigBackend
// receives the cfg.Keyring.Backend value verbatim. We can't easily
// populate jtk's config file in a unit test, so instead this test
// exercises BindBackendFlag directly to mirror what WireBackendSelection
// does — proving the contract our code depends on.
func TestWireBackendSelection_ConfigPassthrough(t *testing.T) {
	t.Setenv(cccredstore.BackendEnvVar(keyring.Service), "")
	opts := &cccredstore.Options{}
	if err := cccredstore.BindBackendFlag(opts, "", false, "memory"); err != nil {
		t.Fatalf("BindBackendFlag: %v", err)
	}
	if opts.Backend != "" {
		t.Errorf("Backend = %q, want empty (no flag)", opts.Backend)
	}
	if opts.ConfigBackend != cccredstore.BackendMemory {
		t.Errorf("ConfigBackend = %q, want %q", opts.ConfigBackend, cccredstore.BackendMemory)
	}
}

// TestWireBackendSelection_InvalidConfigDeferred asserts the
// non-validation contract for config: a bogus config string is passed
// through to Options.ConfigBackend verbatim and the failure surfaces
// later at credstore.Open, not at the helper layer.
// TestWireBackendSelection_ShadowingSubcommand is the regression guard
// for the cobra-doesn't-chain-PersistentPreRunE bug. A subcommand that
// defines its own PersistentPreRunE silently shadows the root's, so
// without an explicit WireBackendSelection call at the top of each
// shadowing PreRunE, --backend would silently stop applying on those
// command paths. Asserts that running through a subcommand whose own
// PreRunE invokes WireBackendSelection produces the expected backend
// state — i.e., the wiring really does run on the shadowed path.
func TestWireBackendSelection_ShadowingSubcommand(t *testing.T) {
	keyring.SetBackendSelection("", "")
	defer keyring.SetBackendSelection("", "")
	t.Setenv(cccredstore.BackendEnvVar(keyring.Service), "")

	rootCmd, _ := NewCmd()
	// Simulate a shadowing subcommand: its own PersistentPreRunE calls
	// WireBackendSelection explicitly (the contract every shadower in
	// jtk's tree must follow — dashboards/boards/automation/sprints).
	shadow := &cobra.Command{
		Use: "shadow",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return WireBackendSelection(cmd)
		},
	}
	leaf := newProbeCmd("leaf")
	shadow.AddCommand(leaf)
	rootCmd.AddCommand(shadow)
	rootCmd.SetArgs([]string{"shadow", "leaf", "--backend", "memory"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute through shadowing PreRunE: %v", err)
	}
	got, _ := keyring.GetBackendSelection()
	if got != cccredstore.BackendMemory {
		t.Errorf("Backend = %q, want %q — shadower's PreRunE failed to invoke WireBackendSelection", got, cccredstore.BackendMemory)
	}
}

func TestWireBackendSelection_InvalidConfigDeferred(t *testing.T) {
	t.Setenv(cccredstore.BackendEnvVar(keyring.Service), "")
	opts := &cccredstore.Options{}
	if err := cccredstore.BindBackendFlag(opts, "", false, "bogus"); err != nil {
		t.Fatalf("BindBackendFlag should NOT validate config: %v", err)
	}
	if string(opts.ConfigBackend) != "bogus" {
		t.Errorf("ConfigBackend = %q, want verbatim passthrough %q", opts.ConfigBackend, "bogus")
	}
}
