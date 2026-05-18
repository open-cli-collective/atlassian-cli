package keyring

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/credstore"
)

// hermetic isolates HOME/XDG and forces the encrypted-file backend so
// these tests never touch (or prompt for) the real OS keychain. It is a
// local copy of credtest.Hermetic — keyring cannot import credtest
// (credtest imports keyring; that would be an import cycle).
func hermetic(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv(BackendEnvVar, "file")
	t.Setenv("ATLASSIAN_CLI_KEYRING_PASSPHRASE", "e2e-passphrase")
	for _, v := range []string{"ATLASSIAN_API_TOKEN", "CFL_API_TOKEN", "JIRA_API_TOKEN"} {
		t.Setenv(v, "")
	}
	ResetMigrationNotice()
	ResetCorruptWarnOnce()
	t.Cleanup(ResetMigrationNotice)
	t.Cleanup(ResetCorruptWarnOnce)
	return dir
}

//nolint:gosec // G101: test fixture string, not a real credential
const secret = "TOK-pqrSTU-suffix" // distinctive so a leak is unmistakable

func TestSetCredential_StdinAndEnv(t *testing.T) {
	hermetic(t)

	// stdin path: trims surrounding whitespace.
	if err := SetCredential(strings.NewReader("  "+secret+"\n"), KeyAPIToken, ""); err != nil {
		t.Fatalf("SetCredential(stdin): %v", err)
	}
	got, ok, err := func() (string, bool, error) {
		s, e := OpenNoMigrate()
		if e != nil {
			return "", false, e
		}
		defer func() { _ = s.Close() }()
		return s.Token(ToolCFL)
	}()
	if err != nil || !ok || got != secret {
		t.Fatalf("stored token mismatch: got=%q ok=%v err=%v", got, ok, err)
	}

	// --from-env path.
	t.Setenv("MY_SECRET_VAR", "env-"+secret)
	if err := SetCredential(nil, KeyJTKAPIToken, "MY_SECRET_VAR"); err != nil {
		t.Fatalf("SetCredential(env): %v", err)
	}
}

func TestSetCredential_Rejections(t *testing.T) {
	hermetic(t)

	if err := SetCredential(strings.NewReader("   \n"), KeyAPIToken, ""); err == nil {
		t.Fatal("expected error for empty token")
	}
	if err := SetCredential(strings.NewReader("x"), "bogus_key", ""); err == nil {
		t.Fatal("expected error for unknown key")
	}
	if err := SetCredential(nil, KeyAPIToken, "DEFINITELY_UNSET_VAR"); err == nil {
		t.Fatal("expected error for unset env var")
	}
	// nil reader + no env var must be a normal error, never a panic.
	if err := SetCredential(nil, KeyAPIToken, ""); err == nil {
		t.Fatal("expected error for nil stdin and no --from-env")
	}
}

// End-to-end §1.8: a plaintext token in the shared config.yml migrates
// into the keyring, the file is scrubbed, the signal fires exactly once,
// and the secret never appears in the signal text.
func TestMigration_EndToEnd_ScrubAndSignal(t *testing.T) {
	dir := hermetic(t)

	sharedPath := filepath.Join(dir, "atlassian-cli", "config.yml")
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

	// Token is now in the keyring.
	s, err := OpenNoMigrate()
	if err != nil {
		t.Fatalf("OpenNoMigrate: %v", err)
	}
	defer func() { _ = s.Close() }()
	tok, ok, err := s.get(KeyAPIToken)
	if err != nil || !ok || tok != secret {
		t.Fatalf("keyring token: got=%q ok=%v err=%v", tok, ok, err)
	}

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

// End-to-end: legacy per-tool plaintext files (cfl yaml + jtk json) with
// NO shared default migrate to their own override keys, are scrubbed in
// place, and the migration is idempotent regardless of tool order.
func TestMigration_LegacyFiles_ScrubAndIdempotent(t *testing.T) {
	hermetic(t)

	cflPath := credstore.LegacyCFLPath()
	jtkPath := credstore.LegacyJTKPath()
	if err := os.MkdirAll(filepath.Dir(cflPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(jtkPath), 0o700); err != nil {
		t.Fatal(err)
	}
	cflTok, jtkTok := "CFL-"+secret, "JTK-"+secret
	if err := os.WriteFile(cflPath,
		[]byte("url: https://acme.atlassian.net\nemail: c@e\napi_token: "+cflTok+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jtkPath,
		[]byte(`{"url":"https://acme.atlassian.net","email":"j@e","api_token":"`+jtkTok+`"}`), 0o600); err != nil {
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

	// No shared default → each legacy file maps to its own override key.
	if v, ok, _ := s.get(KeyCFLAPIToken); !ok || v != cflTok {
		t.Fatalf("cfl_api_token: got=%q ok=%v", v, ok)
	}
	if v, ok, _ := s.get(KeyJTKAPIToken); !ok || v != jtkTok {
		t.Fatalf("jtk_api_token: got=%q ok=%v", v, ok)
	}

	for _, p := range []string{cflPath, jtkPath} {
		raw, rerr := os.ReadFile(p) //nolint:gosec // G304: test reads its own temp file
		if rerr != nil {
			t.Fatal(rerr)
		}
		if strings.Contains(string(raw), secret) || strings.Contains(string(raw), "api_token") {
			t.Fatalf("legacy file %s not scrubbed:\n%s", p, raw)
		}
	}

	// Idempotent across re-run (cfl-first / jtk-first order is internal
	// to gatherEffective; a second pass must be a silent no-op).
	if err := EnsureMigrated(); err != nil {
		t.Fatalf("second EnsureMigrated must be idempotent: %v", err)
	}
}

// ClearAll must FAIL LOUD (naming the path) when a surviving legacy file
// is unparseable — never claim success while plaintext may remain — AND,
// because the plaintext scrub runs before the bundle clear, the keyring
// token must survive the failure (the safer, recoverable state).
func TestClearAll_FailsLoudOnUnparseableLegacy(t *testing.T) {
	hermetic(t)
	if err := SetCredential(strings.NewReader(secret), KeyAPIToken, ""); err != nil {
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

	plan, store, perr := PlanClear(ToolCFL)
	if perr != nil {
		t.Fatalf("PlanClear: %v", perr)
	}
	if store == nil {
		t.Fatal("PlanClear must return an open store on success")
	}
	defer func() { _ = store.Close() }()
	_ = plan

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

// A legacy token shadowed by a non-empty shared default is scrub-only:
// no override key written, and (since nothing was relocated to a NEW
// key beyond the default) the file is still scrubbed.
func TestMigration_ShadowedLegacy_ScrubOnly(t *testing.T) {
	dir := hermetic(t)
	sharedPath := filepath.Join(dir, "atlassian-cli", "config.yml")
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sharedPath,
		[]byte("default:\n  url: https://acme.atlassian.net\n  api_token: DEFAULT-"+secret+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cflPath := credstore.LegacyCFLPath()
	if err := os.MkdirAll(filepath.Dir(cflPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cflPath,
		[]byte("url: https://acme.atlassian.net\napi_token: SHADOWED-"+secret+"\n"), 0o600); err != nil {
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

	if _, ok, _ := s.get(KeyCFLAPIToken); ok {
		t.Fatal("shadowed legacy value must NOT be written to cfl_api_token")
	}
	if v, ok, _ := s.get(KeyAPIToken); !ok || v != "DEFAULT-"+secret {
		t.Fatalf("api_token: got=%q ok=%v", v, ok)
	}
	// Both plaintext sources scrubbed even though the legacy file's value
	// was dead data.
	for _, p := range []string{sharedPath, cflPath} {
		raw, rerr := os.ReadFile(p) //nolint:gosec // G304: test reads its own temp file
		if rerr != nil {
			t.Fatal(rerr)
		}
		if strings.Contains(string(raw), secret) {
			t.Fatalf("%s not scrubbed:\n%s", p, raw)
		}
	}
}

// InspectForTool must report presence/source/backend without ever
// returning the token value.
func TestInspectForTool_NoValue(t *testing.T) {
	hermetic(t)
	if err := SetCredential(strings.NewReader(secret), KeyAPIToken, ""); err != nil {
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
	if err := SetCredential(strings.NewReader(secret), KeyAPIToken, ""); err != nil {
		t.Fatalf("seed: %v", err)
	}
	sharedPath := filepath.Join(dir, "atlassian-cli", "config.yml")
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

// A pre-existing per-tool override key outranks api_token in
// resolveFromStore. When init writes the shared default it must drop that
// override, otherwise the tool keeps resolving the stale override token
// the user just replaced. PersistTokenForTool owns that invariant.
func TestPersistTokenForTool_DefaultWriteClearsStaleOverride(t *testing.T) {
	hermetic(t)

	const stale = "OLD-cfl-override-token"
	if err := PersistTokenForTool(credstore.ToolCFL, true, stale); err != nil {
		t.Fatalf("seed cfl override: %v", err)
	}
	if tok, _, err := ResolveTokenNoMigrate(credstore.ToolCFL); err != nil || tok != stale {
		t.Fatalf("precondition: cfl should resolve the override; got %q err=%v", tok, err)
	}

	// init reuses the shared default → write api_token. The stale
	// cfl_api_token must be removed so cfl resolves the new value.
	if err := PersistTokenForTool(credstore.ToolCFL, false, secret); err != nil {
		t.Fatalf("PersistTokenForTool(default): %v", err)
	}
	if tok, src, err := ResolveTokenNoMigrate(credstore.ToolCFL); err != nil || tok != secret || src != SourceKeyAPI {
		t.Fatalf("cfl must resolve the saved api_token, not the stale override; got %q src=%q err=%v", tok, src, err)
	}

	s, err := OpenNoMigrate()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = s.Close() }()
	if ok, herr := s.HasToken(KeyFor(credstore.ToolCFL)); herr != nil || ok {
		t.Fatalf("cfl override key must be gone after a default write; present=%v err=%v", ok, herr)
	}

	// Sibling override is independent and must survive a cfl default write.
	if err := PersistTokenForTool(credstore.ToolJTK, true, "jtk-only"); err != nil {
		t.Fatalf("seed jtk override: %v", err)
	}
	if err := PersistTokenForTool(credstore.ToolCFL, false, secret); err != nil {
		t.Fatalf("re-write cfl default: %v", err)
	}
	if tok, _, err := ResolveTokenNoMigrate(credstore.ToolJTK); err != nil || tok != "jtk-only" {
		t.Fatalf("jtk override must be untouched by a cfl default write; got %q err=%v", tok, err)
	}
}

// The mismatch "use X for both tools" choice must actually unify: after
// PersistUnifiedToken both tools resolve the one api_token and neither
// per-tool override remains to shadow it. This is the explicit cross-tool
// action where, unlike a normal default write, the sibling override is
// also cleared.
func TestPersistUnifiedToken_BothToolsResolveSharedAndOverridesGone(t *testing.T) {
	hermetic(t)

	// Post-migration steady state of two divergent legacy files: each
	// tool's old token sits under its own per-tool key.
	if err := PersistTokenForTool(credstore.ToolCFL, true, "cfl-old"); err != nil {
		t.Fatalf("seed cfl override: %v", err)
	}
	if err := PersistTokenForTool(credstore.ToolJTK, true, "jtk-old"); err != nil {
		t.Fatalf("seed jtk override: %v", err)
	}

	// User chose "use jtk for both": jtk's token is persisted as the
	// shared default and both overrides are wiped.
	if err := PersistUnifiedToken("jtk-old"); err != nil {
		t.Fatalf("PersistUnifiedToken: %v", err)
	}

	for _, tool := range []string{credstore.ToolCFL, credstore.ToolJTK} {
		tok, src, err := ResolveTokenNoMigrate(tool)
		if err != nil || tok != "jtk-old" || src != SourceKeyAPI {
			t.Fatalf("%s must resolve the unified api_token; got %q src=%q err=%v", tool, tok, src, err)
		}
	}

	s, err := OpenNoMigrate()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = s.Close() }()
	for _, k := range []string{KeyFor(credstore.ToolCFL), KeyFor(credstore.ToolJTK)} {
		if ok, herr := s.HasToken(k); herr != nil || ok {
			t.Fatalf("override key %s must be gone after unify; present=%v err=%v", k, ok, herr)
		}
	}
}
