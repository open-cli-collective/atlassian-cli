package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/credtest"
	"github.com/open-cli-collective/atlassian-go/keyring"
	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/open-cli-collective/atlassian-go/url"
)

// setupTestConfig hermetically isolates the test environment and returns
// the RESOLVED shared config path (credstore.DefaultPath under the
// hermetic env) plus a no-op cleanup. credtest.Hermetic provides the
// canonical 7-var isolation (cross-OS correct — no hand-built layout),
// the file keyring backend, token-env clearing, and migration-notice
// reset; this only adds the non-token JIRA_*/ATLASSIAN_* clears the
// jtk accessors consult.
func setupTestConfig(t *testing.T) (string, func()) {
	t.Helper()

	credtest.Hermetic(t)

	for _, v := range []string{
		"JIRA_URL", "JIRA_DOMAIN", "JIRA_EMAIL", "JIRA_AUTH_METHOD", "JIRA_CLOUD_ID",
		"ATLASSIAN_URL", "ATLASSIAN_EMAIL", "ATLASSIAN_AUTH_METHOD", "ATLASSIAN_CLOUD_ID",
	} {
		t.Setenv(v, "")
	}

	return credtest.SharedConfigPath(t), func() {}
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
	// Asymmetric codec: Save never persists the token (it lives in the
	// keyring); Load reads it only as the one-time migration source.
	testutil.Equal(t, loaded.APIToken, "")
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

// Save is atomic (temp+rename) and leaves no stale .tmp on success
// (the MON-5370 commit-5 hardening; modes are covered above).
func TestConfig_Save_AtomicNoStaleTmp(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	testutil.RequireNoError(t, Save(&Config{URL: "https://acme.atlassian.net"}))
	if _, statErr := os.Stat(Path() + ".tmp"); !os.IsNotExist(statErr) {
		t.Fatal("atomic Save must leave no .tmp on success")
	}
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

	// A plaintext config api_token is NEVER the source anymore.
	testutil.RequireNoError(t, Save(&Config{APIToken: "config-token"}))
	testutil.Equal(t, GetAPIToken(), "")

	// Keyring is the store of truth.
	testutil.RequireNoError(t, keyring.PersistToken("kr-token"))
	testutil.Equal(t, GetAPIToken(), "kr-token")

	// Env outranks the keyring.
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

	// Non-secret config complete but no token yet → still not configured.
	cfg = &Config{
		URL:   "https://test.atlassian.net",
		Email: "test@example.com",
	}
	err = Save(cfg)
	testutil.RequireNoError(t, err)
	testutil.False(t, IsConfigured())

	// Token in the keyring completes it.
	testutil.RequireNoError(t, keyring.PersistToken("kr-token"))
	testutil.True(t, IsConfigured())
}

func TestIsConfigured_LegacyDomain(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Fully configured with legacy domain + keyring token.
	cfg := &Config{
		Domain: "test",
		Email:  "test@example.com",
	}
	err := Save(cfg)
	testutil.RequireNoError(t, err)
	testutil.RequireNoError(t, keyring.PersistToken("kr-token"))
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

func TestGetAuthMethod_Default(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Default should be "basic"
	testutil.Equal(t, GetAuthMethod(), "basic")
}

func TestGetAuthMethod_FromConfig(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	cfg := &Config{
		URL:        "https://test.atlassian.net",
		APIToken:   "token",
		AuthMethod: "bearer",
	}
	err := Save(cfg)
	testutil.RequireNoError(t, err)

	testutil.Equal(t, GetAuthMethod(), "bearer")
}

func TestGetAuthMethod_EnvPrecedence(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	cfg := &Config{AuthMethod: "basic"}
	err := Save(cfg)
	testutil.RequireNoError(t, err)

	// JIRA_AUTH_METHOD takes precedence
	t.Setenv("JIRA_AUTH_METHOD", "bearer")
	testutil.Equal(t, GetAuthMethod(), "bearer")
}

func TestGetAuthMethod_AtlassianFallback(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	t.Setenv("ATLASSIAN_AUTH_METHOD", "bearer")
	testutil.Equal(t, GetAuthMethod(), "bearer")

	// JIRA_AUTH_METHOD takes precedence over ATLASSIAN_AUTH_METHOD
	t.Setenv("JIRA_AUTH_METHOD", "basic")
	testutil.Equal(t, GetAuthMethod(), "basic")
}

func TestGetCloudID_FromConfig(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	cfg := &Config{CloudID: "abc-123"}
	err := Save(cfg)
	testutil.RequireNoError(t, err)

	testutil.Equal(t, GetCloudID(), "abc-123")
}

func TestGetCloudID_EnvPrecedence(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	cfg := &Config{CloudID: "config-cloud"}
	err := Save(cfg)
	testutil.RequireNoError(t, err)

	t.Setenv("JIRA_CLOUD_ID", "env-cloud")
	testutil.Equal(t, GetCloudID(), "env-cloud")
}

func TestGetCloudID_AtlassianFallback(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	t.Setenv("ATLASSIAN_CLOUD_ID", "shared-cloud")
	testutil.Equal(t, GetCloudID(), "shared-cloud")

	t.Setenv("JIRA_CLOUD_ID", "jira-cloud")
	testutil.Equal(t, GetCloudID(), "jira-cloud")
}

func TestIsConfigured_Bearer(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Bearer needs URL + token + cloud ID (no email)
	t.Setenv("JIRA_AUTH_METHOD", "bearer")
	t.Setenv("JIRA_URL", "https://test.atlassian.net")
	t.Setenv("JIRA_API_TOKEN", "token")
	testutil.False(t, IsConfigured()) // missing cloud ID

	t.Setenv("JIRA_CLOUD_ID", "abc-123")
	testutil.True(t, IsConfigured())
}

func TestConfig_SaveAndLoad_WithAuthFields(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	cfg := &Config{
		URL:        "https://test.atlassian.net",
		APIToken:   "scoped-token",
		AuthMethod: "bearer",
		CloudID:    "abc-123-def",
	}

	err := Save(cfg)
	testutil.RequireNoError(t, err)

	loaded, err := Load()
	testutil.RequireNoError(t, err)

	testutil.Equal(t, loaded.AuthMethod, "bearer")
	testutil.Equal(t, loaded.CloudID, "abc-123-def")
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

func TestSharedStore_FillsURLBetweenEnvAndLegacy(t *testing.T) {
	sharedPath, cleanup := setupTestConfig(t)
	defer cleanup()

	// Seed shared store with a URL.
	store := &credstore.Store{
		Default: credstore.Section{URL: "https://shared.atlassian.net"},
	}
	testutil.RequireNoError(t, store.Save(sharedPath))

	// No legacy file, no env vars → shared default wins.
	testutil.Equal(t, "https://shared.atlassian.net", GetURL())

	// Env var beats shared.
	t.Setenv("ATLASSIAN_URL", "https://env.atlassian.net")
	testutil.Equal(t, "https://env.atlassian.net", GetURL())
}

func TestSharedStore_LegacyWinsWhenSharedAbsent(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	cfg := &Config{
		URL:      "https://legacy.atlassian.net",
		Email:    "legacy@example.com",
		APIToken: "legacy-tok",
	}
	testutil.RequireNoError(t, Save(cfg))
	testutil.Equal(t, "https://legacy.atlassian.net", GetURL())
	// The token is never read from the legacy plaintext file at runtime
	// (only the one-time migration consumes it); keyring is empty here.
	testutil.Equal(t, "", GetAPIToken())
}

func TestSharedStore_DefaultProject(t *testing.T) {
	sharedPath, cleanup := setupTestConfig(t)
	defer cleanup()

	store := &credstore.Store{
		JTK: credstore.ToolSection{DefaultProject: "MON"},
	}
	testutil.RequireNoError(t, store.Save(sharedPath))

	testutil.Equal(t, "MON", GetDefaultProject())
	t.Setenv("JIRA_DEFAULT_PROJECT", "ENV")
	testutil.Equal(t, "ENV", GetDefaultProject())
}

func TestSharedStore_AuthMethodWithSource(t *testing.T) {
	sharedPath, cleanup := setupTestConfig(t)
	defer cleanup()

	store := &credstore.Store{
		Default: credstore.Section{AuthMethod: "bearer"},
	}
	testutil.RequireNoError(t, store.Save(sharedPath))

	value, source := GetAuthMethodWithSource()
	testutil.Equal(t, "bearer", value)
	testutil.Equal(t, string(credstore.SourceDefault), source)
}

func TestSharedStore_FullPrecedenceChain(t *testing.T) {
	sharedPath, cleanup := setupTestConfig(t)
	defer cleanup()

	// Layer 1 (lowest): legacy file.
	cfg := &Config{URL: "https://legacy.atlassian.net", APIToken: "legacy-tok"}
	testutil.RequireNoError(t, Save(cfg))

	// Layer 2: shared default.
	store := &credstore.Store{
		Default: credstore.Section{URL: "https://shared.atlassian.net", APIToken: "shared-tok"},
	}
	testutil.RequireNoError(t, store.Save(sharedPath))
	testutil.Equal(t, "https://shared.atlassian.net", GetURL()) // shared default wins over legacy

	// Token precedence is keyring-based (not the YAML store/legacy):
	// the single shared api_token key, then env. There is one key per
	// logical credential (§1.11.10) — no per-tool keyring override.
	testutil.RequireNoError(t, keyring.PersistToken("shared-tok"))
	testutil.Equal(t, "shared-tok", GetAPIToken())

	// Layer 4: ATLASSIAN_* env beats the keyring.
	t.Setenv("ATLASSIAN_API_TOKEN", "atlassian-env-tok")
	testutil.Equal(t, "atlassian-env-tok", GetAPIToken())

	// Layer 5: JIRA_* env beats ATLASSIAN_*.
	t.Setenv("JIRA_API_TOKEN", "jira-env-tok")
	testutil.Equal(t, "jira-env-tok", GetAPIToken())
}

func TestSharedStore_CorruptDoesNotBlockAccessors(t *testing.T) {
	sharedPath, cleanup := setupTestConfig(t)
	defer cleanup()

	// Corrupt shared store.
	testutil.RequireNoError(t, os.MkdirAll(filepath.Dir(sharedPath), 0o700))
	testutil.RequireNoError(t, os.WriteFile(sharedPath, []byte("default: : :: ["), 0o600))

	// Legacy still works.
	cfg := &Config{URL: "https://legacy.atlassian.net", Email: "u@e", APIToken: "tok"}
	testutil.RequireNoError(t, Save(cfg))

	// Non-secret accessor returns the legacy value despite corrupt shared.
	testutil.Equal(t, "https://legacy.atlassian.net", GetURL())
	// GetAPIToken is keyring-only (non-migrating) — a corrupt shared
	// store doesn't block it; the keyring is simply empty here.
	testutil.Equal(t, "", GetAPIToken())
}
