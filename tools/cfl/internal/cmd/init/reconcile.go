package init

import (
	"errors"
	"fmt"
	"strings"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/confluence-cli/internal/config"
)

// reconcileResult captures everything finalizeInit needs after the
// detection phase: a *Config to seed the form, the shared store the user
// already had on disk (so save preserves unrelated fields like the jtk
// section), and the legacy files the user might want to clean up.
//
// Per §2.2 (MON-5328) connection config is single-sourced from the
// shared `default` section — there is no per-tool override and therefore
// no write-target choice; finalizeInit always writes connection to
// `default`.
type reconcileResult struct {
	prefill          *config.Config
	store            *credstore.Store
	consumedLegacies []string // legacy file paths folded into the result
	// affectsSibling is true when the save will mutate connection
	// credentials the sibling tool also reads (always, now that there is
	// one shared default) AND the store already held usable creds — so
	// finalizeInit confirms before overwriting a working shared config.
	affectsSibling bool
}

// detectAndReconcile decides what to do given whatever configs already
// exist on disk. Connection config is single-sourced (§2.2): it gathers
// every connection candidate (shared default, the pre-MON-5328 shared
// per-tool sections via the migration projection, and the legacy cfl/jtk
// files), runs the pure divergence detector, and FAILS LOUD if they
// disagree (naming every source + field, never a value) rather than
// precedence-picking. Aligned → the unified connection is folded into
// the shared default; per-tool non-secret defaults are preserved.
//
// Path arguments are injected so tests can point them at a tempdir.
func detectAndReconcile(
	v *view.View,
	cflLegacyPath, jtkLegacyPath, sharedPath string,
	prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID string,
) (*reconcileResult, error) {
	store, err := credstore.Load(sharedPath)
	if err != nil {
		v.Error("Shared credential store at %s is unreadable: %v", sharedPath, err)
		v.Error("Refusing to overwrite. Fix or remove the file, then re-run cfl init.")
		return nil, err
	}
	// Migration projection retains the pre-MON-5328 per-tool connection
	// fields the canonical Store dropped (EnsureMigrated's token-only
	// scrub preserves them, so they are still readable here).
	proj, err := credstore.LoadSharedLegacyProjection(sharedPath)
	if err != nil {
		v.Error("Shared credential store at %s is unreadable: %v", sharedPath, err)
		v.Error("Refusing to overwrite. Fix or remove the file, then re-run cfl init.")
		return nil, err
	}
	if proj == nil {
		proj = &credstore.SharedLegacyProjection{Path: sharedPath}
	}

	cflLegacy, cflErr := credstore.LoadLegacyCFL(cflLegacyPath)
	if cflErr != nil {
		if errors.Is(cflErr, credstore.ErrCorruptStore) {
			v.Error("Legacy cfl config at %s is unreadable: %v", cflLegacyPath, cflErr)
			v.Error("Refusing to overwrite. Fix or remove the file, then re-run cfl init.")
		}
		return nil, cflErr
	}
	jtkLegacy, jtkErr := credstore.LoadLegacyJTK(jtkLegacyPath)
	if jtkErr != nil {
		// Sibling-corrupt is a warning, not a hard stop.
		v.Info("Note: sibling jtk config at %s is unreadable; ignoring. (%v)", jtkLegacyPath, jtkErr)
		jtkLegacy = nil
	}

	// Build the full named connection candidate set and detect
	// divergence (pure, secret-free, no IO/keyring).
	candidates := connCandidates(sharedPath, store, proj, cflLegacy, jtkLegacy)
	chosen, conflicts := credstore.DetectConnDivergence(candidates)
	if len(conflicts) > 0 {
		return nil, connConflictError(conflicts)
	}

	// Aligned: fold the unified connection into the shared default and
	// preserve per-tool non-secret defaults (cfl's space/output, jtk's
	// project) so neither tool loses them on next read.
	store.Default = credstore.Section{
		URL:        chosen.URL,
		Email:      chosen.Email,
		AuthMethod: chosen.AuthMethod,
		CloudID:    chosen.CloudID,
	}
	consumed := preserveDefaultsAndCollect(store, cflLegacy, jtkLegacy)

	// If the shared store already holds a usable connection, editing it
	// also affects jtk (single shared default) — finalizeInit confirms.
	// Pure (store.HasUsableConfig only): NO keyring I/O in reconcile (the
	// B3 leak-regression rule — keyring access lives in the command
	// layer, never in this pure path).
	affectsSibling := store.HasUsableConfig(credstore.ToolCFL)

	cfg := configFromConn(chosen)
	if store.CFL.DefaultSpace != "" {
		cfg.DefaultSpace = store.CFL.DefaultSpace
	}
	if store.CFL.OutputFormat != "" {
		cfg.OutputFormat = store.CFL.OutputFormat
	}
	applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID)

	return &reconcileResult{
		prefill:          cfg,
		store:            store,
		consumedLegacies: consumed,
		affectsSibling:   affectsSibling,
	}, nil
}

// connCandidates assembles the origin-labeled connection sources for the
// divergence detector: shared default, the pre-MON-5328 shared per-tool
// sections as effective overrides (default ⊕ section), and the legacy
// cfl/jtk files. All-empty candidates are skipped (the detector also
// treats them as "no opinion", but skipping keeps conflict labels tight).
func connCandidates(
	sharedPath string,
	store *credstore.Store,
	proj *credstore.SharedLegacyProjection,
	cflLegacy, jtkLegacy *credstore.LegacyCreds,
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
	if cflLegacy != nil {
		add("legacy cfl config", "", cflLegacy.Path, legacyConn(cflLegacy))
	}
	if jtkLegacy != nil {
		add("legacy jtk config", "", jtkLegacy.Path, legacyConn(jtkLegacy))
	}
	return out
}

// effectiveConn merges a pre-MON-5328 per-tool section over default
// (the old per-field-merge semantics) so the detector compares what the
// tool actually USED to resolve.
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
// field with every contributing source descriptor
// (`<label> <section>.<field> (<path>)`), NEVER a value (§1.12), and an
// actionable remediation pointing at every distinct file.
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
	b.WriteString(" — then re-run cfl init. (No values shown; secrets live only in the OS keyring.)")
	return errors.New(b.String())
}

// preserveDefaultsAndCollect keeps per-tool non-secret defaults from the
// pre-strip projection and legacy files, and returns the legacy file
// paths that contributed a connection (so init can offer to delete them).
func preserveDefaultsAndCollect(
	store *credstore.Store,
	cflLegacy, jtkLegacy *credstore.LegacyCreds,
) []string {
	var consumed []string
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
	if jtkLegacy != nil {
		if jtkLegacy.DefaultProject != "" {
			store.JTK.DefaultProject = jtkLegacy.DefaultProject
		}
		if hasConn(legacyConn(jtkLegacy)) {
			consumed = append(consumed, jtkLegacy.Path)
		}
	}
	return consumed
}

func hasConn(c credstore.ConnProfile) bool {
	return c.URL != "" || c.Email != "" || c.AuthMethod != "" || c.CloudID != ""
}

func configFromConn(c credstore.ConnProfile) *config.Config {
	cfg := &config.Config{
		Email:      c.Email,
		AuthMethod: c.AuthMethod,
		CloudID:    c.CloudID,
	}
	if c.URL != "" {
		cfg.URL = credstore.URLForCFL(c.URL)
	}
	return cfg
}

func applyFlagOverrides(cfg *config.Config, url, email, authMethod, cloudID string) {
	if url != "" {
		cfg.URL = url
	}
	if email != "" {
		cfg.Email = email
	}
	if authMethod != "" {
		cfg.AuthMethod = authMethod
	}
	if cloudID != "" {
		cfg.CloudID = cloudID
	}
}

// applyResultToStore writes the form's final cfg into the shared default
// (connection is single-sourced — §2.2) and preserves/sets the cfl
// per-tool non-secret defaults. The jtk section and jtk defaults are
// left untouched.
func applyResultToStore(store *credstore.Store, cfg *config.Config) {
	store.Default = credstore.Section{
		URL:        credstore.NormalizeBaseURL(cfg.URL),
		Email:      cfg.Email,
		AuthMethod: cfg.AuthMethod,
		CloudID:    cfg.CloudID,
	}
	if cfg.DefaultSpace != "" {
		store.CFL.DefaultSpace = cfg.DefaultSpace
	}
	if cfg.OutputFormat != "" {
		store.CFL.OutputFormat = cfg.OutputFormat
	}
}
