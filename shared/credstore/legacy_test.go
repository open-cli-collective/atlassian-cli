package credstore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestLoadLegacyCFL(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	body := `url: https://acme.atlassian.net/wiki
email: u@example.com
api_token: token
default_space: SPACE
auth_method: bearer
cloud_id: cloud-1
`
	testutil.RequireNoError(t, os.WriteFile(path, []byte(body), 0o600))

	got, err := LoadLegacyCFL(path)
	testutil.RequireNoError(t, err)
	testutil.NotNil(t, got)
	testutil.Equal(t, "https://acme.atlassian.net/wiki", got.URL)
	testutil.Equal(t, "u@example.com", got.Email)
	testutil.Equal(t, "token", got.APIToken)
	testutil.Equal(t, "SPACE", got.DefaultSpace)
	testutil.Equal(t, "bearer", got.AuthMethod)
	testutil.Equal(t, "cloud-1", got.CloudID)

	// Section() normalizes URL to base.
	testutil.Equal(t, "https://acme.atlassian.net", got.Section().URL)
}

func TestLoadLegacyCFL_AbsentReturnsNilNil(t *testing.T) {
	t.Parallel()
	got, err := LoadLegacyCFL(filepath.Join(t.TempDir(), "missing.yml"))
	testutil.RequireNoError(t, err)
	if got != nil {
		t.Fatalf("expected nil for absent file, got %+v", got)
	}
}

func TestLoadLegacyCFL_CorruptReturnsErrCorrupt(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	testutil.RequireNoError(t, os.WriteFile(path, []byte("url: : :: ["), 0o600))

	_, err := LoadLegacyCFL(path)
	testutil.RequireError(t, err)
	if !errors.Is(err, ErrCorruptStore) {
		t.Fatalf("expected ErrCorruptStore, got %v", err)
	}
}

func TestLoadLegacyJTK(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	body := `{"url":"https://acme.atlassian.net","email":"u@e","api_token":"tok","default_project":"PROJ","auth_method":"bearer","cloud_id":"c-1"}`
	testutil.RequireNoError(t, os.WriteFile(path, []byte(body), 0o600))

	got, err := LoadLegacyJTK(path)
	testutil.RequireNoError(t, err)
	testutil.NotNil(t, got)
	testutil.Equal(t, "https://acme.atlassian.net", got.URL)
	testutil.Equal(t, "PROJ", got.DefaultProject)
	testutil.Equal(t, "bearer", got.AuthMethod)
}

func TestLoadLegacyJTK_DomainFallback(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	testutil.RequireNoError(t, os.WriteFile(path, []byte(`{"domain":"acme","email":"u@e","api_token":"t"}`), 0o600))

	got, err := LoadLegacyJTK(path)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "https://acme.atlassian.net", got.URL)
}

func TestLoadLegacyJTK_CorruptReturnsErrCorrupt(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	testutil.RequireNoError(t, os.WriteFile(path, []byte("{not json"), 0o600))

	_, err := LoadLegacyJTK(path)
	testutil.RequireError(t, err)
	if !errors.Is(err, ErrCorruptStore) {
		t.Fatalf("expected ErrCorruptStore, got %v", err)
	}
}

func TestLegacyCFLPath_HonorsXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/xdg")
	testutil.Equal(t, "/custom/xdg/cfl/config.yml", LegacyCFLPath())
}

func TestLegacyJTKPath_MatchesUserConfigDir(t *testing.T) {
	// jtk uses os.UserConfigDir(); we must match exactly so macOS
	// users (~/Library/Application Support) are detected.
	dir, err := os.UserConfigDir()
	testutil.RequireNoError(t, err)
	want := filepath.Join(dir, "jira-ticket-cli", "config.json")
	testutil.Equal(t, want, LegacyJTKPath())
}

func TestLegacyJTKPath_NotXDGFallback(t *testing.T) {
	// On Linux os.UserConfigDir honors XDG_CONFIG_HOME, but on macOS
	// it returns ~/Library/Application Support and ignores XDG. Either
	// way, our LegacyJTKPath must match os.UserConfigDir's choice.
	t.Setenv("XDG_CONFIG_HOME", "/custom/xdg")
	dir, err := os.UserConfigDir()
	testutil.RequireNoError(t, err)
	want := filepath.Join(dir, "jira-ticket-cli", "config.json")
	testutil.Equal(t, want, LegacyJTKPath())
}
