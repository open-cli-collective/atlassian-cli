package root

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/open-cli-collective/atlassian-go/view"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestNewCmd(t *testing.T) {
	t.Parallel()
	cmd, opts := NewCmd()

	testutil.Equal(t, cmd.Use, "jtk")
	testutil.NotEmpty(t, cmd.Short)
	testutil.NotEmpty(t, cmd.Long)
	testutil.NotNil(t, opts)

	// Verify persistent flags exist
	outputFlag := cmd.PersistentFlags().Lookup("output")
	testutil.NotNil(t, outputFlag)

	noColorFlag := cmd.PersistentFlags().Lookup("no-color")
	testutil.NotNil(t, noColorFlag)

	verboseFlag := cmd.PersistentFlags().Lookup("verbose")
	testutil.NotNil(t, verboseFlag)
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
		{"extended flag", "extended"},
		{"fulltext flag", "fulltext"},
		{"id flag", "id"},
		{"verbose flag", "verbose"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := cmd.PersistentFlags().Lookup(tt.flag)
			testutil.NotNil(t, f)
		})
	}
}

func TestNewCmd_FullFlagRemoved(t *testing.T) {
	t.Parallel()
	cmd, _ := NewCmd()
	if f := cmd.PersistentFlags().Lookup("full"); f != nil {
		t.Errorf("--full should have been removed; still registered as %q", f.Name)
	}
}

func TestNewCmd_OutputFlagHidden(t *testing.T) {
	t.Parallel()
	cmd, _ := NewCmd()
	f := cmd.PersistentFlags().Lookup("output")
	testutil.NotNil(t, f)
	testutil.True(t, f.Hidden)
}

func TestNewCmd_FlagDefaults(t *testing.T) {
	t.Parallel()
	cmd, _ := NewCmd()

	outputFlag := cmd.PersistentFlags().Lookup("output")
	testutil.Equal(t, outputFlag.DefValue, "table")

	noColorFlag := cmd.PersistentFlags().Lookup("no-color")
	testutil.Equal(t, noColorFlag.DefValue, "false")

	extendedFlag := cmd.PersistentFlags().Lookup("extended")
	testutil.Equal(t, extendedFlag.DefValue, "false")

	fullTextFlag := cmd.PersistentFlags().Lookup("fulltext")
	testutil.Equal(t, fullTextFlag.DefValue, "false")

	idFlag := cmd.PersistentFlags().Lookup("id")
	testutil.Equal(t, idFlag.DefValue, "false")

	verboseFlag := cmd.PersistentFlags().Lookup("verbose")
	testutil.Equal(t, verboseFlag.DefValue, "false")
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

func TestOptions_SetAPIClient(t *testing.T) {
	client, err := api.New(api.ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@test.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	opts := &Options{}
	opts.SetAPIClient(client)

	got, err := opts.APIClient()
	testutil.RequireNoError(t, err)
	testutil.Equal(t, got, client)
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

func TestOptions_ArtifactMode(t *testing.T) {
	t.Parallel()

	t.Run("returns Agent when Extended is false", func(t *testing.T) {
		t.Parallel()
		opts := &Options{Extended: false}
		testutil.Equal(t, opts.ArtifactMode(), artifact.Agent)
	})

	t.Run("returns Full when Extended is true", func(t *testing.T) {
		t.Parallel()
		opts := &Options{Extended: true}
		testutil.Equal(t, opts.ArtifactMode(), artifact.Full)
	})

	t.Run("returns Agent when IDOnly overrides Extended", func(t *testing.T) {
		t.Parallel()
		opts := &Options{Extended: true, IDOnly: true}
		testutil.Equal(t, opts.ArtifactMode(), artifact.Agent)
	})
}

func TestOptions_IDPrecedence(t *testing.T) {
	t.Parallel()

	t.Run("EmitIDOnly reflects IDOnly field", func(t *testing.T) {
		t.Parallel()
		testutil.True(t, (&Options{IDOnly: true}).EmitIDOnly())
		testutil.False(t, (&Options{IDOnly: false}).EmitIDOnly())
	})

	t.Run("IsExtended is false when IDOnly is set", func(t *testing.T) {
		t.Parallel()
		testutil.False(t, (&Options{IDOnly: true, Extended: true}).IsExtended())
	})

	t.Run("IsExtended is true when Extended is set and IDOnly is not", func(t *testing.T) {
		t.Parallel()
		testutil.True(t, (&Options{IDOnly: false, Extended: true}).IsExtended())
	})

	t.Run("IsFullText is false when IDOnly is set", func(t *testing.T) {
		t.Parallel()
		testutil.False(t, (&Options{IDOnly: true, FullText: true}).IsFullText())
	})

	t.Run("IsFullText is true when FullText is set and IDOnly is not", func(t *testing.T) {
		t.Parallel()
		testutil.True(t, (&Options{IDOnly: false, FullText: true}).IsFullText())
	})
}

func TestOptions_View_UsesAgentPolicy(t *testing.T) {
	t.Parallel()
	opts := &Options{
		Output: "table",
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}
	v := opts.View()

	if v.Policy != view.PolicyAgent {
		t.Errorf("jtk View should use PolicyAgent, got %v", v.Policy)
	}
}

func TestOptions_RenderMode(t *testing.T) {
	t.Parallel()
	opts := &Options{}
	// jtk always uses agent mode for token efficiency
	if got := opts.RenderMode(); got != present.RenderModeAgent {
		t.Errorf("RenderMode() = %v, want RenderModeAgent", got)
	}
}

func TestOptions_RenderStyle(t *testing.T) {
	t.Parallel()
	opts := &Options{}
	// RenderStyle derives from RenderMode via StyleFromMode
	if got := opts.RenderStyle(); got != present.StyleAgent {
		t.Errorf("RenderStyle() = %v, want StyleAgent", got)
	}
}

func TestVersion_BareOutput(t *testing.T) {
	t.Parallel()
	cmd, _ := NewCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})
	_ = cmd.Execute()

	got := strings.TrimSpace(buf.String())
	// Should be just the version number, no "jtk version" prefix
	if strings.HasPrefix(got, "jtk") {
		t.Errorf("version output should be bare, got %q", got)
	}
	// Should match semver pattern or "dev"
	if got != "dev" && !regexp.MustCompile(`^\d+\.\d+\.\d+`).MatchString(got) {
		t.Errorf("version output should be semver or 'dev', got %q", got)
	}
}
