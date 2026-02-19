package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/open-cli-collective/atlassian-go/url"
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

	// Clear any JIRA and ATLASSIAN env vars that might interfere
	t.Setenv("JIRA_URL", "")
	t.Setenv("JIRA_DOMAIN", "")
	t.Setenv("JIRA_EMAIL", "")
	t.Setenv("JIRA_API_TOKEN", "")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("ATLASSIAN_EMAIL", "")
	t.Setenv("ATLASSIAN_API_TOKEN", "")

	// Create macOS-style dir as well for fallback
	libDir := filepath.Join(tempDir, "Library", "Application Support")
	err := os.MkdirAll(libDir, 0700)
	testutil.RequireNoError(t, err)

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
	testutil.RequireNoError(t, err)

	// Load config
	loaded, err := Load()
	testutil.RequireNoError(t, err)

	testutil.Equal(t, loaded.URL, cfg.URL)
	testutil.Equal(t, loaded.Email, cfg.Email)
	testutil.Equal(t, loaded.APIToken, cfg.APIToken)
}

func TestConfig_Load_NotExists(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Load when file doesn't exist should return empty config
	cfg, err := Load()
	testutil.RequireNoError(t, err)
	testutil.NotNil(t, cfg)
	testutil.Empty(t, cfg.URL)
	testutil.Empty(t, cfg.Email)
	testutil.Empty(t, cfg.APIToken)
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
	testutil.RequireNoError(t, err)

	// Clear config
	err = Clear()
	testutil.RequireNoError(t, err)

	// Load should return empty config
	loaded, err := Load()
	testutil.RequireNoError(t, err)
	testutil.Empty(t, loaded.URL)
}

func TestConfig_Clear_NotExists(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Clear when file doesn't exist should not error
	err := Clear()
	testutil.NoError(t, err)
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
	testutil.RequireNoError(t, err)

	// Check file permissions using Path() to get actual config location
	configFile := Path()
	info, err := os.Stat(configFile)
	testutil.RequireNoError(t, err)

	// File should be 0600 (user read/write only)
	testutil.Equal(t, info.Mode().Perm(), os.FileMode(0600))
}

func TestGetURL_EnvOverride(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Save config
	cfg := &Config{URL: "https://config.atlassian.net"}
	err := Save(cfg)
	testutil.RequireNoError(t, err)

	// Without env, should return config value
	testutil.Equal(t, GetURL(), "https://config.atlassian.net")

	// With env, should return env value
	t.Setenv("JIRA_URL", "https://env.atlassian.net")
	testutil.Equal(t, GetURL(), "https://env.atlassian.net")
}

func TestGetURL_LegacyDomainFallback(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Save config with legacy domain only
	cfg := &Config{Domain: "legacy"}
	err := Save(cfg)
	testutil.RequireNoError(t, err)

	// Should construct URL from legacy domain
	testutil.Equal(t, GetURL(), "https://legacy.atlassian.net")

	// JIRA_DOMAIN env should also work
	t.Setenv("JIRA_DOMAIN", "env-legacy")
	testutil.Equal(t, GetURL(), "https://env-legacy.atlassian.net")
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
	testutil.RequireNoError(t, err)

	// URL should take precedence
	testutil.Equal(t, GetURL(), "https://new-url.atlassian.net")
}

func TestNormalizeURL(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			testutil.Equal(t, url.NormalizeURL(tt.input), tt.want)
		})
	}
}

func TestGetDomain_EnvOverride(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Save config
	cfg := &Config{Domain: "config-domain"}
	err := Save(cfg)
	testutil.RequireNoError(t, err)

	// Without env, should return config value
	testutil.Equal(t, GetDomain(), "config-domain")

	// With env, should return env value
	t.Setenv("JIRA_DOMAIN", "env-domain")
	testutil.Equal(t, GetDomain(), "env-domain")
}

func TestGetEmail_EnvOverride(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Save config
	cfg := &Config{Email: "config@example.com"}
	err := Save(cfg)
	testutil.RequireNoError(t, err)

	// Without env, should return config value
	testutil.Equal(t, GetEmail(), "config@example.com")

	// With env, should return env value
	t.Setenv("JIRA_EMAIL", "env@example.com")
	testutil.Equal(t, GetEmail(), "env@example.com")
}

func TestGetAPIToken_EnvOverride(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Save config
	cfg := &Config{APIToken: "config-token"}
	err := Save(cfg)
	testutil.RequireNoError(t, err)

	// Without env, should return config value
	testutil.Equal(t, GetAPIToken(), "config-token")

	// With env, should return env value
	t.Setenv("JIRA_API_TOKEN", "env-token")
	testutil.Equal(t, GetAPIToken(), "env-token")
}

func TestIsConfigured(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Not configured initially
	testutil.False(t, IsConfigured())

	// Partially configured (URL only)
	cfg := &Config{URL: "https://test.atlassian.net"}
	err := Save(cfg)
	testutil.RequireNoError(t, err)
	testutil.False(t, IsConfigured())

	// Fully configured with URL
	cfg = &Config{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "token",
	}
	err = Save(cfg)
	testutil.RequireNoError(t, err)
	testutil.True(t, IsConfigured())
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
	err := Save(cfg)
	testutil.RequireNoError(t, err)
	testutil.True(t, IsConfigured())
}

func TestIsConfigured_EnvOnly(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Set all env vars with JIRA_URL
	t.Setenv("JIRA_URL", "https://env.atlassian.net")
	t.Setenv("JIRA_EMAIL", "env@example.com")
	t.Setenv("JIRA_API_TOKEN", "env-token")

	// Should be configured via env vars only
	testutil.True(t, IsConfigured())
}

func TestIsConfigured_LegacyEnvOnly(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Set all env vars with legacy JIRA_DOMAIN
	t.Setenv("JIRA_DOMAIN", "env-domain")
	t.Setenv("JIRA_EMAIL", "env@example.com")
	t.Setenv("JIRA_API_TOKEN", "env-token")

	// Should be configured via legacy env vars
	testutil.True(t, IsConfigured())
}

func TestPath(t *testing.T) {
	t.Parallel()
	path := Path()
	testutil.Contains(t, path, configDirName)
	testutil.Contains(t, path, configFileName)
}

// Tests for ATLASSIAN_* env var fallbacks

func TestGetURL_AtlassianFallback(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// ATLASSIAN_URL should work when JIRA_URL is not set
	t.Setenv("ATLASSIAN_URL", "https://shared.atlassian.net")
	testutil.Equal(t, GetURL(), "https://shared.atlassian.net")

	// JIRA_URL takes precedence over ATLASSIAN_URL
	t.Setenv("JIRA_URL", "https://jira-specific.atlassian.net")
	testutil.Equal(t, GetURL(), "https://jira-specific.atlassian.net")
}

func TestGetEmail_AtlassianFallback(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// ATLASSIAN_EMAIL should work when JIRA_EMAIL is not set
	t.Setenv("ATLASSIAN_EMAIL", "shared@example.com")
	testutil.Equal(t, GetEmail(), "shared@example.com")

	// JIRA_EMAIL takes precedence over ATLASSIAN_EMAIL
	t.Setenv("JIRA_EMAIL", "jira@example.com")
	testutil.Equal(t, GetEmail(), "jira@example.com")
}

func TestGetAPIToken_AtlassianFallback(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// ATLASSIAN_API_TOKEN should work when JIRA_API_TOKEN is not set
	t.Setenv("ATLASSIAN_API_TOKEN", "shared-token")
	testutil.Equal(t, GetAPIToken(), "shared-token")

	// JIRA_API_TOKEN takes precedence over ATLASSIAN_API_TOKEN
	t.Setenv("JIRA_API_TOKEN", "jira-token")
	testutil.Equal(t, GetAPIToken(), "jira-token")
}

func TestIsConfigured_AtlassianEnvOnly(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Set all ATLASSIAN_* env vars (shared credentials)
	t.Setenv("ATLASSIAN_URL", "https://shared.atlassian.net")
	t.Setenv("ATLASSIAN_EMAIL", "shared@example.com")
	t.Setenv("ATLASSIAN_API_TOKEN", "shared-token")

	// Should be configured via shared env vars
	testutil.True(t, IsConfigured())
}

func TestGetURL_FullPrecedenceChain(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Start with config file only
	cfg := &Config{
		URL:    "https://config-url.atlassian.net",
		Domain: "config-domain",
	}
	err := Save(cfg)
	testutil.RequireNoError(t, err)

	// Config URL should be returned
	testutil.Equal(t, GetURL(), "https://config-url.atlassian.net")

	// Clear config, set legacy JIRA_DOMAIN
	err = Clear()
	testutil.RequireNoError(t, err)
	t.Setenv("JIRA_DOMAIN", "env-domain")
	testutil.Equal(t, GetURL(), "https://env-domain.atlassian.net")

	// ATLASSIAN_URL takes precedence over JIRA_DOMAIN
	t.Setenv("ATLASSIAN_URL", "https://atlassian-url.atlassian.net")
	testutil.Equal(t, GetURL(), "https://atlassian-url.atlassian.net")

	// JIRA_URL takes precedence over ATLASSIAN_URL
	t.Setenv("JIRA_URL", "https://jira-url.atlassian.net")
	testutil.Equal(t, GetURL(), "https://jira-url.atlassian.net")
}
