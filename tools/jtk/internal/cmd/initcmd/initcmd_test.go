package initcmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/credtest"
	"github.com/open-cli-collective/atlassian-go/keyring"
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
	err := runInit(context.Background(), opts, "", "", "", false, "", "Bearer", "", true)
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
		wants    []string
	}{
		{
			name:  "basic auth — missing URL",
			cfg:   &config.Config{},
			wants: []string{"--url"},
		},
		{
			name:  "basic auth — missing email",
			cfg:   &config.Config{URL: "https://acme.atlassian.net"},
			wants: []string{"--email"},
		},
		{
			name:     "bearer — missing cloud-id",
			cfg:      &config.Config{URL: "https://acme.atlassian.net"},
			isBearer: true,
			wants:    []string{"--cloud-id"},
		},
		{
			name:  "basic auth — missing token recommends --token-stdin + --token-from-env + set-credential",
			cfg:   &config.Config{URL: "https://acme.atlassian.net", Email: "u@x.io"},
			wants: []string{"--token-stdin", "--token-from-env", "set-credential"},
		},
		{
			name:     "bearer — missing token recommends --token-stdin + --token-from-env + set-credential",
			cfg:      &config.Config{URL: "https://acme.atlassian.net", CloudID: "cid"},
			isBearer: true,
			wants:    []string{"--token-stdin", "--token-from-env", "set-credential"},
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
			for _, want := range tc.wants {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error must name %s, got %v", want, err)
				}
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
	err := runInit(context.Background(), opts, "", "", "", false, "", "", "", true)
	testutil.RequireError(t, err)
	if !strings.Contains(err.Error(), "--non-interactive") || !strings.Contains(err.Error(), "--url") {
		t.Fatalf("expected --non-interactive missing --url error, got: %v", err)
	}
}

// TestRunInit_NonInteractive_MissingToken_FlagAndKeyringEmpty — the
// fail-loud hint must point to --token-stdin / --token-from-env first
// (the §1.5.1 canonical scripted shape) with `jtk set-credential` as
// the alternate pre-stage path.
func TestRunInit_NonInteractive_MissingToken_FlagAndKeyringEmpty(t *testing.T) {
	credtest.Hermetic(t)
	opts := &root.Options{
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(""),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts, "https://acme.atlassian.net", "u@x.io", "", false, "", "", "", true)
	testutil.RequireError(t, err)
	if !strings.Contains(err.Error(), "--token-stdin") {
		t.Fatalf("error must hint at --token-stdin, got: %v", err)
	}
	if !strings.Contains(err.Error(), "--token-from-env") {
		t.Fatalf("error must hint at --token-from-env, got: %v", err)
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

const initSentinel = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAjtkInitTok"

// TestRunInit_TokenStdin_PopulatesAPIToken — under --non-interactive,
// --token-stdin populates cfg.APIToken and the run proceeds without
// touching the form.
func TestRunInit_TokenStdin_PopulatesAPIToken(t *testing.T) {
	credtest.Hermetic(t)
	opts := &root.Options{
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(initSentinel + "\n"),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts,
		"https://acme.atlassian.net", "u@x.io", "", true, "", "", "", true)
	testutil.RequireNoError(t, err)
}

// TestRunInit_TokenFromEnv_PopulatesAPIToken — same with --token-from-env.
func TestRunInit_TokenFromEnv_PopulatesAPIToken(t *testing.T) {
	credtest.Hermetic(t)
	t.Setenv("JTK_INIT_TOKEN_VAR", initSentinel)
	opts := &root.Options{
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(""),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts,
		"https://acme.atlassian.net", "u@x.io", "", false, "JTK_INIT_TOKEN_VAR", "", "", true)
	testutil.RequireNoError(t, err)
}

// TestRunInit_TokenAndTokenStdin_Fails — mutual exclusion.
func TestRunInit_TokenAndTokenStdin_Fails(t *testing.T) {
	credtest.Hermetic(t)
	opts := &root.Options{
		NoColor: true,
		Stdin:   strings.NewReader(initSentinel),
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts,
		"https://acme.atlassian.net", "u@x.io", "tok-value", true, "", "", "", true)
	testutil.RequireError(t, err)
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("error must mention mutual exclusion, got: %v", err)
	}
}

// TestRunInit_TokenAndTokenFromEnv_Fails — mutual exclusion.
func TestRunInit_TokenAndTokenFromEnv_Fails(t *testing.T) {
	credtest.Hermetic(t)
	t.Setenv("JTK_INIT_TOKEN_VAR", initSentinel)
	opts := &root.Options{
		NoColor: true,
		Stdin:   strings.NewReader(""),
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts,
		"https://acme.atlassian.net", "u@x.io", "tok-value", false, "JTK_INIT_TOKEN_VAR", "", "", true)
	testutil.RequireError(t, err)
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("error must mention mutual exclusion, got: %v", err)
	}
}

// TestRunInit_TokenStdinEmpty_Fails — empty stdin is rejected.
func TestRunInit_TokenStdinEmpty_Fails(t *testing.T) {
	credtest.Hermetic(t)
	opts := &root.Options{
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader("   \n  "),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts,
		"https://acme.atlassian.net", "u@x.io", "", true, "", "", "", true)
	testutil.RequireError(t, err)
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("error must mention empty, got: %v", err)
	}
}

// TestRunInit_DeprecatedTokenFlag_PrintsWarning — --token <value>
// triggers the §1.5.1 deprecation warning to stderr before the run
// proceeds. Asserts the exact prefix so future regressions are loud.
func TestRunInit_DeprecatedTokenFlag_PrintsWarning(t *testing.T) {
	credtest.Hermetic(t)
	var stderr bytes.Buffer
	opts := &root.Options{
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(""),
		Stdout:         &bytes.Buffer{},
		Stderr:         &stderr,
	}
	err := runInit(context.Background(), opts,
		"https://acme.atlassian.net", "u@x.io", "tok-value", false, "", "", "", true)
	testutil.RequireNoError(t, err)
	if !strings.Contains(stderr.String(), "warning: --token is deprecated") {
		t.Fatalf("stderr must contain deprecation warning, got: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "§1.5.1") {
		t.Fatalf("stderr deprecation must reference §1.5.1, got: %q", stderr.String())
	}
}

// TestRunInit_TokenStdinOverridesKeyring — explicit ingress wins over
// keyring backfill. Pre-stage a different value, run with --token-stdin,
// assert the keyring ends up with the NEW value (token-rotation
// contract).
//
// What this pins: the user-visible rotation outcome.
// What this does NOT pin: the keyring-read side effect (whether
// ResolveTokenNoMigrate was called). The production code's guard is
// `if cfg.APIToken == ""` which only triggers the keyring backfill
// when ingress didn't fire. A regression that swapped the order
// (backfill first, then ingress overwrites) would still pass this test
// because the final value is the same. Pinning the side-effect would
// require instrumentation on shared/keyring; that's a separate
// follow-up. The user-facing contract is what matters here.
func TestRunInit_TokenStdinOverridesKeyring(t *testing.T) {
	credtest.Hermetic(t)
	credtest.SeedToken(t, "stale-token-from-keyring")

	opts := &root.Options{
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(initSentinel + "\n"),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts,
		"https://acme.atlassian.net", "u@x.io", "", true, "", "", "", true)
	testutil.RequireNoError(t, err)

	// Assert via the resolver chain that the new value landed.
	got, _, rerr := keyring.ResolveTokenNoMigrate(credstore.ToolJTK)
	testutil.RequireNoError(t, rerr)
	testutil.Equal(t, initSentinel, got)
}

// TestRunInit_TokenStdin_NoDeprecationWarning — --token-stdin is the
// canonical §1.5.1 path; it MUST NOT trigger the --token deprecation
// warning. A regression that emitted the warning for the new flags
// would confuse users into thinking the recommended flag was also
// deprecated.
func TestRunInit_TokenStdin_NoDeprecationWarning(t *testing.T) {
	credtest.Hermetic(t)
	var stderr bytes.Buffer
	opts := &root.Options{
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(initSentinel + "\n"),
		Stdout:         &bytes.Buffer{},
		Stderr:         &stderr,
	}
	err := runInit(context.Background(), opts,
		"https://acme.atlassian.net", "u@x.io", "", true, "", "", "", true)
	testutil.RequireNoError(t, err)
	if strings.Contains(stderr.String(), "deprecated") {
		t.Fatalf("--token-stdin must NOT trigger the --token deprecation warning: %q", stderr.String())
	}
}

// TestRunInit_TokenFromEnv_NoDeprecationWarning — same for
// --token-from-env.
func TestRunInit_TokenFromEnv_NoDeprecationWarning(t *testing.T) {
	credtest.Hermetic(t)
	t.Setenv("JTK_DEPRECATION_NEG_VAR", initSentinel)
	var stderr bytes.Buffer
	opts := &root.Options{
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(""),
		Stdout:         &bytes.Buffer{},
		Stderr:         &stderr,
	}
	err := runInit(context.Background(), opts,
		"https://acme.atlassian.net", "u@x.io", "", false, "JTK_DEPRECATION_NEG_VAR", "", "", true)
	testutil.RequireNoError(t, err)
	if strings.Contains(stderr.String(), "deprecated") {
		t.Fatalf("--token-from-env must NOT trigger the --token deprecation warning: %q", stderr.String())
	}
}

// TestRunInit_TokenStdinAndTokenFromEnv_Fails — both new flags set
// (without --token) must hit the helper's mutual-exclusion guard.
// Pinned at the jtk integration layer so a jtk-specific short-circuit
// regression that bypassed ReadSecretFromIngress would be loud.
func TestRunInit_TokenStdinAndTokenFromEnv_Fails(t *testing.T) {
	credtest.Hermetic(t)
	t.Setenv("JTK_BOTH_VAR", initSentinel)
	opts := &root.Options{
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(initSentinel),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts,
		"https://acme.atlassian.net", "u@x.io", "", true, "JTK_BOTH_VAR", "", "", true)
	testutil.RequireError(t, err)
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("error must mention mutual exclusion, got: %v", err)
	}
}

// TestRunInit_TokenStdinPipedStdin_NoNonInteractiveRequired — the
// canonical CI usage `op read | jtk init --token-stdin --url ... --email ...`
// pipes stdin (non-TTY), which makes WantPrompt false and skips the
// huh form. --non-interactive is therefore NOT required for piped
// usage; the guard only fires when stdin IS a real TTY (would-be
// interactive form). Pinned so a future regression that re-tightens
// the guard to require --non-interactive unconditionally is loud.
func TestRunInit_TokenStdinPipedStdin_NoNonInteractiveRequired(t *testing.T) {
	credtest.Hermetic(t)
	opts := &root.Options{
		NoColor:        true,
		NonInteractive: false, // intentionally — pipe is non-TTY, WantPrompt is false
		Stdin:          strings.NewReader(initSentinel + "\n"),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts,
		"https://acme.atlassian.net", "u@x.io", "", true, "", "", "", true)
	testutil.RequireNoError(t, err)
}
