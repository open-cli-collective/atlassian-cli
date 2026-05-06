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

func TestResultFromSiblingLegacy_ReuseYes(t *testing.T) {
	t.Parallel()
	cfl := &credstore.LegacyCreds{Path: "/cfl.yml", URL: "https://acme.atlassian.net/wiki", Email: "u@e", APIToken: "cfl-tok"}
	r := resultFromSiblingLegacy(cfl, &credstore.Store{}, true, "", "", "", "", "")
	testutil.Equal(t, "cfl-tok", r.prefill.APIToken)
	testutil.Equal(t, []string{"/cfl.yml"}, r.consumedLegacies)
}

func TestResultFromSiblingLegacy_ReuseNo(t *testing.T) {
	t.Parallel()
	cfl := &credstore.LegacyCreds{Path: "/cfl.yml", APIToken: "cfl-tok"}
	r := resultFromSiblingLegacy(cfl, &credstore.Store{}, false, "", "", "", "", "")
	testutil.Equal(t, "", r.prefill.APIToken)
	testutil.Equal(t, 0, len(r.consumedLegacies))
}

func TestResultFromMismatch_UseJTK(t *testing.T) {
	t.Parallel()
	jtk := &credstore.LegacyCreds{Path: "/jtk.json", URL: "https://jtk.atlassian.net", APIToken: "jtk-tok", DefaultProject: "PROJ"}
	cfl := &credstore.LegacyCreds{Path: "/cfl.yml", URL: "https://cfl.atlassian.net/wiki", APIToken: "cfl-tok"}
	v, _, _ := newReconcileView()
	r := resultFromMismatch(jtk, cfl, "use_jtk", &credstore.Store{}, v, "", "", "", "", "")
	testutil.Equal(t, "jtk-tok", r.prefill.APIToken)
	testutil.Equal(t, "PROJ", r.prefill.DefaultProject)
}

func TestResultFromMismatch_UseCFL_PreservesJTKDefaults(t *testing.T) {
	t.Parallel()
	jtk := &credstore.LegacyCreds{Path: "/jtk.json", URL: "https://jtk.atlassian.net", APIToken: "jtk-tok", DefaultProject: "PROJ"}
	cfl := &credstore.LegacyCreds{Path: "/cfl.yml", URL: "https://cfl.atlassian.net/wiki", APIToken: "cfl-tok"}
	v, _, _ := newReconcileView()
	r := resultFromMismatch(jtk, cfl, "use_cfl", &credstore.Store{}, v, "", "", "", "", "")
	testutil.Equal(t, "cfl-tok", r.prefill.APIToken) // cfl creds chosen
	testutil.Equal(t, "PROJ", r.prefill.DefaultProject) // jtk's default_project preserved
}

func TestResultFromMismatch_KeepDifferent(t *testing.T) {
	t.Parallel()
	jtk := &credstore.LegacyCreds{Path: "/jtk.json", URL: "https://jtk.atlassian.net", Email: "u@e", APIToken: "jtk-tok"}
	cfl := &credstore.LegacyCreds{Path: "/cfl.yml", URL: "https://cfl.atlassian.net/wiki", Email: "u@e", APIToken: "cfl-tok", DefaultSpace: "SPACE"}
	v, stdout, _ := newReconcileView()
	r := resultFromMismatch(jtk, cfl, "keep_different", &credstore.Store{}, v, "", "", "", "", "")
	testutil.Equal(t, "https://jtk.atlassian.net", r.store.Default.URL)
	testutil.Equal(t, "jtk-tok", r.store.Default.APIToken)
	testutil.Equal(t, "https://cfl.atlassian.net", r.store.CFL.URL)
	testutil.Equal(t, "cfl-tok", r.store.CFL.APIToken)
	if !strings.Contains(stdout.String(), "Keeping per-tool credentials") {
		t.Errorf("expected keep-different note; got: %s", stdout.String())
	}
}

func TestResultFromSharedNoOverride_ReuseYes(t *testing.T) {
	t.Parallel()
	store := &credstore.Store{
		Default: credstore.Section{
			URL:      "https://acme.atlassian.net",
			Email:    "u@e",
			APIToken: "shared-tok",
		},
		JTK: credstore.ToolSection{DefaultProject: "EXISTING"},
	}
	r := resultFromSharedNoOverride(store, true, "", "", "", "", "")
	testutil.Equal(t, writeDefault, r.target)
	testutil.Equal(t, "shared-tok", r.prefill.APIToken)
	testutil.Equal(t, "EXISTING", r.prefill.DefaultProject)
	testutil.Equal(t, true, r.affectsSibling)
}

func TestResultFromSharedNoOverride_ReuseNo_FreshForm(t *testing.T) {
	t.Parallel()
	store := &credstore.Store{
		Default: credstore.Section{URL: "https://x", APIToken: "shared-tok"},
		JTK:     credstore.ToolSection{DefaultProject: "EXISTING"},
	}
	r := resultFromSharedNoOverride(store, false, "", "", "", "", "")
	testutil.Equal(t, writeJTKOverride, r.target)
	testutil.Equal(t, "", r.prefill.APIToken)
	testutil.Equal(t, "EXISTING", r.prefill.DefaultProject)
	testutil.Equal(t, false, r.affectsSibling)
}

func TestResultFromSharedWithOverride(t *testing.T) {
	t.Parallel()
	store := &credstore.Store{
		Default: credstore.Section{URL: "https://default", APIToken: "default-tok"},
		JTK: credstore.ToolSection{
			Section:        credstore.Section{URL: "https://jtk-override", APIToken: "jtk-tok"},
			DefaultProject: "PROJ",
		},
	}
	r := resultFromSharedWithOverride(store, "", "", "", "", "")
	testutil.Equal(t, writeJTKOverride, r.target)
	testutil.Equal(t, "jtk-tok", r.prefill.APIToken)
	testutil.Equal(t, "PROJ", r.prefill.DefaultProject)
	testutil.Equal(t, false, r.affectsSibling)
}

func TestResultFromSiblingLegacy_PreservesCFLDefaults(t *testing.T) {
	t.Parallel()
	cfl := &credstore.LegacyCreds{
		Path:         "/cfl.yml",
		URL:          "https://x",
		Email:        "u@e",
		APIToken:     "cfl-tok",
		DefaultSpace: "SPACE",
		OutputFormat: "json",
	}
	store := &credstore.Store{}
	r := resultFromSiblingLegacy(cfl, store, true, "", "", "", "", "")
	testutil.Equal(t, "SPACE", r.store.CFL.DefaultSpace)
	testutil.Equal(t, "json", r.store.CFL.OutputFormat)
}

func TestMismatchDescription_EducationalText(t *testing.T) {
	t.Parallel()
	jtk := &credstore.LegacyCreds{Path: "/jtk.json", URL: "https://x", Email: "u@e", APIToken: "tok"}
	cfl := &credstore.LegacyCreds{Path: "/cfl.yml", URL: "https://x", Email: "u@e", APIToken: "different"}
	desc := mismatchDescription(jtk, cfl)
	for _, want := range []string{
		"account-wide",
		"both Jira and Confluence",
		"id.atlassian.com/manage-profile",
		"/jtk.json",
		"/cfl.yml",
	} {
		if !strings.Contains(desc, want) {
			t.Errorf("expected %q in mismatch description; got:\n%s", want, desc)
		}
	}
}
