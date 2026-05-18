package initcmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
)

// reconcileResult captures what finalizeInit needs after detection. Per
// §2.2 (MON-5328) connection config is single-sourced from the shared
// `default` section — no per-tool override, so no write-target choice;
// connection always saves to `default`. jtk's mirror of cfl init's
// reconcileResult (sibling/tool roles swapped).
type reconcileResult struct {
	prefill          *config.Config
	store            *credstore.Store
	consumedLegacies []string
	// affectsSibling: the save mutates the one shared default cfl also
	// reads AND the store already held usable creds.
	affectsSibling bool
}

// detectAndReconcile is jtk's mirror of cfl init's single-source
// reconciliation. It gathers every connection candidate (shared
// default, the pre-MON-5328 shared per-tool sections via the migration
// projection, legacy jtk/cfl files), runs the pure divergence detector,
// and FAILS LOUD if they disagree (naming every source + field, never a
// value) instead of precedence-picking. Aligned → the unified
// connection is folded into the shared default; per-tool non-secret
// defaults are preserved.
func detectAndReconcile(
	v *view.View,
	jtkLegacyPath, cflLegacyPath, sharedPath string,
	prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID string,
) (*reconcileResult, error) {
	store, err := credstore.Load(sharedPath)
	if err != nil {
		v.Error("Shared credential store at %s is unreadable: %v", sharedPath, err)
		v.Error("Refusing to overwrite. Fix or remove the file, then re-run jtk init.")
		return nil, err
	}
	proj, err := credstore.LoadSharedLegacyProjection(sharedPath)
	if err != nil {
		v.Error("Shared credential store at %s is unreadable: %v", sharedPath, err)
		v.Error("Refusing to overwrite. Fix or remove the file, then re-run jtk init.")
		return nil, err
	}
	if proj == nil {
		proj = &credstore.SharedLegacyProjection{Path: sharedPath}
	}

	jtkLegacy, jtkErr := credstore.LoadLegacyJTK(jtkLegacyPath)
	if jtkErr != nil {
		if errors.Is(jtkErr, credstore.ErrCorruptStore) {
			v.Error("Legacy jtk config at %s is unreadable: %v", jtkLegacyPath, jtkErr)
			v.Error("Refusing to overwrite. Fix or remove the file, then re-run jtk init.")
		}
		return nil, jtkErr
	}
	cflLegacy, cflErr := credstore.LoadLegacyCFL(cflLegacyPath)
	if cflErr != nil {
		v.Info("Note: sibling cfl config at %s is unreadable; ignoring. (%v)", cflLegacyPath, cflErr)
		cflLegacy = nil
	}

	candidates := connCandidates(sharedPath, store, proj, jtkLegacy, cflLegacy)
	chosen, conflicts := credstore.DetectConnDivergence(candidates)
	if len(conflicts) > 0 {
		return nil, connConflictError(conflicts)
	}

	store.Default = credstore.Section{
		URL:        chosen.URL,
		Email:      chosen.Email,
		AuthMethod: chosen.AuthMethod,
		CloudID:    chosen.CloudID,
	}
	consumed := preserveDefaultsAndCollect(store, jtkLegacy, cflLegacy)

	// Pure: store.HasUsableConfig only — NO keyring I/O in reconcile
	// (B3 leak-regression rule). finalizeInit confirms before editing a
	// shared connection cfl also reads.
	affectsSibling := store.HasUsableConfig(credstore.ToolJTK)

	cfg := configFromConn(chosen)
	if store.JTK.DefaultProject != "" {
		cfg.DefaultProject = store.JTK.DefaultProject
	}
	applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)

	return &reconcileResult{
		prefill:          cfg,
		store:            store,
		consumedLegacies: consumed,
		affectsSibling:   affectsSibling,
	}, nil
}

// connCandidates assembles the origin-labeled connection sources for the
// detector: shared default, the pre-MON-5328 shared per-tool sections as
// effective overrides (default ⊕ section), and the legacy jtk/cfl files.
func connCandidates(
	sharedPath string,
	store *credstore.Store,
	proj *credstore.SharedLegacyProjection,
	jtkLegacy, cflLegacy *credstore.LegacyCreds,
) []credstore.NamedConn {
	var out []credstore.NamedConn
	add := func(label, section, path string, c credstore.ConnProfile) {
		if c.URL == "" && c.Email == "" && c.AuthMethod == "" && c.CloudID == "" {
			return
		}
		out = append(out, credstore.NamedConn{Label: label, Section: section, Path: path, Conn: c})
	}
	def := credstore.ConnProfile{
		URL: store.Default.URL, Email: store.Default.Email,
		AuthMethod: store.Default.AuthMethod, CloudID: store.Default.CloudID,
	}
	add("shared config", "default", sharedPath, def)
	add("shared config", "cfl", sharedPath, effectiveConn(proj.Default, proj.CFL))
	add("shared config", "jtk", sharedPath, effectiveConn(proj.Default, proj.JTK))
	if jtkLegacy != nil {
		add("legacy jtk config", "", jtkLegacy.Path, legacyConn(jtkLegacy))
	}
	if cflLegacy != nil {
		add("legacy cfl config", "", cflLegacy.Path, legacyConn(cflLegacy))
	}
	return out
}

func effectiveConn(def, sec credstore.SharedLegacyConn) credstore.ConnProfile {
	pick := func(o, d string) string {
		if o != "" {
			return o
		}
		return d
	}
	return credstore.ConnProfile{
		URL:        pick(sec.URL, def.URL),
		Email:      pick(sec.Email, def.Email),
		AuthMethod: pick(sec.AuthMethod, def.AuthMethod),
		CloudID:    pick(sec.CloudID, def.CloudID),
	}
}

func legacyConn(l *credstore.LegacyCreds) credstore.ConnProfile {
	return credstore.ConnProfile{
		URL: l.URL, Email: l.Email, AuthMethod: l.AuthMethod, CloudID: l.CloudID,
	}
}

// connConflictError renders the fail-loud message: every conflicting
// field with every contributing source descriptor, NEVER a value
// (§1.12), plus an actionable remediation pointing at every file.
func connConflictError(conflicts []credstore.ConnConflict) error {
	var b strings.Builder
	b.WriteString("connection config diverges across sources; init will not pick a winner. Conflicts:\n")
	paths := map[string]struct{}{}
	for _, c := range conflicts {
		fmt.Fprintf(&b, "  - %s: %s\n", c.Field, strings.Join(c.Sources, ", "))
		for _, s := range c.Sources {
			if i := strings.LastIndex(s, "("); i >= 0 {
				if j := strings.Index(s[i:], ")"); j > 0 {
					paths[s[i+1:i+j]] = struct{}{}
				}
			}
		}
	}
	b.WriteString("Resolve by editing/removing all but one connection in: ")
	first := true
	for p := range paths {
		if !first {
			b.WriteString(", ")
		}
		b.WriteString(p)
		first = false
	}
	b.WriteString(" — then re-run jtk init. (No values shown; secrets live only in the OS keyring.)")
	return errors.New(b.String())
}

func preserveDefaultsAndCollect(
	store *credstore.Store,
	jtkLegacy, cflLegacy *credstore.LegacyCreds,
) []string {
	var consumed []string
	if jtkLegacy != nil {
		if jtkLegacy.DefaultProject != "" {
			store.JTK.DefaultProject = jtkLegacy.DefaultProject
		}
		if hasConn(legacyConn(jtkLegacy)) {
			consumed = append(consumed, jtkLegacy.Path)
		}
	}
	if cflLegacy != nil {
		if cflLegacy.DefaultSpace != "" {
			store.CFL.DefaultSpace = cflLegacy.DefaultSpace
		}
		if cflLegacy.OutputFormat != "" {
			store.CFL.OutputFormat = cflLegacy.OutputFormat
		}
		if hasConn(legacyConn(cflLegacy)) {
			consumed = append(consumed, cflLegacy.Path)
		}
	}
	return consumed
}

func hasConn(c credstore.ConnProfile) bool {
	return c.URL != "" || c.Email != "" || c.AuthMethod != "" || c.CloudID != ""
}

func configFromConn(c credstore.ConnProfile) *config.Config {
	// jtk uses the bare instance URL (no /wiki suffix).
	return &config.Config{
		URL:        c.URL,
		Email:      c.Email,
		AuthMethod: c.AuthMethod,
		CloudID:    c.CloudID,
	}
}

func applyFlagOverrides(cfg *config.Config, url, email, token, authMethod, cloudID string) {
	if url != "" {
		cfg.URL = url
	}
	if email != "" {
		cfg.Email = email
	}
	if token != "" {
		cfg.APIToken = token
	}
	if authMethod != "" {
		cfg.AuthMethod = authMethod
	}
	if cloudID != "" {
		cfg.CloudID = cloudID
	}
}

// applyResultToStore writes the form's final cfg into the shared default
// (connection is single-sourced — §2.2) and sets the jtk per-tool
// non-secret default. The cfl section and cfl defaults are untouched.
func applyResultToStore(store *credstore.Store, cfg *config.Config) {
	store.Default = credstore.Section{
		URL:        credstore.NormalizeBaseURL(cfg.URL),
		Email:      cfg.Email,
		AuthMethod: cfg.AuthMethod,
		CloudID:    cfg.CloudID,
	}
	if cfg.DefaultProject != "" {
		store.JTK.DefaultProject = cfg.DefaultProject
	}
}
