package initcmd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
)

func TestConfig_GetDefaultProject_Env(t *testing.T) {
	t.Setenv("JIRA_DEFAULT_PROJECT", "ENVPROJ")

	got := config.GetDefaultProject()
	assert.Equal(t, "ENVPROJ", got)
}

func TestConfig_GetDefaultProject_NoConfig(t *testing.T) {
	// Clear env and use temp home dir
	t.Setenv("JIRA_DEFAULT_PROJECT", "")
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	// On Linux, also set XDG_CONFIG_HOME to ensure cross-platform behavior
	t.Setenv("XDG_CONFIG_HOME", homeDir)

	got := config.GetDefaultProject()
	assert.Equal(t, "", got)
}

func TestConfig_DefaultProject_Struct(t *testing.T) {
	// Test that the Config struct has the DefaultProject field
	cfg := &config.Config{
		URL:            "https://test.atlassian.net",
		Email:          "test@example.com",
		APIToken:       "token",
		DefaultProject: "MYPROJ",
	}
	assert.Equal(t, "MYPROJ", cfg.DefaultProject)
}

// Note: Interactive huh form tests are skipped because huh requires a TTY
// The non-interactive paths (all flags provided) still use huh forms internally,
// so we test config loading/saving separately
