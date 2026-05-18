package init

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/confluence-cli/internal/config"
)

// Pure: detectAndReconcile does NO keyring I/O (B3 leak-regression
// rule), so no hermetic harness is needed here.

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
		filepath.Join(tmp, "cfl.yml"), filepath.Join(tmp, "jtk.json"),
		filepath.Join(tmp, "shared.yml"), "", "", "", "")
	testutil.RequireNoError(t, err)
	testutil.NotNil(t, r)
	testutil.Equal(t, "", r.prefill.URL)
	testutil.Equal(t, false, r.affectsSibling)
}

func TestReconcile_OnlyCFLLegacy_FoldsIntoDefault(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	cflPath := filepath.Join(tmp, "cfl.yml")
	testutil.RequireNoError(t, os.WriteFile(cflPath,
		[]byte("url: https://acme.atlassian.net\nemail: u@e\napi_token: cfl-tok\ndefault_space: SP\n"), 0o600))

	v, _, _ := newReconcileView()
	r, err := detectAndReconcile(v, cflPath,
		filepath.Join(tmp, "jtk.json"), filepath.Join(tmp, "shared.yml"),
		"", "", "", "")
	testutil.RequireNoError(t, err)
	// Prefill URL is /wiki-suffixed for cfl; store default is the base.
	testutil.Equal(t, "https://acme.atlassian.net/wiki", r.prefill.URL)
	testutil.Equal(t, "https://acme.atlassian.net", r.store.Default.URL)
	testutil.Equal(t, "u@e", r.store.Default.Email)
	testutil.Equal(t, "SP", r.store.CFL.DefaultSpace)
	testutil.Equal(t, []string{cflPath}, r.consumedLegacies)
	// First-time legacy migration: there was NO usable shared default,
	// so this is not "editing a config the sibling already uses". Pins
	// the documented pre-fold judgement (a post-fold check would wrongly
	// see the just-folded connection and report true).
	testutil.Equal(t, false, r.affectsSibling)
}

func TestReconcile_FlagOverridesPrefill(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	v, _, _ := newReconcileView()
	r, err := detectAndReconcile(v,
		filepath.Join(tmp, "cfl.yml"), filepath.Join(tmp, "jtk.json"),
		filepath.Join(tmp, "shared.yml"),
		"https://flag.atlassian.net", "flag@e.com", "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "https://flag.atlassian.net", r.prefill.URL)
	testutil.Equal(t, "flag@e.com", r.prefill.Email)
}

func TestReconcile_CorruptSharedAborts(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	sharedPath := filepath.Join(tmp, "shared.yml")
	testutil.RequireNoError(t, os.WriteFile(sharedPath, []byte("default: : :: ["), 0o600))
	v, _, stderr := newReconcileView()
	_, err := detectAndReconcile(v,
		filepath.Join(tmp, "cfl.yml"), filepath.Join(tmp, "jtk.json"),
		sharedPath, "", "", "", "")
	testutil.RequireError(t, err)
	if !strings.Contains(stderr.String(), "unreadable") {
		t.Errorf("expected unreadable warning; got: %s", stderr.String())
	}
}

func TestReconcile_CorruptCFLLegacyAborts(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	cflPath := filepath.Join(tmp, "cfl.yml")
	corrupt := []byte(": ::: [")
	testutil.RequireNoError(t, os.WriteFile(cflPath, corrupt, 0o600))
	v, _, stderr := newReconcileView()
	_, err := detectAndReconcile(v, cflPath,
		filepath.Join(tmp, "jtk.json"), filepath.Join(tmp, "shared.yml"),
		"", "", "", "")
	testutil.RequireError(t, err)
	if !strings.Contains(stderr.String(), "unreadable") {
		t.Errorf("corrupt own-legacy must surface an actionable 'unreadable' message; got: %s", stderr.String())
	}
	// Fail-loud must mutate NOTHING: the unreadable file is byte-identical
	// afterwards (a future refactor that truncates/overwrites before the
	// early return would otherwise pass undetected).
	after, _ := os.ReadFile(cflPath) //nolint:gosec // test reads its own temp file
	if string(after) != string(corrupt) {
		t.Errorf("corrupt legacy file was mutated by a failed detect; want byte-identical")
	}
}

func TestReconcile_CorruptJTKLegacyDowngradesToWarning(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	cflPath := filepath.Join(tmp, "cfl.yml")
	testutil.RequireNoError(t, os.WriteFile(cflPath,
		[]byte("url: https://acme.atlassian.net\nemail: u@e\napi_token: t\n"), 0o600))
	jtkPath := filepath.Join(tmp, "jtk.json")
	testutil.RequireNoError(t, os.WriteFile(jtkPath, []byte("{not json"), 0o600))
	v, stdout, stderr := newReconcileView()
	r, err := detectAndReconcile(v, cflPath, jtkPath,
		filepath.Join(tmp, "shared.yml"), "", "", "", "")
	testutil.RequireNoError(t, err) // sibling-corrupt is a warning
	testutil.Equal(t, "https://acme.atlassian.net", r.store.Default.URL)
	if !strings.Contains(stdout.String()+stderr.String(), "sibling jtk config") {
		t.Errorf("expected sibling-ignored note; got stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestReconcile_BothLegaciesAligned_FoldsAndPreservesDefaults(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	cflPath := filepath.Join(tmp, "cfl.yml")
	jtkPath := filepath.Join(tmp, "jtk.json")
	testutil.RequireNoError(t, os.WriteFile(cflPath,
		[]byte("url: https://acme.atlassian.net\nemail: u@e\napi_token: t\ndefault_space: SP\noutput_format: json\n"), 0o600))
	testutil.RequireNoError(t, os.WriteFile(jtkPath,
		[]byte(`{"url":"https://acme.atlassian.net","email":"u@e","api_token":"t","default_project":"PR"}`), 0o600))
	v, _, _ := newReconcileView()
	r, err := detectAndReconcile(v, cflPath, jtkPath,
		filepath.Join(tmp, "shared.yml"), "", "", "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "https://acme.atlassian.net", r.store.Default.URL)
	testutil.Equal(t, "SP", r.store.CFL.DefaultSpace)
	testutil.Equal(t, "json", r.store.CFL.OutputFormat)
	testutil.Equal(t, "PR", r.store.JTK.DefaultProject)
}

func TestReconcile_DivergentLegacies_FailLoudNoValueLeak(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	cflPath := filepath.Join(tmp, "cfl.yml")
	jtkPath := filepath.Join(tmp, "jtk.json")
	testutil.RequireNoError(t, os.WriteFile(cflPath,
		[]byte("url: https://cfl-host.atlassian.net\nemail: u@e\napi_token: t\n"), 0o600))
	testutil.RequireNoError(t, os.WriteFile(jtkPath,
		[]byte(`{"url":"https://jtk-host.atlassian.net","email":"u@e","api_token":"t"}`), 0o600))
	v, _, _ := newReconcileView()
	_, err := detectAndReconcile(v, cflPath, jtkPath,
		filepath.Join(tmp, "shared.yml"), "", "", "", "")
	testutil.RequireError(t, err)
	msg := err.Error()
	if strings.Contains(msg, "cfl-host.atlassian.net") || strings.Contains(msg, "jtk-host.atlassian.net") {
		t.Fatalf("fail-loud must not leak values: %s", msg)
	}
	if !strings.Contains(msg, "url:") || !strings.Contains(msg, cflPath) || !strings.Contains(msg, jtkPath) {
		t.Fatalf("fail-loud must name the field + every source path: %s", msg)
	}
	// email is identical across both sources → it must NOT be reported
	// as a conflict (agreed fields stay folded; only `url` diverges).
	if strings.Contains(msg, "email:") {
		t.Fatalf("agreed field must not spuriously conflict: %s", msg)
	}
}

func TestReconcile_SharedPerToolConnDivergence_FailLoud(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	sharedPath := filepath.Join(tmp, "shared.yml")
	testutil.RequireNoError(t, os.WriteFile(sharedPath,
		[]byte("default:\n  url: https://default.atlassian.net\n  email: u@e\ncfl:\n  url: https://cfl-only.atlassian.net\n"), 0o600))
	v, _, _ := newReconcileView()
	_, err := detectAndReconcile(v,
		filepath.Join(tmp, "cfl.yml"), filepath.Join(tmp, "jtk.json"),
		sharedPath, "", "", "", "")
	testutil.RequireError(t, err)
	if !strings.Contains(err.Error(), "cfl.url") {
		t.Fatalf("must name the shared per-tool section.field: %s", err.Error())
	}
}

// Pins the prior Codex blocker: detectAndReconcile (which init runs
// BEFORE keyring.EnsureMigrated) must fail loud AND mutate nothing when
// a pre-MON-5328 shared config has a divergent per-tool connection plus
// a plaintext api_token — the file is byte-identical afterwards and the
// token is never migrated/scrubbed (init returns on this error before
// EnsureMigrated ever runs).
func TestReconcile_DivergentWithToken_NoMutation(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	sharedPath := filepath.Join(tmp, "shared.yml")
	pre := "default:\n  url: https://default.atlassian.net\n  email: u@e\n  api_token: PLAINTEXT_TOK\ncfl:\n  url: https://cfl-only.atlassian.net\n"
	testutil.RequireNoError(t, os.WriteFile(sharedPath, []byte(pre), 0o600))
	before, _ := os.ReadFile(sharedPath) //nolint:gosec // test reads its own temp file

	v, _, _ := newReconcileView()
	_, err := detectAndReconcile(v,
		filepath.Join(tmp, "cfl.yml"), filepath.Join(tmp, "jtk.json"),
		sharedPath, "", "", "", "")
	testutil.RequireError(t, err)
	if !strings.Contains(err.Error(), "cfl.url") {
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
// affecting the sibling: the resolved connection is byte-equivalent to
// the shared default already on disk (implicit-vs-explicit basic and URL
// normalization are canonicalized away). Pins the §2.2/MON-5328 fix for
// the daemon-flagged UX regression (one shared default would otherwise
// prompt on every re-init).
func TestReconcile_NoNagWhenConnUnchanged(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	sharedPath := filepath.Join(tmp, "shared.yml")
	testutil.RequireNoError(t, os.WriteFile(sharedPath,
		[]byte("default:\n  url: https://acme.atlassian.net\n  email: u@e\n"), 0o600))
	v, _, _ := newReconcileView()
	r, err := detectAndReconcile(v,
		filepath.Join(tmp, "cfl.yml"), filepath.Join(tmp, "jtk.json"),
		sharedPath, "", "", "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, false, r.affectsSibling)
}

// When a usable shared default exists AND the resolved connection
// actually DIFFERS from it (here a legacy file contributes a cloud_id
// the default lacked), the save changes what jtk reads, so the sibling
// confirmation MUST still fire.
func TestReconcile_NagsWhenResolvedConnDiffers(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	sharedPath := filepath.Join(tmp, "shared.yml")
	testutil.RequireNoError(t, os.WriteFile(sharedPath,
		[]byte("default:\n  url: https://acme.atlassian.net\n  email: u@e\n"), 0o600))
	cflPath := filepath.Join(tmp, "cfl.yml")
	testutil.RequireNoError(t, os.WriteFile(cflPath,
		[]byte("url: https://acme.atlassian.net\nemail: u@e\ncloud_id: CID\n"), 0o600))
	v, _, _ := newReconcileView()
	r, err := detectAndReconcile(v, cflPath,
		filepath.Join(tmp, "jtk.json"), sharedPath, "", "", "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, true, r.affectsSibling)
}

// A per-tool default the user already set in the SHARED store must win
// over a stale value in a still-present legacy file: legacy only
// backfills fields the shared store leaves empty. Pins the
// daemon-flagged silent-revert regression in preserveDefaultsAndCollect.
func TestReconcile_SharedPerToolDefaultWinsOverLegacy(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	sharedPath := filepath.Join(tmp, "shared.yml")
	testutil.RequireNoError(t, os.WriteFile(sharedPath,
		[]byte("default:\n  url: https://acme.atlassian.net\n  email: u@e\ncfl:\n  default_space: NEW\n  output_format: json\n"), 0o600))
	cflPath := filepath.Join(tmp, "cfl.yml")
	testutil.RequireNoError(t, os.WriteFile(cflPath,
		[]byte("url: https://acme.atlassian.net\nemail: u@e\ndefault_space: OLD\noutput_format: table\n"), 0o600))
	v, _, _ := newReconcileView()
	r, err := detectAndReconcile(v, cflPath,
		filepath.Join(tmp, "jtk.json"), sharedPath, "", "", "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "NEW", r.store.CFL.DefaultSpace)  // shared store wins
	testutil.Equal(t, "json", r.store.CFL.OutputFormat) // not reverted to legacy
}

func TestApplyResultToStore_WritesDefaultAndCFLDefault(t *testing.T) {
	t.Parallel()
	store := &credstore.Store{JTK: credstore.ToolSection{DefaultProject: "KEEP"}}
	applyResultToStore(store, &config.Config{
		URL: "https://acme.atlassian.net/wiki", Email: "u@e",
		AuthMethod: "basic", DefaultSpace: "SP", OutputFormat: "json",
	})
	testutil.Equal(t, "https://acme.atlassian.net", store.Default.URL) // /wiki stripped
	testutil.Equal(t, "u@e", store.Default.Email)
	testutil.Equal(t, "SP", store.CFL.DefaultSpace)
	testutil.Equal(t, "json", store.CFL.OutputFormat)
	testutil.Equal(t, "KEEP", store.JTK.DefaultProject) // sibling untouched
}
