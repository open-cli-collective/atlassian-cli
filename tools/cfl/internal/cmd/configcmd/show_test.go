package configcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/credtest"
	"github.com/open-cli-collective/atlassian-go/keyring"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	cflconfig "github.com/open-cli-collective/confluence-cli/internal/config"
)

// config show must report token PRESENCE + keyring metadata and never
// the token value (or any slice of it), even with a token configured.
func TestRunShow_TokenPresenceNoLeak(t *testing.T) {
	credtest.Hermetic(t)
	credtest.SeedToken(t, "SUPER-SECRET-show-token")

	out, errBuf := &bytes.Buffer{}, &bytes.Buffer{}
	opts := &root.Options{Output: "table", NoColor: true, Stdout: out, Stderr: errBuf}
	testutil.RequireNoError(t, runShow(opts))

	combined := out.String() + errBuf.String()
	testutil.NotContains(t, combined, "SUPER-SECRET-show-token")
	testutil.NotContains(t, combined, "SUPER") // no prefix slice either
	testutil.Contains(t, combined, "configured")
	testutil.Contains(t, combined, "Keyring Ref")
	testutil.Contains(t, combined, keyring.Ref)
}

func TestRunShow_ExactOutput(t *testing.T) {
	credtest.Hermetic(t)
	t.Setenv("CFL_URL", "")
	t.Setenv("ATLASSIAN_URL", "")
	t.Setenv("CFL_EMAIL", "")
	t.Setenv("ATLASSIAN_EMAIL", "")
	t.Setenv("CFL_DEFAULT_SPACE", "")
	t.Setenv("CFL_AUTH_METHOD", "")
	t.Setenv("ATLASSIAN_AUTH_METHOD", "")
	t.Setenv("CFL_CLOUD_ID", "")
	t.Setenv("ATLASSIAN_CLOUD_ID", "")

	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	cfgPath := filepath.Join(cfgDir, "cfl", "config.yml")
	testutil.RequireNoError(t, (&cflconfig.Config{
		URL:          "https://example.atlassian.net/wiki",
		Email:        "test@example.com",
		DefaultSpace: "TEST",
	}).Save(cfgPath))

	out, errBuf := &bytes.Buffer{}, &bytes.Buffer{}
	opts := &root.Options{Output: "table", NoColor: true, Stdout: out, Stderr: errBuf}
	testutil.RequireNoError(t, runShow(opts))

	testutil.Contains(t, out.String(), "URL: https://example.atlassian.net/wiki  (source: config)\n")
	testutil.Contains(t, out.String(), "Email: test@example.com  (source: config)\n")
	testutil.Contains(t, out.String(), "API Token: not set  (source: unset)\n")
	testutil.Contains(t, out.String(), "Default Space: TEST  (source: config)\n")
	testutil.Contains(t, out.String(), "Auth Method: basic  (source: default)\n")
	testutil.Contains(t, out.String(), "Cloud ID: (source: not set)\n")
	testutil.Contains(t, out.String(), "Keyring Ref: atlassian-cli/default  (source: fixed)\n")
	testutil.Contains(t, out.String(), "Keyring Backend:")
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	testutil.True(t, len(lines) == 8 || len(lines) == 9)
	testutil.Equal(t, "URL: https://example.atlassian.net/wiki  (source: config)", lines[0])
	testutil.Equal(t, "Email: test@example.com  (source: config)", lines[1])
	testutil.Equal(t, "API Token: not set  (source: unset)", lines[2])
	testutil.Equal(t, "Default Space: TEST  (source: config)", lines[3])
	testutil.Equal(t, "Auth Method: basic  (source: default)", lines[4])
	testutil.Equal(t, "Cloud ID: (source: not set)", lines[5])
	testutil.Equal(t, "Keyring Ref: atlassian-cli/default  (source: fixed)", lines[6])
	if len(lines) == 9 {
		testutil.Contains(t, lines[8], "Keyring Passphrase:")
	}
	testutil.Equal(t, "\nConfig file: "+cfgPath+"\n", errBuf.String())
}

func TestRunShow_UnreadableConfigNote(t *testing.T) {
	credtest.Hermetic(t)
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	cfgPath := filepath.Join(cfgDir, "cfl", "config.yml")
	testutil.RequireNoError(t, os.MkdirAll(filepath.Dir(cfgPath), 0o700))
	testutil.RequireNoError(t, os.WriteFile(cfgPath, []byte(":"), 0o600))

	out, errBuf := &bytes.Buffer{}, &bytes.Buffer{}
	opts := &root.Options{Output: "table", NoColor: true, Stdout: out, Stderr: errBuf}
	testutil.RequireNoError(t, runShow(opts))

	testutil.Contains(t, errBuf.String(), "\nConfig file: "+cfgPath+"\n")
	testutil.Contains(t, errBuf.String(), "  (file not found or unreadable)\n")
}
