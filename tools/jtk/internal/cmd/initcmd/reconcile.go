package initcmd

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/keyring"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/internal/config"
)

// hasUsableCreds composes the non-secret config completeness check
// (credstore) with keyring token presence. The token no longer lives in
// the shared store, so neither half alone means "already configured".
func hasUsableCreds(store *credstore.Store, tool string) (bool, error) {
	if !store.HasUsableConfig(tool) {
		return false, nil
	}
	return keyring.HasTokenForTool(tool)
}

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
	// unifyBoth is set by the explicit mismatch choice "use <tool>'s
	// credentials for both tools". It tells the save path to persist the
	// chosen token as the shared api_token AND clear BOTH per-tool
	// override keys, so neither tool stays shadowed on its old token.
	// unifySource is the tool whose token was chosen (credstore.ToolCFL /
	// ToolJTK); the command layer resolves THAT tool's keyring token.
	unifyBoth   bool
	unifySource string
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

	// Case 1: shared store + keyring already hold usable jtk creds.
	usable, err := hasUsableCreds(store, credstore.ToolJTK)
	if err != nil {
		return nil, err
	}
	if usable {
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
		store.JTK.DefaultProject = jtkLegacy.DefaultProject
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

// mismatchDescription is the prompt description for the both-legacies-
// mismatch reconciliation. Extracted so tests can assert the
// educational language without driving huh.
func mismatchDescription(jtkLegacy, cflLegacy *credstore.LegacyCreds) string {
	return fmt.Sprintf(
		"%s\n\n%s\n\nNote: Atlassian API tokens are account-wide. One token usually works for both Jira and Confluence.\nManage tokens: https://id.atlassian.com/manage-profile/security/api-tokens",
		credstore.FormatSection("jtk ("+jtkLegacy.Path+")", jtkLegacy.Section()),
		credstore.FormatSection("cfl ("+cflLegacy.Path+")", cflLegacy.Section()),
	)
}

func promptReconcileMismatch(jtkLegacy, cflLegacy *credstore.LegacyCreds) (string, error) {
	desc := mismatchDescription(jtkLegacy, cflLegacy)
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
	case "use_jtk", "use_cfl":
		// User chose to unify on one tool's creds. Clear both
		// override Sections so each tool resolves to the new default
		// rather than a leftover override from a prior keep_different
		// run. Per-tool defaults (default_space, default_project,
		// output_format) are preserved.
		store.JTK.Section = credstore.Section{}
		store.CFL.Section = credstore.Section{}
		var cfg *config.Config
		chosenTool := credstore.ToolJTK
		if choice == "use_jtk" {
			cfg = configFromLegacy(jtkLegacy)
		} else {
			cfg = configFromLegacy(cflLegacy)
			cfg.DefaultProject = jtkLegacy.DefaultProject
			chosenTool = credstore.ToolCFL
		}
		applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)
		// Signal the cross-tool unify; the command layer (initcmd.go) does
		// the keyring resolve+persist. reconcile.go stays keyring-free so
		// it remains unit-testable without OS-keychain isolation.
		return &reconcileResult{prefill: cfg, target: writeDefault, store: store, consumedLegacies: consumed, unifyBoth: true, unifySource: chosenTool}
	case "keep_different":
		// Both tools land in their override sections so the split is
		// stable: store.Default stays empty, jtk reads its override,
		// cfl reads its override. Save target is writeJTKOverride so
		// post-form edits stay in the jtk section. Per-tool defaults
		// are set here so the function is self-contained — callers
		// don't have to pre-populate the store.
		store.JTK.Section = jtkLegacy.Section()
		store.JTK.DefaultProject = jtkLegacy.DefaultProject
		store.CFL.Section = cflLegacy.Section()
		store.CFL.DefaultSpace = cflLegacy.DefaultSpace
		store.CFL.OutputFormat = cflLegacy.OutputFormat
		cfg := configFromLegacy(jtkLegacy)
		applyFlagOverrides(cfg, prefillURL, prefillEmail, prefillToken, prefillAuthMethod, prefillCloudID)
		v.Info("Keeping per-tool credentials. jtk will use jtk's token; cfl will use cfl's token.")
		return &reconcileResult{prefill: cfg, target: writeJTKOverride, store: store, consumedLegacies: consumed}
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
