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
// exercises BindBackendFlag directly to mirror what wireBackendSelection
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
