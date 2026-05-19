// Package credtest provides a hermetic environment for credential /
// config / init tests across the atlassian-cli modules. It isolates
// HOME/XDG (so no real config.yml or legacy file is touched), forces the
// encrypted-file keyring backend with a fixed passphrase (no OS keychain
// prompts in CI), clears any token env vars that would shadow the
// keyring, and resets the one-time migration sink.
package credtest

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	cccredstore "github.com/open-cli-collective/cli-common/credstore"
	"github.com/open-cli-collective/cli-common/statedirtest"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/keyring"
)

// deprecatedBundleKeys are the removed per-tool override keys. Production
// keyring exposes neither the names nor a writer for them (by design:
// SetToken refuses non-allowlisted keys). Tests still need to fabricate
// pre-migration "B3 upgrade" state — a user upgraded through the per-tool
// build and holds only these — so credtest reaches the bundle through the
// same cli-common credstore the keyring package uses, opened with the
// superset allowlist.
//
// These mirror keyring.deprecatedKeys (unexported there). Duplication is
// acceptable because the set is FROZEN historical state: it is exactly
// the keys the B3 build wrote, and the whole point of MON-5326 is that
// no new per-tool keys will ever be added — so there is nothing to drift
// toward. Authority lives in shared/keyring/migrate.go.
var deprecatedBundleKeys = []string{"cfl_api_token", "jtk_api_token"} //nolint:gosec // G101: bundle key names, not credentials

// openBundle opens the canonical atlassian-cli bundle directly via
// cli-common credstore with the superset allowlist (api_token + the two
// deprecated keys), honoring the same file backend + passphrase env that
// Hermetic configures. Test-harness only.
func openBundle(t *testing.T) (*cccredstore.Store, string) {
	t.Helper()
	// Hermetic precondition guard: opening a BackendFile store with no
	// passphrase rooted in the REAL HOME would read/write real keyring
	// state silently. Fail loud if the caller skipped credtest.Hermetic
	// (which sets the backend + passphrase + isolates HOME/XDG).
	if os.Getenv(keyring.BackendEnvVar) != "file" || os.Getenv("ATLASSIAN_CLI_KEYRING_PASSPHRASE") == "" {
		t.Fatalf("credtest: call credtest.Hermetic(t) before SeedDeprecatedKey/BundleKeys " +
			"(file backend + passphrase + isolated HOME must be set first)")
	}
	service, profile, err := cccredstore.ParseRef(keyring.Ref)
	if err != nil {
		t.Fatalf("credtest: ParseRef(%q): %v", keyring.Ref, err)
	}
	cs, err := cccredstore.Open(service, &cccredstore.Options{
		AllowedKeys:   append([]string{keyring.KeyAPIToken}, deprecatedBundleKeys...),
		ConfigBackend: cccredstore.BackendFile,
		FilePassphrase: func() (string, error) {
			return os.Getenv("ATLASSIAN_CLI_KEYRING_PASSPHRASE"), nil
		},
	})
	if err != nil {
		t.Fatalf("credtest: open bundle: %v", err)
	}
	return cs, profile
}

// SeedDeprecatedKey writes a removed per-tool override key directly into
// the bundle, standing in for B3 upgrade residue (plaintext already
// scrubbed, only the old keyring key left). Production refuses to write
// these, so the fixture goes through the underlying credstore.
func SeedDeprecatedKey(t *testing.T, key, token string) {
	t.Helper()
	cs, profile := openBundle(t)
	defer func() { _ = cs.Close() }()
	if err := cs.Set(profile, key, token, cccredstore.WithOverwrite()); err != nil {
		t.Fatalf("credtest.SeedDeprecatedKey(%s): %v", key, err)
	}
}

// BundleKeys returns the EXACT set of bundle keys that currently hold a
// value — api_token plus the deprecated per-tool keys — sorted. §1.11.11
// conformance asserts against this: a healthy bundle is exactly
// {api_token}; a cleared bundle is exactly empty.
func BundleKeys(t *testing.T) []string {
	t.Helper()
	cs, profile := openBundle(t)
	defer func() { _ = cs.Close() }()
	var present []string
	for _, k := range append([]string{keyring.KeyAPIToken}, deprecatedBundleKeys...) {
		ok, err := cs.Exists(profile, k)
		if err != nil {
			t.Fatalf("credtest.BundleKeys: exists %s: %v", k, err)
		}
		if ok {
			present = append(present, k)
		}
	}
	sort.Strings(present)
	return present
}

// tokenEnvVars are every API-token env var that would otherwise override
// the keyring at runtime; cleared so tests exercise the keyring path.
var tokenEnvVars = []string{
	"ATLASSIAN_API_TOKEN",
	"CFL_API_TOKEN",
	"JIRA_API_TOKEN",
}

// Hermetic isolates the process credential environment for the duration
// of t and returns the temp ROOT. Directory isolation delegates to the
// canonical cli-common statedirtest 7-var harness (HOME/USERPROFILE/
// AppData/LocalAppData/XDG_CONFIG_HOME/XDG_CACHE_HOME/XDG_DATA_HOME) so
// os.UserConfigDir/os.UserCacheDir never resolve to the developer's real
// directories on ANY OS — HOME/XDG-only isolation was a macOS/Windows
// real-dir leak. Callers MUST derive the shared config path from
// SharedConfigPath(t) (the resolver), never hand-build a layout. All
// mutations use t.Setenv / t.Cleanup, so they auto-revert. Not usable
// under t.Parallel (t.Setenv).
func Hermetic(t *testing.T) string {
	t.Helper()
	root := statedirtest.Hermetic(t)

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
	return root
}

// SharedConfigPath resolves the shared store path through the production
// resolver (credstore.DefaultPath) under the active hermetic env and
// ensures its parent dir exists. Tests seed/inspect the shared store
// here instead of hand-building "<root>/atlassian-cli/config.yml", which
// only matched the resolver on Linux. Call after Hermetic(t).
func SharedConfigPath(t *testing.T) string {
	t.Helper()
	p, err := credstore.DefaultPath()
	if err != nil {
		t.Fatalf("credtest.SharedConfigPath: resolving shared path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatalf("credtest.SharedConfigPath: mkdir %s: %v", filepath.Dir(p), err)
	}
	return p
}

// SeedToken stores the shared api_token in the hermetic keyring (the same
// file backend Hermetic configures). Use it to set up "token already in
// the keyring" scenarios without going through init. There is one key per
// logical credential (§1.11.10), so no key argument.
func SeedToken(t *testing.T, token string) {
	t.Helper()
	if err := keyring.PersistToken(token); err != nil {
		t.Fatalf("credtest.SeedToken: %v", err)
	}
}
