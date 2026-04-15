package configcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
)

func newTestRootOptions() *root.Options {
	return &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
		Stdin:   strings.NewReader(""),
	}
}

func clearAuthEnvVars(t *testing.T) {
	t.Helper()
	// Set to empty string rather than unsetting — both os.Getenv and os.LookupEnv
	// treat empty as "not configured" in our env var precedence logic.
	for _, key := range []string{"JIRA_AUTH_METHOD", "JIRA_CLOUD_ID", "ATLASSIAN_AUTH_METHOD", "ATLASSIAN_CLOUD_ID"} {
		t.Setenv(key, "")
	}
}

func TestShowCmd_JSONOutput(t *testing.T) {
	t.Setenv("JIRA_URL", "https://test.atlassian.net")
	t.Setenv("JIRA_EMAIL", "test@example.com")
	t.Setenv("JIRA_API_TOKEN", "token123456")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("ATLASSIAN_EMAIL", "")
	t.Setenv("ATLASSIAN_API_TOKEN", "")
	clearAuthEnvVars(t)

	opts := &root.Options{
		Output:  "json",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
		Stdin:   strings.NewReader(""),
	}

	cmd := newShowCmd(opts)
	err := cmd.Execute()
	testutil.RequireNoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()

	// The entire stdout must be valid JSON — no trailing plain text
	var parsed map[string]any
	err = json.Unmarshal([]byte(stdout), &parsed)
	testutil.RequireNoError(t, err)

	testutil.Equal(t, parsed["url"], "https://test.atlassian.net")
	testutil.Equal(t, parsed["email"], "test@example.com")
	testutil.Equal(t, parsed["auth_method"], "basic")
	testutil.Equal(t, parsed["cloud_id"], "")
	testutil.NotContains(t, stdout, "Config file:")
}

func TestShowCmd_TableOutput(t *testing.T) {
	t.Setenv("JIRA_URL", "https://test.atlassian.net")
	t.Setenv("JIRA_EMAIL", "test@example.com")
	t.Setenv("JIRA_API_TOKEN", "token123456")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("ATLASSIAN_EMAIL", "")
	t.Setenv("ATLASSIAN_API_TOKEN", "")

	opts := newTestRootOptions()

	cmd := newShowCmd(opts)
	err := cmd.Execute()
	testutil.RequireNoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()
	testutil.Contains(t, stdout, "Config file:")
	testutil.Contains(t, stdout, "test@example.com")
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

func TestMaskToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{"normal token", "abcd1234567890wxyz", "abcd********wxyz"},
		{"short token", "abc", "********"},
		{"exactly 8 chars", "12345678", "********"},
		{"9 chars", "123456789", "1234********6789"},
		{"empty token", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := maskToken(tt.token)
			testutil.Equal(t, got, tt.want)
		})
	}
}

func getConfigDir(t *testing.T) string {
	// os.UserConfigDir() returns different paths per platform:
	// - macOS: $HOME/Library/Application Support
	// - Linux: $XDG_CONFIG_HOME or $HOME/.config
	// We set HOME and let the config package derive the path
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempDir, ".config"))

	// Get the actual config path the config package will use
	configPath := config.Path()
	return filepath.Dir(configPath)
}

func TestRunClear_WithConfirmation(t *testing.T) {
	configDir := getConfigDir(t)
	testutil.RequireNoError(t, os.MkdirAll(configDir, 0700))
	configPath := filepath.Join(configDir, "config.json")
	testutil.RequireNoError(t, os.WriteFile(configPath, []byte(`{}`), 0600))

	opts := newTestRootOptions()
	clearOpts := &clearOptions{
		Options: opts,
		force:   false,
		stdin:   strings.NewReader("y\n"),
	}

	err := runClear(context.Background(), clearOpts)
	testutil.RequireNoError(t, err)

	// Verify file was deleted
	_, err = os.Stat(configPath)
	testutil.True(t, os.IsNotExist(err))

	stdout := opts.Stdout.(*bytes.Buffer).String()
	testutil.Contains(t, stdout, "Configuration file removed")
}

func TestRunClear_Cancelled(t *testing.T) {
	configDir := getConfigDir(t)
	testutil.RequireNoError(t, os.MkdirAll(configDir, 0700))
	configPath := filepath.Join(configDir, "config.json")
	testutil.RequireNoError(t, os.WriteFile(configPath, []byte(`{}`), 0600))

	opts := newTestRootOptions()
	clearOpts := &clearOptions{
		Options: opts,
		force:   false,
		stdin:   strings.NewReader("n\n"),
	}

	err := runClear(context.Background(), clearOpts)
	testutil.RequireNoError(t, err)

	// Verify file still exists
	_, err = os.Stat(configPath)
	testutil.NoError(t, err)
}

func TestRunClear_Force(t *testing.T) {
	configDir := getConfigDir(t)
	testutil.RequireNoError(t, os.MkdirAll(configDir, 0700))
	configPath := filepath.Join(configDir, "config.json")
	testutil.RequireNoError(t, os.WriteFile(configPath, []byte(`{}`), 0600))

	opts := newTestRootOptions()
	clearOpts := &clearOptions{
		Options: opts,
		force:   true,
		stdin:   strings.NewReader(""), // No input needed with --force
	}

	err := runClear(context.Background(), clearOpts)
	testutil.RequireNoError(t, err)

	// Verify file was deleted
	_, err = os.Stat(configPath)
	testutil.True(t, os.IsNotExist(err))
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

func TestShowCmd_BearerAuth_JSONOutput(t *testing.T) {
	t.Setenv("JIRA_URL", "https://test.atlassian.net")
	t.Setenv("JIRA_API_TOKEN", "scoped-token-123")
	t.Setenv("JIRA_AUTH_METHOD", "bearer")
	t.Setenv("JIRA_CLOUD_ID", "cloud-abc-123")
	t.Setenv("JIRA_EMAIL", "")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("ATLASSIAN_EMAIL", "")
	t.Setenv("ATLASSIAN_API_TOKEN", "")
	t.Setenv("ATLASSIAN_AUTH_METHOD", "")
	t.Setenv("ATLASSIAN_CLOUD_ID", "")

	opts := &root.Options{
		Output:  "json",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
		Stdin:   strings.NewReader(""),
	}

	cmd := newShowCmd(opts)
	err := cmd.Execute()
	testutil.RequireNoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()
	var parsed map[string]any
	err = json.Unmarshal([]byte(stdout), &parsed)
	testutil.RequireNoError(t, err)

	testutil.Equal(t, parsed["auth_method"], "bearer")
	testutil.Equal(t, parsed["cloud_id"], "cloud-abc-123")
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
