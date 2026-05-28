package initcmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/credtest"
	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
)

func TestConfig_GetDefaultProject_Env(t *testing.T) {
	t.Setenv("JIRA_DEFAULT_PROJECT", "ENVPROJ")

	got := config.GetDefaultProject()
	testutil.Equal(t, got, "ENVPROJ")
}

func TestConfig_GetDefaultProject_NoConfig(t *testing.T) {
	// Clear env and use temp home dir
	t.Setenv("JIRA_DEFAULT_PROJECT", "")
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	// On Linux, also set XDG_CONFIG_HOME to ensure cross-platform behavior
	t.Setenv("XDG_CONFIG_HOME", homeDir)

	got := config.GetDefaultProject()
	testutil.Equal(t, got, "")
}

func TestConfig_DefaultProject_Struct(t *testing.T) {
	t.Parallel()
	// Test that the Config struct has the DefaultProject field
	cfg := &config.Config{
		URL:            "https://test.atlassian.net",
		Email:          "test@example.com",
		APIToken:       "token",
		DefaultProject: "MYPROJ",
	}
	testutil.Equal(t, cfg.DefaultProject, "MYPROJ")
}

func TestRunInit_InvalidAuthMethod(t *testing.T) {
	t.Parallel()
	opts := &root.Options{
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
	// An invalid auth method should be rejected before the interactive form runs
	err := runInit(context.Background(), opts, "", "", "", "Bearer", "", true)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "invalid auth method")
}

// Note: Interactive huh form tests are skipped because huh requires a TTY
// The non-interactive paths (all flags provided) still use huh forms internally,
// so we test config loading/saving separately

// TestRequireNonInteractiveFields_NamesFirstMissing pins the §3.4
// fail-loud message shape so the family pattern (jtk + cfl + future
// nrq-aligned ports) stays consistent. The wizard wrapper isn't tested
// directly because it depends on huh form state we can't easily fake
// — the helper IS the contract for what gets named.
func TestRequireNonInteractiveFields_NamesFirstMissing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      *config.Config
		isBearer bool
		want     string
	}{
		{
			name:     "basic auth — missing URL",
			cfg:      &config.Config{},
			isBearer: false,
			want:     "--url",
		},
		{
			name:     "basic auth — missing email",
			cfg:      &config.Config{URL: "https://acme.atlassian.net"},
			isBearer: false,
			want:     "--email",
		},
		{
			name:     "bearer — missing cloud-id",
			cfg:      &config.Config{URL: "https://acme.atlassian.net"},
			isBearer: true,
			want:     "--cloud-id",
		},
		{
			name:     "basic auth — missing token",
			cfg:      &config.Config{URL: "https://acme.atlassian.net", Email: "u@x.io"},
			isBearer: false,
			want:     "--token",
		},
		{
			name:     "bearer — missing token",
			cfg:      &config.Config{URL: "https://acme.atlassian.net", CloudID: "cid"},
			isBearer: true,
			want:     "--token",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := requireNonInteractiveFields(tc.cfg, tc.isBearer)
			testutil.RequireError(t, err)
			if !strings.Contains(err.Error(), "--non-interactive: missing") {
				t.Fatalf("error must mention --non-interactive prefix: %v", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error must name %s, got %v", tc.want, err)
			}
		})
	}
}

// TestRequireNonInteractiveFields_AllSupplied_NoError — happy path; no
// error returned when every required field is present.
func TestRequireNonInteractiveFields_AllSupplied_NoError(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		URL: "https://acme.atlassian.net", Email: "u@x.io",
		APIToken: "tok-1234567890",
	}
	if err := requireNonInteractiveFields(cfg, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRunInit_NonInteractive_MissingURL_Fails — drives runInit through
// the public surface with --non-interactive but no flags supplied. The
// fail-loud message must surface BEFORE any keyring/migration work runs.
func TestRunInit_NonInteractive_MissingURL_Fails(t *testing.T) {
	credtest.Hermetic(t)
	opts := &root.Options{
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(""), // empty stdin so WantPrompt=false
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts, "", "", "", "", "", true)
	testutil.RequireError(t, err)
	if !strings.Contains(err.Error(), "--non-interactive") || !strings.Contains(err.Error(), "--url") {
		t.Fatalf("expected --non-interactive missing --url error, got: %v", err)
	}
}

// TestRunInit_NonInteractive_MissingToken_FlagAndKeyringEmpty — the
// fail-loud hint must point to `jtk set-credential` when the user has
// neither --token nor a pre-staged keyring entry.
func TestRunInit_NonInteractive_MissingToken_FlagAndKeyringEmpty(t *testing.T) {
	credtest.Hermetic(t)
	opts := &root.Options{
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(""),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts, "https://acme.atlassian.net", "u@x.io", "", "", "", true)
	testutil.RequireError(t, err)
	if !strings.Contains(err.Error(), "--token") {
		t.Fatalf("error must hint at --token, got: %v", err)
	}
	if !strings.Contains(err.Error(), "set-credential") {
		t.Fatalf("error must hint at set-credential pre-staging, got: %v", err)
	}
}

func TestInitCommand_Flags(t *testing.T) {
	t.Parallel()
	rootCmd := &cobra.Command{Use: "jtk", Short: "Test CLI"}

	opts := &root.Options{
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	Register(rootCmd, opts)

	initCmd, _, err := rootCmd.Find([]string{"init"})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "init", initCmd.Use)

	// Verify original flags exist
	urlFlag := initCmd.Flags().Lookup("url")
	testutil.NotNil(t, urlFlag)

	emailFlag := initCmd.Flags().Lookup("email")
	testutil.NotNil(t, emailFlag)

	tokenFlag := initCmd.Flags().Lookup("token")
	testutil.NotNil(t, tokenFlag)

	noVerifyFlag := initCmd.Flags().Lookup("no-verify")
	testutil.NotNil(t, noVerifyFlag)

	// Verify new auth flags exist
	authMethodFlag := initCmd.Flags().Lookup("auth-method")
	testutil.NotNil(t, authMethodFlag)
	testutil.Equal(t, "", authMethodFlag.DefValue)

	cloudIDFlag := initCmd.Flags().Lookup("cloud-id")
	testutil.NotNil(t, cloudIDFlag)
	testutil.Equal(t, "", cloudIDFlag.DefValue)
}
