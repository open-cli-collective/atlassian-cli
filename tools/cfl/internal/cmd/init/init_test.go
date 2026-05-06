package init

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/auth"
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

// finalizeInit tests use t.TempDir() for configPath and an httptest-backed
// clientBuilder so the user's real config is never touched and no real
// network call is made.

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

func TestFinalizeInit_BasicHappyPath(t *testing.T) {
	t.Parallel()
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

	err := finalizeInit(context.Background(), opts, cfg, configPath, false, build)
	testutil.RequireNoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()
	testutil.Contains(t, stdout, "Connected to")
	testutil.Contains(t, stdout, "Configuration saved to")
	testutil.Contains(t, stdout, "abc123 | Rian Stockbower | rian@example.com")

	_, err = os.Stat(configPath)
	testutil.RequireNoError(t, err)
}

func TestFinalizeInit_BearerHappyPath(t *testing.T) {
	t.Parallel()
	server := userResponseServer(t, `{"accountId":"svc456","displayName":"Service Account","email":"svc@example.com"}`, http.StatusOK)
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

	// Builder returns the same httptest-backed client regardless of auth method.
	// This exercises finalizeInit's bearer control flow without depending on
	// api.NewBearerClient's hardcoded gateway URL.
	build := func(_ *config.Config) (*api.Client, error) {
		return api.NewClient(server.URL, "", "scoped-token"), nil
	}

	err := finalizeInit(context.Background(), opts, cfg, configPath, false, build)
	testutil.RequireNoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()
	testutil.Contains(t, stdout, "Connected to")
	testutil.Contains(t, stdout, "Configuration saved to")
	testutil.Contains(t, stdout, "svc456 | Service Account | svc@example.com")
	testutil.Contains(t, stdout, "switch back to basic auth")

	_, err = os.Stat(configPath)
	testutil.RequireNoError(t, err)
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

	err := finalizeInit(context.Background(), opts, cfg, configPath, false, build)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "authentication failed")

	stderr := opts.Stderr.(*bytes.Buffer).String()
	testutil.Contains(t, stderr, "Connection failed")

	_, statErr := os.Stat(configPath)
	testutil.True(t, os.IsNotExist(statErr), "config file should not exist after auth failure")
}

func TestFinalizeInit_NoVerify(t *testing.T) {
	t.Parallel()
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
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

	build := func(_ *config.Config) (*api.Client, error) {
		return api.NewClient(server.URL, "rian@example.com", "test-token"), nil
	}

	err := finalizeInit(context.Background(), opts, cfg, configPath, true, build)
	testutil.RequireNoError(t, err)

	testutil.False(t, called, "no API call should be made when --no-verify is set")

	stdout := opts.Stdout.(*bytes.Buffer).String()
	testutil.Contains(t, stdout, "Configuration saved to")
	// No verify → no fetched user → no one-liner rendered.
	testutil.False(t, strings.Contains(stdout, " | "), "user one-liner should not appear without verify")

	_, err = os.Stat(configPath)
	testutil.RequireNoError(t, err)
}
