package init

import (
	"errors"

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
	// divergence (pure, secret-free, no IO/keyring — shared with jtk).
	candidates := credstore.ConnCandidates(sharedPath, store.Default, proj, cflLegacy, jtkLegacy)
	chosen, conflicts := credstore.DetectConnDivergence(candidates)
	if len(conflicts) > 0 {
		return nil, credstore.ConnConflictError(conflicts, candidates, "cfl")
	}

	// affectsSibling must be judged on the ORIGINAL loaded store, BEFORE
	// folding `chosen` in — otherwise a first-time migration from only a
	// legacy file looks like it is overwriting an already-usable shared
	// default and the user gets a misleading "Save will affect sibling"
	// prompt. Pure (store.HasUsableConfig only): NO keyring I/O in
	// reconcile (the B3 leak-regression rule).
	affectsSibling := store.HasUsableConfig(credstore.ToolCFL)

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
		if legacyHasConn(cflLegacy) {
			consumed = append(consumed, cflLegacy.Path)
		}
	}
	if jtkLegacy != nil {
		if jtkLegacy.DefaultProject != "" {
			store.JTK.DefaultProject = jtkLegacy.DefaultProject
		}
		if legacyHasConn(jtkLegacy) {
			consumed = append(consumed, jtkLegacy.Path)
		}
	}
	return consumed
}

func legacyHasConn(l *credstore.LegacyCreds) bool {
	return l.URL != "" || l.Email != "" || l.AuthMethod != "" || l.CloudID != ""
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
