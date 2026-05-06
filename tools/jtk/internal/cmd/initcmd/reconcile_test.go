package initcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
)

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
		filepath.Join(tmp, "jtk.json"),
		filepath.Join(tmp, "cfl.yml"),
		filepath.Join(tmp, "shared.yml"),
		"", "", "", "", "")
	testutil.RequireNoError(t, err)
	testutil.NotNil(t, r)
	testutil.Equal(t, writeDefault, r.target)
	testutil.Equal(t, "", r.prefill.URL)
}

func TestReconcile_OnlyJTKLegacy_AutoMigrates(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	jtkPath := filepath.Join(tmp, "jtk.json")
	body := `{"url":"https://acme.atlassian.net","email":"u@e","api_token":"jtk-tok","default_project":"PROJ"}`
	testutil.RequireNoError(t, os.WriteFile(jtkPath, []byte(body), 0o600))

	v, stdout, _ := newReconcileView()
	r, err := detectAndReconcile(v, jtkPath,
		filepath.Join(tmp, "cfl.yml"),
		filepath.Join(tmp, "shared.yml"),
		"", "", "", "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, writeDefault, r.target)
	testutil.Equal(t, "https://acme.atlassian.net", r.prefill.URL)
	testutil.Equal(t, "jtk-tok", r.prefill.APIToken)
	testutil.Equal(t, "PROJ", r.prefill.DefaultProject)
	testutil.Equal(t, []string{jtkPath}, r.consumedLegacies)
	if !strings.Contains(stdout.String(), "Migrating existing jtk config") {
		t.Errorf("expected migration message; got: %s", stdout.String())
	}
}

func TestReconcile_FlagOverridesPrefill(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	v, _, _ := newReconcileView()

	r, err := detectAndReconcile(v,
		filepath.Join(tmp, "jtk.json"),
		filepath.Join(tmp, "cfl.yml"),
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
	testutil.RequireNoError(t, os.MkdirAll(filepath.Dir(sharedPath), 0o700))
	testutil.RequireNoError(t, os.WriteFile(sharedPath, []byte("default: : :: ["), 0o600))

	v, _, stderr := newReconcileView()
	_, err := detectAndReconcile(v,
		filepath.Join(tmp, "jtk.json"),
		filepath.Join(tmp, "cfl.yml"),
		sharedPath,
		"", "", "", "", "")
	testutil.RequireError(t, err)
	if !strings.Contains(stderr.String(), "unreadable") {
		t.Errorf("expected unreadable warning; got: %s", stderr.String())
	}
}

func TestApplyResultToStore_DefaultTarget_PreservesCFLSection(t *testing.T) {
	t.Parallel()
	store := &credstore.Store{
		CFL: credstore.ToolSection{
			Section:      credstore.Section{APIToken: "preserved-cfl"},
			DefaultSpace: "SPACE",
		},
	}
	cfg := &config.Config{
		URL: "https://acme.atlassian.net", Email: "u@e", APIToken: "jtk-tok",
		DefaultProject: "PROJ",
	}

	applyResultToStore(store, cfg, writeDefault)

	testutil.Equal(t, "https://acme.atlassian.net", store.Default.URL)
	testutil.Equal(t, "jtk-tok", store.Default.APIToken)
	testutil.Equal(t, "PROJ", store.JTK.DefaultProject)
	// CFL section preserved.
	testutil.Equal(t, "preserved-cfl", store.CFL.APIToken)
	testutil.Equal(t, "SPACE", store.CFL.DefaultSpace)
}

func TestApplyResultToStore_OverrideTarget(t *testing.T) {
	t.Parallel()
	store := &credstore.Store{
		Default: credstore.Section{URL: "https://default.atlassian.net", APIToken: "default-tok"},
	}
	cfg := &config.Config{URL: "https://jtk.atlassian.net", Email: "u@e", APIToken: "jtk-tok"}

	applyResultToStore(store, cfg, writeJTKOverride)

	testutil.Equal(t, "https://jtk.atlassian.net", store.JTK.URL)
	testutil.Equal(t, "jtk-tok", store.JTK.APIToken)
	// Default left alone.
	testutil.Equal(t, "https://default.atlassian.net", store.Default.URL)
	testutil.Equal(t, "default-tok", store.Default.APIToken)
}
