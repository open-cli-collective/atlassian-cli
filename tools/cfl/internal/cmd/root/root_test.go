package root

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/auth"
	sharedclient "github.com/open-cli-collective/atlassian-go/client"
	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/credtest"
	"github.com/open-cli-collective/atlassian-go/keyring"
	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/open-cli-collective/atlassian-go/view"
	cccredstore "github.com/open-cli-collective/cli-common/credstore"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/config"
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

func TestConfigFlagIsAuthoritative(t *testing.T) {
	tests := []struct {
		name        string
		authMethod  string
		url         string
		email       string
		cloudID     string
		checkClient func(*testing.T, *api.Client, string)
	}{
		{
			name:       "basic",
			authMethod: auth.AuthMethodBasic,
			url:        "https://explicit-basic.atlassian.net/wiki",
			email:      "explicit-basic@example.com",
			cloudID:    "explicit-basic-cloud",
			checkClient: func(t *testing.T, client *api.Client, token string) {
				t.Helper()
				if client.GetBaseURL() != "https://explicit-basic.atlassian.net/wiki" ||
					client.GetAuthHeader() != auth.BasicAuthHeader("explicit-basic@example.com", token) {
					t.Fatal("basic client did not use the explicit config and resolved token")
				}
			},
		},
		{
			name:       "bearer",
			authMethod: auth.AuthMethodBearer,
			url:        "https://explicit-bearer.atlassian.net/wiki",
			email:      "explicit-bearer@example.com",
			cloudID:    "explicit-bearer-cloud",
			checkClient: func(t *testing.T, client *api.Client, token string) {
				t.Helper()
				wantURL := fmt.Sprintf("%s/ex/confluence/explicit-bearer-cloud/wiki", sharedclient.GatewayBaseURL)
				if client.GetBaseURL() != wantURL || client.GetAuthHeader() != auth.BearerAuthHeader(token) {
					t.Fatal("bearer client did not use the explicit config and resolved token")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credtest.Hermetic(t)
			keyring.SetBackendSelection("", "")
			t.Cleanup(func() { keyring.SetBackendSelection("", "") })
			for _, name := range []string{
				"CFL_URL", "ATLASSIAN_URL", "CFL_EMAIL", "ATLASSIAN_EMAIL",
				"CFL_AUTH_METHOD", "ATLASSIAN_AUTH_METHOD", "CFL_CLOUD_ID",
				"ATLASSIAN_CLOUD_ID", "CFL_DEFAULT_SPACE",
			} {
				t.Setenv(name, "")
			}

			defaultCfg := &config.Config{
				URL:          "https://default.atlassian.net/wiki",
				Email:        "default@example.com",
				AuthMethod:   auth.AuthMethodBasic,
				CloudID:      "default-cloud",
				DefaultSpace: "DEFAULT",
				OutputFormat: "default-output",
				Keyring:      config.KeyringConfig{Backend: "file"},
			}
			testutil.RequireNoError(t, defaultCfg.Save(config.DefaultConfigPath()))
			testutil.RequireNoError(t, (&credstore.Store{
				Default: credstore.Section{
					URL:        "https://shared.atlassian.net",
					Email:      "shared@example.com",
					AuthMethod: auth.AuthMethodBearer,
					CloudID:    "shared-cloud",
				},
				CFL: credstore.ToolSection{DefaultSpace: "SHARED", OutputFormat: "shared-output"},
			}).Save(credtest.SharedConfigPath(t)))

			explicitPath := filepath.Join(t.TempDir(), "explicit.yml")
			explicitCfg := &config.Config{
				URL:          tt.url,
				Email:        tt.email,
				AuthMethod:   tt.authMethod,
				CloudID:      tt.cloudID,
				DefaultSpace: "EXPLICIT",
				OutputFormat: "explicit-output",
				Keyring:      config.KeyringConfig{Backend: "memory"},
			}
			testutil.RequireNoError(t, explicitCfg.Save(explicitPath))

			secret := "explicit-token-" + tt.name
			t.Setenv("CFL_API_TOKEN", secret)
			var stdout, stderr bytes.Buffer
			rootCmd, opts := NewCmd()
			rootCmd.SetOut(&stdout)
			rootCmd.SetErr(&stderr)
			rootCmd.AddCommand(&cobra.Command{
				Use: "probe",
				RunE: func(*cobra.Command, []string) error {
					cfg, err := opts.Config()
					if err != nil {
						return err
					}
					if cfg.URL != explicitCfg.URL || cfg.Email != explicitCfg.Email ||
						cfg.AuthMethod != explicitCfg.AuthMethod || cfg.CloudID != explicitCfg.CloudID ||
						cfg.DefaultSpace != explicitCfg.DefaultSpace || cfg.OutputFormat != explicitCfg.OutputFormat ||
						cfg.Keyring.Backend != explicitCfg.Keyring.Backend || cfg.APIToken != secret {
						t.Fatal("resolved config did not use every explicit setting and the authoritative token resolver")
					}
					client, err := opts.APIClient()
					if err != nil {
						return err
					}
					tt.checkClient(t, client, secret)
					return nil
				},
			})
			rootCmd.SetArgs([]string{"--config", explicitPath, "probe"})

			err := rootCmd.Execute()
			if err != nil {
				t.Fatalf("Execute failed without exposing credentials: %v", err)
			}
			if opts.ConfigPath != explicitPath {
				t.Fatal("--config did not set the authoritative path")
			}
			_, configBackend := keyring.GetBackendSelection()
			if configBackend != cccredstore.BackendMemory {
				t.Fatal("keyring backend did not come from the explicit config")
			}
			if bytes.Contains(stdout.Bytes(), []byte(secret)) || bytes.Contains(stderr.Bytes(), []byte(secret)) {
				t.Fatal("command output exposed the API token")
			}
		})
	}
}

func TestDefaultConfigLayersSharedValues(t *testing.T) {
	credtest.Hermetic(t)
	keyring.SetBackendSelection("", "")
	t.Cleanup(func() { keyring.SetBackendSelection("", "") })
	t.Setenv("CFL_API_TOKEN", "token")

	legacy := &config.Config{
		URL:          "https://legacy.atlassian.net/wiki",
		Email:        "legacy@example.com",
		AuthMethod:   auth.AuthMethodBasic,
		CloudID:      "legacy-cloud",
		DefaultSpace: "LEGACY",
		OutputFormat: "legacy-output",
	}
	testutil.RequireNoError(t, legacy.Save(config.DefaultConfigPath()))
	testutil.RequireNoError(t, (&credstore.Store{
		Default: credstore.Section{
			URL:        "https://shared.atlassian.net",
			Email:      "shared@example.com",
			AuthMethod: auth.AuthMethodBearer,
			CloudID:    "shared-cloud",
		},
		CFL: credstore.ToolSection{DefaultSpace: "SHARED", OutputFormat: "shared-output"},
	}).Save(credtest.SharedConfigPath(t)))

	rootCmd, opts := NewCmd()
	rootCmd.AddCommand(&cobra.Command{
		Use: "probe",
		RunE: func(*cobra.Command, []string) error {
			cfg, err := opts.Config()
			if err != nil {
				return err
			}
			if cfg.URL != "https://shared.atlassian.net/wiki" || cfg.Email != "shared@example.com" ||
				cfg.AuthMethod != auth.AuthMethodBearer || cfg.CloudID != "shared-cloud" ||
				cfg.DefaultSpace != "SHARED" || cfg.OutputFormat != "shared-output" {
				t.Fatal("default config did not layer shared values")
			}
			return nil
		},
	})
	rootCmd.SetArgs([]string{"probe"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}
