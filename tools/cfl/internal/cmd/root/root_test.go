package root

import (
	"bytes"
	"testing"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/testutil"
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
		{"compact flag", "compact"},
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
		Output:  "json",
		NoColor: true,
		Stdout:  &stdout,
		Stderr:  &stderr,
	}

	v := opts.View()
	testutil.NotNil(t, v)
	testutil.Equal(t, v.Out, &stdout)
	testutil.Equal(t, v.Err, &stderr)
	testutil.True(t, v.NoColor)
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
