package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				URL:      "https://example.atlassian.net",
				Email:    "user@example.com",
				APIToken: "token123",
			},
			wantErr: false,
		},
		{
			name: "missing URL",
			config: Config{
				Email:    "user@example.com",
				APIToken: "token123",
			},
			wantErr: true,
			errMsg:  "url is required",
		},
		{
			name: "missing email",
			config: Config{
				URL:      "https://example.atlassian.net",
				APIToken: "token123",
			},
			wantErr: true,
			errMsg:  "email is required",
		},
		{
			name: "missing API token",
			config: Config{
				URL:   "https://example.atlassian.net",
				Email: "user@example.com",
			},
			wantErr: true,
			errMsg:  "api_token is required",
		},
		{
			name: "invalid URL scheme",
			config: Config{
				URL:      "ftp://example.atlassian.net",
				Email:    "user@example.com",
				APIToken: "token123",
			},
			wantErr: true,
			errMsg:  "url must use https",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_NormalizeURL(t *testing.T) {
	tests := []struct {
		name     string
		inputURL string
		expected string
	}{
		{
			name:     "already has /wiki suffix",
			inputURL: "https://example.atlassian.net/wiki",
			expected: "https://example.atlassian.net/wiki",
		},
		{
			name:     "no /wiki suffix",
			inputURL: "https://example.atlassian.net",
			expected: "https://example.atlassian.net/wiki",
		},
		{
			name:     "trailing slash without /wiki",
			inputURL: "https://example.atlassian.net/",
			expected: "https://example.atlassian.net/wiki",
		},
		{
			name:     "trailing slash with /wiki",
			inputURL: "https://example.atlassian.net/wiki/",
			expected: "https://example.atlassian.net/wiki",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{URL: tt.inputURL}
			cfg.NormalizeURL()
			assert.Equal(t, tt.expected, cfg.URL)
		})
	}
}

func TestConfig_LoadFromEnv(t *testing.T) {
	// Save original env vars
	origURL := os.Getenv("CFL_URL")
	origEmail := os.Getenv("CFL_EMAIL")
	origToken := os.Getenv("CFL_API_TOKEN")
	origSpace := os.Getenv("CFL_DEFAULT_SPACE")

	// Cleanup
	defer func() {
		_ = os.Setenv("CFL_URL", origURL)
		_ = os.Setenv("CFL_EMAIL", origEmail)
		_ = os.Setenv("CFL_API_TOKEN", origToken)
		_ = os.Setenv("CFL_DEFAULT_SPACE", origSpace)
	}()

	t.Run("loads all env vars", func(t *testing.T) {
		_ = os.Setenv("CFL_URL", "https://env.atlassian.net")
		_ = os.Setenv("CFL_EMAIL", "env@example.com")
		_ = os.Setenv("CFL_API_TOKEN", "env-token")
		_ = os.Setenv("CFL_DEFAULT_SPACE", "ENV")

		cfg := &Config{}
		cfg.LoadFromEnv()

		assert.Equal(t, "https://env.atlassian.net", cfg.URL)
		assert.Equal(t, "env@example.com", cfg.Email)
		assert.Equal(t, "env-token", cfg.APIToken)
		assert.Equal(t, "ENV", cfg.DefaultSpace)
	})

	t.Run("env vars override existing values", func(t *testing.T) {
		_ = os.Setenv("CFL_URL", "https://override.atlassian.net")
		_ = os.Setenv("CFL_EMAIL", "")
		_ = os.Setenv("CFL_API_TOKEN", "")
		_ = os.Setenv("CFL_DEFAULT_SPACE", "")

		cfg := &Config{
			URL:   "https://original.atlassian.net",
			Email: "original@example.com",
		}
		cfg.LoadFromEnv()

		// URL should be overridden
		assert.Equal(t, "https://override.atlassian.net", cfg.URL)
		// Email should remain (empty env var doesn't override)
		assert.Equal(t, "original@example.com", cfg.Email)
	})
}

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()

	// Should be under home directory
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(path, home))
	assert.Contains(t, path, "cfl")
	assert.True(t, filepath.Ext(path) == ".yml" || filepath.Ext(path) == ".yaml")
}

func TestConfig_Save_and_Load(t *testing.T) {
	// Create a temp directory for the test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	original := Config{
		URL:          "https://test.atlassian.net",
		Email:        "test@example.com",
		APIToken:     "test-token",
		DefaultSpace: "TEST",
		OutputFormat: "json",
	}

	// Save
	err := original.Save(configPath)
	require.NoError(t, err)

	// Load
	loaded, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, original.URL, loaded.URL)
	assert.Equal(t, original.Email, loaded.Email)
	assert.Equal(t, original.APIToken, loaded.APIToken)
	assert.Equal(t, original.DefaultSpace, loaded.DefaultSpace)
	assert.Equal(t, original.OutputFormat, loaded.OutputFormat)
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yml")
	require.Error(t, err)
}

func TestConfig_LoadFromEnv_AtlassianFallback(t *testing.T) {
	// Clear all relevant env vars
	clearEnvVars := func() {
		os.Unsetenv("CFL_URL")
		os.Unsetenv("CFL_EMAIL")
		os.Unsetenv("CFL_API_TOKEN")
		os.Unsetenv("ATLASSIAN_URL")
		os.Unsetenv("ATLASSIAN_EMAIL")
		os.Unsetenv("ATLASSIAN_API_TOKEN")
	}

	t.Run("ATLASSIAN_* used when CFL_* not set", func(t *testing.T) {
		clearEnvVars()
		defer clearEnvVars()

		os.Setenv("ATLASSIAN_URL", "https://shared.atlassian.net")
		os.Setenv("ATLASSIAN_EMAIL", "shared@example.com")
		os.Setenv("ATLASSIAN_API_TOKEN", "shared-token")

		cfg := &Config{}
		cfg.LoadFromEnv()

		assert.Equal(t, "https://shared.atlassian.net", cfg.URL)
		assert.Equal(t, "shared@example.com", cfg.Email)
		assert.Equal(t, "shared-token", cfg.APIToken)
	})

	t.Run("CFL_* takes precedence over ATLASSIAN_*", func(t *testing.T) {
		clearEnvVars()
		defer clearEnvVars()

		os.Setenv("CFL_URL", "https://cfl.atlassian.net")
		os.Setenv("CFL_EMAIL", "cfl@example.com")
		os.Setenv("CFL_API_TOKEN", "cfl-token")
		os.Setenv("ATLASSIAN_URL", "https://shared.atlassian.net")
		os.Setenv("ATLASSIAN_EMAIL", "shared@example.com")
		os.Setenv("ATLASSIAN_API_TOKEN", "shared-token")

		cfg := &Config{}
		cfg.LoadFromEnv()

		assert.Equal(t, "https://cfl.atlassian.net", cfg.URL)
		assert.Equal(t, "cfl@example.com", cfg.Email)
		assert.Equal(t, "cfl-token", cfg.APIToken)
	})

	t.Run("mixed CFL_* and ATLASSIAN_*", func(t *testing.T) {
		clearEnvVars()
		defer clearEnvVars()

		// Only URL is CFL-specific, rest use shared
		os.Setenv("CFL_URL", "https://cfl.atlassian.net")
		os.Setenv("ATLASSIAN_EMAIL", "shared@example.com")
		os.Setenv("ATLASSIAN_API_TOKEN", "shared-token")

		cfg := &Config{}
		cfg.LoadFromEnv()

		assert.Equal(t, "https://cfl.atlassian.net", cfg.URL)
		assert.Equal(t, "shared@example.com", cfg.Email)
		assert.Equal(t, "shared-token", cfg.APIToken)
	})
}

func TestGetEnvWithFallback(t *testing.T) {
	os.Unsetenv("TEST_PRIMARY")
	os.Unsetenv("TEST_FALLBACK")
	defer func() {
		os.Unsetenv("TEST_PRIMARY")
		os.Unsetenv("TEST_FALLBACK")
	}()

	t.Run("returns primary when set", func(t *testing.T) {
		os.Setenv("TEST_PRIMARY", "primary-value")
		os.Setenv("TEST_FALLBACK", "fallback-value")
		assert.Equal(t, "primary-value", getEnvWithFallback("TEST_PRIMARY", "TEST_FALLBACK"))
	})

	t.Run("returns fallback when primary empty", func(t *testing.T) {
		os.Unsetenv("TEST_PRIMARY")
		os.Setenv("TEST_FALLBACK", "fallback-value")
		assert.Equal(t, "fallback-value", getEnvWithFallback("TEST_PRIMARY", "TEST_FALLBACK"))
	})

	t.Run("returns empty when both empty", func(t *testing.T) {
		os.Unsetenv("TEST_PRIMARY")
		os.Unsetenv("TEST_FALLBACK")
		assert.Equal(t, "", getEnvWithFallback("TEST_PRIMARY", "TEST_FALLBACK"))
	})
}
