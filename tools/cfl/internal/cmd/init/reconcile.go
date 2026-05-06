package init

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/confluence-cli/internal/config"
)

// writeTarget tells the post-form save logic which section of the
// shared store to write credential edits into. Tool-specific bits
// (default_space/output_format) always go to the cfl section
// regardless of target.
type writeTarget int

const (
	writeDefault writeTarget = iota
	writeCFLOverride
)

// reconcileResult captures everything finalizeInit needs after the
// detection + prompt phase: a *Config to seed the form, a write target,
// the shared store the user already had on disk (so save preserves
// unrelated fields like the jtk section), and the list of legacy
// files that the user might want to clean up after migration.
type reconcileResult struct {
	prefill          *config.Config
	target           writeTarget
	store            *credstore.Store
	consumedLegacies []string // paths of legacy files actually read into prefill
	// affectsSibling is true when finalizeInit should confirm before
	// writing because the save will mutate credentials the sibling tool
	// is currently reading from. Set when reuse=yes was chosen on a
	// shared store that already had usable creds.
	affectsSibling bool
}

// detectAndReconcile decides what to do given whatever configs already
// exist on disk. It runs whatever interactive prompts are necessary to
// disambiguate, then returns a deterministic result for finalizeInit.
//
// Path arguments are injected so tests can point them at a tempdir;
// production passes the canonical paths.
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

	cflLegacy, cflErr := credstore.LoadLegacyCFL(cflLegacyPath)
	if cflErr != nil {
		if errors.Is(cflErr, credstore.ErrCorruptStore) {
			v.Error("Legacy cfl config at %s is unreadable: %v", cflLegacyPath, cflErr)
			v.Error("Refusing to overwrite. Fix or remove the file, then re-run cfl init.")
			return nil, cflErr
		}
		return nil, cflErr
	}
	jtkLegacy, jtkErr := credstore.LoadLegacyJTK(jtkLegacyPath)
	if jtkErr != nil {
		// Sibling-corrupt is a warning, not a hard stop — we can still
		// migrate this tool's data without touching the sibling file.
		v.Info("Note: sibling jtk config at %s is unreadable; ignoring. (%v)", jtkLegacyPath, jtkErr)
		jtkLegacy = nil
	}

	// Case 1: shared store has usable creds for cfl already.
	if store.HasUsableCreds(credstore.ToolCFL) {
		// If the user already has a cfl override, edits go back to the
		// override; otherwise the user picks default-vs-override.
		hasOverride := !sectionEmpty(store.CFL.Section)
		if hasOverride {
			return resultFromSharedWithOverride(store, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID), nil
		}
		var reuse bool
		err := huh.NewConfirm().
			Title("Shared Atlassian credentials found").
			Description(fmt.Sprintf(
				"%s\n\nReuse these for cfl? (no = set up cfl-specific credentials)",
				credstore.FormatSection("default", store.Default),
			)).
			Affirmative("Reuse").
			Negative("Set cfl-specific").
			Value(&reuse).
			Run()
		if err != nil {
			return nil, err
		}
		if reuse {
			v.Info("Note: editing these credentials will also affect jtk (writes go to shared default).")
		}
		return resultFromSharedNoOverride(store, reuse, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID), nil
	}

	// Case 2: only this tool's legacy exists.
	if cflLegacy != nil && jtkLegacy == nil {
		v.Info("Migrating existing cfl config at %s to shared store.", cflLegacy.Path)
		return resultFromCFLLegacy(cflLegacy, store, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID), nil
	}

	// Case 3: only sibling legacy exists.
	if cflLegacy == nil && jtkLegacy != nil {
		var reuse bool
		err := huh.NewConfirm().
			Title("Found jtk credentials").
			Description(fmt.Sprintf(
				"%s\n\nReuse these for cfl? (Atlassian API tokens are account-wide and usually work across products.)",
				credstore.FormatSection("jtk", jtkLegacy.Section()),
			)).
			Affirmative("Reuse").
			Negative("Fresh setup").
			Value(&reuse).
			Run()
		if err != nil {
			return nil, err
		}
		return resultFromSiblingLegacy(jtkLegacy, store, reuse, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID), nil
	}

	// Case 4: both legacies exist. Either way, preserve jtk's per-tool
	// defaults into the store so jtk doesn't lose default_project on
	// next read.
	if cflLegacy != nil && jtkLegacy != nil {
		store.JTK.DefaultProject = jtkLegacy.DefaultProject
		if credstore.SectionsEqual(cflLegacy.Section(), jtkLegacy.Section()) {
			v.Info("Found matching cfl and jtk credentials; migrating to shared store.")
			cfg := configFromLegacy(cflLegacy)
			applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID)
			return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: []string{cflLegacy.Path, jtkLegacy.Path}}, nil
		}
		choice, err := promptReconcileMismatch(cflLegacy, jtkLegacy)
		if err != nil {
			return nil, err
		}
		return resultFromMismatch(cflLegacy, jtkLegacy, choice, store, v, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID), nil
	}

	// Case 5: nothing on disk anywhere.
	cfg := &config.Config{}
	applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID)
	return &reconcileResult{prefill: cfg, target: writeDefault, store: store}, nil
}

// mismatchDescription is the prompt description for the both-legacies-
// mismatch reconciliation. Extracted so tests can assert the
// educational language without driving huh.
func mismatchDescription(cflLegacy, jtkLegacy *credstore.LegacyCreds) string {
	return fmt.Sprintf(
		"%s\n\n%s\n\nNote: Atlassian API tokens are account-wide. One token usually works for both Jira and Confluence.\nManage tokens: https://id.atlassian.com/manage-profile/security/api-tokens",
		credstore.FormatSection("cfl ("+cflLegacy.Path+")", cflLegacy.Section()),
		credstore.FormatSection("jtk ("+jtkLegacy.Path+")", jtkLegacy.Section()),
	)
}

func promptReconcileMismatch(cflLegacy, jtkLegacy *credstore.LegacyCreds) (string, error) {
	desc := mismatchDescription(cflLegacy, jtkLegacy)
	var choice string
	err := huh.NewSelect[string]().
		Title("Different Atlassian credentials found").
		Description(desc).
		Options(
			huh.NewOption("Use cfl's credentials for both tools", "use_cfl"),
			huh.NewOption("Use jtk's credentials for both tools", "use_jtk"),
			huh.NewOption("Keep them different (advanced)", "keep_different"),
		).
		Value(&choice).
		Run()
	return choice, err
}

// resultFromCFLLegacy is the post-prompt branch for "only this tool's
// legacy" — lifted out so tests can drive it without huh.
func resultFromCFLLegacy(cflLegacy *credstore.LegacyCreds, store *credstore.Store, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID string) *reconcileResult {
	cfg := configFromLegacy(cflLegacy)
	applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID)
	return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: []string{cflLegacy.Path}}
}

// resultFromSiblingLegacy is the post-prompt branch for "only sibling
// (jtk) legacy" — `reuse` is what huh returned. Either way, jtk's
// per-tool defaults (default_project) are preserved into the store so
// jtk doesn't lose them on next read.
func resultFromSiblingLegacy(jtkLegacy *credstore.LegacyCreds, store *credstore.Store, reuse bool, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID string) *reconcileResult {
	store.JTK.DefaultProject = jtkLegacy.DefaultProject
	var cfg *config.Config
	if reuse {
		cfg = configFromLegacy(jtkLegacy)
	} else {
		cfg = &config.Config{}
	}
	applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID)
	consumed := []string{}
	if reuse {
		consumed = []string{jtkLegacy.Path}
	}
	return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: consumed}
}

// resultFromSharedNoOverride is the post-prompt branch for the
// shared-store-present-no-override case in detectAndReconcile.
// `reuse` is what huh returned.
func resultFromSharedNoOverride(store *credstore.Store, reuse bool, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID string) *reconcileResult {
	var cfg *config.Config
	target := writeCFLOverride
	if reuse {
		cfg = configFromSection(store.Resolve(credstore.ToolCFL))
		copyCFLDefaults(cfg, store.CFL)
		target = writeDefault
	} else {
		cfg = &config.Config{}
		copyCFLDefaults(cfg, store.CFL)
	}
	applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID)
	return &reconcileResult{
		prefill:        cfg,
		target:         target,
		store:          store,
		affectsSibling: reuse, // reuse=yes writes to default → affects jtk
	}
}

// resultFromSharedWithOverride is the (no-prompt) branch when shared
// store already has a populated cfl override section. Edits land in
// the override; default isn't touched.
func resultFromSharedWithOverride(store *credstore.Store, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID string) *reconcileResult {
	cfg := configFromSection(store.Resolve(credstore.ToolCFL))
	copyCFLDefaults(cfg, store.CFL)
	applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID)
	return &reconcileResult{prefill: cfg, target: writeCFLOverride, store: store}
}

// resultFromMismatch is the post-prompt branch for "both legacies
// exist with different creds". `choice` is the huh select value:
// "use_cfl" | "use_jtk" | "keep_different".
func resultFromMismatch(cflLegacy, jtkLegacy *credstore.LegacyCreds, choice string, store *credstore.Store, v *view.View, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID string) *reconcileResult {
	consumed := []string{cflLegacy.Path, jtkLegacy.Path}
	switch choice {
	case "use_cfl", "use_jtk":
		// User chose to unify on one tool's creds. Clear both
		// override Sections so each tool resolves to the new default
		// rather than a leftover override from a prior keep_different
		// run. Per-tool defaults (default_space, default_project,
		// output_format) are preserved.
		store.CFL.Section = credstore.Section{}
		store.JTK.Section = credstore.Section{}
		var cfg *config.Config
		if choice == "use_cfl" {
			cfg = configFromLegacy(cflLegacy)
		} else {
			cfg = configFromLegacy(jtkLegacy)
			cfg.DefaultSpace = cflLegacy.DefaultSpace
			cfg.OutputFormat = cflLegacy.OutputFormat
		}
		applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID)
		return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: consumed}
	case "keep_different":
		// Both tools land in their override sections so the split is
		// stable: store.Default stays empty, cfl reads its override,
		// jtk reads its override. Save target is writeCFLOverride so
		// post-form edits stay in the cfl section.
		store.CFL.Section = cflLegacy.Section()
		store.CFL.DefaultSpace = cflLegacy.DefaultSpace
		store.CFL.OutputFormat = cflLegacy.OutputFormat
		store.JTK.Section = jtkLegacy.Section()
		cfg := configFromLegacy(cflLegacy)
		applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID)
		v.Info("Keeping per-tool credentials. cfl will use cfl's token; jtk will use jtk's token.")
		return &reconcileResult{prefill: cfg, target: writeCFLOverride, store: store, consumedLegacies: consumed}
	}
	cfg := &config.Config{}
	applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID)
	return &reconcileResult{prefill: cfg, target: writeDefault, store: store}
}

func sectionEmpty(s credstore.Section) bool {
	return s.URL == "" && s.Email == "" && s.APIToken == "" && s.AuthMethod == "" && s.CloudID == ""
}

func configFromSection(s credstore.Section) *config.Config {
	cfg := &config.Config{
		Email:      s.Email,
		APIToken:   s.APIToken,
		AuthMethod: s.AuthMethod,
		CloudID:    s.CloudID,
	}
	if s.URL != "" {
		cfg.URL = credstore.URLForCFL(s.URL)
	}
	return cfg
}

func configFromLegacy(l *credstore.LegacyCreds) *config.Config {
	cfg := configFromSection(l.Section())
	if l.DefaultSpace != "" {
		cfg.DefaultSpace = l.DefaultSpace
	}
	if l.OutputFormat != "" {
		cfg.OutputFormat = l.OutputFormat
	}
	return cfg
}

func copyCFLDefaults(cfg *config.Config, t credstore.ToolSection) {
	if t.DefaultSpace != "" {
		cfg.DefaultSpace = t.DefaultSpace
	}
	if t.OutputFormat != "" {
		cfg.OutputFormat = t.OutputFormat
	}
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

// applyResultToStore mutates the shared store so it carries the form's
// final cfg in the right section. It preserves unrelated existing
// fields (jtk section, jtk per-tool defaults) and only overwrites a
// CFL per-tool default when the form actually produced a value, so a
// freshly-set-up cfl doesn't stomp a previously-saved default_space.
func applyResultToStore(store *credstore.Store, cfg *config.Config, target writeTarget) {
	cred := credstore.Section{
		URL:        credstore.NormalizeBaseURL(cfg.URL),
		Email:      cfg.Email,
		APIToken:   cfg.APIToken,
		AuthMethod: cfg.AuthMethod,
		CloudID:    cfg.CloudID,
	}
	switch target {
	case writeDefault:
		store.Default = cred
	case writeCFLOverride:
		store.CFL.Section = cred
	}
	if cfg.DefaultSpace != "" {
		store.CFL.DefaultSpace = cfg.DefaultSpace
	}
	if cfg.OutputFormat != "" {
		store.CFL.OutputFormat = cfg.OutputFormat
	}
}
