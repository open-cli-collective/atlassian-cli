package keyring

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	cccredstore "github.com/open-cli-collective/cli-common/credstore"
	"github.com/open-cli-collective/cli-common/statedirtest"

	"github.com/open-cli-collective/atlassian-go/credstore"
)

// hermetic isolates the full cli-common statedirtest 7-var env set
// (HOME/USERPROFILE/AppData/LocalAppData/XDG_*) so os.UserConfigDir
// never resolves to a real directory on ANY OS, and forces the
// encrypted-file backend so these tests never touch (or prompt for) the
// real OS keychain. keyring cannot import credtest (credtest imports
// keyring — an import cycle), so it composes statedirtest directly.
// Tests MUST derive the shared path from sharedConfigPath(t), never
// hand-build a layout. Returns the temp root.
func hermetic(t *testing.T) string {
	t.Helper()
	root := statedirtest.Hermetic(t)
	t.Setenv(cccredstore.BackendEnvVar(Service), "file")
	t.Setenv("ATLASSIAN_CLI_KEYRING_PASSPHRASE", "e2e-passphrase")
	for _, v := range []string{"ATLASSIAN_API_TOKEN", "CFL_API_TOKEN", "JIRA_API_TOKEN"} {
		t.Setenv(v, "")
	}
	ResetMigrationNotice()
	ResetCorruptWarnOnce()
	t.Cleanup(ResetMigrationNotice)
	t.Cleanup(ResetCorruptWarnOnce)
	return root
}

// sharedConfigPath resolves the shared store path through the production
// resolver under the active hermetic env and ensures its parent exists.
// Replaces hand-building "<root>/atlassian-cli/config.yml", which only
// matched the resolver on Linux.
func sharedConfigPath(t *testing.T) string {
	t.Helper()
	p, err := credstore.DefaultPath()
	if err != nil {
		t.Fatalf("sharedConfigPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatalf("sharedConfigPath mkdir: %v", err)
	}
	return p
}

//nolint:gosec // G101: test fixture string, not a real credential
const secret = "TOK-pqrSTU-suffix" // distinctive so a leak is unmistakable

// bundleKeys returns the EXACT set of bundle keys that currently hold a
// value, across the conforming key (api_token) AND the deprecated B3 keys,
// sorted. §1.11.11 conformance asserts against this: a healthy bundle is
// exactly {api_token}; a cleared bundle is exactly empty; no deprecated
// per-tool key may survive a migration.
func bundleKeys(t *testing.T) []string {
	t.Helper()
	s, err := OpenForClearAll() // migrationAllowedKeys: sees deprecated keys too
	if err != nil {
		t.Fatalf("OpenForClearAll: %v", err)
	}
	defer func() { _ = s.Close() }()
	var present []string
	for _, k := range append([]string{KeyAPIToken}, deprecatedKeys...) {
		ok, herr := s.HasToken(k)
		if herr != nil {
			t.Fatalf("HasToken(%s): %v", k, herr)
		}
		if ok {
			present = append(present, k)
		}
	}
	sort.Strings(present)
	return present
}

// seedDeprecatedKey writes a removed per-tool key directly into the
// bundle to stand in for B3 upgrade state. SetToken refuses non-allowlist
// keys by design, so the fixture goes through the underlying credstore
// (opened with migrationAllowedKeys) — exactly the residue this
// migration's deprecated-key cleanup must absorb.
func seedDeprecatedKey(t *testing.T, key, val string) {
	t.Helper()
	s, err := OpenForClearAll()
	if err != nil {
		t.Fatalf("OpenForClearAll: %v", err)
	}
	defer func() { _ = s.Close() }()
	if err := s.cs.Set(s.profile, key, val, cccredstore.WithOverwrite()); err != nil {
		t.Fatalf("seed deprecated key %s: %v", key, err)
	}
}

// seedRawAPIToken writes api_token WITHOUT running the migration (via
// the no-migrate clear-all store), to stand up a "token already in the
// keyring" precondition for conflict tests.
func seedRawAPIToken(t *testing.T, val string) {
	t.Helper()
	s, err := OpenForClearAll()
	if err != nil {
		t.Fatalf("OpenForClearAll: %v", err)
	}
	defer func() { _ = s.Close() }()
	if err := s.cs.Set(s.profile, KeyAPIToken, val, cccredstore.WithOverwrite()); err != nil {
		t.Fatalf("seed api_token: %v", err)
	}
}

func wantKeys(t *testing.T, got []string, want ...string) {
	t.Helper()
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("bundle key set = %v, want exactly %v", got, want)
	}
}

func TestSetCredential_StdinAndEnv(t *testing.T) {
	hermetic(t)

	// stdin path: trims surrounding whitespace.
	if err := SetCredential(strings.NewReader("  "+secret+"\n"), ""); err != nil {
		t.Fatalf("SetCredential(stdin): %v", err)
	}
	got, ok, err := func() (string, bool, error) {
		s, e := OpenNoMigrate()
		if e != nil {
			return "", false, e
		}
		defer func() { _ = s.Close() }()
		return s.Token()
	}()
	if err != nil || !ok || got != secret {
		t.Fatalf("stored token mismatch: got=%q ok=%v err=%v", got, ok, err)
	}
	// §1.11.11: a single shared key, nothing else.
	wantKeys(t, bundleKeys(t), KeyAPIToken)

	// --from-env path overwrites the same single key.
	t.Setenv("MY_SECRET_VAR", "env-"+secret)
	if err := SetCredential(nil, "MY_SECRET_VAR"); err != nil {
		t.Fatalf("SetCredential(env): %v", err)
	}
	wantKeys(t, bundleKeys(t), KeyAPIToken)
	// Ingress overwrites: the value must actually be replaced, not just
	// "key still present" (guards a regression to no-overwrite ingress).
	got2, ok2, err2 := func() (string, bool, error) {
		s, e := OpenNoMigrate()
		if e != nil {
			return "", false, e
		}
		defer func() { _ = s.Close() }()
		return s.Token()
	}()
	if err2 != nil || !ok2 || got2 != "env-"+secret {
		t.Fatalf("--from-env must overwrite the stored value: got=%q ok=%v err=%v", got2, ok2, err2)
	}
}

func TestSetCredential_Rejections(t *testing.T) {
	hermetic(t)

	if err := SetCredential(strings.NewReader("   \n"), ""); err == nil {
		t.Fatal("expected error for empty token")
	}
	if err := SetCredential(nil, "DEFINITELY_UNSET_VAR"); err == nil {
		t.Fatal("expected error for unset env var")
	}
	// nil reader + no env var must be a normal error, never a panic.
	if err := SetCredential(nil, ""); err == nil {
		t.Fatal("expected error for nil stdin and no --from-env")
	}
}

// End-to-end §1.8: a plaintext token in the shared config.yml migrates
// into the keyring, the file is scrubbed, the signal fires exactly once,
// and the secret never appears in the signal text.
func TestMigration_EndToEnd_ScrubAndSignal(t *testing.T) {
	hermetic(t)

	sharedPath := sharedConfigPath(t)
	// credstore.Save strips the token, so write a pre-migration file by
	// hand to stand in for a real legacy plaintext store.
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0o700); err != nil {
		t.Fatal(err)
	}
	yaml := "default:\n  url: https://acme.atlassian.net\n  email: u@e\n  api_token: " + secret + "\n"
	if err := os.WriteFile(sharedPath, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := EnsureMigrated(); err != nil {
		t.Fatalf("EnsureMigrated: %v", err)
	}

	// Token is now in the single shared keyring key, and nothing else.
	s, err := OpenNoMigrate()
	if err != nil {
		t.Fatalf("OpenNoMigrate: %v", err)
	}
	defer func() { _ = s.Close() }()
	tok, ok, err := s.Token()
	if err != nil || !ok || tok != secret {
		t.Fatalf("keyring token: got=%q ok=%v err=%v", tok, ok, err)
	}
	wantKeys(t, bundleKeys(t), KeyAPIToken)

	// Plaintext file scrubbed (non-secret fields preserved).
	raw, err := os.ReadFile(sharedPath) //nolint:gosec // G304: test reads its own temp file
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), secret) || strings.Contains(string(raw), "api_token") {
		t.Fatalf("shared file not scrubbed:\n%s", raw)
	}
	if !strings.Contains(string(raw), "u@e") {
		t.Fatalf("scrub dropped non-secret fields:\n%s", raw)
	}

	// Signal fired once, and never contains the secret.
	var buf bytes.Buffer
	FlushMigrationNotice(&buf)
	if buf.Len() == 0 {
		t.Fatal("expected a one-time migration notice")
	}
	if strings.Contains(buf.String(), secret) {
		t.Fatalf("migration notice leaked the secret: %s", buf.String())
	}
	// Consume-once: a second flush is empty.
	var buf2 bytes.Buffer
	FlushMigrationNotice(&buf2)
	if buf2.Len() != 0 {
		t.Fatalf("notice flushed twice: %q", buf2.String())
	}

	// Idempotent: re-running migration is a silent no-op (no conflict).
	if err := EnsureMigrated(); err != nil {
		t.Fatalf("second EnsureMigrated must be idempotent: %v", err)
	}
}

// Amended §1.8: two legacy per-tool plaintext files (cfl yml + jtk json)
// holding the SAME token, with no shared default, collapse onto the
// single shared api_token (no per-tool keys), are scrubbed in place, and
// the migration is idempotent.
func TestMigration_DuplicateLegacy_CollapsesToSingleKey(t *testing.T) {
	hermetic(t)

	cflPath := credstore.LegacyCFLPath()
	jtkPath := credstore.LegacyJTKPath()
	if err := os.MkdirAll(filepath.Dir(cflPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(jtkPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cflPath,
		[]byte("url: https://acme.atlassian.net\nemail: c@e\napi_token: "+secret+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jtkPath,
		[]byte(`{"url":"https://acme.atlassian.net","email":"j@e","api_token":"`+secret+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := EnsureMigrated(); err != nil {
		t.Fatalf("EnsureMigrated: %v", err)
	}

	s, err := OpenNoMigrate()
	if err != nil {
		t.Fatalf("OpenNoMigrate: %v", err)
	}
	defer func() { _ = s.Close() }()
	if v, ok, _ := s.Token(); !ok || v != secret {
		t.Fatalf("api_token: got=%q ok=%v", v, ok)
	}
	wantKeys(t, bundleKeys(t), KeyAPIToken)

	for _, p := range []string{cflPath, jtkPath} {
		raw, rerr := os.ReadFile(p) //nolint:gosec // G304: test reads its own temp file
		if rerr != nil {
			t.Fatal(rerr)
		}
		if strings.Contains(string(raw), secret) || strings.Contains(string(raw), "api_token") {
			t.Fatalf("legacy file %s not scrubbed:\n%s", p, raw)
		}
	}

	if err := EnsureMigrated(); err != nil {
		t.Fatalf("second EnsureMigrated must be idempotent: %v", err)
	}
}

// Amended §1.8: divergent plaintext sources (shared default vs a legacy
// per-tool file) is a HARD conflict — the migration never precedence-picks
// a secret winner. Two-phase guarantee: nothing is written and NO source
// is scrubbed; the error names every location and never the value.
func TestMigration_DivergentSources_ConflictNoMutation(t *testing.T) {
	hermetic(t)
	sharedPath := sharedConfigPath(t)
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0o700); err != nil {
		t.Fatal(err)
	}
	defTok := "DEFAULT-" + secret
	legTok := "LEGACY-" + secret
	if err := os.WriteFile(sharedPath,
		[]byte("default:\n  url: https://acme.atlassian.net\n  api_token: "+defTok+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cflPath := credstore.LegacyCFLPath()
	if err := os.MkdirAll(filepath.Dir(cflPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cflPath,
		[]byte("url: https://acme.atlassian.net\napi_token: "+legTok+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := EnsureMigrated()
	if !errors.Is(err, ErrMigrationConflict) {
		t.Fatalf("want ErrMigrationConflict, got %v", err)
	}
	msg := err.Error()
	if strings.Contains(msg, defTok) || strings.Contains(msg, legTok) {
		t.Fatalf("conflict error leaked a secret value: %s", msg)
	}
	if !strings.Contains(msg, sharedPath) || !strings.Contains(msg, cflPath) {
		t.Fatalf("conflict error must name every source location: %s", msg)
	}

	// Two-phase: no source mutated, nothing written to the keyring.
	for _, p := range []string{sharedPath, cflPath} {
		raw, rerr := os.ReadFile(p) //nolint:gosec // G304: test reads its own temp file
		if rerr != nil {
			t.Fatal(rerr)
		}
		if !strings.Contains(string(raw), "api_token") {
			t.Fatalf("conflict must NOT scrub %s:\n%s", p, raw)
		}
	}
	wantKeys(t, bundleKeys(t)) // empty: nothing written
}

// B3 upgrade fixture: a user upgraded through the per-tool-key build and
// holds ONLY deprecated keyring keys (plaintext already scrubbed), both
// with the same value. The migration must consolidate onto api_token,
// delete BOTH deprecated keys, fire the signal, and leave the bundle
// exactly {api_token}. Without this, S1's key-set collapse would silently
// de-authenticate these users.
func TestMigration_B3UpgradeFixture_DeprecatedKeysOnly(t *testing.T) {
	hermetic(t)
	seedDeprecatedKey(t, "cfl_api_token", secret)
	seedDeprecatedKey(t, "jtk_api_token", secret)
	wantKeys(t, bundleKeys(t), "cfl_api_token", "jtk_api_token") // precondition

	if err := EnsureMigrated(); err != nil {
		t.Fatalf("EnsureMigrated: %v", err)
	}

	s, err := OpenNoMigrate()
	if err != nil {
		t.Fatalf("OpenNoMigrate: %v", err)
	}
	defer func() { _ = s.Close() }()
	if v, ok, _ := s.Token(); !ok || v != secret {
		t.Fatalf("api_token: got=%q ok=%v", v, ok)
	}
	wantKeys(t, bundleKeys(t), KeyAPIToken) // deprecated keys gone

	var buf bytes.Buffer
	FlushMigrationNotice(&buf)
	if buf.Len() == 0 {
		t.Fatal("expected a migration notice for the B3 upgrade path")
	}
	if strings.Contains(buf.String(), secret) {
		t.Fatalf("migration notice leaked the secret: %s", buf.String())
	}

	if err := EnsureMigrated(); err != nil {
		t.Fatalf("second EnsureMigrated must be idempotent: %v", err)
	}
}

// B3 upgrade fixture, divergent: the two deprecated keys disagree. Same
// hard-conflict rule — never precedence-pick — and the two-phase
// guarantee means NEITHER deprecated key is deleted and api_token is not
// written.
func TestMigration_B3UpgradeFixture_DivergentDeprecatedKeys_Conflict(t *testing.T) {
	hermetic(t)
	seedDeprecatedKey(t, "cfl_api_token", "CFL-"+secret)
	seedDeprecatedKey(t, "jtk_api_token", "JTK-"+secret)

	err := EnsureMigrated()
	if !errors.Is(err, ErrMigrationConflict) {
		t.Fatalf("want ErrMigrationConflict, got %v", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("conflict error leaked a secret value: %s", err)
	}
	// Two-phase: nothing deleted, nothing written.
	wantKeys(t, bundleKeys(t), "cfl_api_token", "jtk_api_token")
	s, oerr := OpenNoMigrate()
	if oerr != nil {
		t.Fatalf("OpenNoMigrate: %v", oerr)
	}
	defer func() { _ = s.Close() }()
	if _, ok, _ := s.Token(); ok {
		t.Fatal("conflict must not write api_token")
	}

	// Stably reproducible: a second run must fail the same way (the
	// no-mutation + stable-error pair, independent of map iteration
	// order). State must still be untouched.
	if err2 := EnsureMigrated(); !errors.Is(err2, ErrMigrationConflict) {
		t.Fatalf("conflict must be stable on re-run, got: %v", err2)
	}
	wantKeys(t, bundleKeys(t), "cfl_api_token", "jtk_api_token")
}

// §1.11.11: after a default `config clear` the (single-key) bundle is
// exactly empty, and `config clear --all` on an already-clean bundle is a
// no-op that also leaves it empty.
func TestClear_BundleEmptyAfterClear(t *testing.T) {
	hermetic(t)
	if err := SetCredential(strings.NewReader(secret), ""); err != nil {
		t.Fatalf("seed: %v", err)
	}
	wantKeys(t, bundleKeys(t), KeyAPIToken)

	plan, store, err := PlanClear(ToolCFL, false)
	if err != nil {
		t.Fatalf("PlanClear: %v", err)
	}
	defer func() { _ = store.Close() }()
	if plan.ToolKey != KeyAPIToken {
		t.Fatalf("default clear should target api_token; got %q", plan.ToolKey)
	}
	if err := store.DeleteToken(plan.ToolKey); err != nil {
		t.Fatalf("DeleteToken: %v", err)
	}
	wantKeys(t, bundleKeys(t)) // empty
}

// §1.8 recovery path: a user left with only deprecated per-tool keyring
// keys whose values diverge cannot migrate (hard conflict). The
// documented escape hatch is `config clear --all` — it must wipe the
// WHOLE bundle, deprecated keys included, leaving it exactly empty so a
// clean set-credential can follow.
func TestClearAll_RemovesDeprecatedKeys(t *testing.T) {
	hermetic(t)
	seedDeprecatedKey(t, "cfl_api_token", "CFL-"+secret)
	seedDeprecatedKey(t, "jtk_api_token", "JTK-"+secret)
	wantKeys(t, bundleKeys(t), "cfl_api_token", "jtk_api_token") // precondition

	_, store, perr := PlanClear(ToolCFL, true)
	if perr != nil {
		t.Fatalf("PlanClear: %v", perr)
	}
	defer func() { _ = store.Close() }()
	cleared, err := ClearAll(store)
	if err != nil {
		t.Fatalf("ClearAll: %v", err)
	}
	if !cleared {
		t.Fatal("ClearAll must report the bundle cleared")
	}
	wantKeys(t, bundleKeys(t)) // exactly empty — deprecated keys gone
}

// ClearAll must FAIL LOUD (naming the path) when a surviving legacy file
// is unparseable — never claim success while plaintext may remain — AND,
// because the plaintext scrub runs before the bundle clear, the keyring
// token must survive the failure (the safer, recoverable state).
func TestClearAll_FailsLoudOnUnparseableLegacy(t *testing.T) {
	hermetic(t)
	if err := SetCredential(strings.NewReader(secret), ""); err != nil {
		t.Fatalf("seed: %v", err)
	}
	cflPath := credstore.LegacyCFLPath()
	if err := os.MkdirAll(filepath.Dir(cflPath), 0o700); err != nil {
		t.Fatal(err)
	}
	// Not valid YAML and not valid JSON.
	if err := os.WriteFile(cflPath, []byte(":::not yaml: ["), 0o600); err != nil {
		t.Fatal(err)
	}

	_, store, perr := PlanClear(ToolCFL, true)
	if perr != nil {
		t.Fatalf("PlanClear: %v", perr)
	}
	if store == nil {
		t.Fatal("PlanClear must return an open store on success")
	}
	defer func() { _ = store.Close() }()

	cleared, err := ClearAll(store)
	if err == nil {
		t.Fatal("ClearAll must fail loud on an unparseable legacy file")
	}
	if cleared {
		t.Fatal("ClearAll must not report the bundle cleared when scrub failed")
	}
	if !strings.Contains(err.Error(), cflPath) {
		t.Fatalf("error must name the offending path; got: %v", err)
	}
	// The corrupt file is left in place (user-removable), not silently
	// destroyed.
	if _, statErr := os.Stat(cflPath); statErr != nil {
		t.Fatalf("corrupt legacy file should remain for manual removal: %v", statErr)
	}
	// Safer ordering: scrub runs before the bundle clear, so the keyring
	// token must still be present after the failure (recoverable).
	chk, oerr := OpenNoMigrate()
	if oerr != nil {
		t.Fatalf("reopen: %v", oerr)
	}
	defer func() { _ = chk.Close() }()
	if ok, herr := chk.HasToken(KeyAPIToken); herr != nil || !ok {
		t.Fatalf("keyring token must survive a failed scrub (safer ordering); ok=%v err=%v", ok, herr)
	}
}

// InspectForTool must report presence/source/backend without ever
// returning the token value.
func TestInspectForTool_NoValue(t *testing.T) {
	hermetic(t)
	if err := SetCredential(strings.NewReader(secret), ""); err != nil {
		t.Fatalf("seed: %v", err)
	}
	info, err := InspectForTool(ToolCFL)
	if err != nil {
		t.Fatalf("InspectForTool: %v", err)
	}
	if !info.TokenConfigured {
		t.Fatal("expected TokenConfigured=true")
	}
	if info.Ref != Ref || info.Backend != "file" {
		t.Fatalf("unexpected info: %+v", info)
	}
	// The struct must not carry the secret anywhere.
	if strings.Contains(info.TokenSource, secret) ||
		strings.Contains(info.Backend, secret) ||
		strings.Contains(info.BackendSource, secret) {
		t.Fatalf("InspectForTool leaked the secret: %+v", info)
	}
}

// TestResolveToken_CorruptSharedConfig_DegradesGracefully covers the
// graceful-degradation path: a malformed shared config.yml must NOT
// de-authenticate commands — the token still resolves from the keyring,
// and the user is warned exactly once across repeated resolves (the
// one-shot guard), with no secret in the warning text.
func TestResolveToken_CorruptSharedConfig_DegradesGracefully(t *testing.T) {
	dir := hermetic(t)
	if err := SetCredential(strings.NewReader(secret), ""); err != nil {
		t.Fatalf("seed: %v", err)
	}
	sharedPath := sharedConfigPath(t)
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0o700); err != nil {
		t.Fatal(err)
	}
	// Neither valid YAML nor JSON → credstore.Load wraps ErrCorruptStore.
	if err := os.WriteFile(sharedPath, []byte(":::not yaml: ["), 0o600); err != nil {
		t.Fatal(err)
	}

	// Capture stderr to assert the warning fires exactly once across two
	// resolves (the mutex-guarded one-shot).
	tmp, err := os.CreateTemp(dir, "stderr")
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = tmp
	defer func() { os.Stderr = orig }()

	for i := 0; i < 2; i++ {
		got, src, rerr := ResolveToken(ToolCFL)
		if rerr != nil {
			t.Fatalf("ResolveToken must degrade gracefully on a corrupt shared config; got error: %v", rerr)
		}
		if got != secret {
			t.Fatalf("token must still resolve from the keyring; got %q", got)
		}
		if src != SourceKeyAPI {
			t.Fatalf("source must be the keyring api_token; got %q", src)
		}
	}

	os.Stderr = orig
	_ = tmp.Close()
	out, rerr := os.ReadFile(tmp.Name())
	if rerr != nil {
		t.Fatal(rerr)
	}
	if n := strings.Count(string(out), "shared config store is unreadable"); n != 1 {
		t.Fatalf("warning must fire exactly once across two resolves; got %d\n%s", n, out)
	}
	if strings.Contains(string(out), secret) {
		t.Fatal("warning text leaked the secret")
	}
}

// writeLegacyDefault writes a pre-migration shared config.yml with one
// plaintext default api_token (credstore.Save strips tokens, so the
// fixture is hand-written).
func writeLegacyDefault(t *testing.T, token string) string {
	t.Helper()
	p := sharedConfigPath(t)
	if err := os.WriteFile(p,
		[]byte("default:\n  url: https://acme.atlassian.net\n  api_token: "+token+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// Concurrent-writer race, IDENTICAL value: a writer sets api_token to the
// same value between phase-1 read and phase-2 write. The no-overwrite
// write hits ErrExists; re-resolve sees an equal value → benign,
// migration completes (plaintext scrubbed), nothing clobbered.
func TestMigration_ConcurrentWriter_IdenticalValue_Benign(t *testing.T) {
	hermetic(t)
	shared := writeLegacyDefault(t, secret)

	migrationApplyHook = func(s *Store) {
		if err := s.cs.Set(s.profile, KeyAPIToken, secret, cccredstore.WithOverwrite()); err != nil {
			t.Fatalf("race-seed api_token: %v", err)
		}
	}
	t.Cleanup(func() { migrationApplyHook = nil })

	if err := EnsureMigrated(); err != nil {
		t.Fatalf("identical concurrent value must be benign, got: %v", err)
	}
	s, err := OpenNoMigrate()
	if err != nil {
		t.Fatalf("OpenNoMigrate: %v", err)
	}
	defer func() { _ = s.Close() }()
	if v, ok, _ := s.Token(); !ok || v != secret {
		t.Fatalf("api_token: got=%q ok=%v", v, ok)
	}
	wantKeys(t, bundleKeys(t), KeyAPIToken)
	raw, _ := os.ReadFile(shared) //nolint:gosec // G304: test reads its own temp file
	if strings.Contains(string(raw), "api_token") {
		t.Fatalf("plaintext should still be scrubbed on the benign path:\n%s", raw)
	}
	// The consolidation is effective this run → the one-time notice
	// fires (and never leaks the secret), even on the benign race path.
	var nb bytes.Buffer
	FlushMigrationNotice(&nb)
	if nb.Len() == 0 {
		t.Fatal("benign concurrent migration must still emit the notice")
	}
	if strings.Contains(nb.String(), secret) {
		t.Fatalf("notice leaked the secret: %s", nb.String())
	}
}

// Concurrent-writer race, DIVERGENT value: a writer sets api_token to a
// DIFFERENT value mid-migration. The no-overwrite write hits ErrExists;
// re-resolve sees a different value → hard conflict naming the keyring
// value, the racer's token is NOT clobbered, and no plaintext is scrubbed.
func TestMigration_ConcurrentWriter_DivergentValue_ConflictNoClobber(t *testing.T) {
	hermetic(t)
	srcTok := "SRC-" + secret
	raceTok := "RACE-" + secret
	shared := writeLegacyDefault(t, srcTok)

	migrationApplyHook = func(s *Store) {
		if err := s.cs.Set(s.profile, KeyAPIToken, raceTok, cccredstore.WithOverwrite()); err != nil {
			t.Fatalf("race-seed api_token: %v", err)
		}
	}
	t.Cleanup(func() { migrationApplyHook = nil })

	err := EnsureMigrated()
	if !errors.Is(err, ErrMigrationConflict) {
		t.Fatalf("divergent concurrent value must conflict, got: %v", err)
	}
	msg := err.Error()
	if strings.Contains(msg, srcTok) || strings.Contains(msg, raceTok) {
		t.Fatalf("conflict error leaked a secret value: %s", msg)
	}
	if !strings.Contains(msg, shared) || !strings.Contains(msg, KeyAPIToken) {
		t.Fatalf("conflict must name the source path AND the keyring api_token: %s", msg)
	}
	// The racer's token survives (never silently clobbered).
	s, oerr := OpenNoMigrate()
	if oerr != nil {
		t.Fatalf("OpenNoMigrate: %v", oerr)
	}
	defer func() { _ = s.Close() }()
	if v, ok, _ := s.Token(); !ok || v != raceTok {
		t.Fatalf("racer api_token must be intact; got=%q ok=%v", v, ok)
	}
	raw, _ := os.ReadFile(shared) //nolint:gosec // G304: test reads its own temp file
	if !strings.Contains(string(raw), "api_token") {
		t.Fatalf("conflict must NOT scrub the source:\n%s", raw)
	}
}

// A pre-existing keyring api_token that disagrees with a single legacy
// source: the conflict message must name the keyring api_token as a
// disagreeing party (not read as "divergence across" one source), no
// mutation occurs, and no secret leaks.
func TestMigration_ExistingAPIToken_NamedInConflict(t *testing.T) {
	hermetic(t)
	existing := "EXISTING-" + secret
	legacy := "LEGACY-" + secret
	seedRawAPIToken(t, existing)
	shared := writeLegacyDefault(t, legacy)

	err := EnsureMigrated()
	if !errors.Is(err, ErrMigrationConflict) {
		t.Fatalf("want ErrMigrationConflict, got %v", err)
	}
	msg := err.Error()
	if strings.Contains(msg, existing) || strings.Contains(msg, legacy) {
		t.Fatalf("conflict error leaked a secret value: %s", msg)
	}
	if !strings.Contains(msg, shared) || !strings.Contains(msg, KeyAPIToken) || !strings.Contains(msg, Ref) {
		t.Fatalf("conflict must name the legacy path, api_token, and ref: %s", msg)
	}
	// No mutation: existing keyring value intact, plaintext not scrubbed.
	s, oerr := OpenNoMigrate()
	if oerr != nil {
		t.Fatalf("OpenNoMigrate: %v", oerr)
	}
	defer func() { _ = s.Close() }()
	if v, _, _ := s.Token(); v != existing {
		t.Fatalf("existing api_token must be untouched; got %q", v)
	}
	raw, _ := os.ReadFile(shared) //nolint:gosec // G304: test reads its own temp file
	if !strings.Contains(string(raw), "api_token") {
		t.Fatalf("conflict must NOT scrub the source:\n%s", raw)
	}
}

// Realistic B3 upgrade: a deprecated keyring key AND a plaintext file
// both hold the SAME token → collapse to api_token, delete the
// deprecated key, scrub the plaintext, all in one run; bundle exactly
// {api_token}.
func TestMigration_DeprecatedKeyAndPlaintext_SameValue_Collapses(t *testing.T) {
	hermetic(t)
	seedDeprecatedKey(t, "jtk_api_token", secret)
	shared := writeLegacyDefault(t, secret)

	if err := EnsureMigrated(); err != nil {
		t.Fatalf("EnsureMigrated: %v", err)
	}
	s, err := OpenNoMigrate()
	if err != nil {
		t.Fatalf("OpenNoMigrate: %v", err)
	}
	defer func() { _ = s.Close() }()
	if v, ok, _ := s.Token(); !ok || v != secret {
		t.Fatalf("api_token: got=%q ok=%v", v, ok)
	}
	wantKeys(t, bundleKeys(t), KeyAPIToken)
	raw, _ := os.ReadFile(shared) //nolint:gosec // G304: test reads its own temp file
	if strings.Contains(string(raw), "api_token") {
		t.Fatalf("plaintext not scrubbed:\n%s", raw)
	}
}

// B3 upgrade where the deprecated keyring key and the plaintext file
// DISAGREE → hard conflict, nothing mutated.
func TestMigration_DeprecatedKeyAndPlaintext_Divergent_Conflict(t *testing.T) {
	hermetic(t)
	seedDeprecatedKey(t, "jtk_api_token", "DEP-"+secret)
	shared := writeLegacyDefault(t, "PLAIN-"+secret)

	err := EnsureMigrated()
	if !errors.Is(err, ErrMigrationConflict) {
		t.Fatalf("want ErrMigrationConflict, got %v", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("conflict error leaked a secret value: %s", err)
	}
	wantKeys(t, bundleKeys(t), "jtk_api_token") // deprecated key untouched
	raw, _ := os.ReadFile(shared)               //nolint:gosec // G304: test reads its own temp file
	if !strings.Contains(string(raw), "api_token") {
		t.Fatalf("conflict must NOT scrub the source:\n%s", raw)
	}
}

// The overwrite=true apply path (no production caller today, but
// reachable code): an explicit-overwrite migration replaces an existing
// api_token with the single legacy source value and scrubs plaintext.
func TestMigrateOverwrite_ApplyReplacesExisting(t *testing.T) {
	hermetic(t)
	seedRawAPIToken(t, "OLD-"+secret)
	shared := writeLegacyDefault(t, "NEW-"+secret)

	s, err := OpenForClearAll() // migrationAllowedKeys, no auto-migrate
	if err != nil {
		t.Fatalf("OpenForClearAll: %v", err)
	}
	defer func() { _ = s.Close() }()
	if err := migrateLegacyOverwrite(s, true); err != nil {
		t.Fatalf("overwrite migration: %v", err)
	}
	if v, ok, _ := s.Token(); !ok || v != "NEW-"+secret {
		t.Fatalf("overwrite must replace the existing token; got=%q ok=%v", v, ok)
	}
	raw, _ := os.ReadFile(shared) //nolint:gosec // G304: test reads its own temp file
	if strings.Contains(string(raw), "api_token") {
		t.Fatalf("overwrite path must still scrub plaintext:\n%s", raw)
	}
}
