package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestConfig creates a temporary config directory for testing
// Uses t.Setenv for automatic cleanup and t.TempDir for automatic removal
func setupTestConfig(t *testing.T) (string, func()) {
	t.Helper()

	// Use t.TempDir() which auto-cleans after test
	tempDir := t.TempDir()

	// Use t.Setenv which auto-restores after test (Go 1.17+)
	// XDG_CONFIG_HOME is used on Linux, HOME+Library/App Support on macOS
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("HOME", tempDir)

	// Clear any JIRA env vars that might interfere
	t.Setenv("JIRA_URL", "")
	t.Setenv("JIRA_DOMAIN", "")
	t.Setenv("JIRA_EMAIL", "")
	t.Setenv("JIRA_API_TOKEN", "")

	// Create macOS-style dir as well for fallback
	libDir := filepath.Join(tempDir, "Library", "Application Support")
	err := os.MkdirAll(libDir, 0700)
	require.NoError(t, err)

	// Return empty cleanup since t.TempDir and t.Setenv handle it
	return tempDir, func() {}
}

func TestConfig_SaveAndLoad(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	cfg := &Config{
		URL:      "https://example.atlassian.net",
		Email:    "test@example.com",
		APIToken: "secret-token",
	}

	// Save config
	err := Save(cfg)
	require.NoError(t, err)

	// Load config
	loaded, err := Load()
	require.NoError(t, err)

	assert.Equal(t, cfg.URL, loaded.URL)
	assert.Equal(t, cfg.Email, loaded.Email)
	assert.Equal(t, cfg.APIToken, loaded.APIToken)
}

func TestConfig_Load_NotExists(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Load when file doesn't exist should return empty config
	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Empty(t, cfg.URL)
	assert.Empty(t, cfg.Email)
	assert.Empty(t, cfg.APIToken)
}

func TestConfig_Clear(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Save config first
	cfg := &Config{
		URL:      "https://example.atlassian.net",
		Email:    "test@example.com",
		APIToken: "secret-token",
	}
	err := Save(cfg)
	require.NoError(t, err)

	// Clear config
	err = Clear()
	require.NoError(t, err)

	// Load should return empty config
	loaded, err := Load()
	require.NoError(t, err)
	assert.Empty(t, loaded.URL)
}

func TestConfig_Clear_NotExists(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Clear when file doesn't exist should not error
	err := Clear()
	assert.NoError(t, err)
}

func TestConfig_FilePermissions(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	cfg := &Config{
		URL:      "https://example.atlassian.net",
		Email:    "test@example.com",
		APIToken: "secret-token",
	}
	err := Save(cfg)
	require.NoError(t, err)

	// Check file permissions using Path() to get actual config location
	configFile := Path()
	info, err := os.Stat(configFile)
	require.NoError(t, err)

	// File should be 0600 (user read/write only)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestGetURL_EnvOverride(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Save config
	cfg := &Config{URL: "https://config.atlassian.net"}
	err := Save(cfg)
	require.NoError(t, err)

	// Without env, should return config value
	os.Unsetenv("JIRA_URL")
	assert.Equal(t, "https://config.atlassian.net", GetURL())

	// With env, should return env value
	os.Setenv("JIRA_URL", "https://env.atlassian.net")
	defer os.Unsetenv("JIRA_URL")
	assert.Equal(t, "https://env.atlassian.net", GetURL())
}

func TestGetURL_LegacyDomainFallback(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Save config with legacy domain only
	cfg := &Config{Domain: "legacy"}
	err := Save(cfg)
	require.NoError(t, err)

	// Should construct URL from legacy domain
	assert.Equal(t, "https://legacy.atlassian.net", GetURL())

	// JIRA_DOMAIN env should also work
	os.Setenv("JIRA_DOMAIN", "env-legacy")
	defer os.Unsetenv("JIRA_DOMAIN")
	assert.Equal(t, "https://env-legacy.atlassian.net", GetURL())
}

func TestGetURL_URLTakesPrecedence(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Save config with both URL and legacy domain
	cfg := &Config{
		URL:    "https://new-url.atlassian.net",
		Domain: "old-domain",
	}
	err := Save(cfg)
	require.NoError(t, err)

	// URL should take precedence
	assert.Equal(t, "https://new-url.atlassian.net", GetURL())
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"example.atlassian.net", "https://example.atlassian.net"},
		{"https://example.atlassian.net", "https://example.atlassian.net"},
		{"http://example.atlassian.net", "http://example.atlassian.net"},
		{"https://example.atlassian.net/", "https://example.atlassian.net"},
		{"example.atlassian.net/", "https://example.atlassian.net"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, NormalizeURL(tt.input))
		})
	}
}

func TestGetDomain_EnvOverride(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Save config
	cfg := &Config{Domain: "config-domain"}
	err := Save(cfg)
	require.NoError(t, err)

	// Without env, should return config value
	os.Unsetenv("JIRA_DOMAIN")
	assert.Equal(t, "config-domain", GetDomain())

	// With env, should return env value
	os.Setenv("JIRA_DOMAIN", "env-domain")
	defer os.Unsetenv("JIRA_DOMAIN")
	assert.Equal(t, "env-domain", GetDomain())
}

func TestGetEmail_EnvOverride(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Save config
	cfg := &Config{Email: "config@example.com"}
	err := Save(cfg)
	require.NoError(t, err)

	// Without env, should return config value
	os.Unsetenv("JIRA_EMAIL")
	assert.Equal(t, "config@example.com", GetEmail())

	// With env, should return env value
	os.Setenv("JIRA_EMAIL", "env@example.com")
	defer os.Unsetenv("JIRA_EMAIL")
	assert.Equal(t, "env@example.com", GetEmail())
}

func TestGetAPIToken_EnvOverride(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Save config
	cfg := &Config{APIToken: "config-token"}
	err := Save(cfg)
	require.NoError(t, err)

	// Without env, should return config value
	os.Unsetenv("JIRA_API_TOKEN")
	assert.Equal(t, "config-token", GetAPIToken())

	// With env, should return env value
	os.Setenv("JIRA_API_TOKEN", "env-token")
	defer os.Unsetenv("JIRA_API_TOKEN")
	assert.Equal(t, "env-token", GetAPIToken())
}

func TestIsConfigured(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Not configured initially
	assert.False(t, IsConfigured())

	// Partially configured (URL only)
	cfg := &Config{URL: "https://test.atlassian.net"}
	Save(cfg)
	assert.False(t, IsConfigured())

	// Fully configured with URL
	cfg = &Config{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "token",
	}
	Save(cfg)
	assert.True(t, IsConfigured())
}

func TestIsConfigured_LegacyDomain(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Fully configured with legacy domain
	cfg := &Config{
		Domain:   "test",
		Email:    "test@example.com",
		APIToken: "token",
	}
	Save(cfg)
	assert.True(t, IsConfigured())
}

func TestIsConfigured_EnvOnly(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Set all env vars with JIRA_URL
	os.Setenv("JIRA_URL", "https://env.atlassian.net")
	os.Setenv("JIRA_EMAIL", "env@example.com")
	os.Setenv("JIRA_API_TOKEN", "env-token")
	defer func() {
		os.Unsetenv("JIRA_URL")
		os.Unsetenv("JIRA_EMAIL")
		os.Unsetenv("JIRA_API_TOKEN")
	}()

	// Should be configured via env vars only
	assert.True(t, IsConfigured())
}

func TestIsConfigured_LegacyEnvOnly(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Set all env vars with legacy JIRA_DOMAIN
	os.Setenv("JIRA_DOMAIN", "env-domain")
	os.Setenv("JIRA_EMAIL", "env@example.com")
	os.Setenv("JIRA_API_TOKEN", "env-token")
	defer func() {
		os.Unsetenv("JIRA_DOMAIN")
		os.Unsetenv("JIRA_EMAIL")
		os.Unsetenv("JIRA_API_TOKEN")
	}()

	// Should be configured via legacy env vars
	assert.True(t, IsConfigured())
}

func TestPath(t *testing.T) {
	path := Path()
	assert.Contains(t, path, configDirName)
	assert.Contains(t, path, configFileName)
}
