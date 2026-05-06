package initcmd

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
)

// writeTarget tells the post-form save logic which section of the
// shared store to write credential edits into.
type writeTarget int

const (
	writeDefault writeTarget = iota
	writeJTKOverride
)

type reconcileResult struct {
	prefill          *config.Config
	target           writeTarget
	store            *credstore.Store
	consumedLegacies []string
	// affectsSibling is true when finalizeInit should confirm before
	// writing because the save will mutate credentials the sibling tool
	// is currently reading from. Set when reuse=yes was chosen on a
	// shared store that already had usable creds.
	affectsSibling bool
}

// detectAndReconcile is jtk's mirror of cfl init's reconciliation
// flow. See tools/cfl/internal/cmd/init/reconcile.go for the
// canonical commentary; the logic here is symmetric with sibling and
// tool roles swapped.
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

	jtkLegacy, jtkErr := credstore.LoadLegacyJTK(jtkLegacyPath)
	if jtkErr != nil {
		if errors.Is(jtkErr, credstore.ErrCorruptStore) {
			v.Error("Legacy jtk config at %s is unreadable: %v", jtkLegacyPath, jtkErr)
			v.Error("Refusing to overwrite. Fix or remove the file, then re-run jtk init.")
			return nil, jtkErr
		}
		return nil, jtkErr
	}
	cflLegacy, cflErr := credstore.LoadLegacyCFL(cflLegacyPath)
	if cflErr != nil {
		v.Info("Note: sibling cfl config at %s is unreadable; ignoring. (%v)", cflLegacyPath, cflErr)
		cflLegacy = nil
	}

	// Case 1: shared store has usable jtk creds.
	if store.HasUsableCreds(credstore.ToolJTK) {
		hasOverride := !sectionEmpty(store.JTK.Section)
		if hasOverride {
			return resultFromSharedWithOverride(store, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID), nil
		}
		var reuse bool
		err := huh.NewConfirm().
			Title("Shared Atlassian credentials found").
			Description(fmt.Sprintf(
				"%s\n\nReuse these for jtk? (no = set up jtk-specific credentials)",
				credstore.FormatSection("default", store.Default),
			)).
			Affirmative("Reuse").
			Negative("Set jtk-specific").
			Value(&reuse).
			Run()
		if err != nil {
			return nil, err
		}
		if reuse {
			v.Info("Note: editing these credentials will also affect cfl (writes go to shared default).")
		}
		return resultFromSharedNoOverride(store, reuse, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID), nil
	}

	// Case 2: only this tool's legacy.
	if jtkLegacy != nil && cflLegacy == nil {
		v.Info("Migrating existing jtk config at %s to shared store.", jtkLegacy.Path)
		return resultFromJTKLegacy(jtkLegacy, store, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID), nil
	}

	// Case 3: only sibling cfl legacy.
	if jtkLegacy == nil && cflLegacy != nil {
		var reuse bool
		err := huh.NewConfirm().
			Title("Found cfl credentials").
			Description(fmt.Sprintf(
				"%s\n\nReuse these for jtk? (Atlassian API tokens are account-wide and usually work across products.)",
				credstore.FormatSection("cfl", cflLegacy.Section()),
			)).
			Affirmative("Reuse").
			Negative("Fresh setup").
			Value(&reuse).
			Run()
		if err != nil {
			return nil, err
		}
		return resultFromSiblingLegacy(cflLegacy, store, reuse, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID), nil
	}

	// Case 4: both legacies exist. Preserve cfl's per-tool defaults on
	// the store either way so default_space/output_format aren't lost.
	if jtkLegacy != nil && cflLegacy != nil {
		store.CFL.DefaultSpace = cflLegacy.DefaultSpace
		store.CFL.OutputFormat = cflLegacy.OutputFormat
		if credstore.SectionsEqual(jtkLegacy.Section(), cflLegacy.Section()) {
			v.Info("Found matching jtk and cfl credentials; migrating to shared store.")
			cfg := configFromLegacy(jtkLegacy)
			applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)
			return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: []string{jtkLegacy.Path, cflLegacy.Path}}, nil
		}
		choice, err := promptReconcileMismatch(jtkLegacy, cflLegacy)
		if err != nil {
			return nil, err
		}
		return resultFromMismatch(jtkLegacy, cflLegacy, choice, store, v, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID), nil
	}

	cfg := &config.Config{}
	applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)
	return &reconcileResult{prefill: cfg, target: writeDefault, store: store}, nil
}

func promptReconcileMismatch(jtkLegacy, cflLegacy *credstore.LegacyCreds) (string, error) {
	desc := fmt.Sprintf(
		"%s\n\n%s\n\nNote: Atlassian API tokens are account-wide. One token usually works for both Jira and Confluence.\nManage tokens: https://id.atlassian.com/manage-profile/security/api-tokens",
		credstore.FormatSection("jtk ("+jtkLegacy.Path+")", jtkLegacy.Section()),
		credstore.FormatSection("cfl ("+cflLegacy.Path+")", cflLegacy.Section()),
	)
	var choice string
	err := huh.NewSelect[string]().
		Title("Different Atlassian credentials found").
		Description(desc).
		Options(
			huh.NewOption("Use jtk's credentials for both tools", "use_jtk"),
			huh.NewOption("Use cfl's credentials for both tools", "use_cfl"),
			huh.NewOption("Keep them different (advanced)", "keep_different"),
		).
		Value(&choice).
		Run()
	return choice, err
}

// resultFromJTKLegacy / resultFromSiblingLegacy / resultFromMismatch
// are the post-prompt branches lifted out so tests can drive them
// without huh. See the cfl mirror in tools/cfl/internal/cmd/init/reconcile.go.

func resultFromJTKLegacy(jtkLegacy *credstore.LegacyCreds, store *credstore.Store, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID string) *reconcileResult {
	cfg := configFromLegacy(jtkLegacy)
	applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)
	return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: []string{jtkLegacy.Path}}
}

func resultFromSiblingLegacy(cflLegacy *credstore.LegacyCreds, store *credstore.Store, reuse bool, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID string) *reconcileResult {
	store.CFL.DefaultSpace = cflLegacy.DefaultSpace
	store.CFL.OutputFormat = cflLegacy.OutputFormat
	var cfg *config.Config
	if reuse {
		cfg = configFromLegacy(cflLegacy)
	} else {
		cfg = &config.Config{}
	}
	applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)
	consumed := []string{}
	if reuse {
		consumed = []string{cflLegacy.Path}
	}
	return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: consumed}
}

func resultFromSharedNoOverride(store *credstore.Store, reuse bool, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID string) *reconcileResult {
	var cfg *config.Config
	target := writeJTKOverride
	if reuse {
		cfg = configFromSection(store.Resolve(credstore.ToolJTK))
		copyJTKDefaults(cfg, store.JTK)
		target = writeDefault
	} else {
		cfg = &config.Config{}
		copyJTKDefaults(cfg, store.JTK)
	}
	applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)
	return &reconcileResult{
		prefill:        cfg,
		target:         target,
		store:          store,
		affectsSibling: reuse,
	}
}

func resultFromSharedWithOverride(store *credstore.Store, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID string) *reconcileResult {
	cfg := configFromSection(store.Resolve(credstore.ToolJTK))
	copyJTKDefaults(cfg, store.JTK)
	applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)
	return &reconcileResult{prefill: cfg, target: writeJTKOverride, store: store}
}

func resultFromMismatch(jtkLegacy, cflLegacy *credstore.LegacyCreds, choice string, store *credstore.Store, v *view.View, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID string) *reconcileResult {
	consumed := []string{jtkLegacy.Path, cflLegacy.Path}
	switch choice {
	case "use_jtk":
		cfg := configFromLegacy(jtkLegacy)
		applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)
		return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: consumed}
	case "use_cfl":
		cfg := configFromLegacy(cflLegacy)
		cfg.DefaultProject = jtkLegacy.DefaultProject
		applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)
		return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: consumed}
	case "keep_different":
		store.Default = jtkLegacy.Section()
		store.CFL.Section = cflLegacy.Section()
		cfg := configFromLegacy(jtkLegacy)
		applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)
		v.Info("Keeping per-tool credentials. jtk will use jtk's token; cfl will use cfl's token.")
		return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: consumed}
	}
	cfg := &config.Config{}
	applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)
	return &reconcileResult{prefill: cfg, target: writeDefault, store: store}
}

func sectionEmpty(s credstore.Section) bool {
	return s.URL == "" && s.Email == "" && s.APIToken == "" && s.AuthMethod == "" && s.CloudID == ""
}

func configFromSection(s credstore.Section) *config.Config {
	cfg := &config.Config{
		URL:        s.URL,
		Email:      s.Email,
		APIToken:   s.APIToken,
		AuthMethod: s.AuthMethod,
		CloudID:    s.CloudID,
	}
	return cfg
}

func configFromLegacy(l *credstore.LegacyCreds) *config.Config {
	cfg := configFromSection(l.Section())
	if l.DefaultProject != "" {
		cfg.DefaultProject = l.DefaultProject
	}
	return cfg
}

func copyJTKDefaults(cfg *config.Config, t credstore.ToolSection) {
	if t.DefaultProject != "" {
		cfg.DefaultProject = t.DefaultProject
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
	case writeJTKOverride:
		store.JTK.Section = cred
	}
	if cfg.DefaultProject != "" {
		store.JTK.DefaultProject = cfg.DefaultProject
	}
}
