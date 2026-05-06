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
		// override; otherwise edits land in default.
		hasOverride := !sectionEmpty(store.CFL.Section)
		var target writeTarget
		if hasOverride {
			target = writeCFLOverride
		} else {
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
				target = writeDefault
				v.Info("Note: edits will affect both cfl and jtk (writing to shared default).")
			} else {
				target = writeCFLOverride
			}
		}
		cfg := configFromSection(store.Resolve(credstore.ToolCFL))
		copyCFLDefaults(cfg, store.CFL)
		applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID)
		return &reconcileResult{prefill: cfg, target: target, store: store}, nil
	}

	// Case 2: only this tool's legacy exists.
	if cflLegacy != nil && jtkLegacy == nil {
		cfg := configFromLegacy(cflLegacy)
		applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID)
		v.Info("Migrating existing cfl config at %s to shared store.", cflLegacy.Path)
		return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: []string{cflLegacy.Path}}, nil
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
		return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: consumed}, nil
	}

	// Case 4: both legacies exist.
	if cflLegacy != nil && jtkLegacy != nil {
		if credstore.SectionsEqual(cflLegacy.Section(), jtkLegacy.Section()) {
			v.Info("Found matching cfl and jtk credentials; migrating to shared store.")
			cfg := configFromLegacy(cflLegacy)
			applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID)
			return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: []string{cflLegacy.Path, jtkLegacy.Path}}, nil
		}
		// Mismatch: educational reconciliation flow.
		choice, err := promptReconcileMismatch(cflLegacy, jtkLegacy)
		if err != nil {
			return nil, err
		}
		switch choice {
		case "use_cfl":
			cfg := configFromLegacy(cflLegacy)
			applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID)
			return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: []string{cflLegacy.Path, jtkLegacy.Path}}, nil
		case "use_jtk":
			cfg := configFromLegacy(jtkLegacy)
			applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID)
			return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: []string{cflLegacy.Path, jtkLegacy.Path}}, nil
		case "keep_different":
			// Write the cfl creds to default, jtk creds as a jtk override.
			// We do this directly on the store now so the post-form save
			// preserves both halves.
			store.Default = cflLegacy.Section()
			store.CFL.DefaultSpace = cflLegacy.DefaultSpace
			store.CFL.OutputFormat = cflLegacy.OutputFormat
			store.JTK.Section = jtkLegacy.Section()
			store.JTK.DefaultProject = jtkLegacy.DefaultProject
			cfg := configFromLegacy(cflLegacy)
			applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID)
			v.Info("Keeping per-tool credentials. cfl will use cfl's token; jtk will use jtk's token.")
			return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: []string{cflLegacy.Path, jtkLegacy.Path}}, nil
		}
	}

	// Case 5: nothing on disk anywhere.
	cfg := &config.Config{}
	applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillAuthMethod, prefillCloudID)
	return &reconcileResult{prefill: cfg, target: writeDefault, store: store}, nil
}

func promptReconcileMismatch(cflLegacy, jtkLegacy *credstore.LegacyCreds) (string, error) {
	desc := fmt.Sprintf(
		"%s\n\n%s\n\nNote: Atlassian API tokens are account-wide. One token usually works for both Jira and Confluence.\nManage tokens: https://id.atlassian.com/manage-profile/security/api-tokens",
		credstore.FormatSection("cfl ("+cflLegacy.Path+")", cflLegacy.Section()),
		credstore.FormatSection("jtk ("+jtkLegacy.Path+")", jtkLegacy.Section()),
	)
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
// final cfg in the right section. It preserves any unrelated existing
// fields (e.g., jtk section, jtk per-tool defaults).
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
	// Tool-specific bits always live in the cfl section.
	store.CFL.DefaultSpace = cfg.DefaultSpace
	store.CFL.OutputFormat = cfg.OutputFormat
}
