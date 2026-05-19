package initcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/cli-common/statedirtest"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
)

// oldSharedFixture writes a fixture at the prior hand-rolled shared
// location (statedirtest sets $XDG_CONFIG_HOME, so oldSharedPath
// resolves there); the caller passes a DISTINCT sharedPath as "new" so
// §3.2 relocation engages on Linux (resolver would else collapse them).
func oldSharedFixture(t *testing.T, body string) string {
	t.Helper()
	// Explicit $XDG_CONFIG_HOME override (mirrors relocate_test.go's
	// oldBase) so the old-shared path is deterministic and does not
	// silently void if statedirtest's platform behavior changes.
	base := filepath.Join(t.TempDir(), "oldbase")
	t.Setenv("XDG_CONFIG_HOME", base)
	p := filepath.Join(base, "atlassian-cli", "config.yml")
	testutil.RequireNoError(t, os.MkdirAll(filepath.Dir(p), 0o700))
	testutil.RequireNoError(t, os.WriteFile(p, []byte(body), 0o600))
	return p
}

// Major (Codex): old-only shared config at the prior path is COPIED
// (gated apply) into the new path during jtk init and reconciled in.
func TestReconcile_OldSharedOnly_CopiedAtInit(t *testing.T) {
	statedirtest.Hermetic(t)
	tmp := t.TempDir()
	oldSharedFixture(t, "default:\n  url: https://old-shared.atlassian.net\n  email: u@e\njtk:\n  default_project: PROJ\n")
	newPath := filepath.Join(tmp, "newshared", "config.yml")

	v, _, _ := newReconcileView()
	r, err := detectAndReconcile(v,
		filepath.Join(tmp, "jtk.json"), filepath.Join(tmp, "cfl.yml"),
		newPath, "", "", "", "", "")
	testutil.RequireNoError(t, err)
	if _, statErr := os.Stat(newPath); statErr != nil {
		t.Fatalf("old-only must be COPIED to the new path at init: %v", statErr)
	}
	testutil.Equal(t, "https://old-shared.atlassian.net", r.store.Default.URL)
	testutil.Equal(t, "PROJ", r.store.JTK.DefaultProject)
}

// Major (Codex): a pending per-tool connection divergence blocks the
// relocation copy entirely — fail loud, mutate nothing.
func TestReconcile_OldShared_PerToolDivergencePending_NoCopy(t *testing.T) {
	statedirtest.Hermetic(t)
	tmp := t.TempDir()
	oldSharedFixture(t, "default:\n  url: https://old-shared.atlassian.net\n  email: u@e\n")
	newPath := filepath.Join(tmp, "newshared", "config.yml")
	jtkPath := filepath.Join(tmp, "jtk.json")
	testutil.RequireNoError(t, os.WriteFile(jtkPath,
		[]byte(`{"url":"https://divergent.atlassian.net","email":"u@e","api_token":"t"}`), 0o600))

	v, _, _ := newReconcileView()
	_, err := detectAndReconcile(v, jtkPath, filepath.Join(tmp, "cfl.yml"),
		newPath, "", "", "", "", "")
	testutil.RequireError(t, err)
	if _, statErr := os.Stat(newPath); !os.IsNotExist(statErr) {
		t.Fatal("no copy may occur while a per-tool divergence is pending")
	}
}

// These tests are pure: detectAndReconcile does NO keyring I/O (the B3
// leak-regression rule — keyring access lives only in the command
// layer), so no hermetic credtest harness is required.

func newReconcileView() (*view.View, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	v := view.NewWithFormat("table", true)
	v.Out = stdout
	v.Err = stderr
	return v, stdout, stderr
}

func TestReconcile_NoFilesAnywhere(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	v, _, _ := newReconcileView()
	r, err := detectAndReconcile(v,
		filepath.Join(tmp, "jtk.json"), filepath.Join(tmp, "cfl.yml"),
		filepath.Join(tmp, "shared.yml"), "", "", "", "", "")
	testutil.RequireNoError(t, err)
	testutil.NotNil(t, r)
	testutil.Equal(t, "", r.prefill.URL)
	testutil.Equal(t, false, r.affectsSibling)
}

func TestReconcile_OnlyJTKLegacy_FoldsIntoDefault(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	jtkPath := filepath.Join(tmp, "jtk.json")
	body := `{"url":"https://acme.atlassian.net","email":"u@e","api_token":"jtk-tok","default_project":"PROJ"}`
	testutil.RequireNoError(t, os.WriteFile(jtkPath, []byte(body), 0o600))

	v, _, _ := newReconcileView()
	r, err := detectAndReconcile(v, jtkPath,
		filepath.Join(tmp, "cfl.yml"), filepath.Join(tmp, "shared.yml"),
		"", "", "", "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "https://acme.atlassian.net", r.prefill.URL)
	testutil.Equal(t, "PROJ", r.prefill.DefaultProject)
	// Connection folded into the shared default (single-source §2.2).
	testutil.Equal(t, "https://acme.atlassian.net", r.store.Default.URL)
	testutil.Equal(t, "u@e", r.store.Default.Email)
	testutil.Equal(t, "PROJ", r.store.JTK.DefaultProject)
	testutil.Equal(t, []string{jtkPath}, r.consumedLegacies)
	// First-time legacy migration: no usable shared default existed, so
	// affectsSibling must be false. Pins the documented pre-fold
	// judgement (a post-fold check would wrongly report true).
	testutil.Equal(t, false, r.affectsSibling)
}

func TestReconcile_FlagOverridesPrefill(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	v, _, _ := newReconcileView()
	r, err := detectAndReconcile(v,
		filepath.Join(tmp, "jtk.json"), filepath.Join(tmp, "cfl.yml"),
		filepath.Join(tmp, "shared.yml"),
		"https://flag.atlassian.net", "flag@e.com", "flag-tok", "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "https://flag.atlassian.net", r.prefill.URL)
	testutil.Equal(t, "flag-tok", r.prefill.APIToken)
}

func TestReconcile_CorruptSharedAborts(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	sharedPath := filepath.Join(tmp, "shared.yml")
	testutil.RequireNoError(t, os.WriteFile(sharedPath, []byte("default: : :: ["), 0o600))
	v, _, stderr := newReconcileView()
	_, err := detectAndReconcile(v,
		filepath.Join(tmp, "jtk.json"), filepath.Join(tmp, "cfl.yml"),
		sharedPath, "", "", "", "", "")
	testutil.RequireError(t, err)
	if !strings.Contains(stderr.String(), "unreadable") {
		t.Errorf("expected unreadable warning; got: %s", stderr.String())
	}
}

func TestReconcile_CorruptJTKLegacyAborts(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	jtkPath := filepath.Join(tmp, "jtk.json")
	corrupt := []byte("{not json")
	testutil.RequireNoError(t, os.WriteFile(jtkPath, corrupt, 0o600))
	v, _, stderr := newReconcileView()
	_, err := detectAndReconcile(v, jtkPath,
		filepath.Join(tmp, "cfl.yml"), filepath.Join(tmp, "shared.yml"),
		"", "", "", "", "")
	testutil.RequireError(t, err)
	if !strings.Contains(stderr.String(), "unreadable") {
		t.Errorf("corrupt own-legacy must surface an actionable 'unreadable' message; got: %s", stderr.String())
	}
	// Fail-loud must mutate NOTHING: the unreadable file is byte-identical
	// afterwards (a future refactor that truncates/overwrites before the
	// early return would otherwise pass undetected).
	after, _ := os.ReadFile(jtkPath) //nolint:gosec // test reads its own temp file
	if string(after) != string(corrupt) {
		t.Errorf("corrupt legacy file was mutated by a failed detect; want byte-identical")
	}
}

func TestReconcile_CorruptCFLLegacyDowngradesToWarning(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	jtkPath := filepath.Join(tmp, "jtk.json")
	testutil.RequireNoError(t, os.WriteFile(jtkPath,
		[]byte(`{"url":"https://acme.atlassian.net","email":"u@e","api_token":"t"}`), 0o600))
	cflPath := filepath.Join(tmp, "cfl.yml")
	testutil.RequireNoError(t, os.WriteFile(cflPath, []byte(": ::: ["), 0o600))
	v, stdout, stderr := newReconcileView()
	r, err := detectAndReconcile(v, jtkPath, cflPath,
		filepath.Join(tmp, "shared.yml"), "", "", "", "", "")
	testutil.RequireNoError(t, err) // sibling-corrupt is a warning, not fatal
	testutil.Equal(t, "https://acme.atlassian.net", r.store.Default.URL)
	if !strings.Contains(stdout.String()+stderr.String(), "sibling cfl config") {
		t.Errorf("expected sibling-ignored note; got stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestReconcile_BothLegaciesAligned_FoldsAndPreservesDefaults(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	jtkPath := filepath.Join(tmp, "jtk.json")
	cflPath := filepath.Join(tmp, "cfl.yml")
	testutil.RequireNoError(t, os.WriteFile(jtkPath,
		[]byte(`{"url":"https://acme.atlassian.net","email":"u@e","api_token":"t","default_project":"PR"}`), 0o600))
	testutil.RequireNoError(t, os.WriteFile(cflPath,
		[]byte("url: https://acme.atlassian.net\nemail: u@e\napi_token: t\ndefault_space: SP\noutput_format: json\n"), 0o600))
	v, _, _ := newReconcileView()
	r, err := detectAndReconcile(v, jtkPath, cflPath,
		filepath.Join(tmp, "shared.yml"), "", "", "", "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "https://acme.atlassian.net", r.store.Default.URL)
	testutil.Equal(t, "PR", r.store.JTK.DefaultProject)
	testutil.Equal(t, "SP", r.store.CFL.DefaultSpace)
	testutil.Equal(t, "json", r.store.CFL.OutputFormat)
}

func TestReconcile_DivergentLegacies_FailLoudNoValueLeak(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	jtkPath := filepath.Join(tmp, "jtk.json")
	cflPath := filepath.Join(tmp, "cfl.yml")
	testutil.RequireNoError(t, os.WriteFile(jtkPath,
		[]byte(`{"url":"https://jtk-host.atlassian.net","email":"u@e","api_token":"t"}`), 0o600))
	testutil.RequireNoError(t, os.WriteFile(cflPath,
		[]byte("url: https://cfl-host.atlassian.net\nemail: u@e\napi_token: t\n"), 0o600))
	v, _, _ := newReconcileView()
	_, err := detectAndReconcile(v, jtkPath, cflPath,
		filepath.Join(tmp, "shared.yml"), "", "", "", "", "")
	testutil.RequireError(t, err)
	msg := err.Error()
	if strings.Contains(msg, "jtk-host.atlassian.net") || strings.Contains(msg, "cfl-host.atlassian.net") {
		t.Fatalf("fail-loud must not leak values: %s", msg)
	}
	if !strings.Contains(msg, "url:") || !strings.Contains(msg, jtkPath) || !strings.Contains(msg, cflPath) {
		t.Fatalf("fail-loud must name the conflicting field + every source path: %s", msg)
	}
	// email is identical across sources → must NOT spuriously conflict.
	if strings.Contains(msg, "email:") {
		t.Fatalf("agreed field must not spuriously conflict: %s", msg)
	}
}

func TestReconcile_SharedPerToolConnDivergence_FailLoud(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	sharedPath := filepath.Join(tmp, "shared.yml")
	// Pre-MON-5328 file: default vs a divergent per-tool jtk url. The
	// canonical Store drops jtk.url, but the migration projection still
	// sees it → divergence must fail loud.
	testutil.RequireNoError(t, os.WriteFile(sharedPath,
		[]byte("default:\n  url: https://default.atlassian.net\n  email: u@e\njtk:\n  url: https://jtk-only.atlassian.net\n"), 0o600))
	v, _, _ := newReconcileView()
	_, err := detectAndReconcile(v,
		filepath.Join(tmp, "jtk.json"), filepath.Join(tmp, "cfl.yml"),
		sharedPath, "", "", "", "", "")
	testutil.RequireError(t, err)
	if !strings.Contains(err.Error(), "jtk.url") {
		t.Fatalf("must name the shared per-tool section.field: %s", err.Error())
	}
}

// Pins the prior Codex blocker: detectAndReconcile (init runs it BEFORE
// keyring.EnsureMigrated) must fail loud AND mutate nothing on a
// divergent pre-MON-5328 per-tool connection + plaintext api_token.
func TestReconcile_DivergentWithToken_NoMutation(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	sharedPath := filepath.Join(tmp, "shared.yml")
	pre := "default:\n  url: https://default.atlassian.net\n  email: u@e\n  api_token: PLAINTEXT_TOK\njtk:\n  url: https://jtk-only.atlassian.net\n"
	testutil.RequireNoError(t, os.WriteFile(sharedPath, []byte(pre), 0o600))
	before, _ := os.ReadFile(sharedPath) //nolint:gosec // test reads its own temp file

	v, _, _ := newReconcileView()
	_, err := detectAndReconcile(v,
		filepath.Join(tmp, "jtk.json"), filepath.Join(tmp, "cfl.yml"),
		sharedPath, "", "", "", "", "")
	testutil.RequireError(t, err)
	if !strings.Contains(err.Error(), "jtk.url") {
		t.Fatalf("expected connection divergence; got: %v", err)
	}
	after, _ := os.ReadFile(sharedPath) //nolint:gosec // test reads its own temp file
	if string(before) != string(after) {
		t.Fatalf("divergent detect must mutate NOTHING; file changed:\n%s", after)
	}
	if !strings.Contains(string(after), "PLAINTEXT_TOK") {
		t.Fatalf("token must NOT be scrubbed on divergence:\n%s", after)
	}
}

// Re-running init with the connection UNCHANGED must NOT nag about
// affecting the sibling: the resolved connection is canonically
// equivalent to the shared default already on disk. Pins the
// §2.2/MON-5328 fix for the daemon-flagged UX regression.
func TestReconcile_NoNagWhenConnUnchanged(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	sharedPath := filepath.Join(tmp, "shared.yml")
	testutil.RequireNoError(t, os.WriteFile(sharedPath,
		[]byte("default:\n  url: https://acme.atlassian.net\n  email: u@e\n"), 0o600))
	v, _, _ := newReconcileView()
	r, err := detectAndReconcile(v,
		filepath.Join(tmp, "jtk.json"), filepath.Join(tmp, "cfl.yml"),
		sharedPath, "", "", "", "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, false, r.affectsSibling)
}

// When a usable shared default exists AND the resolved connection
// actually DIFFERS from it (a legacy file contributes a cloud_id the
// default lacked), the save changes what cfl reads, so the sibling
// confirmation MUST still fire.
func TestReconcile_NagsWhenResolvedConnDiffers(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	sharedPath := filepath.Join(tmp, "shared.yml")
	testutil.RequireNoError(t, os.WriteFile(sharedPath,
		[]byte("default:\n  url: https://acme.atlassian.net\n  email: u@e\n"), 0o600))
	jtkPath := filepath.Join(tmp, "jtk.json")
	body := `{"url":"https://acme.atlassian.net","email":"u@e","cloud_id":"CID"}`
	testutil.RequireNoError(t, os.WriteFile(jtkPath, []byte(body), 0o600))
	v, _, _ := newReconcileView()
	r, err := detectAndReconcile(v, jtkPath,
		filepath.Join(tmp, "cfl.yml"), sharedPath, "", "", "", "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, true, r.affectsSibling)
}

// A per-tool default the user already set in the SHARED store must win
// over a stale value in a still-present legacy file. Pins the
// daemon-flagged silent-revert regression in preserveDefaultsAndCollect.
func TestReconcile_SharedPerToolDefaultWinsOverLegacy(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	sharedPath := filepath.Join(tmp, "shared.yml")
	testutil.RequireNoError(t, os.WriteFile(sharedPath,
		[]byte("default:\n  url: https://acme.atlassian.net\n  email: u@e\njtk:\n  default_project: NEW\n"), 0o600))
	jtkPath := filepath.Join(tmp, "jtk.json")
	body := `{"url":"https://acme.atlassian.net","email":"u@e","default_project":"OLD"}`
	testutil.RequireNoError(t, os.WriteFile(jtkPath, []byte(body), 0o600))
	v, _, _ := newReconcileView()
	r, err := detectAndReconcile(v, jtkPath,
		filepath.Join(tmp, "cfl.yml"), sharedPath, "", "", "", "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "NEW", r.store.JTK.DefaultProject) // shared store wins
}

func TestApplyResultToStore_WritesDefaultAndJTKDefault(t *testing.T) {
	t.Parallel()
	store := &credstore.Store{CFL: credstore.ToolSection{DefaultSpace: "KEEP"}}
	applyResultToStore(store, &config.Config{
		URL: "https://acme.atlassian.net/", Email: "u@e",
		AuthMethod: "basic", DefaultProject: "PR",
	})
	testutil.Equal(t, "https://acme.atlassian.net", store.Default.URL) // normalized
	testutil.Equal(t, "u@e", store.Default.Email)
	testutil.Equal(t, "PR", store.JTK.DefaultProject)
	testutil.Equal(t, "KEEP", store.CFL.DefaultSpace) // sibling section untouched
}
