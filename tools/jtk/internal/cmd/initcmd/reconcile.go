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
	chosen, conflicts := credstore.DetectConnDivergence(candidates)
	if len(conflicts) > 0 {
		return nil, credstore.ConnConflictError(conflicts, candidates, "jtk")
	}

	// affectsSibling judged on the ORIGINAL loaded store, BEFORE folding
	// `chosen` (else a first-time legacy migration falsely looks like it
	// overwrites a usable shared default). Pure: store.HasUsableConfig
	// only — NO keyring I/O in reconcile (B3 leak-regression rule).
	affectsSibling := store.HasUsableConfig(credstore.ToolJTK)

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

func preserveDefaultsAndCollect(
	store *credstore.Store,
	jtkLegacy, cflLegacy *credstore.LegacyCreds,
) []string {
	var consumed []string
	if jtkLegacy != nil {
		if jtkLegacy.DefaultProject != "" {
			store.JTK.DefaultProject = jtkLegacy.DefaultProject
		}
		if legacyHasConn(jtkLegacy) {
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
