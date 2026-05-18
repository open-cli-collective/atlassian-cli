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
	body, ferr := os.ReadFile(sharedPath) //nolint:gosec // test-controlled tempdir path
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

	body, ferr := os.ReadFile(cflPath) //nolint:gosec // test-controlled tempdir path
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

func TestReconcile_BothLegaciesMatch_AutoMigrates(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	cflPath := filepath.Join(tmp, "cfl.yml")
	jtkPath := filepath.Join(tmp, "jtk.json")
	testutil.RequireNoError(t, os.WriteFile(cflPath, []byte(
		"url: https://acme.atlassian.net/wiki\nemail: u@e\napi_token: tok\ndefault_space: SPACE\n"), 0o600))
	testutil.RequireNoError(t, os.WriteFile(jtkPath, []byte(
		`{"url":"https://acme.atlassian.net","email":"u@e","api_token":"tok","default_project":"PROJ"}`), 0o600))

	v, stdout, _ := newReconcileView()
	r, err := detectAndReconcile(v, cflPath, jtkPath,
		filepath.Join(tmp, "shared.yml"),
		"", "", "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, writeDefault, r.target)
	testutil.Equal(t, "tok", r.prefill.APIToken)
	// jtk's default_project is preserved on the store even though we
	// chose cfl as the migration source.
	testutil.Equal(t, "PROJ", r.store.JTK.DefaultProject)
	if !strings.Contains(stdout.String(), "Found matching cfl and jtk credentials") {
		t.Errorf("expected matched-legacy message; got: %s", stdout.String())
	}
	testutil.Equal(t, false, r.affectsSibling) // migration, not affecting existing sibling state
}

func TestSectionsEqual_AuthMethodCanonicalization(t *testing.T) {
	t.Parallel()
	a := credstore.Section{URL: "https://x", Email: "u@e", APIToken: "t"}                      // empty method
	b := credstore.Section{URL: "https://x", Email: "u@e", APIToken: "t", AuthMethod: "basic"} // explicit
	if !credstore.SectionsEqual(a, b) {
		t.Fatalf("empty auth_method should match explicit basic")
	}
}

func TestApplyResultToStore_PreservesExistingDefaultSpace(t *testing.T) {
	t.Parallel()
	store := &credstore.Store{
		CFL: credstore.ToolSection{DefaultSpace: "EXISTING"},
	}
	cfg := configFixture{URL: "https://x", Email: "u@e", APIToken: "t"}.toConfig() // no DefaultSpace
	applyResultToStore(store, cfg, writeDefault)
	testutil.Equal(t, "EXISTING", store.CFL.DefaultSpace) // not stomped
}

// Post-prompt branch tests — drive each huh-mediated decision by
// calling the extracted helpers directly so we don't have to fake
// stdin.

func TestResultFromSiblingLegacy_ReuseYes(t *testing.T) {
	t.Parallel()
	jtk := &credstore.LegacyCreds{
		Path:           "/jtk.json",
		URL:            "https://acme.atlassian.net",
		Email:          "u@e",
		APIToken:       "jtk-tok",
		DefaultProject: "PROJ",
	}
	r := resultFromSiblingLegacy(jtk, &credstore.Store{}, true, "", "", "", "")
	testutil.Equal(t, "jtk-tok", r.prefill.APIToken)
	testutil.Equal(t, []string{"/jtk.json"}, r.consumedLegacies)
}

func TestResultFromSiblingLegacy_ReuseNo(t *testing.T) {
	t.Parallel()
	jtk := &credstore.LegacyCreds{Path: "/jtk.json", APIToken: "jtk-tok"}
	r := resultFromSiblingLegacy(jtk, &credstore.Store{}, false, "", "", "", "")
	testutil.Equal(t, "", r.prefill.APIToken) // fresh setup, no prefill
	testutil.Equal(t, 0, len(r.consumedLegacies))
}

func TestResultFromMismatch_UseCFL(t *testing.T) {
	t.Parallel()
	cfl := &credstore.LegacyCreds{Path: "/cfl.yml", URL: "https://cfl.atlassian.net/wiki", APIToken: "cfl-tok", DefaultSpace: "SPACE"}
	jtk := &credstore.LegacyCreds{Path: "/jtk.json", URL: "https://jtk.atlassian.net", APIToken: "jtk-tok", DefaultProject: "PROJ"}
	v, _, _ := newReconcileView()
	r := resultFromMismatch(cfl, jtk, "use_cfl", &credstore.Store{}, v, "", "", "", "")
	testutil.Equal(t, "cfl-tok", r.prefill.APIToken)
	testutil.Equal(t, "SPACE", r.prefill.DefaultSpace)
	testutil.Equal(t, []string{"/cfl.yml", "/jtk.json"}, r.consumedLegacies)
}

// TestResultFromMismatch_UseCFL_ClearsStaleJTKOverride pins the
// daemon-r3 fix: if a prior keep_different run populated
// store.JTK.Section, switching to use_cfl must clear it so jtk no
// longer resolves to a stale override that shadows the new default.
func TestResultFromMismatch_UseCFL_ClearsStaleJTKOverride(t *testing.T) {
	t.Parallel()
	cfl := &credstore.LegacyCreds{Path: "/cfl.yml", URL: "https://cfl.atlassian.net/wiki", APIToken: "cfl-tok"}
	jtk := &credstore.LegacyCreds{Path: "/jtk.json", URL: "https://jtk.atlassian.net", APIToken: "jtk-tok"}
	v, _, _ := newReconcileView()
	store := &credstore.Store{
		JTK: credstore.ToolSection{
			Section:        credstore.Section{URL: "https://stale.atlassian.net", APIToken: "stale-tok"},
			DefaultProject: "PROJ", // per-tool default must survive
		},
	}
	r := resultFromMismatch(cfl, jtk, "use_cfl", store, v, "", "", "", "")
	testutil.Equal(t, writeDefault, r.target)
	testutil.Equal(t, "", r.store.JTK.URL)
	testutil.Equal(t, "", r.store.JTK.APIToken)
	testutil.Equal(t, "", r.store.CFL.URL)
	testutil.Equal(t, "", r.store.CFL.APIToken)
	testutil.Equal(t, "PROJ", r.store.JTK.DefaultProject)
	// Signals the command layer to unify on cfl's keyring token and clear
	// both per-tool override keys.
	testutil.Equal(t, true, r.unifyBoth)
	testutil.Equal(t, credstore.ToolCFL, r.unifySource)
}

// Symmetric to UseCFL_ClearsStaleJTKOverride: use_jtk also clears
// both overrides so cfl falls through to the new default.
func TestResultFromMismatch_UseJTK_ClearsBothOverrides(t *testing.T) {
	t.Parallel()
	cfl := &credstore.LegacyCreds{Path: "/cfl.yml", URL: "https://cfl.atlassian.net/wiki", APIToken: "cfl-tok", DefaultSpace: "SPACE"}
	jtk := &credstore.LegacyCreds{Path: "/jtk.json", URL: "https://jtk.atlassian.net", APIToken: "jtk-tok"}
	v, _, _ := newReconcileView()
	store := &credstore.Store{
		CFL: credstore.ToolSection{
			Section:      credstore.Section{URL: "https://stale.atlassian.net", APIToken: "stale-cfl"},
			DefaultSpace: "STALE",
		},
		JTK: credstore.ToolSection{
			Section: credstore.Section{URL: "https://stale-jtk.atlassian.net", APIToken: "stale-jtk"},
		},
	}
	r := resultFromMismatch(cfl, jtk, "use_jtk", store, v, "", "", "", "")
	testutil.Equal(t, writeDefault, r.target)
	testutil.Equal(t, "", r.store.CFL.URL)
	testutil.Equal(t, "", r.store.CFL.APIToken)
	testutil.Equal(t, "", r.store.JTK.URL)
	testutil.Equal(t, "", r.store.JTK.APIToken)
	testutil.Equal(t, "STALE", r.store.CFL.DefaultSpace) // per-tool default survives
	testutil.Equal(t, true, r.unifyBoth)
	testutil.Equal(t, credstore.ToolJTK, r.unifySource) // unify on jtk's token
}

func TestResultFromMismatch_UseJTK_PreservesCFLDefaults(t *testing.T) {
	t.Parallel()
	cfl := &credstore.LegacyCreds{Path: "/cfl.yml", URL: "https://cfl.atlassian.net/wiki", APIToken: "cfl-tok", DefaultSpace: "SPACE"}
	jtk := &credstore.LegacyCreds{Path: "/jtk.json", URL: "https://jtk.atlassian.net", APIToken: "jtk-tok"}
	v, _, _ := newReconcileView()
	r := resultFromMismatch(cfl, jtk, "use_jtk", &credstore.Store{}, v, "", "", "", "")
	testutil.Equal(t, "jtk-tok", r.prefill.APIToken)   // jtk creds chosen
	testutil.Equal(t, "SPACE", r.prefill.DefaultSpace) // but cfl's default_space preserved
}

func TestResultFromMismatch_KeepDifferent(t *testing.T) {
	t.Parallel()
	cfl := &credstore.LegacyCreds{Path: "/cfl.yml", URL: "https://cfl.atlassian.net/wiki", Email: "u@e", APIToken: "cfl-tok", DefaultSpace: "SPACE"}
	jtk := &credstore.LegacyCreds{Path: "/jtk.json", URL: "https://jtk.atlassian.net", Email: "u@e", APIToken: "jtk-tok"}
	v, stdout, _ := newReconcileView()
	store := &credstore.Store{}
	jtk.DefaultProject = "PROJ"
	store.JTK.DefaultProject = jtk.DefaultProject // mirrors detectAndReconcile Case 4 preamble
	r := resultFromMismatch(cfl, jtk, "keep_different", store, v, "", "", "", "")
	// cfl creds → CFL override; jtk creds → JTK override; default empty.
	testutil.Equal(t, writeCFLOverride, r.target)
	testutil.Equal(t, "", r.store.Default.URL)
	testutil.Equal(t, "", r.store.Default.APIToken)
	testutil.Equal(t, "https://cfl.atlassian.net", r.store.CFL.URL)
	testutil.Equal(t, "cfl-tok", r.store.CFL.APIToken)
	testutil.Equal(t, "https://jtk.atlassian.net", r.store.JTK.URL)
	testutil.Equal(t, "jtk-tok", r.store.JTK.APIToken)
	testutil.Equal(t, "SPACE", r.store.CFL.DefaultSpace)
	testutil.Equal(t, "PROJ", r.store.JTK.DefaultProject)
	// keep_different keeps per-tool overrides — it is NOT a unify action.
	testutil.Equal(t, false, r.unifyBoth)
	testutil.Equal(t, "", r.unifySource)
	if !strings.Contains(stdout.String(), "Keeping per-tool credentials") {
		t.Errorf("expected keep-different note; got: %s", stdout.String())
	}
}

func TestResultFromCFLLegacy_BasicMigration(t *testing.T) {
	t.Parallel()
	cfl := &credstore.LegacyCreds{Path: "/cfl.yml", URL: "https://acme.atlassian.net/wiki", APIToken: "tok"}
	r := resultFromCFLLegacy(cfl, &credstore.Store{}, "", "", "", "")
	testutil.Equal(t, writeDefault, r.target)
	testutil.Equal(t, "tok", r.prefill.APIToken)
	testutil.Equal(t, []string{"/cfl.yml"}, r.consumedLegacies)
}

func TestMismatchDescription_EducationalText(t *testing.T) {
	t.Parallel()
	cfl := &credstore.LegacyCreds{Path: "/cfl.yml", URL: "https://x", Email: "u@e", APIToken: "tok"}
	jtk := &credstore.LegacyCreds{Path: "/jtk.json", URL: "https://x", Email: "u@e", APIToken: "different"}
	desc := mismatchDescription(cfl, jtk)
	for _, want := range []string{
		"account-wide",
		"both Jira and Confluence",
		"id.atlassian.com/manage-profile",
		"/cfl.yml",
		"/jtk.json",
	} {
		if !strings.Contains(desc, want) {
			t.Errorf("expected %q in mismatch description; got:\n%s", want, desc)
		}
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
		CFL: credstore.ToolSection{DefaultSpace: "EXISTING"},
	}
	r := resultFromSharedNoOverride(store, true, "", "", "", "")
	testutil.Equal(t, writeDefault, r.target)
	testutil.Equal(t, "shared-tok", r.prefill.APIToken)
	testutil.Equal(t, "EXISTING", r.prefill.DefaultSpace) // carried over
	testutil.Equal(t, true, r.affectsSibling)
}

func TestResultFromSharedNoOverride_ReuseNo_FreshForm(t *testing.T) {
	t.Parallel()
	store := &credstore.Store{
		Default: credstore.Section{URL: "https://x", APIToken: "shared-tok"},
		CFL:     credstore.ToolSection{DefaultSpace: "EXISTING"},
	}
	r := resultFromSharedNoOverride(store, false, "", "", "", "")
	testutil.Equal(t, writeCFLOverride, r.target)
	testutil.Equal(t, "", r.prefill.APIToken)             // fresh, not prefilled
	testutil.Equal(t, "EXISTING", r.prefill.DefaultSpace) // tool defaults still carried so user doesn't retype
	testutil.Equal(t, false, r.affectsSibling)
}

func TestResultFromSharedWithOverride(t *testing.T) {
	t.Parallel()
	store := &credstore.Store{
		Default: credstore.Section{URL: "https://default", APIToken: "default-tok"},
		CFL: credstore.ToolSection{
			Section:      credstore.Section{URL: "https://cfl-override", APIToken: "cfl-tok"},
			DefaultSpace: "SPACE",
		},
	}
	r := resultFromSharedWithOverride(store, "", "", "", "")
	testutil.Equal(t, writeCFLOverride, r.target)
	testutil.Equal(t, "cfl-tok", r.prefill.APIToken) // override wins
	testutil.Equal(t, "SPACE", r.prefill.DefaultSpace)
	testutil.Equal(t, false, r.affectsSibling) // override doesn't affect sibling
}

func TestResultFromSiblingLegacy_PreservesJTKDefaults(t *testing.T) {
	t.Parallel()
	jtk := &credstore.LegacyCreds{
		Path:           "/jtk.json",
		URL:            "https://x",
		Email:          "u@e",
		APIToken:       "jtk-tok",
		DefaultProject: "PROJ",
	}
	store := &credstore.Store{}
	r := resultFromSiblingLegacy(jtk, store, true, "", "", "", "")
	// Even though cfl init is the running tool, jtk's default_project
	// must survive the migration.
	testutil.Equal(t, "PROJ", r.store.JTK.DefaultProject)
}
