package init

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/auth"
	sharedclient "github.com/open-cli-collective/atlassian-go/client"
	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/credtest"
	"github.com/open-cli-collective/atlassian-go/keyring"
	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/api"
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
	err := runInit(context.Background(), opts, "", "", "Bearer", "", true)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "invalid auth method")
}

// TestRequireNonInteractiveFields_NamesFirstMissing — cfl variant.
// Critically, the token error names `cfl set-credential` rather than
// `--token` because cfl init has no --token flag.
func TestRequireNonInteractiveFields_NamesFirstMissing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		cfg      *config.Config
		isBearer bool
		want     string
	}{
		{"basic — missing URL", &config.Config{}, false, "--url"},
		{"basic — missing email", &config.Config{URL: "https://acme.atlassian.net"}, false, "--email"},
		{"bearer — missing cloud-id", &config.Config{URL: "https://acme.atlassian.net"}, true, "--cloud-id"},
		{
			name: "basic — missing token directs to set-credential",
			cfg:  &config.Config{URL: "https://acme.atlassian.net", Email: "u@x.io"},
			want: "set-credential",
		},
		{
			name:     "bearer — missing token directs to set-credential",
			cfg:      &config.Config{URL: "https://acme.atlassian.net", CloudID: "cid"},
			isBearer: true,
			want:     "set-credential",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := requireNonInteractiveFields(tc.cfg, tc.isBearer)
			testutil.RequireError(t, err)
			if !strings.Contains(err.Error(), "--non-interactive") {
				t.Fatalf("error must mention --non-interactive: %v", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error must mention %s, got %v", tc.want, err)
			}
		})
	}
}

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
	err := runInit(context.Background(), opts, "", "", "", "", true)
	testutil.RequireError(t, err)
	if !strings.Contains(err.Error(), "--non-interactive") || !strings.Contains(err.Error(), "--url") {
		t.Fatalf("expected --non-interactive missing --url error, got: %v", err)
	}
}

// TestRunInit_NonInteractive_MissingToken_DirectsToSetCredential — cfl
// has no --token flag so the fail-loud hint points to the canonical
// pre-staging path.
func TestRunInit_NonInteractive_MissingToken_DirectsToSetCredential(t *testing.T) {
	credtest.Hermetic(t)
	opts := &root.Options{
		Output:         "table",
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(""),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	err := runInit(context.Background(), opts, "https://acme.atlassian.net", "u@x.io", "", "", true)
	testutil.RequireError(t, err)
	if !strings.Contains(err.Error(), "set-credential") {
		t.Fatalf("error must direct user to set-credential, got: %v", err)
	}
}

// finalizeInit tests use t.TempDir() for paths and an httptest-backed
// clientBuilder so the user's real config is never touched and no real
// network call is made.

func newFinalizeReconcileResult() *reconcileResult {
	return &reconcileResult{store: &credstore.Store{}}
}

func newFinalizeOpts() *root.Options {
	return &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
}

func userResponseServer(t *testing.T, body string, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/wiki/rest/api/user/current", r.URL.Path)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

// TestFinalizeInit_WritesConnectionToDefault verifies the §2.2
// (MON-5328) single-source model: connection always lands in the shared
// `default` section (no per-tool override target), the token NEVER
// touches the plaintext store (keyring only, single api_token), and the
// cfl section carries no connection fields.
func TestFinalizeInit_WritesConnectionToDefault(t *testing.T) {
	credtest.Hermetic(t) // t.Setenv → no t.Parallel
	server := userResponseServer(t, `{"accountId":"abc","displayName":"X","email":"x@e"}`, http.StatusOK)
	defer server.Close()
	build := func(_ *config.Config) (*api.Client, error) {
		return api.NewClient(server.URL, "x@e", "tok"), nil
	}
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	cfg := &config.Config{URL: server.URL + "/wiki", Email: "x@e", APIToken: "tok", DefaultSpace: "SP"}
	result := &reconcileResult{store: &credstore.Store{}}

	testutil.RequireNoError(t,
		finalizeInit(context.Background(), newFinalizeOpts(), cfg, result, configPath, false, build))

	loaded, err := credstore.Load(configPath)
	testutil.RequireNoError(t, err)

	// Connection in default; token never in plaintext.
	testutil.Equal(t, "", loaded.Default.APIToken)
	testutil.Equal(t, "x@e", loaded.Default.Email)
	testutil.Equal(t, server.URL, loaded.Default.URL) // /wiki stripped
	// cfl section carries only the non-secret default, no connection.
	testutil.Equal(t, "SP", loaded.CFL.DefaultSpace)

	// Raw file must contain no api_token and no per-tool connection key.
	raw, rerr := os.ReadFile(configPath) //nolint:gosec // test reads its own temp file
	testutil.RequireNoError(t, rerr)
	if strings.Contains(string(raw), "api_token") {
		t.Fatalf("plaintext store must never contain api_token:\n%s", raw)
	}

	s, err := keyring.OpenNoMigrate()
	testutil.RequireNoError(t, err)
	defer func() { _ = s.Close() }()
	ok, err := s.HasToken(keyring.KeyAPIToken)
	testutil.RequireNoError(t, err)
	testutil.True(t, ok)
}

func TestFinalizeInit_BasicHappyPath(t *testing.T) {
	credtest.Hermetic(t) // t.Setenv → no t.Parallel
	server := userResponseServer(t, `{"accountId":"abc123","displayName":"Rian Stockbower","email":"rian@example.com"}`, http.StatusOK)
	defer server.Close()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	opts := newFinalizeOpts()
	cfg := &config.Config{
		URL:      server.URL,
		Email:    "rian@example.com",
		APIToken: "test-token",
	}

	build := func(_ *config.Config) (*api.Client, error) {
		return api.NewClient(server.URL, "rian@example.com", "test-token"), nil
	}

	err := finalizeInit(context.Background(), opts, cfg, newFinalizeReconcileResult(), configPath, false, build)
	testutil.RequireNoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()
	testutil.Contains(t, stdout, "Connected to")
	testutil.Contains(t, stdout, "Configuration saved to")
	testutil.Contains(t, stdout, "abc123 | Rian Stockbower | rian@example.com")

	_, err = os.Stat(configPath)
	testutil.RequireNoError(t, err)
}

func TestFinalizeInit_BearerHappyPath(t *testing.T) {
	credtest.Hermetic(t) // t.Setenv → no t.Parallel
	// Server asserts that the verify request actually carries a Bearer
	// Authorization header — i.e. the bearer code path emits bearer auth on
	// the wire, not just bearer-themed UI copy.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/wiki/rest/api/user/current", r.URL.Path)
		testutil.Equal(t, "Bearer scoped-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accountId":"svc456","displayName":"Service Account","email":"svc@example.com"}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	opts := newFinalizeOpts()
	cfg := &config.Config{
		URL:        server.URL,
		APIToken:   "scoped-token",
		AuthMethod: auth.AuthMethodBearer,
		CloudID:    "test-cloud-id",
	}

	// Construct a real bearer-style client (auth header injected via Options)
	// pointed at the httptest URL. This mirrors what api.NewBearerClient
	// produces, just with a routable base URL.
	build := func(c *config.Config) (*api.Client, error) {
		return &api.Client{
			Client: sharedclient.New(server.URL, "", "", &sharedclient.Options{
				AuthHeader: auth.BearerAuthHeader(c.APIToken),
			}),
		}, nil
	}

	err := finalizeInit(context.Background(), opts, cfg, newFinalizeReconcileResult(), configPath, false, build)
	testutil.RequireNoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()
	testutil.Contains(t, stdout, "Connected to")
	testutil.Contains(t, stdout, "Configuration saved to")
	testutil.Contains(t, stdout, "svc456 | Service Account | svc@example.com")
	testutil.Contains(t, stdout, "switch back to basic auth")

	_, err = os.Stat(configPath)
	testutil.RequireNoError(t, err)
}

// TestDefaultClientBuilder verifies the production wiring between
// cfg.AuthMethod and which client constructor runs. In the finalizeInit
// tests the builder is always replaced; this test pins the default.
func TestDefaultClientBuilder(t *testing.T) {
	t.Parallel()

	t.Run("basic constructs basic-auth client", func(t *testing.T) {
		t.Parallel()
		cfg := &config.Config{
			URL:      "https://example.atlassian.net",
			Email:    "user@example.com",
			APIToken: "secret",
		}
		c, err := defaultClientBuilder(cfg)
		testutil.RequireNoError(t, err)
		testutil.Equal(t, "https://example.atlassian.net", c.BaseURL)
		// Basic auth header is "Basic <base64(email:token)>"; presence of the
		// "Basic " prefix is enough to confirm dispatch.
		testutil.True(t, strings.HasPrefix(c.AuthHeader, "Basic "), "expected Basic prefix, got: "+c.AuthHeader)
	})

	t.Run("bearer constructs bearer-auth client at gateway", func(t *testing.T) {
		t.Parallel()
		cfg := &config.Config{
			APIToken:   "scoped-token",
			AuthMethod: auth.AuthMethodBearer,
			CloudID:    "cloud-abc",
		}
		c, err := defaultClientBuilder(cfg)
		testutil.RequireNoError(t, err)
		testutil.Contains(t, c.BaseURL, "/ex/confluence/cloud-abc/wiki")
		testutil.Equal(t, "Bearer scoped-token", c.AuthHeader)
	})

	t.Run("bearer rejects empty cloud ID", func(t *testing.T) {
		t.Parallel()
		cfg := &config.Config{
			APIToken:   "scoped-token",
			AuthMethod: auth.AuthMethodBearer,
		}
		_, err := defaultClientBuilder(cfg)
		testutil.RequireError(t, err)
	})
}

func TestFinalizeInit_AuthFailure(t *testing.T) {
	t.Parallel()
	server := userResponseServer(t, `{"message":"Unauthorized"}`, http.StatusUnauthorized)
	defer server.Close()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	opts := newFinalizeOpts()
	cfg := &config.Config{
		URL:      server.URL,
		Email:    "rian@example.com",
		APIToken: "wrong-token",
	}

	build := func(_ *config.Config) (*api.Client, error) {
		return api.NewClient(server.URL, "rian@example.com", "wrong-token"), nil
	}

	err := finalizeInit(context.Background(), opts, cfg, newFinalizeReconcileResult(), configPath, false, build)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "authentication failed")

	// Both the error and the remediation hint must land on stderr — splitting
	// them across stdout/stderr would mean a script capturing only stderr
	// sees the failure with no actionable next step.
	stderr := opts.Stderr.(*bytes.Buffer).String()
	testutil.Contains(t, stderr, "Connection failed")
	testutil.Contains(t, stderr, "Check your credentials and try again")

	_, statErr := os.Stat(configPath)
	testutil.True(t, os.IsNotExist(statErr), "config file should not exist after auth failure")
}

func TestFinalizeInit_BuildFailureSurfacesError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	opts := newFinalizeOpts()
	cfg := &config.Config{
		URL:      "https://example.atlassian.net",
		Email:    "rian@example.com",
		APIToken: "test-token",
	}

	build := func(_ *config.Config) (*api.Client, error) {
		return nil, errors.New("simulated builder failure")
	}

	err := finalizeInit(context.Background(), opts, cfg, newFinalizeReconcileResult(), configPath, false, build)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "simulated builder failure")

	// User must see WHY init failed, not just a non-zero exit.
	stderr := opts.Stderr.(*bytes.Buffer).String()
	testutil.Contains(t, stderr, "Could not construct API client")

	_, statErr := os.Stat(configPath)
	testutil.True(t, os.IsNotExist(statErr), "config should not be saved when builder fails")
}

func TestFinalizeInit_NoVerify(t *testing.T) {
	credtest.Hermetic(t) // t.Setenv → no t.Parallel
	httpCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		httpCalled = true
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	opts := newFinalizeOpts()
	cfg := &config.Config{
		URL:      server.URL,
		Email:    "rian@example.com",
		APIToken: "test-token",
	}

	// Track builder invocation directly. If the noVerify guard regresses
	// (e.g. moves below the build call), the server-not-called assertion
	// alone wouldn't catch it — the builder running but never being used
	// would still leave httpCalled=false.
	builderCalled := false
	build := func(_ *config.Config) (*api.Client, error) {
		builderCalled = true
		return api.NewClient(server.URL, "rian@example.com", "test-token"), nil
	}

	err := finalizeInit(context.Background(), opts, cfg, newFinalizeReconcileResult(), configPath, true, build)
	testutil.RequireNoError(t, err)

	testutil.False(t, builderCalled, "clientBuilder should not be invoked when --no-verify is set")
	testutil.False(t, httpCalled, "no API call should be made when --no-verify is set")

	stdout := opts.Stdout.(*bytes.Buffer).String()
	testutil.Contains(t, stdout, "Configuration saved to")
	// No verify → no "Connected to" confirmation, no user one-liner.
	testutil.False(t, strings.Contains(stdout, "Connected to"), "verify confirmation should not appear without verify")

	_, err = os.Stat(configPath)
	testutil.RequireNoError(t, err)
}
