package configcmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
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

func TestNewTestCmd_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/myself")
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
	require.NoError(t, err)
	opts.SetAPIClient(client)

	cmd := newTestCmd(opts)
	err = cmd.Execute()
	require.NoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()
	assert.Contains(t, stdout, "Authentication successful")
	assert.Contains(t, stdout, "API access verified")
	assert.Contains(t, stdout, "Test User")
}

func TestNewTestCmd_AuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	require.NoError(t, err)
	opts.SetAPIClient(client)

	cmd := newTestCmd(opts)
	err = cmd.Execute()
	// Command doesn't return error, it prints error message
	require.NoError(t, err)

	// Error messages go to stderr
	stderr := opts.Stderr.(*bytes.Buffer).String()
	assert.Contains(t, stderr, "Authentication failed")
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
	require.NoError(t, err)

	// Error messages go to stderr
	stderr := opts.Stderr.(*bytes.Buffer).String()
	assert.Contains(t, stderr, "No Jira URL configured")
}

func TestMaskToken(t *testing.T) {
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
			got := maskToken(tt.token)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRunClear_WithConfirmation(t *testing.T) {
	// Create a temp config file
	// On macOS, UserConfigDir() returns ~/Library/Application Support
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	configDir := filepath.Join(homeDir, "Library", "Application Support", "jira-ticket-cli")
	require.NoError(t, os.MkdirAll(configDir, 0700))
	configPath := filepath.Join(configDir, "config.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{}`), 0600))

	opts := newTestRootOptions()
	clearOpts := &clearOptions{
		Options: opts,
		force:   false,
		stdin:   strings.NewReader("y\n"),
	}

	err := runClear(clearOpts)
	require.NoError(t, err)

	// Verify file was deleted
	_, err = os.Stat(configPath)
	assert.True(t, os.IsNotExist(err))

	stdout := opts.Stdout.(*bytes.Buffer).String()
	assert.Contains(t, stdout, "Configuration file removed")
}

func TestRunClear_Cancelled(t *testing.T) {
	// Create a temp config file
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	configDir := filepath.Join(homeDir, "Library", "Application Support", "jira-ticket-cli")
	require.NoError(t, os.MkdirAll(configDir, 0700))
	configPath := filepath.Join(configDir, "config.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{}`), 0600))

	opts := newTestRootOptions()
	clearOpts := &clearOptions{
		Options: opts,
		force:   false,
		stdin:   strings.NewReader("n\n"),
	}

	err := runClear(clearOpts)
	require.NoError(t, err)

	// Verify file still exists
	_, err = os.Stat(configPath)
	assert.NoError(t, err)
}

func TestRunClear_Force(t *testing.T) {
	// Create a temp config file
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	configDir := filepath.Join(homeDir, "Library", "Application Support", "jira-ticket-cli")
	require.NoError(t, os.MkdirAll(configDir, 0700))
	configPath := filepath.Join(configDir, "config.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{}`), 0600))

	opts := newTestRootOptions()
	clearOpts := &clearOptions{
		Options: opts,
		force:   true,
		stdin:   strings.NewReader(""), // No input needed with --force
	}

	err := runClear(clearOpts)
	require.NoError(t, err)

	// Verify file was deleted
	_, err = os.Stat(configPath)
	assert.True(t, os.IsNotExist(err))
}

func TestGetDefaultProjectSource(t *testing.T) {
	// Clear env vars
	t.Setenv("JIRA_DEFAULT_PROJECT", "")

	// Use temp home dir (macOS uses ~/Library/Application Support)
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// No config, no env
	assert.Equal(t, "-", getDefaultProjectSource())

	// With env var
	t.Setenv("JIRA_DEFAULT_PROJECT", "PROJ")
	assert.Equal(t, "env (JIRA_DEFAULT_PROJECT)", getDefaultProjectSource())
}
