package config

import (
	"os"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/keyring"
)

// ValuesWithSources holds all config values with their source information.
// This is a projection helper that inspects env vars and config file to determine
// where each value came from. Used by commands to pass resolved values to presenters.
type ValuesWithSources struct {
	URL         string
	URLSource   string
	Email       string
	EmailSource string
	// The API token VALUE is never projected — the keyring is the source
	// of truth and §1.12 forbids displaying it (or any prefix/suffix).
	// Only presence + source + non-secret keyring metadata are shown.
	TokenConfigured   bool
	TokenSource       string
	KeyringRef        string
	KeyringBackend    string
	KeyringPassphrase string // file backend only; "" otherwise
	DefaultProject    string
	ProjectSource     string
	AuthMethod        string
	AuthMethodSrc     string
	CloudID           string
	CloudIDSrc        string
	Path              string
}

// GetValuesWithSources returns all config values with their source information.
func GetValuesWithSources() ValuesWithSources {
	url, urlSrc := GetURLWithSource()
	email, emailSrc := GetEmailWithSource()
	project, projectSrc := GetDefaultProjectWithSource()
	authMethod, authMethodSrc := GetAuthMethodWithSource()
	cloudID, cloudIDSrc := GetCloudIDWithSource()

	// Non-migrating: `config show` is diagnostic and must stay usable
	// even during an unresolved §1.8 conflict. A keyring error is folded
	// into a clear source label rather than crashing show.
	kr, err := keyring.InspectForTool(credstore.ToolJTK)
	if err != nil {
		kr.TokenSource = "keyring error: " + err.Error()
	}

	return ValuesWithSources{
		URL:               url,
		URLSource:         urlSrc,
		Email:             email,
		EmailSource:       emailSrc,
		TokenConfigured:   kr.TokenConfigured,
		TokenSource:       kr.TokenSource,
		KeyringRef:        kr.Ref,
		KeyringBackend:    keyringBackendLabel(kr),
		KeyringPassphrase: kr.PassphraseSource,
		DefaultProject:    project,
		ProjectSource:     projectSrc,
		AuthMethod:        authMethod,
		AuthMethodSrc:     authMethodSrc,
		CloudID:           cloudID,
		CloudIDSrc:        cloudIDSrc,
		Path:              Path(),
	}
}

// keyringBackendLabel renders the backend and how it was selected, e.g.
// "keychain (auto)". Empty when the keyring could not be opened.
func keyringBackendLabel(kr keyring.Info) string {
	if kr.Backend == "" {
		return ""
	}
	if kr.BackendSource == "" {
		return kr.Backend
	}
	return kr.Backend + " (" + kr.BackendSource + ")"
}

// GetURLWithSource returns the URL and its source.
// Precedence: JIRA_URL → ATLASSIAN_URL → config url → JIRA_DOMAIN (legacy) → config domain (legacy)
func GetURLWithSource() (value, source string) {
	if os.Getenv("JIRA_URL") != "" {
		return GetURL(), "env (JIRA_URL)"
	}
	if os.Getenv("ATLASSIAN_URL") != "" {
		return GetURL(), "env (ATLASSIAN_URL)"
	}
	cfg, err := Load()
	if err != nil {
		return "", "-"
	}
	if cfg.URL != "" {
		return GetURL(), "config"
	}
	// Check legacy domain sources
	if os.Getenv("JIRA_DOMAIN") != "" {
		return GetURL(), "env (JIRA_DOMAIN, deprecated)"
	}
	if cfg.Domain != "" {
		return GetURL(), "config (domain, deprecated)"
	}
	return "", "-"
}

// GetEmailWithSource returns the email and its source.
// Precedence: JIRA_EMAIL → ATLASSIAN_EMAIL → config email
func GetEmailWithSource() (value, source string) {
	if os.Getenv("JIRA_EMAIL") != "" {
		return GetEmail(), "env (JIRA_EMAIL)"
	}
	if os.Getenv("ATLASSIAN_EMAIL") != "" {
		return GetEmail(), "env (ATLASSIAN_EMAIL)"
	}
	cfg, err := Load()
	if err != nil {
		return "", "-"
	}
	if cfg.Email != "" {
		return cfg.Email, "config"
	}
	return "", "-"
}

// GetDefaultProjectWithSource returns the default project and its source.
// Precedence: JIRA_DEFAULT_PROJECT → config default_project
func GetDefaultProjectWithSource() (value, source string) {
	if os.Getenv("JIRA_DEFAULT_PROJECT") != "" {
		return GetDefaultProject(), "env (JIRA_DEFAULT_PROJECT)"
	}
	cfg, err := Load()
	if err != nil {
		return "", "-"
	}
	if cfg.DefaultProject != "" {
		return cfg.DefaultProject, "config"
	}
	return "", "-"
}
