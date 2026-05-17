// Package credtest provides a hermetic environment for credential /
// config / init tests across the atlassian-cli modules. It isolates
// HOME/XDG (so no real config.yml or legacy file is touched), forces the
// encrypted-file keyring backend with a fixed passphrase (no OS keychain
// prompts in CI), clears any token env vars that would shadow the
// keyring, and resets the one-time migration sink.
package credtest

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/keyring"
)

// tokenEnvVars are every API-token env var that would otherwise override
// the keyring at runtime; cleared so tests exercise the keyring path.
var tokenEnvVars = []string{
	"ATLASSIAN_API_TOKEN",
	"CFL_API_TOKEN",
	"JIRA_API_TOKEN",
}

// Hermetic isolates the process credential environment for the duration
// of t and returns the temp directory used as HOME/XDG_CONFIG_HOME.
// All mutations use t.Setenv / t.Cleanup, so they auto-revert.
func Hermetic(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	// Force the portable encrypted-file backend so tests never touch (or
	// prompt for) the real OS keychain.
	t.Setenv(keyring.BackendEnvVar, "file")
	t.Setenv("ATLASSIAN_CLI_KEYRING_PASSPHRASE", "credtest-passphrase")
	for _, v := range tokenEnvVars {
		t.Setenv(v, "")
	}

	keyring.ResetMigrationNotice()
	keyring.ResetCorruptWarnOnce()
	t.Cleanup(keyring.ResetMigrationNotice)
	t.Cleanup(keyring.ResetCorruptWarnOnce)
	return dir
}

// SeedToken stores a token under key in the hermetic keyring (the same
// file backend Hermetic configures). Use it to set up "token already in
// the keyring" scenarios without going through init.
func SeedToken(t *testing.T, key, token string) {
	t.Helper()
	if err := keyring.PersistToken(key, token); err != nil {
		t.Fatalf("credtest.SeedToken(%q): %v", key, err)
	}
}
