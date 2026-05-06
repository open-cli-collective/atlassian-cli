package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	sharedconfig "github.com/open-cli-collective/atlassian-go/config"
	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestConfig_Validate(t *testing.T) {
	t.Parallel()
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
		{
			name: "valid bearer config",
			config: Config{
				URL:        "https://example.atlassian.net",
				APIToken:   "scoped-token",
				AuthMethod: "bearer",
				CloudID:    "abc-123",
			},
			wantErr: false,
		},
		{
			name: "bearer missing cloud ID",
			config: Config{
				URL:        "https://example.atlassian.net",
				APIToken:   "scoped-token",
				AuthMethod: "bearer",
			},
			wantErr: true,
			errMsg:  "cloud_id is required for bearer auth",
		},
		{
			name: "bearer without email is valid",
			config: Config{
				URL:        "https://example.atlassian.net",
				APIToken:   "scoped-token",
				AuthMethod: "bearer",
				CloudID:    "abc-123",
			},
			wantErr: false,
		},
		{
			name: "invalid auth method",
			config: Config{
				URL:        "https://example.atlassian.net",
				Email:      "user@example.com",
				APIToken:   "token",
				AuthMethod: "oauth",
			},
			wantErr: true,
			errMsg:  "invalid auth method",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.config.Validate()
			if tt.wantErr {
				testutil.RequireError(t, err)
				testutil.Contains(t, err.Error(), tt.errMsg)
			} else {
				testutil.NoError(t, err)
			}
		})
	}
}

func TestConfig_NormalizeURL(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			cfg := Config{URL: tt.inputURL}
			cfg.NormalizeURL()
			testutil.Equal(t, tt.expected, cfg.URL)
		})
	}
}

func TestConfig_LoadFromEnv(t *testing.T) {
	t.Parallel()
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
		t.Parallel()
		_ = os.Setenv("CFL_URL", "https://env.atlassian.net")
		_ = os.Setenv("CFL_EMAIL", "env@example.com")
		_ = os.Setenv("CFL_API_TOKEN", "env-token")
		_ = os.Setenv("CFL_DEFAULT_SPACE", "ENV")

		cfg := &Config{}
		cfg.LoadFromEnv()

		testutil.Equal(t, "https://env.atlassian.net", cfg.URL)
		testutil.Equal(t, "env@example.com", cfg.Email)
		testutil.Equal(t, "env-token", cfg.APIToken)
		testutil.Equal(t, "ENV", cfg.DefaultSpace)
	})

	t.Run("env vars override existing values", func(t *testing.T) {
		t.Parallel()
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
		testutil.Equal(t, "https://override.atlassian.net", cfg.URL)
		// Email should remain (empty env var doesn't override)
		testutil.Equal(t, "original@example.com", cfg.Email)
	})
}

func TestDefaultConfigPath(t *testing.T) {
	t.Parallel()
	path := DefaultConfigPath()

	// Should be under home directory
	home, err := os.UserHomeDir()
	testutil.RequireNoError(t, err)

	testutil.True(t, strings.HasPrefix(path, home))
	testutil.Contains(t, path, "cfl")
	testutil.True(t, filepath.Ext(path) == ".yml" || filepath.Ext(path) == ".yaml")
}

func TestConfig_Save_and_Load(t *testing.T) {
	t.Parallel()
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
	testutil.RequireNoError(t, err)

	// Load
	loaded, err := Load(configPath)
	testutil.RequireNoError(t, err)

	testutil.Equal(t, original.URL, loaded.URL)
	testutil.Equal(t, original.Email, loaded.Email)
	testutil.Equal(t, original.APIToken, loaded.APIToken)
	testutil.Equal(t, original.DefaultSpace, loaded.DefaultSpace)
	testutil.Equal(t, original.OutputFormat, loaded.OutputFormat)
}

func TestLoad_FileNotFound(t *testing.T) {
	t.Parallel()
	_, err := Load("/nonexistent/path/config.yml")
	testutil.RequireError(t, err)
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

		t.Setenv("ATLASSIAN_URL", "https://shared.atlassian.net")
		t.Setenv("ATLASSIAN_EMAIL", "shared@example.com")
		t.Setenv("ATLASSIAN_API_TOKEN", "shared-token")

		cfg := &Config{}
		cfg.LoadFromEnv()

		testutil.Equal(t, "https://shared.atlassian.net", cfg.URL)
		testutil.Equal(t, "shared@example.com", cfg.Email)
		testutil.Equal(t, "shared-token", cfg.APIToken)
	})

	t.Run("CFL_* takes precedence over ATLASSIAN_*", func(t *testing.T) {
		clearEnvVars()
		defer clearEnvVars()

		t.Setenv("CFL_URL", "https://cfl.atlassian.net")
		t.Setenv("CFL_EMAIL", "cfl@example.com")
		t.Setenv("CFL_API_TOKEN", "cfl-token")
		t.Setenv("ATLASSIAN_URL", "https://shared.atlassian.net")
		t.Setenv("ATLASSIAN_EMAIL", "shared@example.com")
		t.Setenv("ATLASSIAN_API_TOKEN", "shared-token")

		cfg := &Config{}
		cfg.LoadFromEnv()

		testutil.Equal(t, "https://cfl.atlassian.net", cfg.URL)
		testutil.Equal(t, "cfl@example.com", cfg.Email)
		testutil.Equal(t, "cfl-token", cfg.APIToken)
	})

	t.Run("mixed CFL_* and ATLASSIAN_*", func(t *testing.T) {
		clearEnvVars()
		defer clearEnvVars()

		// Only URL is CFL-specific, rest use shared
		t.Setenv("CFL_URL", "https://cfl.atlassian.net")
		t.Setenv("ATLASSIAN_EMAIL", "shared@example.com")
		t.Setenv("ATLASSIAN_API_TOKEN", "shared-token")

		cfg := &Config{}
		cfg.LoadFromEnv()

		testutil.Equal(t, "https://cfl.atlassian.net", cfg.URL)
		testutil.Equal(t, "shared@example.com", cfg.Email)
		testutil.Equal(t, "shared-token", cfg.APIToken)
	})
}

func TestConfig_Save_and_Load_WithAuthFields(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	original := Config{
		URL:        "https://test.atlassian.net",
		APIToken:   "scoped-token",
		AuthMethod: "bearer",
		CloudID:    "abc-123-def",
	}

	err := original.Save(configPath)
	testutil.RequireNoError(t, err)

	loaded, err := Load(configPath)
	testutil.RequireNoError(t, err)

	testutil.Equal(t, original.AuthMethod, loaded.AuthMethod)
	testutil.Equal(t, original.CloudID, loaded.CloudID)
	testutil.Equal(t, original.URL, loaded.URL)
	testutil.Equal(t, original.APIToken, loaded.APIToken)
}

func TestConfig_LoadFromEnv_AuthFields(t *testing.T) {
	clearEnvVars := func() {
		os.Unsetenv("CFL_AUTH_METHOD")
		os.Unsetenv("CFL_CLOUD_ID")
		os.Unsetenv("ATLASSIAN_AUTH_METHOD")
		os.Unsetenv("ATLASSIAN_CLOUD_ID")
	}

	t.Run("CFL_* auth env vars", func(t *testing.T) {
		clearEnvVars()
		defer clearEnvVars()

		t.Setenv("CFL_AUTH_METHOD", "bearer")
		t.Setenv("CFL_CLOUD_ID", "cloud-123")

		cfg := &Config{}
		cfg.LoadFromEnv()

		testutil.Equal(t, "bearer", cfg.AuthMethod)
		testutil.Equal(t, "cloud-123", cfg.CloudID)
	})

	t.Run("ATLASSIAN_* fallback for auth fields", func(t *testing.T) {
		clearEnvVars()
		defer clearEnvVars()

		t.Setenv("ATLASSIAN_AUTH_METHOD", "bearer")
		t.Setenv("ATLASSIAN_CLOUD_ID", "shared-cloud")

		cfg := &Config{}
		cfg.LoadFromEnv()

		testutil.Equal(t, "bearer", cfg.AuthMethod)
		testutil.Equal(t, "shared-cloud", cfg.CloudID)
	})

	t.Run("CFL_* takes precedence over ATLASSIAN_* for auth fields", func(t *testing.T) {
		clearEnvVars()
		defer clearEnvVars()

		t.Setenv("CFL_AUTH_METHOD", "bearer")
		t.Setenv("CFL_CLOUD_ID", "cfl-cloud")
		t.Setenv("ATLASSIAN_AUTH_METHOD", "basic")
		t.Setenv("ATLASSIAN_CLOUD_ID", "shared-cloud")

		cfg := &Config{}
		cfg.LoadFromEnv()

		testutil.Equal(t, "bearer", cfg.AuthMethod)
		testutil.Equal(t, "cfl-cloud", cfg.CloudID)
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
		t.Setenv("TEST_PRIMARY", "primary-value")
		t.Setenv("TEST_FALLBACK", "fallback-value")
		testutil.Equal(t, "primary-value", sharedconfig.GetEnvWithFallback("TEST_PRIMARY", "TEST_FALLBACK"))
	})

	t.Run("returns fallback when primary empty", func(t *testing.T) {
		os.Unsetenv("TEST_PRIMARY")
		t.Setenv("TEST_FALLBACK", "fallback-value")
		testutil.Equal(t, "fallback-value", sharedconfig.GetEnvWithFallback("TEST_PRIMARY", "TEST_FALLBACK"))
	})

	t.Run("returns empty when both empty", func(t *testing.T) {
		os.Unsetenv("TEST_PRIMARY")
		os.Unsetenv("TEST_FALLBACK")
		testutil.Equal(t, "", sharedconfig.GetEnvWithFallback("TEST_PRIMARY", "TEST_FALLBACK"))
	})
}

func TestConfig_LoadFromShared(t *testing.T) {
	t.Parallel()

	t.Run("default fills empty fields and appends /wiki to URL", func(t *testing.T) {
		t.Parallel()
		store := &credstore.Store{
			Default: credstore.Section{
				URL:      "https://acme.atlassian.net",
				Email:    "default@example.com",
				APIToken: "default-tok",
			},
		}
		cfg := &Config{}
		cfg.LoadFromShared(store)
		testutil.Equal(t, "https://acme.atlassian.net/wiki", cfg.URL)
		testutil.Equal(t, "default@example.com", cfg.Email)
		testutil.Equal(t, "default-tok", cfg.APIToken)
	})

	t.Run("cfl override beats default per field", func(t *testing.T) {
		t.Parallel()
		store := &credstore.Store{
			Default: credstore.Section{
				URL:      "https://acme.atlassian.net",
				Email:    "default@example.com",
				APIToken: "default-tok",
			},
			CFL: credstore.ToolSection{
				Section: credstore.Section{APIToken: "cfl-only-tok"},
			},
		}
		cfg := &Config{}
		cfg.LoadFromShared(store)
		testutil.Equal(t, "default@example.com", cfg.Email) // from default
		testutil.Equal(t, "cfl-only-tok", cfg.APIToken)     // from override
	})

	t.Run("default_space and output_format come from cfl section only", func(t *testing.T) {
		t.Parallel()
		store := &credstore.Store{
			CFL: credstore.ToolSection{
				DefaultSpace: "MYSPACE",
				OutputFormat: "json",
			},
		}
		cfg := &Config{}
		cfg.LoadFromShared(store)
		testutil.Equal(t, "MYSPACE", cfg.DefaultSpace)
		testutil.Equal(t, "json", cfg.OutputFormat)
	})

	t.Run("nil store is no-op", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{URL: "preserve"}
		cfg.LoadFromShared(nil)
		testutil.Equal(t, "preserve", cfg.URL)
	})

	t.Run("empty shared store leaves config alone", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{URL: "https://existing/wiki", Email: "e@x"}
		cfg.LoadFromShared(&credstore.Store{})
		testutil.Equal(t, "https://existing/wiki", cfg.URL)
		testutil.Equal(t, "e@x", cfg.Email)
	})

	t.Run("shared overlays existing legacy values", func(t *testing.T) {
		t.Parallel()
		// Loader contract: shared values replace existing fields when set,
		// because legacy precedence is below shared.
		cfg := &Config{URL: "https://legacy.atlassian.net/wiki", APIToken: "legacy-tok"}
		store := &credstore.Store{
			Default: credstore.Section{
				URL:      "https://shared.atlassian.net",
				APIToken: "shared-tok",
			},
		}
		cfg.LoadFromShared(store)
		testutil.Equal(t, "https://shared.atlassian.net/wiki", cfg.URL)
		testutil.Equal(t, "shared-tok", cfg.APIToken)
	})
}

func TestLoadWithEnv_PrecedenceLegacyToSharedToEnv(t *testing.T) {
	// Isolate XDG so credstore.DefaultPath points into a tempdir.
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	// Seed legacy file (cfl-only).
	legacyDir := t.TempDir()
	legacyPath := filepath.Join(legacyDir, "config.yml")
	legacy := &Config{
		URL:      "https://legacy.atlassian.net/wiki",
		Email:    "legacy@example.com",
		APIToken: "legacy-tok",
	}
	testutil.RequireNoError(t, legacy.Save(legacyPath))

	// Seed shared store (overrides URL + token via default).
	sharedPath := filepath.Join(xdg, "atlassian-cli", "config.yml")
	store := &credstore.Store{
		Default: credstore.Section{
			URL:      "https://shared.atlassian.net",
			APIToken: "shared-tok",
		},
	}
	testutil.RequireNoError(t, store.Save(sharedPath))

	// Set CFL_API_TOKEN env (highest precedence).
	t.Setenv("CFL_API_TOKEN", "env-tok")

	cfg, err := LoadWithEnv(legacyPath)
	testutil.RequireNoError(t, err)

	// URL: shared wins over legacy (env not set for URL).
	testutil.Equal(t, "https://shared.atlassian.net/wiki", cfg.URL)
	// Email: only legacy has it; preserved.
	testutil.Equal(t, "legacy@example.com", cfg.Email)
	// API token: env wins over shared and legacy.
	testutil.Equal(t, "env-tok", cfg.APIToken)
}

func TestLoadWithEnv_CorruptSharedFallsBackToLegacy(t *testing.T) {
	// Runtime LoadWithEnv must keep working even when the shared file is
	// broken — every cfl command would otherwise fail. Init uses a
	// separate path that does surface corruption.
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	sharedPath := filepath.Join(xdg, "atlassian-cli", "config.yml")
	testutil.RequireNoError(t, os.MkdirAll(filepath.Dir(sharedPath), 0o700))
	testutil.RequireNoError(t, os.WriteFile(sharedPath, []byte("default: : :: ["), 0o600))

	legacyDir := t.TempDir()
	legacyPath := filepath.Join(legacyDir, "config.yml")
	legacy := &Config{URL: "https://x.atlassian.net/wiki", Email: "e@x", APIToken: "t"}
	testutil.RequireNoError(t, legacy.Save(legacyPath))

	cfg, err := LoadWithEnv(legacyPath)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "https://x.atlassian.net/wiki", cfg.URL)
}
