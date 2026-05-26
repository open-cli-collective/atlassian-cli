package root

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/keyring"
	cccredstore "github.com/open-cli-collective/cli-common/credstore"
)

func newProbeCmd(name string) *cobra.Command {
	return &cobra.Command{
		Use:  name,
		RunE: func(*cobra.Command, []string) error { return nil },
	}
}

func TestWireBackendSelection_FlagSet(t *testing.T) {
	keyring.SetBackendSelection("", "")
	defer keyring.SetBackendSelection("", "")
	t.Setenv(cccredstore.BackendEnvVar(keyring.Service), "")

	rootCmd, _ := NewCmd()
	rootCmd.AddCommand(newProbeCmd("probe"))
	rootCmd.SetArgs([]string{"probe", "--backend", "memory"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	gotBackend, _ := keyring.GetBackendSelection()
	if gotBackend != cccredstore.BackendMemory {
		t.Errorf("Backend = %q, want %q", gotBackend, cccredstore.BackendMemory)
	}
}

func TestWireBackendSelection_FlagInvalid(t *testing.T) {
	keyring.SetBackendSelection("", "")
	defer keyring.SetBackendSelection("", "")
	t.Setenv(cccredstore.BackendEnvVar(keyring.Service), "")

	rootCmd, _ := NewCmd()
	rootCmd.AddCommand(newProbeCmd("probe"))
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
