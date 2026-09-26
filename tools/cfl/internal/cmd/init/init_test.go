package init

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/auth"
	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/credtest"
	"github.com/open-cli-collective/atlassian-go/keyring"
	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	"github.com/open-cli-collective/confluence-cli/internal/config"
)

func TestConfigFilePermissions(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	cfg := config.Config{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "secret-token",
	}

	err := cfg.Save(configPath)
	testutil.RequireNoError(t, err)

	info, err := os.Stat(configPath)
	testutil.RequireNoError(t, err)

	perm := info.Mode().Perm()
	testutil.Equal(t, perm, os.FileMode(0600))
}

func TestConfigFilePermissions_DirectoryCreation(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nested", "deeply", "config.yml")

	cfg := config.Config{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "secret-token",
	}

	err := cfg.Save(configPath)
	testutil.RequireNoError(t, err)

	_, err = os.Stat(configPath)
	testutil.RequireNoError(t, err)

	dirInfo, err := os.Stat(filepath.Dir(configPath))
	testutil.RequireNoError(t, err)
	testutil.True(t, dirInfo.IsDir())
}

func TestInitCommand_Flags(t *testing.T) {
	t.Parallel()
	rootCmd := &cobra.Command{
		Use:   "cfl",
		Short: "Test CLI",
	}

	opts := &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	Register(rootCmd, opts)

	initCmd, _, err := rootCmd.Find([]string{"init"})
	testutil.RequireNoError(t, err)

	testutil.Equal(t, "init", initCmd.Use)
	testutil.NotEmpty(t, initCmd.Short)
	testutil.NotEmpty(t, initCmd.Long)

	urlFlag := initCmd.Flags().Lookup("url")
	testutil.NotNil(t, urlFlag)
	testutil.Equal(t, "", urlFlag.DefValue)

	emailFlag := initCmd.Flags().Lookup("email")
	testutil.NotNil(t, emailFlag)
	testutil.Equal(t, "", emailFlag.DefValue)

	noVerifyFlag := initCmd.Flags().Lookup("no-verify")
	testutil.NotNil(t, noVerifyFlag)
	testutil.Equal(t, "false", noVerifyFlag.DefValue)

	authMethodFlag := initCmd.Flags().Lookup("auth-method")
	testutil.NotNil(t, authMethodFlag)
	testutil.Equal(t, "", authMethodFlag.DefValue)

	cloudIDFlag := initCmd.Flags().Lookup("cloud-id")
	testutil.NotNil(t, cloudIDFlag)
	testutil.Equal(t, "", cloudIDFlag.DefValue)
}

func TestRunInit_InvalidAuthMethod(t *testing.T) {
	t.Parallel()
	opts := &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts, "", "", false, "", "Bearer", "", true)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "invalid auth method")
}

// TestRunInit_NonInteractive_MissingURL_Fails — drives runInit through
// the public surface; fail-loud surfaces before any keyring work runs.
func TestRunInit_NonInteractive_MissingURL_Fails(t *testing.T) {
	credtest.Hermetic(t)
	opts := &root.Options{
		Output:         "table",
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(""),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts, "", "", false, "", "", "", true)
	testutil.RequireError(t, err)
	if !strings.Contains(err.Error(), "--non-interactive") || !strings.Contains(err.Error(), "--url") {
		t.Fatalf("expected --non-interactive missing --url error, got: %v", err)
	}
}

// TestRunInit_NonInteractive_MissingToken_RecommendsAllPaths — cfl
// init has no --token flag; the §1.5.1 fail-loud hint must recommend
// --token-stdin / --token-from-env (added in this PR) AND point to
// `cfl set-credential` as the alternate pre-stage path.
func TestRunInit_NonInteractive_MissingToken_RecommendsAllPaths(t *testing.T) {
	credtest.Hermetic(t)
	opts := &root.Options{
		Output:         "table",
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(""),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts, "https://acme.atlassian.net", "u@x.io", false, "", "", "", true)
	testutil.RequireError(t, err)
	for _, want := range []string{"--token-stdin", "--token-from-env", "set-credential"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error must mention %s, got: %v", want, err)
		}
	}
}

func TestRunInit_UsesConfiguredPathForReconciliation(t *testing.T) {
	credtest.Hermetic(t)
	defaultPath := config.DefaultConfigPath()
	explicitPath := filepath.Join(t.TempDir(), "explicit.yml")
	testutil.RequireNoError(t, (&config.Config{
		URL: "https://default.atlassian.net/wiki", Email: "default@example.com", DefaultSpace: "DEFAULT",
	}).Save(defaultPath))
	testutil.RequireNoError(t, (&config.Config{
		URL: "https://explicit.atlassian.net/wiki", Email: "explicit@example.com", DefaultSpace: "EXPLICIT",
	}).Save(explicitPath))

	opts := &root.Options{
		ConfigPath:     explicitPath,
		Output:         "table",
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(cflInitSentinel + "\n"),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	testutil.RequireNoError(t, runInit(context.Background(), opts,
		"https://configured.atlassian.net", "configured@example.com", true, "", "", "", true))

	store, err := credstore.Load(credtest.SharedConfigPath(t))
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "EXPLICIT", store.CFL.DefaultSpace)
	testutil.Equal(t, "https://configured.atlassian.net", store.Default.URL)
}

func TestRunInit_Proxy_NoTokenRequiredOrPersisted(t *testing.T) {
	credtest.Hermetic(t)
	opts := &root.Options{
		Output:         "table",
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(""),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts,
		"http://127.0.0.1:8080/atlassian", "", false, "", auth.AuthMethodProxy, "", true)
	testutil.RequireNoError(t, err)

	store, err := credstore.Load(credtest.SharedConfigPath(t))
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "http://127.0.0.1:8080/atlassian", store.Default.URL)
	testutil.Equal(t, "", store.Default.Email)
	testutil.Equal(t, auth.AuthMethodProxy, store.Default.AuthMethod)
	testutil.Equal(t, "", store.Default.CloudID)

	s, err := keyring.OpenNoMigrate()
	testutil.RequireNoError(t, err)
	defer func() { _ = s.Close() }()
	ok, err := s.HasToken(keyring.KeyAPIToken)
	testutil.RequireNoError(t, err)
	testutil.False(t, ok)
}

const cflInitSentinel = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAcflInitTok"

// TestRunInit_TokenStdin_PopulatesAPIToken — under --non-interactive,
// --token-stdin populates cfg.APIToken so the run proceeds without
// requiring a pre-staged keyring entry.
func TestRunInit_TokenStdin_PopulatesAPIToken(t *testing.T) {
	credtest.Hermetic(t)
	opts := &root.Options{
		Output:         "table",
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(cflInitSentinel + "\n"),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts,
		"https://acme.atlassian.net", "u@x.io", true, "", "", "", true)
	testutil.RequireNoError(t, err)
}

// TestRunInit_TokenFromEnv_PopulatesAPIToken — same with --token-from-env.
func TestRunInit_TokenFromEnv_PopulatesAPIToken(t *testing.T) {
	credtest.Hermetic(t)
	t.Setenv("CFL_INIT_TOKEN_VAR", cflInitSentinel)
	opts := &root.Options{
		Output:         "table",
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(""),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts,
		"https://acme.atlassian.net", "u@x.io", false, "CFL_INIT_TOKEN_VAR", "", "", true)
	testutil.RequireNoError(t, err)
}

// TestRunInit_TokenStdinAndFromEnv_Fails — mutual exclusion.
func TestRunInit_TokenStdinAndFromEnv_Fails(t *testing.T) {
	credtest.Hermetic(t)
	t.Setenv("CFL_INIT_TOKEN_VAR", cflInitSentinel)
	opts := &root.Options{
		Output:         "table",
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(cflInitSentinel),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts,
		"https://acme.atlassian.net", "u@x.io", true, "CFL_INIT_TOKEN_VAR", "", "", true)
	testutil.RequireError(t, err)
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("error must mention mutual exclusion, got: %v", err)
	}
}

// TestRunInit_TokenStdinEmpty_Fails — empty stdin is rejected.
func TestRunInit_TokenStdinEmpty_Fails(t *testing.T) {
	credtest.Hermetic(t)
	opts := &root.Options{
		Output:         "table",
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader("   \n  "),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts,
		"https://acme.atlassian.net", "u@x.io", true, "", "", "", true)
	testutil.RequireError(t, err)
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("error must mention empty, got: %v", err)
	}
}

// TestRunInit_TokenStdinOverridesKeyring — explicit ingress wins over
// keyring backfill (token-rotation contract).
func TestRunInit_TokenStdinOverridesKeyring(t *testing.T) {
	credtest.Hermetic(t)
	credtest.SeedToken(t, "stale-token-from-keyring")

	opts := &root.Options{
		Output:         "table",
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(cflInitSentinel + "\n"),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts,
		"https://acme.atlassian.net", "u@x.io", true, "", "", "", true)
	testutil.RequireNoError(t, err)

	got, _, rerr := keyring.ResolveTokenNoMigrate(credstore.ToolCFL)
	testutil.RequireNoError(t, rerr)
	testutil.Equal(t, cflInitSentinel, got)
}

// TestRunInit_TokenStdinPipedStdin_NoNonInteractiveRequired — mirrors
// the jtk test: canonical CI usage `op read | cfl init --token-stdin ...`
// pipes stdin (non-TTY), so WantPrompt is false and the form skips
// regardless of --non-interactive. The TTY-only guard does NOT fire on
// a piped stdin.
func TestRunInit_TokenStdinPipedStdin_NoNonInteractiveRequired(t *testing.T) {
	credtest.Hermetic(t)
	opts := &root.Options{
		Output:         "table",
		NoColor:        true,
		NonInteractive: false,
		Stdin:          strings.NewReader(cflInitSentinel + "\n"),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts,
		"https://acme.atlassian.net", "u@x.io", true, "", "", "", true)
	testutil.RequireNoError(t, err)
}
