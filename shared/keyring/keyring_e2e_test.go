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
	t.Cleanup(ResetMigrationNotice)
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
// is unparseable — never claim success while plaintext may remain.
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

	err := ClearAll()
	if err == nil {
		t.Fatal("ClearAll must fail loud on an unparseable legacy file")
	}
	if !strings.Contains(err.Error(), cflPath) {
		t.Fatalf("error must name the offending path; got: %v", err)
	}
	// The corrupt file is left in place (user-removable), not silently
	// destroyed.
	if _, statErr := os.Stat(cflPath); statErr != nil {
		t.Fatalf("corrupt legacy file should remain for manual removal: %v", statErr)
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
