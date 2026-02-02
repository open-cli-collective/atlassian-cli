package initcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
)

func TestConfig_DefaultProject(t *testing.T) {
	// Test that DefaultProject field is preserved in config
	// Note: On macOS, UserConfigDir() returns ~/Library/Application Support
	// so we need to mock that path
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Create the config directory structure macOS expects
	configDir := filepath.Join(homeDir, "Library", "Application Support", "jira-ticket-cli")
	require.NoError(t, os.MkdirAll(configDir, 0700))

	// Write config with default project
	configPath := filepath.Join(configDir, "config.json")
	configContent := `{"url":"https://test.atlassian.net","email":"test@example.com","api_token":"token","default_project":"MYPROJ"}`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0600))

	// Load config and verify default project
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "MYPROJ", cfg.DefaultProject)
}

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

	got := config.GetDefaultProject()
	assert.Equal(t, "", got)
}

// Note: Interactive huh form tests are skipped because huh requires a TTY
// The non-interactive paths (all flags provided) still use huh forms internally,
// so we test config loading/saving separately
