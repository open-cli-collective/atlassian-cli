package root

import (
	"bytes"
	"testing"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/open-cli-collective/atlassian-go/view"
	"github.com/spf13/cobra"
)

func TestNewCmd(t *testing.T) {
	t.Parallel()
	cmd, opts := NewCmd()

	testutil.Equal(t, cmd.Use, "cfl")
	testutil.NotEmpty(t, cmd.Short)
	testutil.NotEmpty(t, cmd.Long)
	testutil.NotNil(t, opts)

	// Verify persistent flags exist
	outputFlag := cmd.PersistentFlags().Lookup("output")
	testutil.NotNil(t, outputFlag)

	noColorFlag := cmd.PersistentFlags().Lookup("no-color")
	testutil.NotNil(t, noColorFlag)

	fullFlag := cmd.PersistentFlags().Lookup("full")
	testutil.NotNil(t, fullFlag)
}

func TestNewCmd_Flags(t *testing.T) {
	t.Parallel()
	cmd, _ := NewCmd()

	tests := []struct {
		name string
		flag string
	}{
		{"output flag", "output"},
		{"no-color flag", "no-color"},
		{"full flag", "full"},
		{"config flag", "config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := cmd.PersistentFlags().Lookup(tt.flag)
			testutil.NotNil(t, f)
		})
	}
}

func TestNewCmd_FlagDefaults(t *testing.T) {
	t.Parallel()
	cmd, _ := NewCmd()

	outputFlag := cmd.PersistentFlags().Lookup("output")
	testutil.Equal(t, outputFlag.DefValue, "table")

	noColorFlag := cmd.PersistentFlags().Lookup("no-color")
	testutil.Equal(t, noColorFlag.DefValue, "false")

	fullFlag := cmd.PersistentFlags().Lookup("full")
	testutil.Equal(t, fullFlag.DefValue, "false")
}

func TestOptions_View(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	opts := &Options{
		Output:  "plain",
		NoColor: true,
		Stdout:  &stdout,
		Stderr:  &stderr,
	}

	v := opts.View()
	testutil.NotNil(t, v)
	testutil.Equal(t, v.Out, &stdout)
	testutil.Equal(t, v.Err, &stderr)
	testutil.True(t, v.NoColor)
	testutil.Equal(t, view.FormatPlain, v.Format)
}

func TestOptions_ArtifactMode(t *testing.T) {
	t.Parallel()

	t.Run("returns Agent when Full is false", func(t *testing.T) {
		t.Parallel()
		opts := &Options{Full: false}
		testutil.Equal(t, opts.ArtifactMode(), artifact.Agent)
	})

	t.Run("returns Full when Full is true", func(t *testing.T) {
		t.Parallel()
		opts := &Options{Full: true}
		testutil.Equal(t, opts.ArtifactMode(), artifact.Full)
	})
}

func TestRegisterCommands(t *testing.T) {
	cmd, opts := NewCmd()

	called := false
	registrar := func(parent *cobra.Command, o *Options) {
		called = true
		testutil.Equal(t, parent, cmd)
		testutil.Equal(t, o, opts)
	}

	RegisterCommands(cmd, opts, registrar)
	testutil.True(t, called)
}

func TestValidateOutputFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"table accepted", "table", false},
		{"plain accepted", "plain", false},
		{"json rejected — §2 control-plane carve-out", "json", true},
		{"yaml rejected — outside closed set", "yaml", true},
		{"empty rejected", "", true},
		{"random rejected", "csv", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateOutputFormat(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("validateOutputFormat(%q) returned nil, want error", tt.input)
				}
				testutil.Contains(t, err.Error(), "valid formats: table, plain")
				return
			}
			if err != nil {
				t.Fatalf("validateOutputFormat(%q) returned error: %v", tt.input, err)
			}
		})
	}
}

func TestRoot_RejectsInvalidOutput_AtPreRun(t *testing.T) {
	t.Parallel()

	for _, format := range []string{"json", "yaml", ""} {
		format := format
		t.Run("rejects "+format, func(t *testing.T) {
			t.Parallel()
			cmd, _ := NewCmd()
			// Stub child so cobra reaches PersistentPreRunE.
			cmd.AddCommand(&cobra.Command{
				Use:  "probe",
				RunE: func(*cobra.Command, []string) error { return nil },
			})
			cmd.SetArgs([]string{"-o", format, "probe"})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			err := cmd.Execute()
			if err == nil {
				t.Fatalf("-o %q should be rejected, got nil error", format)
			}
			testutil.Contains(t, err.Error(), "invalid output format")
			testutil.Contains(t, err.Error(), "valid formats: table, plain")
		})
	}
}

// Local --json flags (e.g. set-credential's control-plane envelope) must
// not be confused with the global -o json guard. The guard inspects
// opts.Output (the global -o/--output value); a child command's local
// boolean --json flag has no bearing on it. This pins the invariant so a
// future "uniform JSON rejection" sweep doesn't accidentally reach into
// child command flags.
func TestRoot_LocalJSONFlag_NotRejectedByOutputGuard(t *testing.T) {
	t.Parallel()
	cmd, _ := NewCmd()
	probeJSON := false
	probe := &cobra.Command{
		Use: "probe",
		RunE: func(*cobra.Command, []string) error {
			return nil
		},
	}
	probe.Flags().BoolVar(&probeJSON, "json", false, "local boolean flag (e.g. set-credential's control-plane envelope)")
	cmd.AddCommand(probe)
	cmd.SetArgs([]string{"probe", "--json"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("local --json should not be rejected by the global -o guard, got: %v", err)
	}
	if !probeJSON {
		t.Fatalf("local --json flag did not flip; cobra registration regression?")
	}
}

func TestOptions_View_UsesDefaultPolicy(t *testing.T) {
	t.Parallel()
	opts := &Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
	v := opts.View()

	// cfl should NOT use PolicyAgent - it keeps human-oriented output
	if v.Policy != view.PolicyDefault {
		t.Errorf("cfl View should use PolicyDefault, got %v", v.Policy)
	}
	testutil.Equal(t, view.FormatTable, v.Format)
	testutil.True(t, v.NoColor)
}

func TestOptions_RenderMode(t *testing.T) {
	t.Parallel()
	opts := &Options{}
	if got := opts.RenderMode(); got != present.RenderModeHuman {
		t.Errorf("RenderMode() = %v, want RenderModeHuman", got)
	}
}

func TestOptions_RenderStyle(t *testing.T) {
	t.Parallel()
	opts := &Options{}
	if got := opts.RenderStyle(); got != present.StyleHuman {
		t.Errorf("RenderStyle() = %v, want StyleHuman", got)
	}
}

func TestOptions_RenderStyle_PlainOutput(t *testing.T) {
	t.Parallel()
	opts := &Options{Output: "plain"}
	if got := opts.RenderStyle(); got != present.StyleHumanPlain {
		t.Errorf("RenderStyle() = %v, want StyleHumanPlain", got)
	}
}
