package configcmd

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/credtest"
	"github.com/open-cli-collective/atlassian-go/keyring"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
)

func newTestRootOptions() *root.Options {
	return &root.Options{
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
		Stdin:   strings.NewReader(""),
	}
}

func TestShowCmd_TableOutput(t *testing.T) {
	credtest.Hermetic(t) // deterministic file keyring; clears token env
	t.Setenv("JIRA_URL", "https://test.atlassian.net")
	t.Setenv("JIRA_EMAIL", "test@example.com")
	t.Setenv("JIRA_API_TOKEN", "token123456")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("ATLASSIAN_EMAIL", "")

	opts := newTestRootOptions()

	cmd := newShowCmd(opts)
	err := cmd.Execute()
	testutil.RequireNoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()
	testutil.Contains(t, stdout, "Config file:")
	testutil.Contains(t, stdout, "test@example.com")
	// The token value (here from env) must never be rendered.
	testutil.NotContains(t, stdout, "token123456")
	testutil.Contains(t, stdout, "configured")
}

func TestRunClear_All(t *testing.T) {
	xdg := credtest.Hermetic(t)
	credtest.SeedToken(t, "shared-secret")

	sharedPath := filepath.Join(xdg, "atlassian-cli", "config.yml")
	testutil.RequireNoError(t, os.MkdirAll(filepath.Dir(sharedPath), 0o700))
	testutil.RequireNoError(t, os.WriteFile(sharedPath, []byte("default:\n  url: https://x\n"), 0o600))

	opts, _, _ := newClearOpts(t, true, "")
	opts.all = true
	testutil.RequireNoError(t, runClear(context.Background(), opts))

	testutil.False(t, jtkTokenPresent(t, keyring.KeyAPIToken))
	_, statErr := os.Stat(sharedPath)
	testutil.True(t, os.IsNotExist(statErr))
}

func TestNewTestCmd_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Contains(t, r.URL.Path, "/myself")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accountId": "123", "displayName": "Test User", "emailAddress": "test@example.com"}`))
	}))
	defer server.Close()

	// Clear any real env vars and set test vars
	t.Setenv("JIRA_URL", server.URL)
	t.Setenv("JIRA_EMAIL", "test@example.com")
	t.Setenv("JIRA_API_TOKEN", "token123")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("ATLASSIAN_EMAIL", "")
	t.Setenv("ATLASSIAN_API_TOKEN", "")

	opts := newTestRootOptions()
	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token123",
	})
	testutil.RequireNoError(t, err)
	opts.SetAPIClient(client)

	cmd := newTestCmd(opts)
	err = cmd.Execute()
	testutil.RequireNoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()
	testutil.Contains(t, stdout, "Authentication successful")
	testutil.Contains(t, stdout, "API access verified")
	testutil.Contains(t, stdout, "Test User")
}

func TestNewTestCmd_AuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message": "Unauthorized"}`))
	}))
	defer server.Close()

	// Clear any real env vars and set test vars
	t.Setenv("JIRA_URL", server.URL)
	t.Setenv("JIRA_EMAIL", "test@example.com")
	t.Setenv("JIRA_API_TOKEN", "bad-token")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("ATLASSIAN_EMAIL", "")
	t.Setenv("ATLASSIAN_API_TOKEN", "")

	opts := newTestRootOptions()
	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "bad-token",
	})
	testutil.RequireNoError(t, err)
	opts.SetAPIClient(client)

	cmd := newTestCmd(opts)
	err = cmd.Execute()
	// Command doesn't return error, it prints error message
	testutil.RequireNoError(t, err)

	// Error messages go to stderr
	stderr := opts.Stderr.(*bytes.Buffer).String()
	testutil.Contains(t, stderr, "Authentication failed")
}

func TestNewTestCmd_NoURL(t *testing.T) {
	// Clear ALL URL env vars
	t.Setenv("JIRA_URL", "")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("JIRA_DOMAIN", "")
	t.Setenv("JIRA_EMAIL", "")
	t.Setenv("JIRA_API_TOKEN", "")
	t.Setenv("ATLASSIAN_EMAIL", "")
	t.Setenv("ATLASSIAN_API_TOKEN", "")

	// Use temp config dir to avoid picking up real config
	// Must set both HOME and XDG_CONFIG_HOME for cross-platform support
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	opts := newTestRootOptions()

	cmd := newTestCmd(opts)
	err := cmd.Execute()
	testutil.RequireNoError(t, err)

	// Error messages go to stderr
	stderr := opts.Stderr.(*bytes.Buffer).String()
	testutil.Contains(t, stderr, "No Jira URL configured")
}

func newClearOpts(t *testing.T, force bool, stdin string) (*clearOptions, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	out, errBuf := &bytes.Buffer{}, &bytes.Buffer{}
	return &clearOptions{
		Options: &root.Options{NoColor: true, Stdout: out, Stderr: errBuf, Stdin: strings.NewReader("")},
		force:   force,
		stdin:   strings.NewReader(stdin),
	}, out, errBuf
}

func jtkTokenPresent(t *testing.T, key string) bool {
	t.Helper()
	s, err := keyring.OpenNoMigrate()
	testutil.RequireNoError(t, err)
	defer func() { _ = s.Close() }()
	ok, err := s.HasToken(key)
	testutil.RequireNoError(t, err)
	return ok
}

func TestRunClear_DeletesSharedKey_Confirmed(t *testing.T) {
	credtest.Hermetic(t)
	credtest.SeedToken(t, "shared-secret")

	opts, _, errBuf := newClearOpts(t, false, "y\n")
	testutil.RequireNoError(t, runClear(context.Background(), opts))

	testutil.False(t, jtkTokenPresent(t, keyring.KeyAPIToken))
	testutil.Contains(t, errBuf.String(), "cfl will also lose access")
	// Removed per-tool override keys must never be advised again.
	testutil.NotContains(t, errBuf.String(), "jtk_api_token")
	testutil.NotContains(t, errBuf.String(), "override")
	// §1.11.11 via the REAL command flow: exactly empty (no stray
	// deprecated key survives a default clear).
	testutil.Equal(t, 0, len(credtest.BundleKeys(t)))
}

func TestRunClear_Cancelled(t *testing.T) {
	credtest.Hermetic(t)
	credtest.SeedToken(t, "shared-secret")

	opts, _, _ := newClearOpts(t, false, "n\n")
	testutil.RequireNoError(t, runClear(context.Background(), opts))

	testutil.True(t, jtkTokenPresent(t, keyring.KeyAPIToken))
}

func TestRunClear_Force_DeletesSharedKey(t *testing.T) {
	credtest.Hermetic(t)
	credtest.SeedToken(t, "shared-secret")

	opts, _, _ := newClearOpts(t, true, "")
	testutil.RequireNoError(t, runClear(context.Background(), opts))

	// One key per logical credential (§1.11.10): the force path deletes
	// the single shared api_token without prompting.
	testutil.False(t, jtkTokenPresent(t, keyring.KeyAPIToken))
}

func TestRunClear_NothingToClear(t *testing.T) {
	credtest.Hermetic(t)
	opts, out, _ := newClearOpts(t, true, "")
	testutil.RequireNoError(t, runClear(context.Background(), opts))
	testutil.Contains(t, out.String(), "nothing to clear")
}

func TestGetDefaultProjectWithSource(t *testing.T) {
	// Clear env vars
	t.Setenv("JIRA_DEFAULT_PROJECT", "")

	// Use temp dir for cross-platform behavior
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	// No config, no env
	_, source := config.GetDefaultProjectWithSource()
	testutil.Equal(t, source, "-")

	// With env var
	t.Setenv("JIRA_DEFAULT_PROJECT", "PROJ")
	_, source = config.GetDefaultProjectWithSource()
	testutil.Equal(t, source, "env (JIRA_DEFAULT_PROJECT)")
}

func TestGetAuthMethodWithSource(t *testing.T) {
	t.Setenv("JIRA_AUTH_METHOD", "")
	t.Setenv("ATLASSIAN_AUTH_METHOD", "")

	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	// No config, no env → default
	_, source := config.GetAuthMethodWithSource()
	testutil.Equal(t, source, "default")

	// With JIRA_AUTH_METHOD env var
	t.Setenv("JIRA_AUTH_METHOD", "bearer")
	_, source = config.GetAuthMethodWithSource()
	testutil.Equal(t, source, "env (JIRA_AUTH_METHOD)")

	// With ATLASSIAN_AUTH_METHOD fallback
	t.Setenv("JIRA_AUTH_METHOD", "")
	t.Setenv("ATLASSIAN_AUTH_METHOD", "bearer")
	_, source = config.GetAuthMethodWithSource()
	testutil.Equal(t, source, "env (ATLASSIAN_AUTH_METHOD)")

	// Invalid value is ignored, falls through to default
	t.Setenv("JIRA_AUTH_METHOD", "Bearer")
	t.Setenv("ATLASSIAN_AUTH_METHOD", "")
	val, source := config.GetAuthMethodWithSource()
	testutil.Equal(t, val, "basic")
	testutil.Equal(t, source, "default")
}

func TestGetCloudIDWithSource(t *testing.T) {
	t.Setenv("JIRA_CLOUD_ID", "")
	t.Setenv("ATLASSIAN_CLOUD_ID", "")

	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	// No config, no env
	_, source := config.GetCloudIDWithSource()
	testutil.Equal(t, source, "-")

	// With JIRA_CLOUD_ID env var
	t.Setenv("JIRA_CLOUD_ID", "cloud-123")
	_, source = config.GetCloudIDWithSource()
	testutil.Equal(t, source, "env (JIRA_CLOUD_ID)")

	// With ATLASSIAN_CLOUD_ID fallback
	t.Setenv("JIRA_CLOUD_ID", "")
	t.Setenv("ATLASSIAN_CLOUD_ID", "shared-cloud")
	_, source = config.GetCloudIDWithSource()
	testutil.Equal(t, source, "env (ATLASSIAN_CLOUD_ID)")
}

func TestNewTestCmd_BearerAuth_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify bearer auth header is sent (constructed by api.New with bearer config)
		authHeader := r.Header.Get("Authorization")
		testutil.Equal(t, "Bearer scoped-token", authHeader)

		testutil.Contains(t, r.URL.Path, "/myself")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accountId": "123", "displayName": "Service Account", "emailAddress": ""}`))
	}))
	defer server.Close()

	t.Setenv("JIRA_URL", server.URL)
	t.Setenv("JIRA_AUTH_METHOD", "bearer")
	t.Setenv("JIRA_CLOUD_ID", "test-cloud")
	t.Setenv("JIRA_API_TOKEN", "scoped-token")
	t.Setenv("JIRA_EMAIL", "")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("ATLASSIAN_EMAIL", "")
	t.Setenv("ATLASSIAN_API_TOKEN", "")
	t.Setenv("ATLASSIAN_AUTH_METHOD", "")
	t.Setenv("ATLASSIAN_CLOUD_ID", "")

	opts := newTestRootOptions()

	// Create a real bearer auth client via api.New to exercise the full bearer
	// construction path, then redirect both BaseURLs to the test server.
	client, err := api.New(api.ClientConfig{
		URL:        server.URL,
		APIToken:   "scoped-token",
		AuthMethod: "bearer",
		CloudID:    "test-cloud",
	})
	testutil.RequireNoError(t, err)
	// Point both outer and embedded BaseURL at the test server so either
	// code path (absolute URL construction or embedded client methods) works.
	testBaseURL := server.URL + "/rest/api/3"
	client.BaseURL = testBaseURL
	client.Client.BaseURL = testBaseURL
	opts.SetAPIClient(client)

	cmd := newTestCmd(opts)
	err = cmd.Execute()
	testutil.RequireNoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()
	testutil.Contains(t, stdout, "Authentication successful")
	testutil.Contains(t, stdout, "Service Account")
}
