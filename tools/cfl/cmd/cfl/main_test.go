package main

import (
	"bytes"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/credtest"
)

// unreachableURL returns a URL whose dial fails fast and
// deterministically on any host: the listener stays bound for the test's
// lifetime (no close-then-rebind TOCTOU) but every accepted connection is
// closed immediately, so the HTTP client gets an instant EOF/reset
// instead of a port-9-style stall or a spurious success.
func unreachableURL(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		for {
			c, aerr := l.Accept()
			if aerr != nil {
				return
			}
			_ = c.Close()
		}
	}()
	t.Cleanup(func() { _ = l.Close() })
	return "http://" + l.Addr().String()
}

// §1.11.6 acceptance: these tests invoke the REAL cfl entrypoint (this
// package's main(), re-executed as a subprocess) on a migrating
// invocation and assert the one-time §1.8 notice reaches stderr —
// crucially even when the command then exits non-zero, since main()
// flushes the notice before os.Exit. They also pin §1.11.11: after a
// successful migration the keyring bundle key set is exactly {api_token}
// (no deprecated per-tool keys survive), including the B3 upgrade path.
//
// `_migration` JSON signal is not applicable for atlassian-cli command
// paths today (no command emits a structured migration envelope), so it
// is intentionally not asserted here.

// migrationLine is the stable substring recordMigration writes; it never
// contains the secret (asserted below).
const migrationLine = "consolidated the API token into the OS keyring"

// entrypointEnv re-execs this test binary as the real cfl CLI.
const entrypointEnv = "CFL_ENTRYPOINT_TEST"

func TestMain(m *testing.M) {
	if os.Getenv(entrypointEnv) == "1" {
		// Subprocess: behave exactly as the installed cfl binary. main()
		// reads os.Args (the CLI args we exec with) and calls os.Exit, so
		// the testing framework's flag parsing is never reached.
		main()
		return
	}
	os.Exit(m.Run())
}

// runCLI re-execs the test binary as cfl with args, in a hermetic
// HOME/XDG rooted at dir and the encrypted-file keyring backend. It
// returns stderr and the process exit code.
func runCLI(t *testing.T, dir string, stdin string, args ...string) (stderr string, code int) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	cmd := exec.Command(exe, args...) //nolint:gosec // G204: exe is this test binary
	cmd.Env = append(os.Environ(),
		entrypointEnv+"=1",
		"HOME="+dir,
		"XDG_CONFIG_HOME="+dir,
		"ATLASSIAN_CLI_KEYRING_BACKEND=file",
		"ATLASSIAN_CLI_KEYRING_PASSPHRASE=credtest-passphrase",
		// Token/URL env must not shadow the keyring or the fixture file.
		"ATLASSIAN_API_TOKEN=", "CFL_API_TOKEN=", "JIRA_API_TOKEN=",
		"ATLASSIAN_URL=", "CFL_URL=",
	)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var errBuf bytes.Buffer
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	// A nil ProcessState means the subprocess never started (exec
	// failure) — a test-infra error, not a CLI exit. Fail loud rather
	// than panic in ExitCode().
	if cmd.ProcessState == nil {
		t.Fatalf("subprocess did not start: %v", runErr)
	}
	return errBuf.String(), cmd.ProcessState.ExitCode()
}

func writeLegacyShared(t *testing.T, dir, url, token string) string {
	t.Helper()
	p := filepath.Join(dir, "atlassian-cli", "config.yml")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	yaml := "default:\n  url: " + url + "\n  email: u@e\n  api_token: " + token + "\n"
	if err := os.WriteFile(p, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

//nolint:gosec // G101: test fixture token, not a real credential
const legacyTok = "LEGACY-entrypoint-TOK"

// Plaintext migration via the real entrypoint, exit 0: set-credential
// opens the keyring (running the §1.8 migration on the seeded legacy
// plaintext), the notice reaches stderr, the plaintext is scrubbed, and
// the bundle is exactly {api_token}.
func TestEntrypoint_PlaintextMigration_Exit0(t *testing.T) {
	dir := credtest.Hermetic(t)
	shared := writeLegacyShared(t, dir, "https://acme.atlassian.net", legacyTok)

	// §1.5.2: set-credential needs --ref / --key / --stdin explicitly;
	// --overwrite is required because the §1.8 migration writes legacyTok
	// to the keyring during Open() before the new value lands.
	stderr, code := runCLI(t, dir, "NEW-token-from-stdin\n",
		"set-credential",
		"--ref", "atlassian-cli/default",
		"--key", "api_token",
		"--stdin", "--overwrite")
	if code != 0 {
		t.Fatalf("set-credential should exit 0; got %d\nstderr:\n%s", code, stderr)
	}
	if !strings.Contains(stderr, migrationLine) {
		t.Fatalf("migration notice missing from stderr:\n%s", stderr)
	}
	if strings.Contains(stderr, legacyTok) {
		t.Fatalf("stderr leaked the secret: %s", stderr)
	}
	if got := credtest.BundleKeys(t); strings.Join(got, ",") != "api_token" {
		t.Fatalf("§1.11.11: bundle key set = %v, want exactly [api_token]", got)
	}
	raw, _ := os.ReadFile(shared) //nolint:gosec // G304: test reads its own temp file
	if strings.Contains(string(raw), "api_token") {
		t.Fatalf("legacy plaintext not scrubbed:\n%s", raw)
	}
}

// The migration signal must survive a non-zero exit: a migrating
// invocation whose subsequent command fails (unreachable URL) still
// flushes the notice to stderr before os.Exit.
func TestEntrypoint_Migration_SurvivesNonZeroExit(t *testing.T) {
	dir := credtest.Hermetic(t)
	// Closed ephemeral port → `me` fails fast after migration ran.
	writeLegacyShared(t, dir, unreachableURL(t), legacyTok)

	stderr, code := runCLI(t, dir, "", "me")
	if code == 0 {
		t.Fatalf("me against an unreachable URL must exit non-zero\nstderr:\n%s", stderr)
	}
	if !strings.Contains(stderr, migrationLine) {
		t.Fatalf("migration notice must be flushed BEFORE the non-zero exit:\n%s", stderr)
	}
	if strings.Contains(stderr, legacyTok) {
		t.Fatalf("stderr leaked the secret: %s", stderr)
	}
}

// B3 upgrade fixture via the real entrypoint: a user holding ONLY the
// deprecated per-tool keyring keys (no plaintext) is consolidated onto
// the single api_token, the deprecated keys are deleted, and the bundle
// is exactly {api_token}.
func TestEntrypoint_B3UpgradeFixture_DeprecatedKeysOnly(t *testing.T) {
	dir := credtest.Hermetic(t)
	credtest.SeedDeprecatedKey(t, "cfl_api_token", legacyTok)
	credtest.SeedDeprecatedKey(t, "jtk_api_token", legacyTok)

	// B3 upgrade: migration consolidates deprecated per-tool keys onto
	// api_token, so by the time set-credential's write fires the entry
	// already exists — --overwrite is required for the new value to land.
	stderr, code := runCLI(t, dir, "NEW-token-from-stdin\n",
		"set-credential",
		"--ref", "atlassian-cli/default",
		"--key", "api_token",
		"--stdin", "--overwrite")
	if code != 0 {
		t.Fatalf("set-credential should exit 0; got %d\nstderr:\n%s", code, stderr)
	}
	if !strings.Contains(stderr, migrationLine) {
		t.Fatalf("B3 upgrade must emit the migration notice:\n%s", stderr)
	}
	if got := credtest.BundleKeys(t); strings.Join(got, ",") != "api_token" {
		t.Fatalf("§1.11.11: B3 upgrade must leave exactly [api_token]; got %v", got)
	}
}

// §2.2/MON-5328 detect-before-mutate at the REAL entrypoint: a
// pre-MON-5328 shared config whose per-tool connection diverges from
// `default`, plus a plaintext api_token, must make `cfl init` fail loud
// from detectAndReconcile BEFORE keyring.EnsureMigrated runs — so the
// token is NOT migrated/scrubbed and the file is untouched. Pins the
// prior Codex blocker at the runInit level (re-introducing
// EnsureMigrated-first would fail this test).
func TestEntrypoint_DivergentInit_NoMutationBeforeFailLoud(t *testing.T) {
	dir := credtest.Hermetic(t)
	p := filepath.Join(dir, "atlassian-cli", "config.yml")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	pre := "default:\n  url: https://default.atlassian.net\n  email: u@e\n  api_token: PLAINTEXT_TOK\ncfl:\n  url: https://cfl-only.atlassian.net\n"
	if err := os.WriteFile(p, []byte(pre), 0o600); err != nil {
		t.Fatal(err)
	}

	stderr, code := runCLI(t, dir, "", "init", "--no-verify")
	if code == 0 {
		t.Fatalf("divergent init must exit non-zero; stderr:\n%s", stderr)
	}
	if !strings.Contains(stderr, "cfl.url") {
		t.Fatalf("must fail loud naming the divergent field/source; stderr:\n%s", stderr)
	}
	if strings.Contains(stderr, "cfl-only.atlassian.net") || strings.Contains(stderr, "PLAINTEXT_TOK") {
		t.Fatalf("fail-loud must not leak values; stderr:\n%s", stderr)
	}
	// EnsureMigrated never ran: keyring empty, token still on disk.
	if got := credtest.BundleKeys(t); len(got) != 0 {
		t.Fatalf("token must NOT be migrated before the fail-loud; bundle=%v", got)
	}
	raw, _ := os.ReadFile(p) //nolint:gosec // test reads its own temp file
	if !strings.Contains(string(raw), "PLAINTEXT_TOK") || !strings.Contains(string(raw), "cfl-only.atlassian.net") {
		t.Fatalf("divergent init must mutate NOTHING on disk:\n%s", raw)
	}
}
