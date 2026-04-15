package config

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestGetURLWithSource_EnvPrecedence(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// No config, no env
	value, source := GetURLWithSource()
	testutil.Equal(t, value, "")
	testutil.Equal(t, source, "-")

	// Config file
	cfg := &Config{URL: "https://config.atlassian.net"}
	err := Save(cfg)
	testutil.RequireNoError(t, err)

	value, source = GetURLWithSource()
	testutil.Equal(t, value, "https://config.atlassian.net")
	testutil.Equal(t, source, "config")

	// ATLASSIAN_URL takes precedence over config
	t.Setenv("ATLASSIAN_URL", "https://atlassian.atlassian.net")
	value, source = GetURLWithSource()
	testutil.Equal(t, value, "https://atlassian.atlassian.net")
	testutil.Equal(t, source, "env (ATLASSIAN_URL)")

	// JIRA_URL takes precedence over ATLASSIAN_URL
	t.Setenv("JIRA_URL", "https://jira.atlassian.net")
	value, source = GetURLWithSource()
	testutil.Equal(t, value, "https://jira.atlassian.net")
	testutil.Equal(t, source, "env (JIRA_URL)")
}

func TestGetURLWithSource_LegacyDomain(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Config with legacy domain
	cfg := &Config{Domain: "legacy"}
	err := Save(cfg)
	testutil.RequireNoError(t, err)

	value, source := GetURLWithSource()
	testutil.Equal(t, value, "https://legacy.atlassian.net")
	testutil.Equal(t, source, "config (domain, deprecated)")

	// JIRA_DOMAIN env var
	err = Clear()
	testutil.RequireNoError(t, err)
	t.Setenv("JIRA_DOMAIN", "env-legacy")

	value, source = GetURLWithSource()
	testutil.Equal(t, value, "https://env-legacy.atlassian.net")
	testutil.Equal(t, source, "env (JIRA_DOMAIN, deprecated)")
}

func TestGetEmailWithSource_EnvPrecedence(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// No config, no env
	value, source := GetEmailWithSource()
	testutil.Equal(t, value, "")
	testutil.Equal(t, source, "-")

	// Config file
	cfg := &Config{Email: "config@example.com"}
	err := Save(cfg)
	testutil.RequireNoError(t, err)

	value, source = GetEmailWithSource()
	testutil.Equal(t, value, "config@example.com")
	testutil.Equal(t, source, "config")

	// ATLASSIAN_EMAIL takes precedence over config
	t.Setenv("ATLASSIAN_EMAIL", "atlassian@example.com")
	value, source = GetEmailWithSource()
	testutil.Equal(t, value, "atlassian@example.com")
	testutil.Equal(t, source, "env (ATLASSIAN_EMAIL)")

	// JIRA_EMAIL takes precedence over ATLASSIAN_EMAIL
	t.Setenv("JIRA_EMAIL", "jira@example.com")
	value, source = GetEmailWithSource()
	testutil.Equal(t, value, "jira@example.com")
	testutil.Equal(t, source, "env (JIRA_EMAIL)")
}

func TestGetAPITokenWithSource_EnvPrecedence(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// No config, no env
	value, source := GetAPITokenWithSource()
	testutil.Equal(t, value, "")
	testutil.Equal(t, source, "-")

	// Config file
	cfg := &Config{APIToken: "config-token"}
	err := Save(cfg)
	testutil.RequireNoError(t, err)

	value, source = GetAPITokenWithSource()
	testutil.Equal(t, value, "config-token")
	testutil.Equal(t, source, "config")

	// ATLASSIAN_API_TOKEN takes precedence over config
	t.Setenv("ATLASSIAN_API_TOKEN", "atlassian-token")
	value, source = GetAPITokenWithSource()
	testutil.Equal(t, value, "atlassian-token")
	testutil.Equal(t, source, "env (ATLASSIAN_API_TOKEN)")

	// JIRA_API_TOKEN takes precedence over ATLASSIAN_API_TOKEN
	t.Setenv("JIRA_API_TOKEN", "jira-token")
	value, source = GetAPITokenWithSource()
	testutil.Equal(t, value, "jira-token")
	testutil.Equal(t, source, "env (JIRA_API_TOKEN)")
}

func TestGetDefaultProjectWithSource_EnvPrecedence(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// No config, no env
	value, source := GetDefaultProjectWithSource()
	testutil.Equal(t, value, "")
	testutil.Equal(t, source, "-")

	// Config file
	cfg := &Config{DefaultProject: "PROJ"}
	err := Save(cfg)
	testutil.RequireNoError(t, err)

	value, source = GetDefaultProjectWithSource()
	testutil.Equal(t, value, "PROJ")
	testutil.Equal(t, source, "config")

	// JIRA_DEFAULT_PROJECT takes precedence over config
	t.Setenv("JIRA_DEFAULT_PROJECT", "ENV-PROJ")
	value, source = GetDefaultProjectWithSource()
	testutil.Equal(t, value, "ENV-PROJ")
	testutil.Equal(t, source, "env (JIRA_DEFAULT_PROJECT)")
}

func TestGetValuesWithSources_AllFields(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Set up mixed sources
	cfg := &Config{
		URL:            "https://config.atlassian.net",
		Email:          "config@example.com",
		DefaultProject: "PROJ",
	}
	err := Save(cfg)
	testutil.RequireNoError(t, err)

	t.Setenv("JIRA_API_TOKEN", "env-token")
	t.Setenv("JIRA_AUTH_METHOD", "bearer")
	t.Setenv("JIRA_CLOUD_ID", "cloud-123")

	result := GetValuesWithSources()

	// Verify all fields are populated
	testutil.Equal(t, result.URL, "https://config.atlassian.net")
	testutil.Equal(t, result.URLSource, "config")

	testutil.Equal(t, result.Email, "config@example.com")
	testutil.Equal(t, result.EmailSource, "config")

	testutil.Equal(t, result.APIToken, "env-token")
	testutil.Equal(t, result.TokenSource, "env (JIRA_API_TOKEN)")

	testutil.Equal(t, result.DefaultProject, "PROJ")
	testutil.Equal(t, result.ProjectSource, "config")

	testutil.Equal(t, result.AuthMethod, "bearer")
	testutil.Equal(t, result.AuthMethodSrc, "env (JIRA_AUTH_METHOD)")

	testutil.Equal(t, result.CloudID, "cloud-123")
	testutil.Equal(t, result.CloudIDSrc, "env (JIRA_CLOUD_ID)")

	testutil.NotEmpty(t, result.Path)
}

func TestGetValuesWithSources_AllEmpty(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	result := GetValuesWithSources()

	testutil.Equal(t, result.URL, "")
	testutil.Equal(t, result.URLSource, "-")

	testutil.Equal(t, result.Email, "")
	testutil.Equal(t, result.EmailSource, "-")

	testutil.Equal(t, result.APIToken, "")
	testutil.Equal(t, result.TokenSource, "-")

	testutil.Equal(t, result.DefaultProject, "")
	testutil.Equal(t, result.ProjectSource, "-")

	// AuthMethod defaults to "basic"
	testutil.Equal(t, result.AuthMethod, "basic")
	testutil.Equal(t, result.AuthMethodSrc, "default")

	testutil.Equal(t, result.CloudID, "")
	testutil.Equal(t, result.CloudIDSrc, "-")
}
