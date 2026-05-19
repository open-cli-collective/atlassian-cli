package initcmd

import (
	"errors"

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
	// §3.2 PURE pre-token relocation gate: runs BEFORE per-tool
	// divergence detection and BEFORE any mutation. A corrupt old/new
	// file or a divergent old↔new shared config fails loud here naming
	// both paths, mutating nothing.
	rel, relErr := credstore.DetectSharedRelocation(sharedPath)
	if relErr != nil {
		if errors.Is(relErr, credstore.ErrCorruptStore) {
			// A corrupt old OR new shared file: same contract/UX as the
			// Load path below — unreadable, refuse to overwrite.
			v.Error("Shared credential store at %s is unreadable: %v", sharedPath, relErr)
			v.Error("Refusing to overwrite. Fix or remove the file, then re-run jtk init.")
		} else {
			v.Error("Shared credential store relocation check failed: %v", relErr)
			v.Error("Refusing to mutate anything. Reconcile the named file(s), then re-run jtk init.")
		}
		return nil, relErr
	}

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

	candidates := credstore.ConnCandidates(sharedPath, store.Default, proj, cflLegacy, jtkLegacy)
	// Old-shared is ADDITIVE so a relocation copy is gated on the
	// per-tool divergence check passing (no copy while a divergence is
	// pending).
	candidates = append(candidates, credstore.OldSharedConnCandidates(rel)...)
	chosen, conflicts := credstore.DetectConnDivergence(candidates)
	if len(conflicts) > 0 {
		return nil, credstore.ConnConflictError(conflicts, candidates, "jtk")
	}

	// Gated apply: every conflict gate (relocation + per-tool
	// divergence) has passed — materialize the old shared file at the
	// new path (copy-leave-old), then reload so the remainder
	// reconciles the materialized file exactly as a returning user.
	if rel.CopyNeeded {
		if aerr := credstore.ApplySharedRelocation(rel); aerr != nil {
			v.Error("Could not relocate the shared credential store: %v", aerr)
			return nil, aerr
		}
		// Reload only the canonical store: the divergence candidates
		// (incl. old-shared) were already built and checked above, so
		// `proj` is not read again — the materialized store is what the
		// fold + preserveDefaults below operate on.
		store, err = credstore.Load(sharedPath)
		if err != nil {
			v.Error("Shared credential store at %s is unreadable: %v", sharedPath, err)
			return nil, err
		}
	}

	// affectsSibling judged on the ORIGINAL loaded store, BEFORE folding
	// `chosen` (else a first-time legacy migration falsely looks like it
	// overwrites a usable shared default). True only when the store
	// already held usable creds AND the resolved connection actually
	// DIFFERS from disk: re-running `jtk init` without changing the
	// connection is a no-op for cfl and must not nag (one shared default
	// would otherwise prompt on every re-init). Pure (HasUsableConfig +
	// value compare only): NO keyring I/O in reconcile (B3 rule).
	origDefault := store.Default
	affectsSibling := store.HasUsableConfig(credstore.ToolJTK) &&
		!credstore.ConnEqualsSection(chosen, origDefault)

	// Fold the unified connection into the shared default. This in-place
	// write is intentionally redundant with applyResultToStore (called
	// by finalizeInit after the form, with the final URL-normalized
	// values); result.store.Default is never read between the two, so
	// the transient state is not observable.
	store.Default = credstore.Section{
		URL:        chosen.URL,
		Email:      chosen.Email,
		AuthMethod: chosen.AuthMethod,
		CloudID:    chosen.CloudID,
	}
	consumed := preserveDefaultsAndCollect(store, jtkLegacy, cflLegacy)

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

// preserveDefaultsAndCollect fills per-tool non-secret defaults from the
// legacy files. Legacy values only fill fields the shared store leaves
// EMPTY: a value the user already set in the shared store (e.g. a prior
// init that changed default_project) must not be silently reverted to a
// stale legacy value just because the old file still exists. Shared
// store wins; legacy backfills absent.
func preserveDefaultsAndCollect(
	store *credstore.Store,
	jtkLegacy, cflLegacy *credstore.LegacyCreds,
) []string {
	var consumed []string
	if jtkLegacy != nil {
		if store.JTK.DefaultProject == "" && jtkLegacy.DefaultProject != "" {
			store.JTK.DefaultProject = jtkLegacy.DefaultProject
		}
		if legacyHasConn(jtkLegacy) {
			consumed = append(consumed, jtkLegacy.Path)
		}
	}
	if cflLegacy != nil {
		if store.CFL.DefaultSpace == "" && cflLegacy.DefaultSpace != "" {
			store.CFL.DefaultSpace = cflLegacy.DefaultSpace
		}
		if store.CFL.OutputFormat == "" && cflLegacy.OutputFormat != "" {
			store.CFL.OutputFormat = cflLegacy.OutputFormat
		}
		if legacyHasConn(cflLegacy) {
			consumed = append(consumed, cflLegacy.Path)
		}
	}
	return consumed
}

func legacyHasConn(l *credstore.LegacyCreds) bool {
	return l.URL != "" || l.Email != "" || l.AuthMethod != "" || l.CloudID != ""
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
