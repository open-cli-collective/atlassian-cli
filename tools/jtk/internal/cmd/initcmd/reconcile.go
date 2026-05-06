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
		var target writeTarget
		if hasOverride {
			target = writeJTKOverride
		} else {
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
				target = writeDefault
				v.Info("Note: edits will affect both jtk and cfl (writing to shared default).")
			} else {
				target = writeJTKOverride
			}
		}
		cfg := configFromSection(store.Resolve(credstore.ToolJTK))
		copyJTKDefaults(cfg, store.JTK)
		applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)
		return &reconcileResult{prefill: cfg, target: target, store: store}, nil
	}

	// Case 2: only this tool's legacy.
	if jtkLegacy != nil && cflLegacy == nil {
		cfg := configFromLegacy(jtkLegacy)
		applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)
		v.Info("Migrating existing jtk config at %s to shared store.", jtkLegacy.Path)
		return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: []string{jtkLegacy.Path}}, nil
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
		return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: consumed}, nil
	}

	// Case 4: both legacies exist.
	if jtkLegacy != nil && cflLegacy != nil {
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
		switch choice {
		case "use_jtk":
			cfg := configFromLegacy(jtkLegacy)
			applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)
			return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: []string{jtkLegacy.Path, cflLegacy.Path}}, nil
		case "use_cfl":
			cfg := configFromLegacy(cflLegacy)
			applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)
			return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: []string{jtkLegacy.Path, cflLegacy.Path}}, nil
		case "keep_different":
			store.Default = jtkLegacy.Section()
			store.JTK.DefaultProject = jtkLegacy.DefaultProject
			store.CFL.Section = cflLegacy.Section()
			store.CFL.DefaultSpace = cflLegacy.DefaultSpace
			store.CFL.OutputFormat = cflLegacy.OutputFormat
			cfg := configFromLegacy(jtkLegacy)
			applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)
			v.Info("Keeping per-tool credentials. jtk will use jtk's token; cfl will use cfl's token.")
			return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: []string{jtkLegacy.Path, cflLegacy.Path}}, nil
		}
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
	store.JTK.DefaultProject = cfg.DefaultProject
}
