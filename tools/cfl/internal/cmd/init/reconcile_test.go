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

type configFixture struct {
	URL          string
	Email        string
	APIToken     string
	AuthMethod   string
	CloudID      string
	DefaultSpace string
}

func (c configFixture) toConfig() *config.Config {
	return &config.Config{
		URL:          c.URL,
		Email:        c.Email,
		APIToken:     c.APIToken,
		AuthMethod:   c.AuthMethod,
		CloudID:      c.CloudID,
		DefaultSpace: c.DefaultSpace,
	}
}

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
		filepath.Join(tmp, "cfl.yml"),
		filepath.Join(tmp, "jtk.json"),
		filepath.Join(tmp, "shared.yml"),
		"", "", "", "")
	testutil.RequireNoError(t, err)
	testutil.NotNil(t, r)
	testutil.Equal(t, writeDefault, r.target)
	testutil.Equal(t, "", r.prefill.URL)
	testutil.Equal(t, 0, len(r.consumedLegacies))
}

func TestReconcile_OnlyCFLLegacy_AutoMigrates(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	cflPath := filepath.Join(tmp, "cfl.yml")
	body := `url: https://acme.atlassian.net/wiki
email: u@e.com
api_token: tok
default_space: SPACE
`
	testutil.RequireNoError(t, os.WriteFile(cflPath, []byte(body), 0o600))

	v, stdout, _ := newReconcileView()
	r, err := detectAndReconcile(v, cflPath,
		filepath.Join(tmp, "jtk.json"),
		filepath.Join(tmp, "shared.yml"),
		"", "", "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, writeDefault, r.target)
	testutil.Equal(t, "https://acme.atlassian.net/wiki", r.prefill.URL)
	testutil.Equal(t, "tok", r.prefill.APIToken)
	testutil.Equal(t, "SPACE", r.prefill.DefaultSpace)
	testutil.Equal(t, []string{cflPath}, r.consumedLegacies)
	if !strings.Contains(stdout.String(), "Migrating existing cfl config") {
		t.Errorf("expected migration message; got: %s", stdout.String())
	}
}

func TestReconcile_FlagOverridesPrefill(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	v, _, _ := newReconcileView()

	r, err := detectAndReconcile(v,
		filepath.Join(tmp, "cfl.yml"),
		filepath.Join(tmp, "jtk.json"),
		filepath.Join(tmp, "shared.yml"),
		"https://flag.atlassian.net", "flag@example.com", "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "https://flag.atlassian.net", r.prefill.URL)
	testutil.Equal(t, "flag@example.com", r.prefill.Email)
}

func TestReconcile_CorruptSharedAborts(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	sharedPath := filepath.Join(tmp, "shared.yml")
	testutil.RequireNoError(t, os.MkdirAll(filepath.Dir(sharedPath), 0o700))
	testutil.RequireNoError(t, os.WriteFile(sharedPath, []byte("default: : :: ["), 0o600))

	v, _, stderr := newReconcileView()
	_, err := detectAndReconcile(v,
		filepath.Join(tmp, "cfl.yml"),
		filepath.Join(tmp, "jtk.json"),
		sharedPath,
		"", "", "", "")
	testutil.RequireError(t, err)

	// The shared file was not modified by detection.
	body, ferr := os.ReadFile(sharedPath)
	testutil.RequireNoError(t, ferr)
	testutil.Equal(t, "default: : :: [", string(body))

	if !strings.Contains(stderr.String(), "unreadable") {
		t.Errorf("expected unreadable warning on stderr; got: %s", stderr.String())
	}
}

func TestReconcile_CorruptCFLLegacyAborts(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	cflPath := filepath.Join(tmp, "cfl.yml")
	testutil.RequireNoError(t, os.WriteFile(cflPath, []byte("url: : :: ["), 0o600))

	v, _, stderr := newReconcileView()
	_, err := detectAndReconcile(v, cflPath,
		filepath.Join(tmp, "jtk.json"),
		filepath.Join(tmp, "shared.yml"),
		"", "", "", "")
	testutil.RequireError(t, err)

	body, ferr := os.ReadFile(cflPath)
	testutil.RequireNoError(t, ferr)
	testutil.Equal(t, "url: : :: [", string(body))

	if !strings.Contains(stderr.String(), "unreadable") {
		t.Errorf("expected unreadable warning; got: %s", stderr.String())
	}
}

func TestReconcile_CorruptJTKLegacyDowngradesToWarning(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	jtkPath := filepath.Join(tmp, "jtk.json")
	testutil.RequireNoError(t, os.WriteFile(jtkPath, []byte("{not json"), 0o600))

	v, stdout, _ := newReconcileView()
	r, err := detectAndReconcile(v,
		filepath.Join(tmp, "cfl.yml"),
		jtkPath,
		filepath.Join(tmp, "shared.yml"),
		"", "", "", "")
	testutil.RequireNoError(t, err)
	testutil.NotNil(t, r)
	if !strings.Contains(stdout.String(), "ignoring") {
		t.Errorf("expected ignore note for sibling-corrupt; got: %s", stdout.String())
	}
}

func TestApplyResultToStore_DefaultTarget(t *testing.T) {
	t.Parallel()
	store := &credstore.Store{
		// pre-existing JTK section that we must not stomp
		JTK: credstore.ToolSection{
			Section:        credstore.Section{APIToken: "preserved-jtk"},
			DefaultProject: "PROJ",
		},
	}
	cfg := configFixture{
		URL: "https://acme.atlassian.net/wiki", Email: "u@e", APIToken: "t",
		DefaultSpace: "SPACE",
	}.toConfig()

	applyResultToStore(store, cfg, writeDefault)

	testutil.Equal(t, "https://acme.atlassian.net", store.Default.URL) // base form
	testutil.Equal(t, "t", store.Default.APIToken)
	testutil.Equal(t, "SPACE", store.CFL.DefaultSpace)
	// JTK section preserved.
	testutil.Equal(t, "preserved-jtk", store.JTK.APIToken)
	testutil.Equal(t, "PROJ", store.JTK.DefaultProject)
}

func TestApplyResultToStore_OverrideTarget(t *testing.T) {
	t.Parallel()
	store := &credstore.Store{
		Default: credstore.Section{URL: "https://default.atlassian.net", APIToken: "default-tok"},
	}
	cfg := configFixture{
		URL: "https://cfl.atlassian.net/wiki", Email: "u@e", APIToken: "cfl-tok",
	}.toConfig()

	applyResultToStore(store, cfg, writeCFLOverride)

	// Override section was written.
	testutil.Equal(t, "https://cfl.atlassian.net", store.CFL.URL)
	testutil.Equal(t, "cfl-tok", store.CFL.APIToken)
	// Default left alone.
	testutil.Equal(t, "https://default.atlassian.net", store.Default.URL)
	testutil.Equal(t, "default-tok", store.Default.APIToken)
}
