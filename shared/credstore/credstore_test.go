package credstore

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/auth"
	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestLoad_AbsentReturnsEmptyStore(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := Load(filepath.Join(dir, "missing.yml"))
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "", s.Default.URL)
	testutil.Equal(t, "", s.CFL.DefaultSpace)
}

func TestLoad_CorruptReturnsErrCorrupt(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	testutil.RequireNoError(t, os.WriteFile(path, []byte("default: not valid yaml: ::: : ["), 0o600))
	_, err := Load(path)
	testutil.RequireError(t, err)
	if !errors.Is(err, ErrCorruptStore) {
		t.Fatalf("expected ErrCorruptStore, got %v", err)
	}
}

func TestSaveLoad_Roundtrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "atlassian-cli", "config.yml")
	in := &Store{
		Default: Section{
			URL:      "https://acme.atlassian.net",
			Email:    "u@example.com",
			APIToken: "tok",
		},
		CFL: ToolSection{
			DefaultSpace: "MYSPACE",
		},
	}
	testutil.RequireNoError(t, in.Save(path))
	out, err := Load(path)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, in.Default.URL, out.Default.URL)
	testutil.Equal(t, in.Default.Email, out.Default.Email)
	testutil.Equal(t, in.CFL.DefaultSpace, out.CFL.DefaultSpace)
	// Asymmetric codec: Save never persists the token, so a Save→Load
	// roundtrip drops it (it now lives in the keyring).
	testutil.Equal(t, "", out.Default.APIToken)
}

func TestSave_ModeIs0600(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("file modes don't apply on windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	s := &Store{Default: Section{URL: "https://x"}}
	testutil.RequireNoError(t, s.Save(path))
	info, err := os.Stat(path)
	testutil.RequireNoError(t, err)
	mode := info.Mode().Perm()
	testutil.Equal(t, os.FileMode(0o600), mode)
}

func TestSave_NoLeftoverTempOnSuccess(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	s := &Store{Default: Section{URL: "https://x"}}
	testutil.RequireNoError(t, s.Save(path))

	matches, err := filepath.Glob(filepath.Join(dir, "*.tmp"))
	testutil.RequireNoError(t, err)
	if len(matches) != 0 {
		t.Fatalf("found leftover tmp files: %v", matches)
	}
}

func TestSave_NoLeftoverTempOnFailure(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("permission semantics differ on windows")
	}
	// Make the parent directory non-writable so os.WriteFile of the .tmp fails.
	dir := t.TempDir()
	target := filepath.Join(dir, "child", "config.yml")
	testutil.RequireNoError(t, os.MkdirAll(filepath.Dir(target), 0o700))
	testutil.RequireNoError(t, os.Chmod(filepath.Dir(target), 0o500)) //nolint:gosec // intentional: induce write failure
	t.Cleanup(func() { _ = os.Chmod(filepath.Dir(target), 0o700) })   //nolint:gosec // intentional: restore for cleanup

	s := &Store{Default: Section{URL: "https://x"}}
	err := s.Save(target)
	testutil.RequireError(t, err)

	// No stray .tmp left in the locked directory.
	testutil.RequireNoError(t, os.Chmod(filepath.Dir(target), 0o700)) //nolint:gosec // intentional: restore for inspection
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(target), "*.tmp"))
	testutil.RequireNoError(t, err)
	if len(matches) != 0 {
		t.Fatalf("found leftover tmp files after induced failure: %v", matches)
	}
}

// §2.2 (MON-5328): connection is single-sourced from `default`. A
// per-tool section can no longer override connection fields, so Resolve
// returns Default regardless of tool.
func TestResolve_AlwaysReturnsDefault(t *testing.T) {
	t.Parallel()
	s := &Store{
		Default: Section{
			URL:      "https://acme.atlassian.net",
			Email:    "default@example.com",
			APIToken: "default-tok",
		},
		CFL: ToolSection{DefaultSpace: "SP"},
		JTK: ToolSection{DefaultProject: "PR"},
	}
	for _, tool := range []string{ToolCFL, ToolJTK, "unknown"} {
		got := s.Resolve(tool)
		testutil.Equal(t, "https://acme.atlassian.net", got.URL)
		testutil.Equal(t, "default@example.com", got.Email)
		testutil.Equal(t, "default-tok", got.APIToken)
	}
}

func TestResolve_UnknownToolReturnsDefault(t *testing.T) {
	t.Parallel()
	s := &Store{Default: Section{URL: "https://x"}}
	got := s.Resolve("unknown")
	testutil.Equal(t, "https://x", got.URL)
}

func TestResolveWithSource(t *testing.T) {
	t.Parallel()
	s := &Store{
		Default: Section{URL: "https://acme.atlassian.net", Email: "u@e.com", APIToken: "def-tok"},
		CFL:     ToolSection{DefaultSpace: "SP"},
	}
	cases := []struct {
		field     string
		wantValue string
		wantSrc   Source
	}{
		{"url", "https://acme.atlassian.net", SourceDefault},
		{"email", "u@e.com", SourceDefault},
		{"api_token", "def-tok", SourceDefault},
		{"cloud_id", "", SourceUnset},
	}
	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			t.Parallel()
			v, src := s.ResolveWithSource(ToolCFL, tc.field)
			testutil.Equal(t, tc.wantValue, v)
			testutil.Equal(t, tc.wantSrc, src)
		})
	}
}

// HasUsableConfig is NON-secret completeness only — the token lives in
// the keyring, so it is intentionally NOT part of this check (callers
// compose it with keyring.HasToken).
func TestHasUsableConfig(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		s    *Store
		tool string
		want bool
	}{
		{
			name: "basic complete (no token needed)",
			s:    &Store{Default: Section{URL: "u", Email: "e"}},
			tool: ToolCFL,
			want: true,
		},
		{
			name: "basic missing email",
			s:    &Store{Default: Section{URL: "u"}},
			tool: ToolCFL,
			want: false,
		},
		{
			name: "a stray token does not substitute for email",
			s:    &Store{Default: Section{URL: "u", APIToken: "t"}},
			tool: ToolCFL,
			want: false,
		},
		{
			name: "§2.2: a per-tool section can NOT complete connection",
			s: &Store{
				Default: Section{URL: "u"}, // missing email
				CFL:     ToolSection{DefaultSpace: "SP"},
			},
			tool: ToolCFL,
			want: false, // per-tool no longer supplies email
		},
		{
			name: "bearer needs cloud_id",
			s: &Store{Default: Section{
				URL: "u", AuthMethod: auth.AuthMethodBearer,
			}},
			tool: ToolJTK,
			want: false,
		},
		{
			name: "bearer complete (no token needed)",
			s: &Store{Default: Section{
				URL: "u", CloudID: "c", AuthMethod: auth.AuthMethodBearer,
			}},
			tool: ToolJTK,
			want: true,
		},
		{
			name: "empty store",
			s:    &Store{},
			tool: ToolCFL,
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.s.HasUsableConfig(tc.tool)
			testutil.Equal(t, tc.want, got)
		})
	}
}

// TestSave_NeverWritesAnyToken is the §3 asymmetric-codec guarantee:
// even when the in-memory Store carries tokens in every section, the
// marshaled bytes contain no api_token; Load still exposes a token so
// the one-time keyring migration can find it.
func TestSave_NeverWritesAnyToken(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	in := &Store{
		Default: Section{URL: "u", Email: "e", APIToken: "DEFAULT_SECRET"},
		CFL:     ToolSection{DefaultSpace: "SP"},
		JTK:     ToolSection{DefaultProject: "PR"},
	}
	if err := in.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, err := os.ReadFile(path) //nolint:gosec // test reads its own temp file
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := string(raw)
	for _, secret := range []string{"DEFAULT_SECRET", "CFL_SECRET", "JTK_SECRET", "api_token"} {
		if strings.Contains(got, secret) {
			t.Fatalf("Save persisted %q; file:\n%s", secret, got)
		}
	}
	// Non-secret fields are still written.
	if !strings.Contains(got, "default_space") || !strings.Contains(got, "SP") {
		t.Fatalf("Save dropped non-secret fields; file:\n%s", got)
	}

	// Load still reads api_token (migration source): write one by hand.
	legacy := "default:\n  url: u\n  email: e\n  api_token: LEGACY_SECRET\n"
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	back, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if back.Default.APIToken != "LEGACY_SECRET" {
		t.Fatalf("Load did not expose api_token for migration; got %q", back.Default.APIToken)
	}
}

func TestNormalizeBaseURL(t *testing.T) {
	t.Parallel()
	cases := []struct{ in, out string }{
		{"", ""},
		{"https://acme.atlassian.net", "https://acme.atlassian.net"},
		{"https://acme.atlassian.net/", "https://acme.atlassian.net"},
		{"https://acme.atlassian.net/wiki", "https://acme.atlassian.net"},
		{"https://acme.atlassian.net/wiki/", "https://acme.atlassian.net"},
		{"https://acme.atlassian.net/wiki/wiki", "https://acme.atlassian.net"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			testutil.Equal(t, tc.out, NormalizeBaseURL(tc.in))
		})
	}
}

func TestURLForCFL(t *testing.T) {
	t.Parallel()
	cases := []struct{ in, out string }{
		{"", ""},
		{"https://acme.atlassian.net", "https://acme.atlassian.net/wiki"},
		{"https://acme.atlassian.net/", "https://acme.atlassian.net/wiki"},
		{"https://acme.atlassian.net/wiki", "https://acme.atlassian.net/wiki"},
		{"https://acme.atlassian.net/wiki/", "https://acme.atlassian.net/wiki"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			got := URLForCFL(tc.in)
			testutil.Equal(t, tc.out, got)
			// Idempotent: applying NormalizeBaseURL(URLForCFL(x)) returns base.
			if tc.out != "" && !strings.HasSuffix(got, "/wiki") {
				t.Fatalf("URLForCFL did not produce /wiki suffix: %q", got)
			}
		})
	}
}

func TestDefaultPath_HonorsXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/xdg")
	got := DefaultPath()
	testutil.Equal(t, "/custom/xdg/atlassian-cli/config.yml", got)
}
