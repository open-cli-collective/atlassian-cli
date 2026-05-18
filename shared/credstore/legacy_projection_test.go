package credstore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

// The migration projection must still see pre-MON-5328 per-tool
// connection + token fields the canonical Store no longer decodes —
// "never destroy evidence before the migration reads it".
func TestLoadSharedLegacyProjection(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	yml := `default:
  url: https://acme.atlassian.net
  email: d@e
  api_token: DEF_TOK
cfl:
  url: https://cfl.atlassian.net
  api_token: CFL_TOK
  default_space: SP
jtk:
  email: j@e
  api_token: JTK_TOK
  default_project: PR
`
	testutil.RequireNoError(t, os.WriteFile(path, []byte(yml), 0o600))

	p, err := LoadSharedLegacyProjection(path)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "DEF_TOK", p.Default.APIToken)
	testutil.Equal(t, "https://acme.atlassian.net", p.Default.URL)
	testutil.Equal(t, "https://cfl.atlassian.net", p.CFL.URL) // canonical Store drops this
	testutil.Equal(t, "CFL_TOK", p.CFL.APIToken)
	testutil.Equal(t, "j@e", p.JTK.Email)
	testutil.Equal(t, "JTK_TOK", p.JTK.APIToken)

	// Canonical Load must NOT expose per-tool connection/token.
	s, err := Load(path)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "DEF_TOK", s.Default.APIToken) // default still readable (migration source)
	testutil.Equal(t, "SP", s.CFL.DefaultSpace)      // per-tool non-secret default kept
	testutil.Equal(t, "PR", s.JTK.DefaultProject)
}

func TestLoadSharedLegacyProjection_AbsentAndCorrupt(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	p, err := LoadSharedLegacyProjection(filepath.Join(dir, "nope.yml"))
	testutil.RequireNoError(t, err)
	if p != nil {
		t.Fatalf("absent file must return nil projection, got %+v", p)
	}

	bad := filepath.Join(dir, "bad.yml")
	testutil.RequireNoError(t, os.WriteFile(bad, []byte("default: : :: ["), 0o600))
	bp, err := LoadSharedLegacyProjection(bad)
	if !errors.Is(err, ErrCorruptStore) {
		t.Fatalf("corrupt file must return ErrCorruptStore, got %v", err)
	}
	if bp != nil {
		t.Fatalf("corrupt file must return a nil projection (no half-decoded struct), got %+v", bp)
	}
}
