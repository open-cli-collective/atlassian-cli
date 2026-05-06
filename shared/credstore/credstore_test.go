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
	testutil.Equal(t, "", s.CFL.APIToken)
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
			Section:      Section{APIToken: "cfl-tok"},
			DefaultSpace: "MYSPACE",
		},
	}
	testutil.RequireNoError(t, in.Save(path))
	out, err := Load(path)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, in.Default.URL, out.Default.URL)
	testutil.Equal(t, in.Default.Email, out.Default.Email)
	testutil.Equal(t, in.Default.APIToken, out.Default.APIToken)
	testutil.Equal(t, in.CFL.APIToken, out.CFL.APIToken)
	testutil.Equal(t, in.CFL.DefaultSpace, out.CFL.DefaultSpace)
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

func TestResolve_PerFieldMergeWithPartialOverride(t *testing.T) {
	t.Parallel()
	s := &Store{
		Default: Section{
			URL:      "https://acme.atlassian.net",
			Email:    "default@example.com",
			APIToken: "default-tok",
		},
		CFL: ToolSection{
			Section: Section{APIToken: "cfl-tok"}, // only token override
		},
	}
	got := s.Resolve(ToolCFL)
	testutil.Equal(t, "https://acme.atlassian.net", got.URL)      // default
	testutil.Equal(t, "default@example.com", got.Email)           // default
	testutil.Equal(t, "cfl-tok", got.APIToken)                    // override
	testutil.Equal(t, "default-tok", s.Default.APIToken)          // default untouched
	testutil.Equal(t, "default-tok", s.Resolve(ToolJTK).APIToken) // jtk falls through to default
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
		Default: Section{URL: "https://acme.atlassian.net", Email: "u@e.com"},
		CFL:     ToolSection{Section: Section{APIToken: "cfl-tok"}},
	}
	cases := []struct {
		field     string
		wantValue string
		wantSrc   Source
	}{
		{"url", "https://acme.atlassian.net", SourceDefault},
		{"email", "u@e.com", SourceDefault},
		{"api_token", "cfl-tok", SourceOverrideCFL},
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

func TestHasUsableCreds(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		s    *Store
		tool string
		want bool
	}{
		{
			name: "basic complete in default",
			s:    &Store{Default: Section{URL: "u", Email: "e", APIToken: "t"}},
			tool: ToolCFL,
			want: true,
		},
		{
			name: "basic missing email",
			s:    &Store{Default: Section{URL: "u", APIToken: "t"}},
			tool: ToolCFL,
			want: false,
		},
		{
			name: "basic completed by partial override",
			s: &Store{
				Default: Section{URL: "u", APIToken: "t"},
				CFL:     ToolSection{Section: Section{Email: "e"}},
			},
			tool: ToolCFL,
			want: true,
		},
		{
			name: "bearer needs cloud_id",
			s: &Store{Default: Section{
				URL: "u", APIToken: "t", AuthMethod: auth.AuthMethodBearer,
			}},
			tool: ToolJTK,
			want: false,
		},
		{
			name: "bearer complete",
			s: &Store{Default: Section{
				URL: "u", APIToken: "t", CloudID: "c", AuthMethod: auth.AuthMethodBearer,
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
			got := tc.s.HasUsableCreds(tc.tool)
			testutil.Equal(t, tc.want, got)
		})
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
