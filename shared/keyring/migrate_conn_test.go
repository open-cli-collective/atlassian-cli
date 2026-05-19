package keyring

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// §2.2/MON-5328 S2 — the runtime keyring token migration must be
// TOKEN-ONLY and connection-preserving: driving the REAL migrating path
// (EnsureMigrated → scrubSharedStore) against a pre-strip shared file
// must (a) move the token into the keyring, (b) DELETE every api_token
// node (not blank it to ""), and (c) leave legacy per-tool connection
// fields + non-secret defaults intact on disk — no stripped/whole-profile
// save on a plain runtime path. Without this, S1's schema strip would
// silently drop a user's per-tool url/email on the first command.
func TestRuntimeMigration_TokenOnly_PreservesPerToolConn(t *testing.T) {
	hermetic(t)
	path := sharedConfigPath(t)
	pre := "default:\n" +
		"  url: https://acme.atlassian.net\n" +
		"  email: d@e\n" +
		"  api_token: " + secret + "\n" +
		"cfl:\n" +
		"  url: https://cfl.example.net\n" +
		"  email: c@e\n" +
		"  default_space: SP\n" +
		"jtk:\n" +
		"  api_token: " + secret + "\n" +
		"  default_project: PR\n"
	if err := os.WriteFile(path, []byte(pre), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := EnsureMigrated(); err != nil {
		t.Fatalf("EnsureMigrated: %v", err)
	}

	// (a) token now in the keyring.
	s, err := OpenNoMigrate()
	if err != nil {
		t.Fatalf("OpenNoMigrate: %v", err)
	}
	defer func() { _ = s.Close() }()
	if v, ok, _ := s.Token(); !ok || v != secret {
		t.Fatalf("token not migrated to keyring: got=%q ok=%v", v, ok)
	}

	raw, err := os.ReadFile(path) //nolint:gosec // test reads its own temp file
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)

	// (b) every api_token node DELETED — not present at all, not "" .
	if strings.Contains(got, "api_token") || strings.Contains(got, secret) {
		t.Fatalf("scrub must DELETE api_token nodes, file still has it:\n%s", got)
	}

	// (c) legacy per-tool connection + non-secret defaults preserved
	// (no stripped/whole-profile save on the runtime path).
	var doc map[string]map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if doc["cfl"]["url"] != "https://cfl.example.net" {
		t.Fatalf("runtime scrub dropped legacy per-tool cfl.url:\n%s", got)
	}
	if doc["cfl"]["email"] != "c@e" {
		t.Fatalf("runtime scrub dropped legacy per-tool cfl.email:\n%s", got)
	}
	if doc["cfl"]["default_space"] != "SP" {
		t.Fatalf("runtime scrub dropped cfl.default_space:\n%s", got)
	}
	if doc["jtk"]["default_project"] != "PR" {
		t.Fatalf("runtime scrub dropped jtk.default_project:\n%s", got)
	}
	if doc["default"]["url"] != "https://acme.atlassian.net" {
		t.Fatalf("runtime scrub dropped default.url:\n%s", got)
	}
}

// deleteYAMLKey must excise api_token ANYWHERE — nested mappings and
// inside sequences — not just top-level sections (the "excise a single
// api_token anywhere" claim).
func TestScrubSharedStore_NestedAndSequenceRecursion(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yml")
	in := "default:\n  url: https://acme.atlassian.net\n  api_token: T0\n" +
		"nested:\n  inner:\n    api_token: T1\n    keep: yes\n" +
		"list:\n  - name: a\n    api_token: T2\n  - name: b\n"
	if err := os.WriteFile(p, []byte(in), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := scrubSharedStore(p); err != nil {
		t.Fatalf("scrub: %v", err)
	}
	raw, _ := os.ReadFile(p) //nolint:gosec // test reads its own temp file
	got := string(raw)
	for _, leaked := range []string{"api_token", "T0", "T1", "T2"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("api_token must be excised everywhere (incl. nested/sequence); still has %q:\n%s", leaked, got)
		}
	}
	for _, kept := range []string{"https://acme.atlassian.net", "keep: yes", "name: a", "name: b"} {
		if !strings.Contains(got, kept) {
			t.Fatalf("non-token content must be preserved; missing %q:\n%s", kept, got)
		}
	}
}

// No api_token present → scrubSharedStore is a no-op and leaves the file
// byte-identical (no needless yaml.Node re-emit churn). Absent file is
// also a no-op; unparseable yaml is a hard error.
func TestScrubSharedStore_NoopAndError(t *testing.T) {
	dir := t.TempDir()

	clean := filepath.Join(dir, "clean.yml")
	body := "default:\n  url: https://acme.atlassian.net\n  email: u@e\n"
	if err := os.WriteFile(clean, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := scrubSharedStore(clean); err != nil {
		t.Fatalf("no-op scrub must succeed: %v", err)
	}
	raw, _ := os.ReadFile(clean) //nolint:gosec // test reads its own temp file
	if string(raw) != body {
		t.Fatalf("token-free file must be byte-identical after scrub:\ngot:\n%s\nwant:\n%s", raw, body)
	}

	if err := scrubSharedStore(filepath.Join(dir, "absent.yml")); err != nil {
		t.Fatalf("absent file must be a no-op, got: %v", err)
	}

	bad := filepath.Join(dir, "bad.yml")
	if err := os.WriteFile(bad, []byte("default: : :: ["), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := scrubSharedStore(bad); err == nil {
		t.Fatal("unparseable yaml must be a hard error")
	}
}
