package initcmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
)

func TestConfig_GetDefaultProject_Env(t *testing.T) {
	t.Setenv("JIRA_DEFAULT_PROJECT", "ENVPROJ")

	got := config.GetDefaultProject()
	testutil.Equal(t, got, "ENVPROJ")
}

func TestConfig_GetDefaultProject_NoConfig(t *testing.T) {
	// Clear env and use temp home dir
	t.Setenv("JIRA_DEFAULT_PROJECT", "")
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	// On Linux, also set XDG_CONFIG_HOME to ensure cross-platform behavior
	t.Setenv("XDG_CONFIG_HOME", homeDir)

	got := config.GetDefaultProject()
	testutil.Equal(t, got, "")
}

func TestConfig_DefaultProject_Struct(t *testing.T) {
	t.Parallel()
	// Test that the Config struct has the DefaultProject field
	cfg := &config.Config{
		URL:            "https://test.atlassian.net",
		Email:          "test@example.com",
		APIToken:       "token",
		DefaultProject: "MYPROJ",
	}
	testutil.Equal(t, cfg.DefaultProject, "MYPROJ")
}

func TestRunInit_InvalidAuthMethod(t *testing.T) {
	t.Parallel()
	opts := &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
	// An invalid auth method should be rejected before the interactive form runs
	err := runInit(context.Background(), opts, "", "", "", "Bearer", "", true)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "invalid auth method")
}

// Note: Interactive huh form tests are skipped because huh requires a TTY
// The non-interactive paths (all flags provided) still use huh forms internally,
// so we test config loading/saving separately

func TestInitCommand_Flags(t *testing.T) {
	t.Parallel()
	rootCmd := &cobra.Command{Use: "jtk", Short: "Test CLI"}

	opts := &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	Register(rootCmd, opts)

	initCmd, _, err := rootCmd.Find([]string{"init"})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "init", initCmd.Use)

	// Verify original flags exist
	urlFlag := initCmd.Flags().Lookup("url")
	testutil.NotNil(t, urlFlag)

	emailFlag := initCmd.Flags().Lookup("email")
	testutil.NotNil(t, emailFlag)

	tokenFlag := initCmd.Flags().Lookup("token")
	testutil.NotNil(t, tokenFlag)

	noVerifyFlag := initCmd.Flags().Lookup("no-verify")
	testutil.NotNil(t, noVerifyFlag)

	// Verify new auth flags exist
	authMethodFlag := initCmd.Flags().Lookup("auth-method")
	testutil.NotNil(t, authMethodFlag)
	testutil.Equal(t, "", authMethodFlag.DefValue)

	cloudIDFlag := initCmd.Flags().Lookup("cloud-id")
	testutil.NotNil(t, cloudIDFlag)
	testutil.Equal(t, "", cloudIDFlag.DefValue)
}
